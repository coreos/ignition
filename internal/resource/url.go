// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	configErrors "github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/util"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pin/tftp"
	"github.com/vincent-petithory/dataurl"
)

var (
	ErrSchemeUnsupported      = errors.New("unsupported source scheme")
	ErrPathNotAbsolute        = errors.New("path is not absolute")
	ErrNotFound               = errors.New("resource not found")
	ErrFailed                 = errors.New("failed to fetch resource")
	ErrCompressionUnsupported = errors.New("compression is not supported with that scheme")
	ErrNeedNet                = errors.New("resource requires networking")

	// ConfigHeaders are the HTTP headers that should be used when the Ignition
	// config is being fetched
	configHeaders = http.Header{
		"Accept-Encoding": []string{"identity"},
		"Accept":          []string{"application/vnd.coreos.ignition+json;version=3.2.0, */*;q=0.1"},
	}
)

// Fetcher holds settings for fetching resources from URLs
type Fetcher struct {
	// The logger object to use when logging information.
	Logger *log.Logger

	// client is the http client that will be used when fetching http(s)
	// resources. If left nil, one will be created and used, but this means any
	// timeouts Ignition was configured to used will be ignored.
	client *HttpClient

	// The AWS Session to use when fetching resources from S3. If left nil, the
	// first S3 object that is fetched will initialize the field. This can be
	// used to set credentials.
	AWSSession *session.Session

	// The region where the AWS machine trying to fetch is.
	// This is used as a hint to fetch the S3 bucket from the right partition and region.
	S3RegionHint string

	// GCSSession is a client for interacting with Google Cloud Storage.
	// It is used when fetching resources from GCS.
	GCSSession *storage.Client

	// Whether to only attempt fetches which can be performed offline. This
	// currently only includes the "data" scheme. Other schemes will result in
	// ErrNeedNet. In the future, we can improve on this by dropping this
	// and just making sure that we canonicalize all "insufficient
	// network"-related errors to ErrNeedNet. That way, distro integrators
	// could distinguish between "partial" and full network bring-up.
	Offline bool
}

type FetchOptions struct {
	// Headers are the http headers that will be used when fetching http(s)
	// resources. They have no effect on other fetching schemes.
	Headers http.Header

	// Hash is the hash to use when calculating a fetched resource's hash. If
	// left as nil, no hash will be calculated.
	Hash hash.Hash

	// The expected sum to be produced by the given hasher. If the Hash field is
	// nil, this field is ignored.
	ExpectedSum []byte

	// Compression specifies the type of compression to use when decompressing
	// the fetched object. If left empty, no decompression will be used.
	Compression string
}

// FetchToBuffer will fetch the given url into a temporary file, and then read
// in the contents of the file and delete it. It will return the downloaded
// contents, or an error if one was encountered.
func (f *Fetcher) FetchToBuffer(u url.URL, opts FetchOptions) ([]byte, error) {
	if f.Offline && util.UrlNeedsNet(u) {
		return nil, ErrNeedNet
	}

	var err error
	dest := new(bytes.Buffer)
	switch u.Scheme {
	case "http", "https":
		err = f.fetchFromHTTP(u, dest, opts)
	case "tftp":
		err = f.fetchFromTFTP(u, dest, opts)
	case "data":
		err = f.fetchFromDataURL(u, dest, opts)
	case "s3":
		buf := &s3buf{
			WriteAtBuffer: aws.NewWriteAtBuffer([]byte{}),
		}
		err = f.fetchFromS3(u, buf, opts)
		return buf.Bytes(), err
	case "gs":
		err = f.fetchFromGCS(u, dest, opts)
	case "":
		return nil, nil
	default:
		return nil, ErrSchemeUnsupported
	}
	return dest.Bytes(), err
}

// s3buf is a wrapper around the aws.WriteAtBuffer that also allows reading and seeking.
// Read() and Seek() are only safe to call after the download call is made. This is only for
// use with fetchFromS3* functions.
type s3buf struct {
	*aws.WriteAtBuffer
	// only safe to call read/seek after finishing writing. Not safe for parallel use
	reader io.ReadSeeker
}

func (s *s3buf) Read(p []byte) (int, error) {
	if s.reader == nil {
		s.reader = bytes.NewReader(s.Bytes())
	}
	return s.reader.Read(p)
}

func (s *s3buf) Seek(offset int64, whence int) (int64, error) {
	if s.reader == nil {
		s.reader = bytes.NewReader(s.Bytes())
	}
	return s.reader.Seek(offset, whence)
}

// Fetch calls the appropriate FetchFrom* function based on the scheme of the
// given URL. The results will be decompressed if compression is set in opts,
// and written into dest. If opts.Hash is set the data stream will also be
// hashed and compared against opts.ExpectedSum, and any match failures will
// result in an error being returned.
//
// Fetch expects dest to be an empty file and for the cursor in the file to be
// at the beginning. Since some url schemes (ex: s3) use chunked downloads and
// fetch chunks out of order, Fetch's behavior when dest is not an empty file is
// undefined.
func (f *Fetcher) Fetch(u url.URL, dest *os.File, opts FetchOptions) error {
	if f.Offline && util.UrlNeedsNet(u) {
		return ErrNeedNet
	}

	switch u.Scheme {
	case "http", "https":
		return f.fetchFromHTTP(u, dest, opts)
	case "tftp":
		return f.fetchFromTFTP(u, dest, opts)
	case "data":
		return f.fetchFromDataURL(u, dest, opts)
	case "s3":
		return f.fetchFromS3(u, dest, opts)
	case "gs":
		return f.fetchFromGCS(u, dest, opts)
	case "":
		return nil
	default:
		return ErrSchemeUnsupported
	}
}

// FetchFromTFTP fetches a resource from u via TFTP into dest, returning an
// error if one is encountered.
func (f *Fetcher) fetchFromTFTP(u url.URL, dest io.Writer, opts FetchOptions) error {
	if !strings.ContainsRune(u.Host, ':') {
		u.Host = u.Host + ":69"
	}
	c, err := tftp.NewClient(u.Host)
	if err != nil {
		return err
	}
	wt, err := c.Receive(u.Path, "octet")
	if err != nil {
		return err
	}
	// The TFTP library takes an io.Writer to send data in to, but to decompress
	// the stream the gzip library wraps an io.Reader, so let's create a pipe to
	// connect these two things
	pReader, pWriter := io.Pipe()
	doneChan := make(chan error, 2)

	checkForDoneChanErr := func(err error) error {
		// If an error is encountered while decompressing or copying data out of
		// the pipe, there's probably an error from writing into the pipe that
		// will better describe what went wrong. This function does a
		// non-blocking read of doneChan, overriding the returned error val if
		// there's anything in doneChan.
		select {
		case writeErr := <-doneChan:
			if writeErr != nil {
				return writeErr
			}
			return err
		default:
			return err
		}
	}

	// A goroutine is used to handle writing the fetched data into the pipe
	// while also copying it out of the pipe concurrently
	go func() {
		_, err := wt.WriteTo(pWriter)
		doneChan <- err
		err = pWriter.Close()
		doneChan <- err
	}()
	err = f.decompressCopyHashAndVerify(dest, pReader, opts)
	if err != nil {
		return checkForDoneChanErr(err)
	}
	// receive the error from wt.WriteTo()
	err = <-doneChan
	if err != nil {
		return err
	}
	// receive the error from pWriter.Close()
	err = <-doneChan
	if err != nil {
		return err
	}
	return nil
}

// FetchFromHTTP fetches a resource from u via HTTP(S) into dest, returning an
// error if one is encountered.
func (f *Fetcher) fetchFromHTTP(u url.URL, dest io.Writer, opts FetchOptions) error {
	// for the case when "config is not valid"
	// this if necessary if not spawned through kola (e.g. Packet Dashboard)
	if f.client == nil {
		logger := log.New(true)
		f.Logger = &logger
		if err := f.newHttpClient(); err != nil {
			return err
		}
	}

	// We do not want to redirect HTTP headers
	f.client.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		req.Header = make(http.Header)
		return nil
	}

	// TODO use .Clone() when we have a new enough golang
	// (With Rust, we'd have immutability and wouldn't need to defensively clone)
	headers := make(http.Header)
	for k, va := range configHeaders {
		for _, v := range va {
			headers.Set(k, v)
		}
	}
	for k, va := range opts.Headers {
		for _, v := range va {
			headers.Set(k, v)
		}
	}

	dataReader, status, ctxCancel, err := f.client.getReaderWithHeader(u.String(), headers)
	if ctxCancel != nil {
		// whatever context getReaderWithHeader created for the request should
		// be cancelled once we're done reading the response
		defer ctxCancel()
	}
	if err != nil {
		return err
	}
	defer dataReader.Close()

	switch status {
	case http.StatusOK, http.StatusNoContent:
		break
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return ErrFailed
	}

	return f.decompressCopyHashAndVerify(dest, dataReader, opts)
}

// FetchFromDataURL writes the data stored in the dataurl u into dest, returning
// an error if one is encountered.
func (f *Fetcher) fetchFromDataURL(u url.URL, dest io.Writer, opts FetchOptions) error {
	url, err := dataurl.DecodeString(u.String())
	if err != nil {
		return err
	}

	return f.decompressCopyHashAndVerify(dest, bytes.NewBuffer(url.Data), opts)
}

// FetchFromGCS writes the data stored in a GCS bucket as described by u into dest, returning
// an error if one is encountered. It looks for the default credentials by querying metadata
// server on GCE. If it fails to get the credentials, then it will fall back to anonymous
// credentials to fetch the object content.
func (f *Fetcher) fetchFromGCS(u url.URL, dest io.Writer, opts FetchOptions) error {
	ctx := context.Background()
	var clientOption option.ClientOption
	if f.GCSSession == nil {
		if metadata.OnGCE() {
			id, _ := metadata.ProjectID()
			creds := &google.Credentials{
				ProjectID:   id,
				TokenSource: google.ComputeTokenSource("", storage.ScopeReadOnly),
			}
			clientOption = option.WithCredentials(creds)
		} else {
			f.Logger.Debug("falling back to unauthenticated GCS access")
			clientOption = option.WithoutAuthentication()
		}

		var err error
		f.GCSSession, err = storage.NewClient(ctx, clientOption)
		if err != nil {
			return err
		}
	}

	path := strings.TrimLeft(u.Path, "/")
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()
	rc, err := f.GCSSession.Bucket(u.Host).Object(path).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("error while reading content from (%q): %v", u.String(), err)
	}

	return f.decompressCopyHashAndVerify(dest, rc, opts)
}

type s3target interface {
	io.WriterAt
	io.ReadSeeker
}

// FetchFromS3 gets data from an S3 bucket as described by u and writes it into
// dest, returning an error if one is encountered. It will attempt to acquire
// IAM credentials from the EC2 metadata service, and if this fails will attempt
// to fetch the object with anonymous credentials.
func (f *Fetcher) fetchFromS3(u url.URL, dest s3target, opts FetchOptions) error {
	if opts.Compression != "" {
		return ErrCompressionUnsupported
	}
	ctx := context.Background()
	if f.client != nil && f.client.timeout != 0 {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, f.client.timeout)
		defer cancelFn()
	}

	if f.AWSSession == nil {
		var err error
		f.AWSSession, err = session.NewSession(&aws.Config{
			Credentials: credentials.AnonymousCredentials,
		})
		if err != nil {
			return err
		}
	}
	sess := f.AWSSession.Copy()

	// Determine the partition and region this bucket is in
	regionHint := "us-east-1"
	if f.S3RegionHint != "" {
		regionHint = f.S3RegionHint
	}
	region, err := s3manager.GetBucketRegion(ctx, sess, u.Host, regionHint)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return fmt.Errorf("couldn't determine the region for bucket %q: %v", u.Host, err)
		}
		return err
	}

	sess.Config.Region = aws.String(region)

	var versionId *string
	if v, ok := u.Query()["versionId"]; ok && len(v) > 0 {
		versionId = aws.String(v[0])
	}

	input := &s3.GetObjectInput{
		Bucket:    &u.Host,
		Key:       &u.Path,
		VersionId: versionId,
	}
	err = f.fetchFromS3WithCreds(ctx, dest, input, sess)
	if err != nil {
		return err
	}
	if opts.Hash != nil {
		opts.Hash.Reset()
		_, err = dest.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		_, err = io.Copy(opts.Hash, dest)
		if err != nil {
			return err
		}
		calculatedSum := opts.Hash.Sum(nil)
		if !bytes.Equal(calculatedSum, opts.ExpectedSum) {
			return util.ErrHashMismatch{
				Calculated: hex.EncodeToString(calculatedSum),
				Expected:   hex.EncodeToString(opts.ExpectedSum),
			}
		}
		f.Logger.Debug("file matches expected sum of: %s", hex.EncodeToString(opts.ExpectedSum))
	}
	return nil
}

func (f *Fetcher) fetchFromS3WithCreds(ctx context.Context, dest s3target, input *s3.GetObjectInput, sess *session.Session) error {
	httpClient, err := defaultHTTPClient()
	if err != nil {
		return err
	}

	awsConfig := aws.NewConfig().WithHTTPClient(httpClient)
	s3Client := s3.New(sess, awsConfig)
	downloader := s3manager.NewDownloaderWithClient(s3Client)
	if _, err := downloader.DownloadWithContext(ctx, dest, input); err != nil {
		if awserrval, ok := err.(awserr.Error); ok && awserrval.Code() == "EC2RoleRequestError" {
			// If this error was due to an EC2 role request error, try again
			// with the anonymous credentials.
			sess.Config.Credentials = credentials.AnonymousCredentials
			return f.fetchFromS3WithCreds(ctx, dest, input, sess)
		}
		return err
	}
	return nil
}

// uncompress will wrap the given io.Reader in a decompresser specified in the
// FetchOptions, and return an io.ReadCloser with the decompressed data stream.
func (f *Fetcher) uncompress(r io.Reader, opts FetchOptions) (io.ReadCloser, error) {
	switch opts.Compression {
	case "":
		return ioutil.NopCloser(r), nil
	case "gzip":
		return gzip.NewReader(r)
	default:
		return nil, configErrors.ErrCompressionInvalid
	}
}

// decompressCopyHashAndVerify will decompress src if necessary, copy src into
// dest until src returns an io.EOF while also calculating a hash if one is set,
// and will return an error if there's any problems with any of this or if the
// hash doesn't match the expected hash in the opts.
func (f *Fetcher) decompressCopyHashAndVerify(dest io.Writer, src io.Reader, opts FetchOptions) error {
	decompressor, err := f.uncompress(src, opts)
	if err != nil {
		return err
	}
	defer decompressor.Close()
	if opts.Hash != nil {
		opts.Hash.Reset()
		dest = io.MultiWriter(dest, opts.Hash)
	}
	_, err = io.Copy(dest, decompressor)
	if err != nil {
		return err
	}
	if opts.Hash != nil {
		calculatedSum := opts.Hash.Sum(nil)
		if !bytes.Equal(calculatedSum, opts.ExpectedSum) {
			return util.ErrHashMismatch{
				Calculated: hex.EncodeToString(calculatedSum),
				Expected:   hex.EncodeToString(opts.ExpectedSum),
			}
		}
		f.Logger.Debug("file matches expected sum of: %s", hex.EncodeToString(opts.ExpectedSum))
	}
	return nil
}
