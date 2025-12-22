// Copyright 2020 Red Hat, Inc.
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
	"compress/gzip"
	"crypto/sha512"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/util"
)

func TestDataUrl(t *testing.T) {
	type in struct {
		url  string
		opts FetchOptions
	}
	type out struct {
		data []byte
		err  error
	}

	tests := []struct {
		in  in
		out out
	}{
		// data url, no compression
		{
			in: in{
				url: "data:,hello%20world%0a",
				opts: FetchOptions{
					ExpectedSum: []byte("\xdb\x39\x74\xa9\x7f\x24\x07\xb7\xca\xe1\xae\x63\x7c\x00\x30\x68\x7a\x11\x91\x32\x74\xd5\x78\x49\x25\x58\xe3\x9c\x16\xc0\x17\xde\x84\xea\xcd\xc8\xc6\x2f\xe3\x4e\xe4\xe1\x2b\x4b\x14\x28\x81\x7f\x09\xb6\xa2\x76\x0c\x3f\x8a\x66\x4c\xea\xe9\x4d\x24\x34\xa5\x93"),
				},
			},
			out: out{data: []byte("hello world\n")},
		},
		// data url, bad hash
		{
			in: in{
				url: "data:,hello%20world%0a",
				opts: FetchOptions{
					ExpectedSum: []byte("\xdb\x39\x74\xa9\x7f\x24\x07\xb7\xca\xe1\xae\x63\x7c\x00\x30\x68\x7a\x11\x91\x32\x74\xd5\x78\x49\x25\x58\xe3\x9c\x16\xc0\x17\xde\x84\xea\xcd\xc8\xc6\x2f\xe3\x4e\xe4\xe1\x2b\x4b\x14\x28\x81\x7f\x09\xb6\xa2\x76\x0c\x3f\x8a\x66\x4c\xea\xe9\x4d\x24\x34\xa5\x00"),
				},
			},
			out: out{err: util.ErrHashMismatch{
				Calculated: "db3974a97f2407b7cae1ae637c0030687a11913274d578492558e39c16c017de84eacdc8c62fe34ee4e12b4b1428817f09b6a2760c3f8a664ceae94d2434a593",
				Expected:   "db3974a97f2407b7cae1ae637c0030687a11913274d578492558e39c16c017de84eacdc8c62fe34ee4e12b4b1428817f09b6a2760c3f8a664ceae94d2434a500",
			}},
		},
		// data url, gzipped
		{
			in: in{
				url: "data:,%1F%8B%08%08%90e%AB%5E%02%03z%00K%ADH%CC-%C8IUH%CB%CCI%E5%02%00tp%A6%CB%0D%00%00%00",
				opts: FetchOptions{
					Compression: "gzip",
					// digest of decompressed data
					ExpectedSum: []byte("\x80\x7e\x8f\xf9\x49\xe6\x1d\x23\xf5\xee\x42\xa6\x29\xec\x96\xe9\xfc\x52\x6b\x62\xf0\x30\xcd\x70\xba\x2c\xd5\xb9\xd9\x79\x35\x46\x1e\xac\xc2\x9b\xf5\x8b\xcd\x04\x26\xe9\xe1\xfd\xb0\xed\xa9\x39\x60\x3e\xd5\x2c\x9c\x06\xd0\x71\x22\x08\xa1\x5c\xd5\x82\xc6\x0e"),
				},
			},
			out: out{data: []byte("example file\n")},
		},
		// data url, gzipped, bad hash
		{
			in: in{
				url: "data:,%1F%8B%08%08%90e%AB%5E%02%03z%00K%ADH%CC-%C8IUH%CB%CCI%E5%02%00tp%A6%CB%0D%00%00%00",
				opts: FetchOptions{
					Compression: "gzip",
					ExpectedSum: []byte("\x80\x7e\x8f\xf9\x49\xe6\x1d\x23\xf5\xee\x42\xa6\x29\xec\x96\xe9\xfc\x52\x6b\x62\xf0\x30\xcd\x70\xba\x2c\xd5\xb9\xd9\x79\x35\x46\x1e\xac\xc2\x9b\xf5\x8b\xcd\x04\x26\xe9\xe1\xfd\xb0\xed\xa9\x39\x60\x3e\xd5\x2c\x9c\x06\xd0\x71\x22\x08\xa1\x5c\xd5\x82\xc6\x00"),
				},
			},
			out: out{err: util.ErrHashMismatch{
				Calculated: "807e8ff949e61d23f5ee42a629ec96e9fc526b62f030cd70ba2cd5b9d97935461eacc29bf58bcd0426e9e1fdb0eda939603ed52c9c06d0712208a15cd582c60e",
				Expected:   "807e8ff949e61d23f5ee42a629ec96e9fc526b62f030cd70ba2cd5b9d97935461eacc29bf58bcd0426e9e1fdb0eda939603ed52c9c06d0712208a15cd582c600",
			}},
		},
		// data url, invalid compressed data
		{
			in: in{
				url: "data:,hello%20world%0a",
				opts: FetchOptions{
					Compression: "gzip",
				},
			},
			out: out{err: gzip.ErrHeader},
		},
		// data url, bad compression type
		{
			in: in{
				url: "data:,hello%20world%0a",
				opts: FetchOptions{
					Compression: "xor",
				},
			},
			out: out{err: errors.ErrCompressionInvalid},
		},
	}

	logger := log.New(true)
	f := Fetcher{
		Logger: &logger,
	}

	for i, test := range tests {
		u, err := url.Parse(test.in.url)
		if err != nil {
			t.Errorf("#%d: parsing URL: %v", i, err)
			continue
		}
		test.in.opts.Hash = sha512.New()
		result, err := f.FetchToBuffer(*u, test.in.opts)
		if !reflect.DeepEqual(test.out.err, err) {
			t.Errorf("#%d: fetching URL: expected error %+v, got %+v", i, test.out.err, err)
			continue
		}
		if test.out.err == nil && !reflect.DeepEqual(test.out.data, result) {
			t.Errorf("#%d: expected output %+v, got %+v", i, test.out.data, result)
			continue
		}
	}
}

func TestFetchOffline(t *testing.T) {
	type in struct {
		url  string
		opts FetchOptions
	}
	type out struct {
		data []byte
		err  error
	}

	tests := []struct {
		in  in
		out out
	}{
		// data url, no compression
		{
			in: in{
				url: "data:,hello%20world%0a",
				opts: FetchOptions{
					ExpectedSum: []byte("\xdb\x39\x74\xa9\x7f\x24\x07\xb7\xca\xe1\xae\x63\x7c\x00\x30\x68\x7a\x11\x91\x32\x74\xd5\x78\x49\x25\x58\xe3\x9c\x16\xc0\x17\xde\x84\xea\xcd\xc8\xc6\x2f\xe3\x4e\xe4\xe1\x2b\x4b\x14\x28\x81\x7f\x09\xb6\xa2\x76\x0c\x3f\x8a\x66\x4c\xea\xe9\x4d\x24\x34\xa5\x93"),
				},
			},
			out: out{data: []byte("hello world\n")},
		},
		// empty
		{
			in: in{
				url: "",
			},
			out: out{data: nil},
		},
		// http url
		{
			in: in{
				url: "http://google.com",
			},
			out: out{err: ErrNeedNet},
		},
		// https url
		{
			in: in{
				url: "https://google.com",
			},
			out: out{err: ErrNeedNet},
		},
		// tftp url
		{
			in: in{
				url: "tftp://127.0.0.1/tftp",
			},
			out: out{err: ErrNeedNet},
		},
		// s3 url
		{
			in: in{
				url: "s3://kola-fixtures/resources/anonymous",
			},
			out: out{err: ErrNeedNet},
		},
		// arn url specifying bucket
		{
			in: in{
				url: "arn:aws:s3:::kola-fixtures/resources/anonymous",
			},
			out: out{err: ErrNeedNet},
		},
		// arn url specifying s3 access point
		{
			in: in{
				url: "arn:aws:s3:us-west-2:123456789012:accesspoint/test/object/name",
			},
			out: out{err: ErrNeedNet},
		},
		// gs url
		{
			in: in{
				url: "gs://foo/bar",
			},
			out: out{err: ErrNeedNet},
		},
	}

	logger := log.New(true)
	f := Fetcher{
		Logger:  &logger,
		Offline: true,
	}

	for i, test := range tests {
		u, err := url.Parse(test.in.url)
		if err != nil {
			t.Errorf("#%d: parsing URL: %v", i, err)
			continue
		}
		test.in.opts.Hash = sha512.New()
		result, err := f.FetchToBuffer(*u, test.in.opts)
		if !reflect.DeepEqual(test.out.err, err) {
			t.Errorf("#%d: fetching URL: expected error %+v, got %+v", i, test.out.err, err)
			continue
		}
		if test.out.err == nil && !reflect.DeepEqual(test.out.data, result) {
			t.Errorf("#%d: expected output %+v, got %+v", i, test.out.data, result)
			continue
		}
	}
}

func TestParseARN(t *testing.T) {
	tests := []struct {
		url        string
		bucket     string
		key        string
		region     string
		regionHint string
		err        error
	}{
		{
			url: "arn:aws:iam:us-west-2:123456789012:resource",
			err: errors.ErrInvalidS3ARN,
		},
		{
			url: "arn:aws:s3:us-east-1:123456789012:accesspoint/test/object",
			err: errors.ErrInvalidS3ARN,
		},
		{
			url: "arn:aws:s3:us-east-1:123456789012:accesspoint/test/name",
			err: errors.ErrInvalidS3ARN,
		},
		{
			url:        "arn:aws:s3:::kola-fixtures/resources/anonymous",
			bucket:     "kola-fixtures",
			key:        "resources/anonymous",
			regionHint: "us-east-1",
		},
		{
			url:        "arn:aws-cn:s3:::kola-fixtures/resources/anonymous",
			bucket:     "kola-fixtures",
			key:        "resources/anonymous",
			regionHint: "cn-north-1",
		},
		{
			url:        "arn:aws-us-gov:s3:::kola-fixtures/resources/anonymous",
			bucket:     "kola-fixtures",
			key:        "resources/anonymous",
			regionHint: "us-gov-west-1",
		},
		{
			url:    "arn:invalid:s3:::kola-fixtures/resources/anonymous",
			bucket: "kola-fixtures",
			key:    "resources/anonymous",
		},
		{
			url:        "arn:aws:s3:us-west-2:123456789012:accesspoint/test/object/name",
			bucket:     "arn:aws:s3:us-west-2:123456789012:accesspoint/test",
			key:        "name",
			region:     "us-west-2",
			regionHint: "us-east-1",
		},
		{
			url:        "arn:aws-cn:s3:cn-northwest-1:123456789012:accesspoint/test/object/name",
			bucket:     "arn:aws-cn:s3:cn-northwest-1:123456789012:accesspoint/test",
			key:        "name",
			region:     "cn-northwest-1",
			regionHint: "cn-north-1",
		},
		{
			url:        "arn:aws-us-gov:s3:us-gov-east-1:123456789012:accesspoint/test/object/name",
			bucket:     "arn:aws-us-gov:s3:us-gov-east-1:123456789012:accesspoint/test",
			key:        "name",
			region:     "us-gov-east-1",
			regionHint: "us-gov-west-1",
		},
		{
			url:    "arn:invalid:s3:us-west-2:123456789012:accesspoint/test/object/name",
			bucket: "arn:invalid:s3:us-west-2:123456789012:accesspoint/test",
			key:    "name",
			region: "us-west-2",
		},
		{
			url:        "arn:aws:s3:us-west-2:123456789012:accesspoint/test/object/path/object",
			bucket:     "arn:aws:s3:us-west-2:123456789012:accesspoint/test",
			key:        "path/object",
			region:     "us-west-2",
			regionHint: "us-east-1",
		},
	}

	logger := log.New(true)
	f := Fetcher{
		Logger: &logger,
	}

	for i, test := range tests {
		bucket, key, region, regionHint, err := f.parseARN(test.url)
		assert.Equal(t, test.err, err, "#%d: bad err", i)
		assert.Equal(t, test.bucket, bucket, "#%d: bad bucket", i)
		assert.Equal(t, test.key, key, "#%d: bad key", i)
		assert.Equal(t, test.region, region, "#%d: bad region", i)
		assert.Equal(t, test.regionHint, regionHint, "#%d: bad region hint", i)
	}
}

func TestParseAzureStorageUrl(t *testing.T) {
	tests := []struct {
		url            url.URL
		storageAccount string
		container      string
		file           string
		err            error
	}{
		{
			url: url.URL{
				Scheme: "https",
				Host:   "example.blob.core.windows.net",
				Path:   "/my-container/file.ign",
			},
			storageAccount: "https://example.blob.core.windows.net/",
			container:      "my-container",
			file:           "file.ign",
			err:            nil,
		},
		{
			url: url.URL{
				Scheme: "https",
				Host:   "example.blob.core.windows.net",
				Path:   "/invalid-url",
			},
			storageAccount: "",
			container:      "",
			file:           "",
			err:            fmt.Errorf("invalid URL path, ensure url has a structure of /container/filename.ign: /invalid-url"),
		},
		{
			url: url.URL{
				Scheme: "https",
				Host:   "example.blob.core.windows.net",
				Path:   "/invalid-url/another-blob/myfile.ign",
			},
			storageAccount: "",
			container:      "",
			file:           "",
			err:            fmt.Errorf("invalid URL path, ensure url has a structure of /container/filename.ign: /invalid-url/another-blob/myfile.ign"),
		},
	}

	logger := log.New(true)
	f := Fetcher{
		Logger: &logger,
	}
	for i, test := range tests {
		storageAccount, container, file, err := f.parseAzureStorageUrl(test.url)
		assert.Equal(t, test.err, err, "#%d: bad err", i)
		assert.Equal(t, test.storageAccount, storageAccount, "#%d: bad storageAccount", i)
		assert.Equal(t, test.container, container, "#%d: bad container", i)
		assert.Equal(t, test.file, file, "#%d: bad file", i)
	}

}

func TestFetchConfigDualStack(t *testing.T) {
	logger := log.New(true)
	f := Fetcher{
		Logger: &logger,
	}

	tests := []struct {
		name         string
		userdataURLs map[string]url.URL
		fetchConfig  func(*Fetcher, url.URL) ([]byte, error)
		expectError  bool
		expectIPv    string
	}{
		{
			name: "IPv4 only success",
			userdataURLs: map[string]url.URL{
				IPv4: {
					Scheme: "data",
					Opaque: "," + url.PathEscape(`{"ignition":{"version":"3.5.0"},"storage":{}}`),
				},
			},
			fetchConfig: func(f *Fetcher, u url.URL) ([]byte, error) {
				return f.FetchToBuffer(u, FetchOptions{})
			},
			expectError: false,
			expectIPv:   IPv4,
		},
		{
			name: "IPv6 only success",
			userdataURLs: map[string]url.URL{
				IPv6: {
					Scheme: "data",
					Opaque: "," + url.PathEscape(`{"ignition":{"version":"3.5.0"},"storage":{}}`),
				},
			},
			fetchConfig: func(f *Fetcher, u url.URL) ([]byte, error) {
				return f.FetchToBuffer(u, FetchOptions{})
			},
			expectError: false,
			expectIPv:   IPv6,
		},
		{
			name: "both IPv4 and IPv6 success",
			userdataURLs: map[string]url.URL{
				IPv4: {
					Scheme: "data",
					Opaque: "," + url.PathEscape(`{"ignition":{"version":"3.5.0"},"storage":{}}`),
				},
				IPv6: {
					Scheme: "data",
					Opaque: "," + url.PathEscape(`{"ignition":{"version":"3.5.0"},"passwd":{"users":[{"name":"test"}]}}`),
				},
			},
			fetchConfig: func(f *Fetcher, u url.URL) ([]byte, error) {
				return f.FetchToBuffer(u, FetchOptions{})
			},
			expectError: false,
		},
		{
			name: "both IPv4 and IPv6 fail - use invalid JSON to avoid network retries",
			userdataURLs: map[string]url.URL{
				IPv4: {
					Scheme: "data",
					Opaque: ",invalid-json-content",
				},
				IPv6: {
					Scheme: "data",
					Opaque: ",also-invalid-json",
				},
			},
			fetchConfig: func(f *Fetcher, u url.URL) ([]byte, error) {
				return f.FetchToBuffer(u, FetchOptions{})
			},
			expectError: true,
		},

		{
			name: "IPv4 success with invalid JSON",
			userdataURLs: map[string]url.URL{
				IPv4: {
					Scheme: "data",
					Opaque: ",invalid-json",
				},
			},
			fetchConfig: func(f *Fetcher, u url.URL) ([]byte, error) {
				return f.FetchToBuffer(u, FetchOptions{})
			},
			expectError: true,
		},
		{
			name: "IPv4 fails, IPv6 succeeds",
			userdataURLs: map[string]url.URL{
				IPv4: {
					Scheme: "data",
					Opaque: ",invalid-json",
				},
				IPv6: {
					Scheme: "data",
					Opaque: "," + url.PathEscape(`{"ignition":{"version":"3.5.0"},"storage":{}}`),
				},
			},
			fetchConfig: func(f *Fetcher, u url.URL) ([]byte, error) {
				return f.FetchToBuffer(u, FetchOptions{})
			},
			expectError: false,
			expectIPv:   IPv6,
		},
		{
			name: "IPv4 returns empty data",
			userdataURLs: map[string]url.URL{
				IPv4: {
					Scheme: "data",
					Opaque: ",",
				},
			},
			fetchConfig: func(f *Fetcher, u url.URL) ([]byte, error) {
				// Return empty bytes to simulate empty metadata response
				return []byte{}, nil
			},
			expectError: true,
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, rpt, err := FetchConfigDualStack(&f, test.userdataURLs, test.fetchConfig)

			if test.expectError {
				if err == nil {
					t.Errorf("#%d: FetchConfigDualStack() expected error, got nil", i)
				}
				// For empty data case, verify it returns ErrEmpty
				if test.name == "IPv4 returns empty data" && err != errors.ErrEmpty {
					t.Errorf("#%d: FetchConfigDualStack() expected errors.ErrEmpty for empty data, got %v", i, err)
				}
				// Config should be empty on error
				if cfg.Ignition.Version != "" {
					t.Errorf("#%d: FetchConfigDualStack() should return empty config on error, got version %q", i, cfg.Ignition.Version)
				}
			} else {
				if err != nil {
					t.Errorf("#%d: FetchConfigDualStack() unexpected error: %v", i, err)
				}
				// Should have a valid config
				if cfg.Ignition.Version == "" {
					t.Errorf("#%d: FetchConfigDualStack() should return non-zero config for valid URLs", i)
				}
			}

			_ = rpt
		})
	}
}
