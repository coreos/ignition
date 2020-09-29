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

// The zVM provider fetches a local configuration from the virtual unit
// record devices.

package zvm

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/vcontext/report"
)

const readerDevice string = "000c"

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	// Fetch config files directly from reader device.
	_, err := f.Logger.LogCmd(exec.Command(distro.ModprobeCmd(), "vmur"), "Loading zVM control program module")
	if err != nil {
		f.Logger.Err("Couldn't install vmur module: %v", err)
		errors := fmt.Errorf("Couldn't install vmur module: %v", err)
		return types.Config{}, report.Report{}, errors
	}
	// Online the reader device.
	logger := f.Logger
	err = onlineDevice(logger)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}
	// Read files from the z/VM reader queue.
	readerInfo, err := exec.Command(distro.VmurCmd(), "li").CombinedOutput()
	if err != nil {
		f.Logger.Err("Can not get reader device: %v", err)
		errors := fmt.Errorf("Can not get reader device: %v", err)
		return types.Config{}, report.Report{}, errors
	}
	for _, records := range strings.Split(string(readerInfo), "\n") {
		record := strings.Fields(records)
		// The records format: ORIGINID FILE CLASSA CLASSB RECORDS CPY HOLD DATE TIME NAME TYPE DIST
		if len(record) <= 1 {
			break
		}
		if len(record) < 11 {
			continue
		}
		spoolid := record[1]
		ftype := record[10]
		file := record[9] + "." + ftype
		// Receive the spool file.
		if ftype == "ign" {
			_, err := f.Logger.LogCmd(exec.Command(distro.VmurCmd(), "re", "-f", spoolid, file), "Receive the spool file")
			if err != nil {
				return types.Config{}, report.Report{}, err
			}
			f.Logger.Info("using config file at %q", file)
			rawConfig, err := ioutil.ReadFile(file)
			if err != nil {
				f.Logger.Err("Couldn't read config from configFile %q: %v", file, err)
				break
			}
			jsonConfig := bytes.Trim(rawConfig, string(byte(0)))
			return util.ParseConfig(f.Logger, jsonConfig)
		}
	}
	return types.Config{}, report.Report{}, errors.ErrEmpty
}

func onlineDevice(logger *log.Logger) error {
	_, err := logger.LogCmd(exec.Command(distro.ChccwdevCmd(), "-e", readerDevice), "Brings a Linux device online")
	if err != nil {
		// If online failed, expose the device firstly.
		_, err = logger.LogCmd(exec.Command(distro.CioIgnoreCmd(), "-r", readerDevice), "Expose reader device")
		if err != nil {
			logger.Err("Couldn't expose reader device %q: %v", readerDevice, err)
			errors := fmt.Errorf("Couldn't expose reader device %q: %v", readerDevice, err)
			return errors
		}
		_, err = logger.LogCmd(exec.Command(distro.ChccwdevCmd(), "-e", readerDevice), "Brings a Linux device online")
		if err != nil {
			logger.Err("Couldn't online reader device")
			errors := fmt.Errorf("Couldn't online reader device")
			return errors
		}
	}
	_, err = logger.LogCmd(exec.Command(distro.UdevadmCmd(), "settle"), "Settle udev device")
	return err
}
