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

// The OpenStack provider fetches configurations from the userdata available in
// both the config-drive as well as the network metadata service. Whichever
// responds first is the config that is used.
// NOTE: This provider is still EXPERIMENTAL.

package proxmoxve

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	ut "github.com/coreos/ignition/v2/internal/util"

	"github.com/coreos/vcontext/report"
)

const (
	ciuserdataPath = "/user-data"
	// See https://bugzilla.proxmox.com/show_bug.cgi?id=2429 for more details about vendordata
	civendordataPath = "/vendor-data"
	deviceLabel      = "cidata"
)

func init() {
	platform.Register(
		platform.Provider{
			Name:  "proxmoxve",
			Fetch: fetchConfig,
		},
	)
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

	go dispatch(
		"config drive (cidata)", func() ([]byte, error) {
			return fetchConfigFromDevice(f.Logger, ctx, filepath.Join(distro.DiskByLabelDir(), deviceLabel))
		},
	)

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		f.Logger.Info("cidata drive was not available in time. Continuing without a config...")
	}

	return util.ParseConfig(f.Logger, data)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return (err == nil)
}

func fetchConfigFromDevice(logger *log.Logger, ctx context.Context, path string) ([]byte, error) {
	for !fileExists(path) {
		logger.Debug("config drive (%q) not found. Waiting...", path)
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	logger.Debug("creating temporary mount point")
	mnt, err := os.MkdirTemp("", "ignition-configdrive")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(mnt); removeErr != nil {
			logger.Warning("failed to remove temp directory %q: %v", mnt, removeErr)
		}
	}()

	cmd := exec.Command(distro.MountCmd(), "-o", "ro", "-t", "auto", path, mnt)
	if _, err := logger.LogCmd(cmd, "mounting config drive"); err != nil {
		return nil, err
	}
	defer func() {
		_ = logger.LogOp(
			func() error {
				return ut.UmountPath(mnt)
			},
			"unmounting %q at %q", path, mnt,
		)
	}()

	paths := []string{ciuserdataPath, civendordataPath}
	header := []byte("#cloud-config\n")

	for _, path := range paths {
		fullPath := filepath.Join(mnt, path)
		if !fileExists(fullPath) {
			continue
		}

		contents, err := os.ReadFile(fullPath)
		if err != nil {
			// Log the error but continue to next file
			logger.Debug("failed to read %q: %v", fullPath, err)
			continue
		}

		// Skip if it's a cloud-config file
		if bytes.HasPrefix(contents, header) {
			logger.Debug("config drive (%q) contains a cloud-config configuration, ignoring", fullPath)
			continue
		}

		// Check if there's actual content in the file
		if len(contents) > 0 {
			logger.Debug("config drive (%q) contains data", fullPath)
			return contents, nil
		}

		logger.Debug("config drive (%q) is empty, ignoring", fullPath)
	}

	// No valid configuration found in any of the files
	return nil, nil
}
