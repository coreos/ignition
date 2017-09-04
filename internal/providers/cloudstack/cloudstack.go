// Copyright 2017 CoreOS, Inc.
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

// The CloudStack provider fetches configurations from the userdata available in
// the config-drive.
// NOTE: This provider is still EXPERIMENTAL.

package cloudstack

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/config/validate/report"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/resource"

	"golang.org/x/net/context"
)

const (
	diskByLabelPath         = "/dev/disk/by-label/"
	configDriveUserdataPath = "/cloudstack/userdata/user_data.txt"
)

func FetchConfig(f resource.Fetcher) (types.Config, report.Report, error) {
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

	go dispatch("config drive (config)", func() ([]byte, error) {
		return fetchConfigFromDevice(f.Logger, ctx, "config-2")
	})

	go dispatch("config drive (CONFIG)", func() ([]byte, error) {
		return fetchConfigFromDevice(f.Logger, ctx, "CONFIG-2")
	})

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		f.Logger.Info("Config drive was not available in time. Continuing without a config...")
	}

	return config.Parse(data)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return (err == nil)
}

func labelExists(label string) bool {
	_, err := getPath(label)
	return (err == nil)
}

func getPath(label string) (string, error) {
	path := diskByLabelPath + label

	if fileExists(path) {
		return path, nil
	}

	return "", fmt.Errorf("label not found: %s", label)
}

func fetchConfigFromDevice(logger *log.Logger, ctx context.Context, label string) ([]byte, error) {
	for !labelExists(label) {
		logger.Debug("config drive (%q) not found. Waiting...", label)
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	path, err := getPath(label)
	if err != nil {
		return nil, err
	}

	logger.Debug("creating temporary mount point")
	mnt, err := ioutil.TempDir("", "ignition-configdrive")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.Remove(mnt)

	cmd := exec.Command("/bin/mount", "-o", "ro", "-t", "auto", path, mnt)
	if _, err := logger.LogCmd(cmd, "mounting config drive"); err != nil {
		return nil, err
	}
	defer logger.LogOp(
		func() error { return syscall.Unmount(mnt, 0) },
		"unmounting %q at %q", path, mnt,
	)

	if !fileExists(filepath.Join(mnt, configDriveUserdataPath)) {
		return nil, nil
	}

	return ioutil.ReadFile(filepath.Join(mnt, configDriveUserdataPath))
}
