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

// The vultr provider fetches a remote configuration from the vultr
// user-data metadata service URL.
// https://web.archive.org/web/20190513194756/https://www.vultr.com/metadata/#user

package vultr

import (
	"net/url"

	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var (
	userdataUrl = url.URL{
		Scheme: "http",
		Host:   "169.254.169.254",
		Path:   "user-data/user-data",
	}
)

func init() {
	platform.Register(platform.Provider{
		Name:  "vultr",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	data, err := f.FetchToBuffer(userdataUrl, resource.FetchOptions{})
	if err != nil && err != resource.ErrNotFound {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}
