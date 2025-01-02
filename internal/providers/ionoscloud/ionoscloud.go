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
// The IONOS Cloud provider fetches the ignition config from the user-data
// available in an injected file at /var/lib/cloud/seed/nocloud/user-data.
// This file is created by the IONOS Cloud VM handler before the first boot
// through the cloud init user data handling.
//
// User data with the directive #cloud-config will be ignored
// See for more: https://docs.ionos.com/cloud/compute-services/compute-engine/how-tos/boot-cloud-init

package ionoscloud

import (
	"context"
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
	rootLabelEnvVar  = "IGNITION_CONFIG_ROOT_LABEL"
	defaultRootLabel = "ROOT"
	userDataPath     = "/var/lib/cloud/seed/nocloud/user-data"
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

	deviceLabel := os.Getenv(rootLabelEnvVar)
	if deviceLabel == "" {
		deviceLabel = defaultRootLabel
	}

	go dispatch(
		"load config from root partition", func() ([]byte, error) {
			return fetchConfigFromDevice(f.Logger, ctx, filepath.Join(distro.DiskByLabelDir(), deviceLabel))
		},
	)

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		f.Logger.Info("root partition was not available in time. Continuing without a config...")
	}

	return util.ParseConfig(f.Logger, data)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return (err == nil)
}

func fetchConfigFromDevice(logger *log.Logger, ctx context.Context, device string) ([]byte, error) {
	for !fileExists(device) {
		logger.Debug("root partition (%q) not found. Waiting...", device)
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
	if _, err := logger.LogCmd(cmd, "mounting root partition"); err != nil {
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

	if !fileExists(filepath.Join(mnt, userDataPath)) {
		return nil, nil
	}

	contents, err := os.ReadFile(filepath.Join(mnt, userDataPath))
	if err != nil {
		return nil, err
	}

	if util.IsCloudConfig(contents) {
		logger.Debug("root partition (%q) contains a cloud-config configuration, ignoring", device)
		return nil, nil
	}

	return contents, nil
}
