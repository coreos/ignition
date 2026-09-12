package blobfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	azblobblob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/awsimdsfs"
	"github.com/hairyhenderson/go-fsimpl/internal"
	"github.com/hairyhenderson/go-fsimpl/internal/env"
	"gocloud.dev/blob"
	"gocloud.dev/blob/azureblob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"
	"gocloud.dev/gcp"
)

type blobFS struct {
	ctx     context.Context
	base    *url.URL
	hclient *http.Client
	bucket  *blob.Bucket
	envfs   fs.FS
	imdsfs  fs.FS
	root    string
}

// Some blob APIs don't return valid modTimes, and some do. To conform to fstest
// set this to a fake value
//
//nolint:gochecknoglobals
var fakeModTime *time.Time

// New provides a filesystem (an fs.FS) backed by an blob storage bucket,
// rooted at the given URL.
//
// A context can be given by using WithContextFS.
func New(u *url.URL) (fs.FS, error) {
	switch u.Scheme {
	case s3blob.Scheme, gcsblob.Scheme, azureblob.Scheme:
	default:
		return nil, fmt.Errorf("invalid URL scheme %q", u.Scheme)
	}

	root := strings.TrimPrefix(u.Path, "/")

	iu, _ := url.Parse("aws+imds:")

	imdsfs, err := awsimdsfs.New(iu)
	if err != nil {
		return nil, fmt.Errorf("couldn't create IMDS filesystem: %w", err)
	}

	return &blobFS{
		ctx:     context.Background(),
		base:    u,
		hclient: http.DefaultClient,
		root:    root,
		envfs:   os.DirFS("/"),
		imdsfs:  imdsfs,
	}, nil
}

// FS is used to register this filesystem with an fsimpl.FSMux
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, s3blob.Scheme, gcsblob.Scheme, azureblob.Scheme)

var (
	_ fs.FS                     = (*blobFS)(nil)
	_ fs.ReadFileFS             = (*blobFS)(nil)
	_ fs.SubFS                  = (*blobFS)(nil)
	_ internal.WithContexter    = (*blobFS)(nil)
	_ internal.WithHTTPClienter = (*blobFS)(nil)
	_ internal.WithIMDSFSer     = (*blobFS)(nil)
)

func (f blobFS) URL() string {
	return f.base.String()
}

func (f *blobFS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *blobFS) WithHTTPClient(client *http.Client) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.hclient = client

	return &fsys
}

func (f *blobFS) WithIMDSFS(imdsfs fs.FS) fs.FS {
	if imdsfs == nil {
		return f
	}

	fsys := *f
	fsys.imdsfs = imdsfs

	return &fsys
}

func (f *blobFS) openBucket() (*blob.Bucket, error) {
	o, err := f.newOpener(f.ctx, f.base.Scheme)
	if err != nil {
		return nil, fmt.Errorf("bucket opener: %w", err)
	}

	u := f.cleanCdkURL(*f.base)

	bucket, err := o.OpenBucketURL(f.ctx, &u)
	if err != nil {
		return nil, fmt.Errorf("open bucket: %w", err)
	}

	return bucket, nil
}

func (f *blobFS) Sub(name string) (fs.FS, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "sub", Path: name, Err: fs.ErrInvalid}
	}

	if name == "." || name == "" {
		return f, nil
	}

	fsys := *f
	fsys.root = path.Join(fsys.root, name)

	return &fsys, nil
}

func (f *blobFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	if f.bucket == nil {
		bucket, err := f.openBucket()
		if err != nil {
			return nil, fmt.Errorf("open: %w", err)
		}

		f.bucket = bucket
	}

	file := &blobFile{
		ctx:       f.ctx,
		name:      strings.TrimPrefix(path.Base(name), "."),
		bucket:    f.bucket,
		root:      strings.TrimPrefix(path.Join(f.root, path.Dir(name)), "."),
		pageToken: blob.FirstPageToken,
	}

	if name == "." {
		mt := time.Time{}
		if fakeModTime != nil {
			mt = *fakeModTime
		}

		file.fi = internal.DirInfo(file.name, mt)

		return file, nil
	}

	_, err := file.Stat()
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}

	return file, nil
}

func (f *blobFS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: fs.ErrInvalid}
	}

	if f.bucket == nil {
		bucket, err := f.openBucket()
		if err != nil {
			return nil, fmt.Errorf("readFile: %w", err)
		}

		f.bucket = bucket
	}

	return f.bucket.ReadAll(f.ctx, path.Join(f.root, name))
}

// create the correct kind of blob.BucketURLOpener for the given scheme
func (f *blobFS) newOpener(ctx context.Context, scheme string) (opener blob.BucketURLOpener, err error) {
	switch scheme {
	case s3blob.Scheme:
		// see https://gocloud.dev/concepts/urls/#muxes
		return &s3v2URLOpener{imdsfs: f.imdsfs}, nil
	case gcsblob.Scheme:
		if env.GetenvFS(f.envfs, "GOOGLE_ANON") == "true" {
			return &gcsblob.URLOpener{
				Client: gcp.NewAnonymousHTTPClient(f.hclient.Transport),
			}, nil
		}

		creds, err := gcp.DefaultCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve GCP credentials: %w", err)
		}

		client, err := gcp.NewHTTPClient(
			f.hclient.Transport,
			gcp.CredentialsTokenSource(creds))
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP HTTP client: %w", err)
		}

		return &gcsblob.URLOpener{Client: client}, nil
	case azureblob.Scheme:
		return blob.DefaultURLMux(), nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}
}

// copy/sanitize the URL for the Go CDK - it doesn't like params it can't parse
func (f *blobFS) cleanCdkURL(u url.URL) url.URL {
	switch u.Scheme {
	case s3blob.Scheme:
		return f.cleanS3URL(u)
	case gcsblob.Scheme:
		return f.cleanGSURL(u)
	case azureblob.Scheme:
		return f.cleanAzBlobURL(u)
	default:
		return u
	}
}

func (f *blobFS) cleanAzBlobURL(u url.URL) url.URL {
	q := u.Query()
	for param := range q {
		switch param {
		case "domain":
		default:
			q.Del(param)
		}
	}

	u.RawQuery = q.Encode()

	return u
}

func (f *blobFS) cleanGSURL(u url.URL) url.URL {
	q := u.Query()
	for param := range q {
		switch param {
		case "access_id", "private_key_path":
		default:
			q.Del(param)
		}
	}

	u.RawQuery = q.Encode()

	return u
}

type blobFile struct {
	ctx       context.Context
	reader    *blob.Reader
	bucket    *blob.Bucket
	fi        fs.FileInfo
	listIter  *blob.ListIterator
	name      string
	root      string
	pageToken []byte
}

var _ fs.ReadDirFile = (*blobFile)(nil)

func (f *blobFile) Close() error {
	if f.reader == nil {
		return nil
	}

	return f.reader.Close()
}

func (f *blobFile) Read(p []byte) (int, error) {
	if f.reader == nil {
		r, err := f.bucket.NewReader(f.ctx, path.Join(f.root, f.name), nil)
		if err != nil {
			return 0, &fs.PathError{Op: "read", Path: f.name, Err: err}
		}

		f.reader = r
	}

	return f.reader.Read(p)
}

func (f *blobFile) Stat() (fs.FileInfo, error) {
	if f.fi != nil {
		return f.fi, nil
	}

	out, err := f.bucket.Attributes(f.ctx, path.Join(f.root, f.name))
	if gcerrors.Code(err) == gcerrors.NotFound {
		return blobFindDir(f.ctx, f.bucket, f.root, f.name)
	}

	if err != nil {
		return nil, err
	}

	mode := fs.FileMode(0o444)

	azResp := azblobblob.GetPropertiesResponse{}
	if out.As(&azResp) && *azResp.ContentType == "" {
		// this is likely a directory
		mode |= fs.ModeDir | 0o111
	}

	if fakeModTime != nil {
		out.ModTime = *fakeModTime
	}

	f.fi = internal.FileInfo(f.name, out.Size, mode, out.ModTime, out.ContentType)

	return f.fi, nil
}

func blobFindDir(ctx context.Context, bucket *blob.Bucket, root, name string) (fs.FileInfo, error) {
	// Prefix is not suffixed with /, so that we find only a single entry, if a
	// dir with that name exists
	opts := blob.ListOptions{Delimiter: "/", Prefix: path.Join(root, name)}

	list, _, err := bucket.ListPage(ctx, blob.FirstPageToken, 1, &opts)
	if err != nil {
		return nil, err
	}

	if len(list) != 1 {
		return nil, fs.ErrNotExist
	}

	dir := list[0]

	if !dir.IsDir {
		return nil, fmt.Errorf("not a directory: %s", name)
	}

	mt := time.Time{}
	if fakeModTime != nil {
		mt = *fakeModTime
	}

	fi := internal.DirInfo(path.Base(name), mt)

	return fi, nil
}

//nolint:gocyclo
func (f *blobFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.listIter == nil {
		opts := blob.ListOptions{Delimiter: "/", Prefix: path.Join(f.root, f.name)}
		if opts.Prefix != "" {
			opts.Prefix += "/"
		}

		f.listIter = f.bucket.List(&opts)
	}

	dirents := []fs.DirEntry{}

	for i := 0; (n > 0 && i < n) || n <= 0; i++ {
		obj, err := f.listIter.Next(f.ctx)
		if errors.Is(err, io.EOF) {
			if n <= 0 {
				err = nil
			}

			return dirents, err
		}

		if err != nil {
			return nil, err
		}

		mode := fs.FileMode(0o444)
		if obj.IsDir {
			mode |= fs.ModeDir | 0o111
		}

		// container.BlobItem for objects, container.BlobPrefix for "directories"
		azItem := container.BlobItem{}
		if obj.As(&azItem) && (azItem.Properties.ContentType == nil || *azItem.Properties.ContentType == "") {
			// this is likely a directory, so ignore it
			continue
		}

		name := strings.TrimSuffix(path.Base(obj.Key), "/")

		if fakeModTime != nil {
			obj.ModTime = *fakeModTime
		}

		fi := internal.FileInfo(name, obj.Size, mode, obj.ModTime, "")
		dirents = append(dirents, internal.FileInfoDirEntry(fi))
	}

	return dirents, nil
}
