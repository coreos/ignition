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

package qemu

import (
	"io/ioutil"
	"os/exec"

	"github.com/coreos/ignition/v2/internal/resource"
)

const (
	firmwareConfigPath = "/sys/firmware/qemu_fw_cfg/by_name/opt/com.coreos/config/raw"
)

func fwCfgSupported() bool {
	return true
}

func fetchFromFwCfg(f *resource.Fetcher) ([]byte, error) {
	_, err := f.Logger.LogCmd(exec.Command("modprobe", "qemu_fw_cfg"), "loading QEMU firmware config module")
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(firmwareConfigPath)
}
