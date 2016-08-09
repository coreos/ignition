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

// The packet provider fetches a remote configuration from the packet.net
// userdata metadata service URL.

package packet

import (
	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/config/validate/report"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/util"

	"github.com/packethost/packngo/metadata"
)

func FetchConfig(logger *log.Logger, _ *util.HttpClient) (types.Config, report.Report, error) {
	logger.Debug("fetching config from packet metadata")
	data, err := metadata.GetUserData()
	if err != nil {
		logger.Err("failed to fetch config: %v", err)
		return types.Config{}, report.Report{}, err
	}

	return config.Parse(data)
}
