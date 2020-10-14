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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	ignerrors "github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/earlyrand"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/util"
	"github.com/coreos/ignition/v2/internal/version"

	"github.com/vincent-petithory/dataurl"

	"golang.org/x/net/http/httpproxy"
)

const (
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 5 * time.Second

	defaultHttpResponseHeaderTimeout = 10
	defaultHttpTotalTimeout          = 0
)

var (
	ErrTimeout         = errors.New("unable to fetch resource in time")
	ErrPEMDecodeFailed = errors.New("unable to decode PEM block")
)

// HttpClient is a simple wrapper around the Go HTTP client that standardizes
// the process and logging of fetching payloads.
type HttpClient struct {
	client  *http.Client
	logger  *log.Logger
	timeout time.Duration

	transport *http.Transport
	cas       map[string][]byte
}

func (f *Fetcher) UpdateHttpTimeoutsAndCAs(timeouts types.Timeouts, cas []types.Resource, proxy types.Proxy) error {
	if f.client == nil {
		if err := f.newHttpClient(); err != nil {
			return err
		}
	}

	// Update timeouts
	responseHeader := defaultHttpResponseHeaderTimeout
	total := defaultHttpTotalTimeout
	if timeouts.HTTPResponseHeaders != nil {
		responseHeader = *timeouts.HTTPResponseHeaders
	}
	if timeouts.HTTPTotal != nil {
		total = *timeouts.HTTPTotal
	}

	f.client.client.Timeout = time.Duration(total) * time.Second
	f.client.timeout = f.client.client.Timeout

	f.client.transport.ResponseHeaderTimeout = time.Duration(responseHeader) * time.Second
	f.client.client.Transport = f.client.transport

	// Update proxy
	f.client.transport.Proxy = func(req *http.Request) (*url.URL, error) {
		return proxyFuncFromIgnitionConfig(proxy)(req.URL)
	}
	f.client.client.Transport = f.client.transport

	// Update CAs
	if len(cas) == 0 {
		return nil
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		f.Logger.Err("Unable to read system certificate pool: %s", err)
		return err
	}

	for _, ca := range cas {
		cablob, err := f.getCABlob(ca)
		if err != nil {
			return err
		}
		if err := f.parseCABundle(cablob, ca, pool); err != nil {
			f.Logger.Err("Unable to parse CA bundle: %s", err)
			return err
		}
	}
	f.client.transport.TLSClientConfig = &tls.Config{RootCAs: pool}
	return nil
}

// parseCABundle parses a CA bundle which includes multiple CAs.
func (f *Fetcher) parseCABundle(cablob []byte, ca types.Resource, pool *x509.CertPool) error {
	for len(cablob) > 0 {
		block, rest := pem.Decode(cablob)
		if block == nil {
			f.Logger.Err("Unable to decode CA (%v)", ca.Source)
			return ErrPEMDecodeFailed
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			f.Logger.Err("Unable to parse CA (%v): %s", ca.Source, err)
			return err
		}
		f.Logger.Info("Adding %q to list of CAs", cert.Subject.CommonName)
		pool.AddCert(cert)
		cablob = rest
	}
	return nil
}

func (f *Fetcher) getCABlob(ca types.Resource) ([]byte, error) {
	// this is also already checked at validation time
	if ca.Source == nil {
		f.Logger.Crit("invalid CA: %v", ignerrors.ErrSourceRequired)
		return nil, ignerrors.ErrSourceRequired
	}
	if blob, ok := f.client.cas[*ca.Source]; ok {
		return blob, nil
	}
	u, err := url.Parse(*ca.Source)
	if err != nil {
		f.Logger.Crit("Unable to parse CA URL: %s", err)
		return nil, err
	}
	hasher, err := util.GetHasher(ca.Verification)
	if err != nil {
		f.Logger.Crit("Unable to get hasher: %s", err)
		return nil, err
	}

	var expectedSum []byte
	if hasher != nil {
		// explicitly ignoring the error here because the config should already
		// be validated by this point
		_, expectedSumString, _ := util.HashParts(ca.Verification)
		expectedSum, err = hex.DecodeString(expectedSumString)
		if err != nil {
			f.Logger.Crit("Error parsing verification string %q: %v", expectedSumString, err)
			return nil, err
		}
	}

	var headers http.Header
	if ca.HTTPHeaders != nil && len(ca.HTTPHeaders) > 0 {
		headers, err = ca.HTTPHeaders.Parse()
		if err != nil {
			return nil, err
		}
	}

	var compression string
	if ca.Compression != nil {
		compression = *ca.Compression
	}

	cablob, err := f.FetchToBuffer(*u, FetchOptions{
		Hash:        hasher,
		Headers:     headers,
		ExpectedSum: expectedSum,
		Compression: compression,
	})
	if err != nil {
		f.Logger.Err("Unable to fetch CA (%s): %s", u, err)
		return nil, err
	}
	f.client.cas[*ca.Source] = cablob
	return cablob, nil

}

// RewriteCAsWithDataUrls will modify the passed in slice of CA references to
// contain the actual CA file via a dataurl in their source field.
func (f *Fetcher) RewriteCAsWithDataUrls(cas []types.Resource) error {
	for i, ca := range cas {
		blob, err := f.getCABlob(ca)
		if err != nil {
			return err
		}

		// Clean HTTP headers
		cas[i].HTTPHeaders = nil
		// the rewrite wipes the compression
		cas[i].Compression = nil

		encoded := dataurl.EncodeBytes(blob)
		cas[i].Source = &encoded
	}
	return nil
}

// DefaultHTTPClient builds the default `http.client` for Ignition.
func defaultHTTPClient() (*http.Client, error) {
	urand, err := earlyrand.UrandomReader()
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{
		Rand: urand,
	}
	transport := http.Transport{
		ResponseHeaderTimeout: time.Duration(defaultHttpResponseHeaderTimeout) * time.Second,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			Resolver: &net.Resolver{
				PreferGo: true,
			},
		}).Dial,
		TLSClientConfig:     &tlsConfig,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := http.Client{
		Transport: &transport,
	}
	return &client, nil
}

// newHttpClient populates the fetcher with the default HTTP client.
func (f *Fetcher) newHttpClient() error {
	defaultClient, err := defaultHTTPClient()
	if err != nil {
		return err
	}

	f.client = &HttpClient{
		client:    defaultClient,
		logger:    f.Logger,
		timeout:   time.Duration(defaultHttpTotalTimeout) * time.Second,
		transport: defaultClient.Transport.(*http.Transport),
		cas:       make(map[string][]byte),
	}
	return nil
}

// getReaderWithHeader performs an HTTP GET on the provided URL with the
// provided request header and returns the response body Reader, HTTP status
// code, a cancel function for the result's context, and error (if any). By
// default, User-Agent is added to the header but this can be overridden.
func (c HttpClient) getReaderWithHeader(url string, header http.Header) (io.ReadCloser, int, context.CancelFunc, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, nil, err
	}

	req.Header.Set("User-Agent", "Ignition/"+version.Raw)

	for key, values := range header {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	if c.timeout != 0 {
		cancelFn()
		ctx, cancelFn = context.WithTimeout(context.Background(), c.timeout)
	}

	duration := initialBackoff
	for attempt := 1; ; attempt++ {
		c.logger.Info("GET %s: attempt #%d", url, attempt)
		resp, err := c.client.Do(req.WithContext(ctx))

		if err == nil {
			c.logger.Info("GET result: %s", http.StatusText(resp.StatusCode))
			if resp.StatusCode < 500 {
				return resp.Body, resp.StatusCode, cancelFn, nil
			}
			resp.Body.Close()
		} else {
			c.logger.Info("GET error: %v", err)
		}

		duration = duration * 2
		if duration > maxBackoff {
			duration = maxBackoff
		}

		// Wait before next attempt or exit if we timeout while waiting
		select {
		case <-time.After(duration):
		case <-ctx.Done():
			return nil, 0, cancelFn, ErrTimeout
		}
	}
}

func proxyFuncFromIgnitionConfig(proxy types.Proxy) func(*url.URL) (*url.URL, error) {
	noProxy := translateNoProxySliceToString(proxy.NoProxy)

	if proxy.HTTPProxy == nil {
		proxy.HTTPProxy = new(string)
	}

	if proxy.HTTPSProxy == nil {
		proxy.HTTPSProxy = new(string)
	}

	cfg := &httpproxy.Config{
		HTTPProxy:  *proxy.HTTPProxy,
		HTTPSProxy: *proxy.HTTPSProxy,
		NoProxy:    noProxy,
	}

	return cfg.ProxyFunc()
}

func translateNoProxySliceToString(items []types.NoProxyItem) string {
	newItems := make([]string, len(items))
	for i, o := range items {
		newItems[i] = string(o)
	}
	return strings.Join(newItems, ",")
}
