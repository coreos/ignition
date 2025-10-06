// Copyright 2021 Red Hat.
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

// The KubeVirt (https://kubevirt.io) provider fetches
// configuration from the userdata available in the config-drive. It is similar
// to Openstack and uses the same APIs.

package kubevirt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	ut "github.com/coreos/ignition/v2/internal/util"

	"github.com/coreos/vcontext/report"
)

const (
	configDriveUserdataPath = "/openstack/latest/user_data"
)

func init() {
	platform.Register(platform.Provider{
		Name:  "kubevirt",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	var data []byte
	var err error
	configPath := filepath.Join(distro.DiskByLabelDir(), "config-2")
	data, err = fetchConfigFromDevice(f.Logger, configPath)
	if err != nil {
		f.Logger.Err("failed to fetch config from %s: %v", configPath, err)
	}
	return util.ParseConfig(f.Logger, data)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return (err == nil)
}

func fetchConfigFromDevice(logger *log.Logger, path string) ([]byte, error) {
	// There is not always a config drive in kubevirt, but we can limit ignition usage
	// to VMs with config drives. Block forever if there is none.
	for !fileExists(path) {
		logger.Debug("config drive (%q) not found. Waiting...", path)
		time.Sleep(time.Second)
	}

	logger.Debug("creating temporary mount point")
	mnt, err := os.MkdirTemp("", "ignition-configdrive")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(mnt); removeErr != nil {
			logger.Warning("failed to remove temp directory %q: %v", mnt, removeErr)
		}
	}()

	cmd := exec.Command(distro.MountCmd(), "-o", "ro", "-t", "auto", path, mnt)
	if _, err := logger.LogCmd(cmd, "mounting config drive"); err != nil {
		return nil, err
	}
	defer func() {
		_ = logger.LogOp(
			func() error {
				return ut.UmountPath(mnt)
			},
			"unmounting %q at %q", path, mnt,
		)
	}()

	mntConfigDriveUserdataPath := filepath.Join(mnt, configDriveUserdataPath)
	if !fileExists(mntConfigDriveUserdataPath) {
		return nil, nil
	}

	return os.ReadFile(mntConfigDriveUserdataPath)
}
