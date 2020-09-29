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

// +build !s390x,!ppc64le

// The default QEMU provider fetches a local configuration from the firmware
// config interface (opt/com.coreos/config). Platforms without support for
// qemu_fw_cfg should use the blockdev implementation instead.

package qemu

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

const (
	firmwareConfigPath = "/sys/firmware/qemu_fw_cfg/by_name/opt/com.coreos/config/raw"
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	_, err := f.Logger.LogCmd(exec.Command("modprobe", "qemu_fw_cfg"), "loading QEMU firmware config module")
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	data, err := ioutil.ReadFile(firmwareConfigPath)
	if os.IsNotExist(err) {
		f.Logger.Info("QEMU firmware config was not found. Ignoring...")
	} else if err != nil {
		f.Logger.Err("couldn't read QEMU firmware config: %v", err)
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}
