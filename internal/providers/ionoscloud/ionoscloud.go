// Copyright 2024 Red Hat, Inc.
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
// NOTE: This provider is still EXPERIMENTAL.
//
// The IONOS Cloud provider fetches the ignition config from a user-data file.
// This file is created by the IONOS Cloud VM handler before the first boot
// and gets injected into a device at /config/user-data by default.
//
// The kernel parameters deviceLabelKernelFlag and userDataKernelFlag can be
// used during the build process of images and for the VM initialization to
// specify on which disk or partition the user-data is going to be injected.
//
// User data files with the directive #cloud-config and #!/bin/ will be ignored
// See for more: https://docs.ionos.com/cloud/compute-services/compute-engine/how-tos/boot-cloud-init

package ionoscloud

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	deviceLabelKernelFlag = "ignition.config.device"
	defaultDeviceLabel    = "OEM"
	userDataKernelFlag    = "ignition.config.path"
	defaultUserDataPath   = "config/user-data"
)

func init() {
	platform.Register(platform.Provider{
		Name:  "ionoscloud",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	var data []byte
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	dispatch := func(name string, fn func() ([]byte, error)) {
		raw, err := fn()
		if err != nil {
			switch err {
			case context.Canceled:
			case context.DeadlineExceeded:
				f.Logger.Err("timed out while fetching config from %s", name)
			default:
				f.Logger.Err("failed to fetch config from %s: %v", name, err)
			}
			return
		}

		data = raw
		cancel()
	}

	deviceLabel, userDataPath, err := readFromKernelParams(f.Logger)

	if err != nil {
		f.Logger.Err("couldn't read kernel parameters: %v", err)
		return types.Config{}, report.Report{}, err
	}

	if deviceLabel == "" {
		deviceLabel = defaultDeviceLabel
	}

	if userDataPath == "" {
		userDataPath = defaultUserDataPath
	}

	go dispatch(
		"load config from disk", func() ([]byte, error) {
			return fetchConfigFromDevice(f.Logger, ctx, deviceLabel, userDataPath)
		},
	)

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		f.Logger.Info("disk was not available in time. Continuing without a config...")
	}

	return util.ParseConfig(f.Logger, data)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return (err == nil)
}

func fetchConfigFromDevice(logger *log.Logger,
	ctx context.Context,
	deviceLabel string,
	dataPath string,
) ([]byte, error) {
	device := filepath.Join(distro.DiskByLabelDir(), deviceLabel)
	for !fileExists(device) {
		logger.Debug("disk (%q) not found. Waiting...", device)
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	logger.Debug("creating temporary mount point")
	mnt, err := os.MkdirTemp("", "ignition-config")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.Remove(mnt)

	cmd := exec.Command(distro.MountCmd(), "-o", "ro", "-t", "auto", device, mnt)
	if _, err := logger.LogCmd(cmd, "mounting disk"); err != nil {
		return nil, err
	}
	defer func() {
		_ = logger.LogOp(
			func() error {
				return ut.UmountPath(mnt)
			},
			"unmounting %q at %q", device, mnt,
		)
	}()

	if !fileExists(filepath.Join(mnt, dataPath)) {
		return nil, nil
	}

	contents, err := os.ReadFile(filepath.Join(mnt, dataPath))
	if err != nil {
		return nil, err
	}

	if util.IsCloudConfig(contents) {
		logger.Debug("disk (%q) contains a cloud-config configuration, ignoring", device)
		return nil, nil
	}

	if util.IsShellScript(contents) {
		logger.Debug("disk (%q) contains a shell script, ignoring", device)
		return nil, nil
	}

	return contents, nil
}

func readFromKernelParams(logger *log.Logger) (string, string, error) {
	args, err := os.ReadFile(distro.KernelCmdlinePath())
	if err != nil {
		return "", "", err
	}

	deviceLabel, userDataPath := parseParams(args)
	logger.Debug("parsed device label from parameters: %s", deviceLabel)
	logger.Debug("parsed user-data path from parameters: %s", userDataPath)
	return deviceLabel, userDataPath, nil
}

func parseParams(args []byte) (deviceLabel, userDataPath string) {
	for _, arg := range strings.Split(string(args), " ") {
		parts := strings.SplitN(strings.TrimSpace(arg), "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		if key == deviceLabelKernelFlag {
			deviceLabel = value
		}

		if key == userDataKernelFlag {
			userDataPath = value
		}
	}

	return
}
