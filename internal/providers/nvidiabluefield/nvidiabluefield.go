// Copyright 2025 CoreOS, Inc.
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
//
// The NVIDIA BlueField [1] provider fetches configurations from the bootfifo interface.
// [1] https://www.nvidia.com/en-eu/networking/products/data-processing-unit

package nvidiabluefield

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

// bootfifo paths are exposed in sysfs by the mlxbf_bootctl platform driver.
// https://github.com/torvalds/linux/blob/2fbe820/drivers/platform/mellanox/mlxbf-bootctl.c#L954
var bootfifoPaths = []string{
	"/sys/bus/platform/devices/MLNXBF04:00/bootfifo",
}

func init() {
	platform.Register(platform.Provider{
		Name:  "nvidiabluefield",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (cfg types.Config, rpt report.Report, err error) {
	// load mlxbf_bootctl module
	if _, err = f.Logger.LogCmd(exec.Command(distro.ModprobeCmd(), "mlxbf_bootctl"), "loading mlxbf-booctl kernel module"); err != nil {
		return
	}

	for _, bootfifoPath := range bootfifoPaths {
		f.Logger.Debug("Attempting to read bootfifo at %s", bootfifoPath)
		raw, err := os.ReadFile(bootfifoPath)

		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			f.Logger.Err("Could not read bootfifo at %s: %v", bootfifoPath, err)
			continue
		}

		if len(raw) == 0 {
			f.Logger.Info("%s is empty", bootfifoPath)
			continue
		}

		data := bytes.Trim(raw, "\x00")
		return util.ParseConfig(f.Logger, data)
	}

	f.Logger.Info("No config found in any of the NVIDIA BlueField bootfifo interfaces")
	return types.Config{}, report.Report{}, err
}
