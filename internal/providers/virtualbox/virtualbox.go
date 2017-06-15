// Copyright 2015 CoreOS, Inc.
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

// The virtualbox provider fetches the configuration from raw data on a partition
// with the GUID 99570a8a-f826-4eb0-ba4e-9dd72d55ea13

package virtualbox

import (
	"fmt"
	"io/ioutil"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/config/validate/report"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/providers/util"
	"github.com/coreos/ignition/internal/resource"
)

const (
	configPath = "/dev/disk/by-partuuid/99570a8a-f826-4eb0-ba4e-9dd72d55ea13"
)

func FetchConfig(logger *log.Logger, _ *resource.HttpClient) (types.Config, report.Report, error) {
	logger.Debug("Attempting to read config drive")
	rawConfig, err := ioutil.ReadFile(configPath)
	if err != nil {
		logger.Debug("Failed to read config drive, assuming no config")
		return types.Config{}, report.Report{}, config.ErrEmpty
	}
	nilLocation := -1
	for i := 0; i < len(rawConfig); i++ {
		// All configs must be nil terminated
		if rawConfig[i] == byte(0) {
			nilLocation = i
			break
		}
	}
	if nilLocation == -1 {
		logger.Debug("Nil terminator not found; invalid config")
		return types.Config{}, report.Report{}, fmt.Errorf("Invalid config (no nil terminator)")
	}
	trimmedConfig := rawConfig[:nilLocation]
	return util.ParseConfig(logger, trimmedConfig)
}
