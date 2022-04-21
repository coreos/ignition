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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	execUtil "github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
	"golang.org/x/sys/unix"
)

const (
	configPath = "/CustomData.bin"
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

var (
	imdsUserdataURL = url.URL{
		Scheme:   "http",
		Host:     "169.254.169.254",
		Path:     "metadata/instance/compute/userData",
		RawQuery: "api-version=2021-01-01&format=text",
	}
)

// FetchConfig wraps fetchFromAzureMetadata to implement the platform.NewFetcher interface.
func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	return fetchFromAzureMetadata(f)
}

// fetchFromAzureMetadata first tries to fetch userData from IMDS then fallback on customData in case
// of empty config.
func fetchFromAzureMetadata(f *resource.Fetcher) (types.Config, report.Report, error) {
	// fetch-offline is not supported since we first try to get config from Azure IMDS.
	// this config fetching can only happen during fetch stage.
	if f.Offline {
		return types.Config{}, report.Report{}, resource.ErrNeedNet
	}

	logger := f.Logger

	// we first try to fetch config from IMDS, in case of failure we fallback on the custom data.
	userData, err := fetchFromIMDS(f)
	if err == nil {
		logger.Info("config has been read from IMDS userdata")
		return util.ParseConfig(logger, userData)
	}

	if err != errors.ErrEmpty {
		return types.Config{}, report.Report{}, err
	}

	logger.Debug("failed to retrieve userdata from IMDS, falling back to custom data: %v", err)
	return FetchFromOvfDevice(f, []string{CDS_FSTYPE_UDF})
}

// fetchFromIMDS requests the Azure IMDS to fetch userdata and decode it.
func fetchFromIMDS(f *resource.Fetcher) ([]byte, error) {
	headers := make(http.Header)
	headers.Set("Metadata", "true")

	data, err := f.FetchToBuffer(imdsUserdataURL, resource.FetchOptions{Headers: headers})
	if err != nil {
		return nil, fmt.Errorf("fetching to buffer: %w", err)
	}

	n := len(data)

	if n == 0 {
		return nil, errors.ErrEmpty
	}

	// data is base64 encoded by the IMDS
	userData := make([]byte, base64.StdEncoding.DecodedLen(n))

	// we keep the number of bytes written to return [:l] only.
	// otherwise last byte will be 0x00 which makes fail the JSON's unmarshalling.
	l, err := base64.StdEncoding.Decode(userData, data)
	if err != nil {
		return nil, fmt.Errorf("decoding userdata: %w", err)
	}

	return userData[:l], nil
}

// FetchFromOvfDevice has the return signature of platform.NewFetcher. It is
// wrapped by this and AzureStack packages.
func FetchFromOvfDevice(f *resource.Fetcher, ovfFsTypes []string) (types.Config, report.Report, error) {
	logger := f.Logger
	checkedDevices := make(map[string]struct{})
	for {
		for _, ovfFsType := range ovfFsTypes {
			devices, err := execUtil.GetBlockDevices(ovfFsType)
			if err != nil {
				return types.Config{}, report.Report{}, fmt.Errorf("failed to retrieve block devices with FSTYPE=%q: %v", ovfFsType, err)
			}
			for _, dev := range devices {
				_, checked := checkedDevices[dev]
				// verify that this is a CD-ROM drive. This helps
				// to avoid reading data from an arbitrary block
				// device attached to the VM by the user.
				if !checked && isCdromPresent(logger, dev) {
					rawConfig, err := getRawConfig(f, dev, ovfFsType)
					if err != nil {
						logger.Debug("failed to retrieve config from device %q: %v", dev, err)
					} else {
						logger.Info("config has been read from custom data")
						return util.ParseConfig(logger, rawConfig)
					}
				}
				checkedDevices[dev] = struct{}{}
			}
		}
		// wait for the actual config drive to appear
		// if it's not shown up yet
		time.Sleep(time.Second)
	}
}

// getRawConfig returns the config by mounting the given block device
func getRawConfig(f *resource.Fetcher, devicePath string, fstype string) ([]byte, error) {
	logger := f.Logger
	mnt, err := ioutil.TempDir("", "ignition-azure")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.Remove(mnt)

	logger.Debug("mounting config device")
	if err := logger.LogOp(
		func() error { return unix.Mount(devicePath, mnt, fstype, unix.MS_RDONLY, "") },
		"mounting %q at %q", devicePath, mnt,
	); err != nil {
		return nil, fmt.Errorf("failed to mount device %q at %q: %v", devicePath, mnt, err)
	}
	defer func() {
		_ = logger.LogOp(
			func() error { return unix.Unmount(mnt, 0) },
			"unmounting %q at %q", devicePath, mnt,
		)
	}()

	// detect the config drive by looking for a file which is always present
	logger.Debug("checking for config drive")
	if _, err := os.Stat(filepath.Join(mnt, "ovf-env.xml")); err != nil {
		return nil, fmt.Errorf("device %q does not appear to be a config drive: %v", devicePath, err)
	}

	logger.Debug("reading config")
	rawConfig, err := ioutil.ReadFile(filepath.Join(mnt, configPath))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config from device %q: %v", devicePath, err)
	}
	return rawConfig, nil
}

// isCdromPresent verifies if the given config drive is CD-ROM
func isCdromPresent(logger *log.Logger, devicePath string) bool {
	logger.Debug("opening config device: %q", devicePath)
	device, err := os.Open(devicePath)
	if err != nil {
		logger.Info("failed to open config device: %v", err)
		return false
	}
	defer device.Close()

	logger.Debug("getting drive status for %q", devicePath)
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
