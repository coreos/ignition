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

// The vmware provider fetches a configuration from the VMware Guest Info
// interface.

package vmware

import (
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/config/validate/report"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/providers"
	"github.com/coreos/ignition/internal/providers/util"
	"github.com/coreos/ignition/internal/resource"

	"github.com/sigma/vmw-guestinfo/rpcvmx"
	"github.com/sigma/vmw-guestinfo/vmcheck"
	"github.com/vmware/vmw-ovflib"
)

func FetchConfig(logger *log.Logger, _ *resource.HttpClient) (types.Config, report.Report, error) {
	if !vmcheck.IsVirtualWorld() {
		return types.Config{}, report.Report{}, providers.ErrNoProvider
	}

	config, err := fetchRawConfig(logger)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	decodedData, err := decodeConfig(config)
	if err != nil {
		logger.Debug("failed to decode config: %v", err)
		return types.Config{}, report.Report{}, err
	}

	logger.Debug("config successfully fetched")
	return util.ParseConfig(logger, decodedData)
}

func fetchRawConfig(logger *log.Logger) (config, error) {
	info := rpcvmx.NewConfig()

	ovfEnv, err := info.String("ovfenv", "")
	if err != nil {
		logger.Debug("failed to fetch ovfenv: %v. Continuing...", err)
	} else if ovfEnv != "" {
		logger.Debug("using OVF environment from guestinfo")
		env, err := ovf.ReadEnvironment([]byte(ovfEnv))
		if err != nil {
			logger.Err("failed to parse OVF environment: %v", err)
			return config{}, err
		}

		return config{
			data:     env.Properties["guestinfo.coreos.config.data"],
			encoding: env.Properties["guestinfo.coreos.config.data.encoding"],
		}, nil
	}

	data, err := info.String("coreos.config.data", "")
	if err != nil {
		logger.Debug("failed to fetch config: %v", err)
		return config{}, err
	}

	encoding, err := info.String("coreos.config.data.encoding", "")
	if err != nil {
		logger.Debug("failed to fetch config encoding: %v", err)
		return config{}, err
	}

	return config{
		data:     data,
		encoding: encoding,
	}, nil
}
