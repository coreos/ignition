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

// The Scaleway provider fetches a remote configuration from the Scaleway
// user-data metadata service URL.
// NOTE: For security reason, Scaleway requires to query user data with a source port below 1024.

package scaleway

import (
	"math/rand"
	"net/url"

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var (
	userdataURL = url.URL{
		Scheme: "http",
		Host:   "169.254.42.42",
		Path:   "user_data/cloud-init",
	}
)

func init() {
	platform.Register(platform.Provider{
		Name:  "scaleway",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	// For security reason, Scaleway requires to query user data with a source port below 1024.
	port := func() int {
		return rand.Intn(1022) + 1
	}

	data, err := f.FetchToBuffer(userdataURL, resource.FetchOptions{
		LocalPort: port,
	})
	if err != nil && err != resource.ErrNotFound {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}
