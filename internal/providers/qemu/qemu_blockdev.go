// Copyright 2020 Red Hat, Inc.
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

// +build s390x ppc64le

// The QEMU provider on s390x and ppc64le fetches a configuration file from an
// attached block device with id 'virtio-ignition'.

package qemu

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

const (
	ignitionBlockDevicePath    = "/dev/disk/by-id/virtio-ignition"
	blockDeviceTimeout         = 5 * time.Minute
	blockDevicePollingInterval = 5 * time.Second
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	f.Logger.Warning("Fetching the Ignition config via the Virtio block driver is currently experimental and subject to change.")

	_, err := f.Logger.LogCmd(exec.Command("modprobe", "virtio_blk"), "loading Virtio block driver module")
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	data, err := fetchConfigFromBlockDevice(f.Logger)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}

func fetchConfigFromBlockDevice(logger *log.Logger) ([]byte, error) {
	var data []byte
	c := make(chan error)
	go func() {
		var err error
		for {
			if data, err = ioutil.ReadFile(ignitionBlockDevicePath); err != nil {
				if !os.IsNotExist(err) {
					break
				}
				logger.Debug("block device (%q) not found. Waiting...", ignitionBlockDevicePath)
				time.Sleep(blockDevicePollingInterval)
			} else {
				err = nil
				break
			}
		}
		c <- err
	}()

	select {
	case err := <-c:
		if err != nil {
			return nil, err
		}
	case <-time.After(blockDeviceTimeout):
		return nil, fmt.Errorf("timed out after %v waiting for block device %q to appear", blockDeviceTimeout, ignitionBlockDevicePath)
	}

	return bytes.TrimRight(data, "\x00"), nil
}
