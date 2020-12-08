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

// The azure provider fetches a configuration from the Azure OVF DVD.

package azure

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	execUtil "github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
	"golang.org/x/sys/unix"
)

const (
	configDeviceID = "ata-Virtual_CD"
	configPath     = "/CustomData.bin"
)

// These constants come from <cdrom.h>.
const (
	CDROM_DRIVE_STATUS = 0x5326
)

// These constants come from <cdrom.h>.
const (
	CDS_NO_INFO = iota
	CDS_NO_DISC
	CDS_TRAY_OPEN
	CDS_DRIVE_NOT_READY
	CDS_DISC_OK
)

// Azure uses a UDF volume for the OVF configuration.
const (
	CDS_FSTYPE_UDF = "udf"
)

// FetchConfig wraps FetchOvfDevice to implement the platform.NewFetcher interface.
func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	return FetchFromOvfDevice(f, []string{CDS_FSTYPE_UDF})
}

// FetchFromOvfDevice has the return signature of platform.NewFetcher. It is
// wrapped by this and AzureStack packages.
func FetchFromOvfDevice(f *resource.Fetcher, ovfFsTypes []string) (types.Config, report.Report, error) {
	devicePath := filepath.Join(distro.DiskByIDDir(), configDeviceID)

	logger := f.Logger
	logger.Debug("waiting for config DVD...")
	waitForCdrom(logger, devicePath)

	fsType, err := checkOvfFsType(logger, devicePath, ovfFsTypes)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	mnt, err := ioutil.TempDir("", "ignition-azure")
	if err != nil {
		return types.Config{}, report.Report{}, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.Remove(mnt)

	logger.Debug("mounting config device")
	if err := logger.LogOp(
		func() error { return unix.Mount(devicePath, mnt, fsType, unix.MS_RDONLY, "") },
		"mounting %q at %q", devicePath, mnt,
	); err != nil {
		return types.Config{}, report.Report{}, fmt.Errorf("failed to mount device %q at %q: %v", devicePath, mnt, err)
	}
	defer func() {
		_ = logger.LogOp(
			func() error { return unix.Unmount(mnt, 0) },
			"unmounting %q at %q", devicePath, mnt,
		)
	}()

	logger.Debug("reading config")
	rawConfig, err := ioutil.ReadFile(filepath.Join(mnt, configPath))
	if err != nil && !os.IsNotExist(err) {
		return types.Config{}, report.Report{}, fmt.Errorf("failed to read config: %v", err)
	}

	return util.ParseConfig(logger, rawConfig)
}

func waitForCdrom(logger *log.Logger, devicePath string) {
	for !isCdromPresent(logger, devicePath) {
		time.Sleep(time.Second)
	}
}

func isCdromPresent(logger *log.Logger, devicePath string) bool {
	logger.Debug("opening config device")
	device, err := os.Open(devicePath)
	if err != nil {
		logger.Info("failed to open config device: %v", err)
		return false
	}
	defer device.Close()

	logger.Debug("getting drive status")
	status, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(device.Fd()),
		uintptr(CDROM_DRIVE_STATUS),
		uintptr(0),
	)

	switch status {
	case CDS_NO_INFO:
		logger.Info("drive status: no info")
	case CDS_NO_DISC:
		logger.Info("drive status: no disc")
	case CDS_TRAY_OPEN:
		logger.Info("drive status: open")
	case CDS_DRIVE_NOT_READY:
		logger.Info("drive status: not ready")
	case CDS_DISC_OK:
		logger.Info("drive status: OK")
	default:
		logger.Err("failed to get drive status: %s", errno.Error())
	}

	return (status == CDS_DISC_OK)
}

func checkOvfFsType(logger *log.Logger, devicePath string, fsTypes []string) (string, error) {
	fs, err := execUtil.GetFilesystemInfo(devicePath, false)
	if err != nil {
		return fs.Type, fmt.Errorf("failed to detect filesystem on ovf device %q: %v", devicePath, err)
	}
	for _, f := range fsTypes {
		if f == fs.Type {
			return fs.Type, nil
		}
	}
	return fs.Type, fmt.Errorf("filesystem %q is not a supported ovf device", fs.Type)
}
