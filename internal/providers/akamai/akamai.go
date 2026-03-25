// Copyright 2024 CoreOS, Inc.
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

// Package akamai provides platform support for Akamai Connected Cloud
// (previously known as Linode).
package akamai

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

func init() {
	platform.Register(platform.Provider{
		Name:  "akamai",
		Fetch: fetchConfig,
	})
}

// HTTP headers.
const (
	// tokenTTLHeader is the name of the HTTP request header that must be
	// set when making requests to [tokenURL] or [tokenURL6].
	tokenTTLHeader = "Metadata-Token-Expiry-Seconds"

	// tokenHeader is the name of the HTTP request header that callers must
	// set when making requests to [userdataURL] or [userdataURL6].
	tokenHeader = "Metadata-Token"
)

var (
	// IPv4 URLs.
	tokenURL    = url.URL{Scheme: "http", Host: "169.254.169.254", Path: "/v1/token"}
	userdataURL = url.URL{Scheme: "http", Host: "169.254.169.254", Path: "/v1/user-data"}

	// IPv6 URLs (for reference).
	// tokenURL6    = url.URL{Scheme: "http", Host: "[fd00:a9fe:a9fe::1]", Path: "/v1/token"}
	// userdataURL6 = url.URL{Scheme: "http", Host: "[fd00:a9fe:a9fe::1]", Path: "/v1/user-data"}
)

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	if f.Offline {
		return types.Config{}, report.Report{}, resource.ErrNeedNet
	}

	token, err := getToken(f)
	if err != nil {
		return types.Config{}, report.Report{}, fmt.Errorf("get token: %w", err)
	}

	// NOTE: If we do not explicitly set the "Accept" header, it will be
	// set by FetchToBuffer to a value that the Linode Metadata Service
	// does not accept.
	encoded, err := f.FetchToBuffer(userdataURL, resource.FetchOptions{
		Headers: http.Header{
			"Accept":    []string{"*/*"},
			tokenHeader: []string{string(token)},
		},
	})
	if err != nil {
		return types.Config{}, report.Report{}, fmt.Errorf("fetch userdata: %w", err)
	}

	// The Linode Metadata Service requires userdata to be base64-encoded
	// when it is uploaded, so we will have to decode the response.
	data := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	n, err := base64.StdEncoding.Decode(data, encoded)
	if err != nil {
		return types.Config{}, report.Report{}, fmt.Errorf("decode base64: %w", err)
	}

	// The Linode Metadata Service can compress userdata.
	// We have to gunzip if needed.
	unzipData, err := util.TryUnzip(data[:n])
	if err != nil {
		return types.Config{}, report.Report{}, fmt.Errorf("unzip: %w", err)
	}

	return util.ParseConfig(f.Logger, unzipData)
}

// defaultTokenTTL is the time-to-live (TTL; in seconds) for an authorization
// token retrieved from the Metadata Service API's "PUT /v1/token" endpoint.
const defaultTokenTTL = "300"

// getToken retrieves an authorization token to use for subsequent requests to
// Linode's Metadata Service.
// The returned token must be provided in the [tokenHeader] request header.
func getToken(f *resource.Fetcher) (token string, err error) {
	// NOTE: This is using "text/plain" for content negotiation, just to
	// skip the need to decode a JSON response.
	// In the future, the accepted content type should probably become
	// "application/vnd.coreos.ignition+json", but that will require
	// support from Linode's Metadata Service API.
	p, err := f.FetchToBuffer(tokenURL, resource.FetchOptions{
		HTTPVerb: http.MethodPut,
		Headers: http.Header{
			"Accept":       []string{"text/plain"},
			tokenTTLHeader: []string{defaultTokenTTL},
		},
	})
	if err != nil {
		return "", fmt.Errorf("fetch to buffer: %w", err)
	}

	p = bytes.TrimSpace(p)
	if len(p) == 0 {
		return "", errors.New("received an empty token")
	}

	return string(p), nil
}
