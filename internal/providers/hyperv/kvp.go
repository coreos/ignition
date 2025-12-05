// Copyright 2023 Red Hat
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

package hyperv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/containers/libhvee/pkg/kvp"
	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/vcontext/report"
)

const singleKey = "ignition.config"

// Prefix for multiple config fragments to reassemble.  The suffix is a
// sequential integer starting from 0.
const splitKeyPrefix = "ignition.config."

func init() {
	platform.Register(platform.Provider{
		Name:           "hyperv",
		FetchWithFiles: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) ([]types.File, types.Config, report.Report, error) {
	var kvpFiles []types.File

	// To read key-value pairs from the Windows host, the hv_util kernel
	// module must be loaded to create the kernel device
	if _, statErr := os.Stat("/sys/bus/vmbus/drivers/hv_utils"); statErr != nil {
		if _, err := f.Logger.LogCmd(exec.Command(distro.ModprobeCmd(), "hv_utils"), "loading hv_utils kernel module"); err != nil {
			return nil, types.Config{}, report.Report{}, fmt.Errorf("loading hv_utils kernel module: %w", err)
		}
	}

	keyValuePairs, err := kvp.GetKeyValuePairs()
	if err != nil {
		return nil, types.Config{}, report.Report{}, fmt.Errorf("reading key-value pairs: %w", err)
	}

	var ign string
	kv, err := keyValuePairs[kvp.DefaultKVPPoolID].GetValueByKey(singleKey)
	if err == nil {
		f.Logger.Debug("found single KVP key")
		ign = kv.Value
	} else if err != kvp.ErrKeyNotFound {
		return nil, types.Config{}, report.Report{}, fmt.Errorf("looking up single KVP key: %w", err)
	}

	if ign == "" {
		ign, err = keyValuePairs.GetSplitKeyValues(splitKeyPrefix, kvp.DefaultKVPPoolID)
		if err == nil {
			f.Logger.Debug("found concatenated KVP keys")
		} else if err != kvp.ErrNoKeyValuePairsFound {
			return nil, types.Config{}, report.Report{}, fmt.Errorf("reassembling split config: %w", err)
		}
	}

	// hv_kvp_daemon writes pools to the filesystem in /var/lib/hyperv.
	// We've already read the pool data, and the host won't send it again
	// on this boot, so we need to write the files ourselves.
	for poolID := range keyValuePairs {
		// hv_kvp_daemon writes the pool files with mode 644 in a
		// directory with mode 755.  This isn't safe for us, since
		// it leaks the config to non-root users, including on
		// subsequent boots.
		// - There's no API that lets us delete the KVPs from the host.
		// - We could filter out the KVPs when writing the pools,
		//   but if hv_kvp_daemon runs on subsequent boots, it could
		//   re-add them.
		// - The caller doesn't give us a way to create directory
		//   entries, only files; and we probably shouldn't set
		//   restrictive permissions on /var/lib/hyperv because it
		//   hypothetically might be used for other purposes.
		// Avoid the issue by setting the files to mode 600.
		// hv_kvp_daemon won't change the mode afterward.
		poolPath := filepath.Join(kvp.DefaultKVPFilePath, fmt.Sprintf("%s%d", kvp.DefaultKVPBaseName, poolID))
		kvpFiles = append(kvpFiles, util.MakeProviderOutputFile(poolPath, 0600, keyValuePairs.EncodePoolFile(poolID)))
	}

	if ign == "" {
		return kvpFiles, types.Config{}, report.Report{}, errors.ErrEmpty
	}

	c, r, err := util.ParseConfig(f.Logger, []byte(ign))
	return kvpFiles, c, r, err
}
