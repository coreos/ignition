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

// The cmdline provider fetches a remote configuration from the URL specified
// in the kernel boot option "ignition.config.url".

package cmdline

import (
	"context"
	"fmt"
	"net/url"
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

type cmdlineFlag string

const (
	flagUrl          cmdlineFlag = "ignition.config.url"
	flagDeviceLabel  cmdlineFlag = "ignition.config.device"
	flagUserDataPath cmdlineFlag = "ignition.config.path"
)

type cmdlineOpts struct {
	Url          *url.URL
	UserDataPath string
	DeviceLabel  string
}

var (
	// we are a special-cased system provider; don't register ourselves
	// for lookup by name
	Config = platform.NewConfig(platform.Provider{
		Name:  "cmdline",
		Fetch: fetchConfig,
	})
)

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	opts, err := parseCmdline(f.Logger)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	var data []byte

	if opts.Url != nil {
		data, err = f.FetchToBuffer(*opts.Url, resource.FetchOptions{})
		if err != nil {
			return types.Config{}, report.Report{}, err
		}

		return util.ParseConfig(f.Logger, data)
	}

	if opts.UserDataPath != "" && opts.DeviceLabel != "" {
		return fetchConfigFromDevice(f.Logger, opts)
	}

	return types.Config{}, report.Report{}, platform.ErrNoProvider
}

func parseCmdline(logger *log.Logger) (*cmdlineOpts, error) {
	cmdline, err := os.ReadFile(distro.KernelCmdlinePath())
	if err != nil {
		logger.Err("couldn't read cmdline: %v", err)
		return nil, err
	}

	opts := &cmdlineOpts{}

	for _, arg := range strings.Split(string(cmdline), " ") {
		parts := strings.SplitN(strings.TrimSpace(arg), "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := cmdlineFlag(parts[0])
		value := parts[1]

		switch key {
		case flagUrl:
			if value == "" {
				logger.Info("url flag found but no value provided")
				continue
			}

			url, err := url.Parse(value)
			if err != nil {
				logger.Err("failed to parse url: %v", err)
				continue
			}
			opts.Url = url
		case flagDeviceLabel:
			if value == "" {
				logger.Info("device label flag found but no value provided")
				continue
			}
			opts.DeviceLabel = value
		case flagUserDataPath:
			if value == "" {
				logger.Info("user data path flag found but no value provided")
				continue
			}
			opts.DeviceLabel = value
		}
	}

	return opts, nil
}

func fetchConfigFromDevice(logger *log.Logger, opts *cmdlineOpts) (types.Config, report.Report, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var data []byte

	dispatch := func(name string, fn func() ([]byte, error)) {
		raw, err := fn()
		if err != nil {
			switch err {
			case context.Canceled:
			case context.DeadlineExceeded:
				logger.Err("timed out while fetching config from %s", name)
			default:
				logger.Err("failed to fetch config from %s: %v", name, err)
			}
			return
		}

		data = raw
		cancel()
	}

	go dispatch(
		"load config from disk", func() ([]byte, error) {
			return tryMounting(logger, ctx, opts)
		},
	)

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		logger.Info("disk was not available in time. Continuing without a config...")
	}

	return util.ParseConfig(logger, data)
}

func tryMounting(logger *log.Logger, ctx context.Context, opts *cmdlineOpts) ([]byte, error) {
	device := filepath.Join(distro.DiskByLabelDir(), opts.DeviceLabel)
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

	if !fileExists(filepath.Join(mnt, opts.UserDataPath)) {
		return nil, nil
	}

	contents, err := os.ReadFile(filepath.Join(mnt, opts.UserDataPath))
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return (err == nil)
}
