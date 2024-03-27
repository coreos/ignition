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
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/coreos/ignition/v2/config/v3_5_experimental/types"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

func init() {
	platform.Register(platform.Provider{
		Name:  "akamai",
		Fetch: fetchConfig,
		Init:  initFetcher,
	})
}

var (
	tokenURL    = url.URL{Scheme: "http", Host: "169.254.169.254", Path: "/v1/token"}
	userdataURL = url.URL{Scheme: "http", Host: "169.254.169.254", Path: "/v1/user-data"}
)

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	if f.AkamaiMetadataToken == "" {
		return types.Config{}, report.Report{}, errors.New("akamai metadata token not set")
	}

	encoded, err := f.FetchToBuffer(userdataURL, resource.FetchOptions{
		Headers: http.Header{
			"Metadata-Token": []string{f.AkamaiMetadataToken},
		},
	})
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	// The Linode Metadata Service requires userdata to be base64-encoded
	// when it is uploaded.
	data := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	if _, err := base64.StdEncoding.Decode(data, encoded); err != nil {
		return types.Config{}, report.Report{}, fmt.Errorf("decode base64: %w", err)
	}

	return util.ParseConfig(f.Logger, data)
}

func initFetcher(f *resource.Fetcher) error {
	token, err := f.FetchToBuffer(tokenURL, resource.FetchOptions{
		HTTPVerb: http.MethodPut,
		Headers: http.Header{
			"Metadata-Token-Expiry-Seconds": []string{"3600"},
		},
	})

	// NOTE: ErrNotFound could mean the instance is running in a region
	// where the Metadata Service has not been deployed.
	if err != nil {
		return fmt.Errorf("generate metadata api token: %w", err)
	}

	f.AkamaiMetadataToken = string(token)

	return nil
}
