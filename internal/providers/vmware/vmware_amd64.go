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
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/providers"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
	"github.com/vmware/vmw-guestinfo/rpcvmx"
	"github.com/vmware/vmw-guestinfo/vmcheck"
	"github.com/vmware/vmw-ovflib"
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	if isVM, err := vmcheck.IsVirtualWorld(); err != nil {
		return types.Config{}, report.Report{}, err
	} else if !isVM {
		return types.Config{}, report.Report{}, providers.ErrNoProvider
	}

	config, err := fetchRawConfig(f)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	decodedData, err := decodeConfig(config)
	if err != nil {
		f.Logger.Debug("failed to decode config: %v", err)
		return types.Config{}, report.Report{}, err
	}

	f.Logger.Debug("config successfully fetched")
	return util.ParseConfig(f.Logger, decodedData)
}

func fetchRawConfig(f *resource.Fetcher) (config, error) {
	info := rpcvmx.NewConfig()

	var ovfData string
	var ovfEncoding string

	ovfEnv, err := info.String("ovfenv", "")
	if err != nil {
		f.Logger.Warning("failed to fetch ovfenv: %v. Continuing...", err)
	} else if ovfEnv != "" {
		f.Logger.Debug("using OVF environment from guestinfo")
		env, err := ovf.ReadEnvironment([]byte(ovfEnv))
		if err != nil {
			f.Logger.Warning("failed to parse OVF environment: %v. Continuing...", err)
		}

		ovfData = env.Properties["guestinfo.ignition.config.data"]
		ovfEncoding = env.Properties["guestinfo.ignition.config.data.encoding"]
	}

	data, err := info.String("ignition.config.data", ovfData)
	if err != nil {
		f.Logger.Debug("failed to fetch config: %v", err)
		return config{}, err
	}

	encoding, err := info.String("ignition.config.data.encoding", ovfEncoding)
	if err != nil {
		f.Logger.Debug("failed to fetch config encoding: %v", err)
		return config{}, err
	}

	return config{
		data:     data,
		encoding: encoding,
	}, nil
}
