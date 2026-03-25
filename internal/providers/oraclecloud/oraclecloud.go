// Copyright 2020 Red Hat
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

// The oraclecloud provider fetches a remote configuration from Oracle's
// Instance Metadata Service v2.
// https://docs.oracle.com/en-us/iaas/Content/Compute/Tasks/gettingmetadata.htm

package oraclecloud

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var userdataUrl = url.URL{
	Scheme: "http",
	Host:   "169.254.169.254",
	Path:   "opc/v2/instance/metadata/user_data",
}

func init() {
	platform.Register(platform.Provider{
		Name:  "oraclecloud",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	data, err := f.FetchToBuffer(userdataUrl, resource.FetchOptions{
		Headers: http.Header{
			"Authorization": []string{"Bearer Oracle"},
		},
		RetryCodes: []int{http.StatusTooManyRequests},
	})
	if err != nil && err != resource.ErrNotFound {
		// Do not wrap these errors, ignition uses direct comparsion to distinguish them.
		return types.Config{}, report.Report{}, err
	}

	userdata := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	size, err := base64.StdEncoding.Decode(userdata, data)
	if err != nil {
		return types.Config{}, report.Report{}, fmt.Errorf("decoding userdata: %w", err)
	}
	userdata = userdata[:size]

	return util.ParseConfig(f.Logger, userdata)
}
