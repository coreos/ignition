// Copyright 2019 Red Hat, Inc.
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

// The IONOSCloud provider fetches configurations from the userdata available in
// the injected file /var/lib/cloud/seed/nocloud/user-data.
// NOTE: This provider is still EXPERIMENTAL.

package ionoscloud

import (
	"bytes"
	"os"

	"github.com/coreos/ignition/v2/config/v3_5_experimental/types"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

const (
	defaultFilename = "/var/lib/cloud/seed/nocloud/user-data"
)

func init() {
	platform.Register(platform.Provider{
		Name:  "ionoscloud",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	f.Logger.Info("using config file at %q", defaultFilename)

	rawConfig, err := os.ReadFile(defaultFilename)
	if err != nil {
		f.Logger.Err("couldn't read config %q: %v", defaultFilename, err)
		return types.Config{}, report.Report{}, err
	}

	header := []byte("#cloud-config\n")
	if bytes.HasPrefix(rawConfig, header) {
		f.Logger.Debug("config drive (%q) contains a cloud-config configuration, ignoring", defaultFilename)
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, rawConfig)
}
