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

// The configdrive provider fetches a user_data from a config-2 file system.

package configdrive

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/providers"
	"github.com/coreos/ignition/internal/providers/util"
)

const (
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 30 * time.Second
	configDevice   = "/dev/disk/by-label/config-2"
	configPath     = "/openstack/latest/user_data:/ec2/latest/user-data"
)

type Creator struct{}

func (Creator) Create(logger *log.Logger) providers.Provider {
	return &provider{
		logger:  logger,
		backoff: initialBackoff,
	}
}

type provider struct {
	logger  *log.Logger
	backoff time.Duration
}

func (p provider) FetchConfig() (types.Config, error) {
	p.logger.Debug("creating temporary mount point")
	mnt, err := ioutil.TempDir("", "ignition-configdrive")
	if err != nil {
		return types.Config{}, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.Remove(mnt)

	p.logger.Debug("mounting config drive")
	if cmd := exec.Command("/usr/bin/mount", "-o", "ro", "-t", "auto", configDevice, mnt); cmd.Run() != nil {
		return types.Config{}, fmt.Errorf("failed to mount device %q at %q: %v", configDevice, mnt, err)
	}
	defer p.logger.LogOp(
		func() error { return syscall.Unmount(mnt, 0) },
		"unmounting %q at %q", configDevice, mnt,
	)

	p.logger.Debug("reading config")
	for _, path := range strings.Split(configPath, ":") {
		rawConfig, err := ioutil.ReadFile(filepath.Join(mnt, path))
		if err == nil {
			return config.Parse(rawConfig)
		}
	}

	return types.Config{}, fmt.Errorf("failed to find config in %s", configPath)
}

func (p provider) IsOnline() bool {
	p.logger.Debug("opening config drive: %s", configDevice)
	err := util.AssertCdromOnline(configDevice)
	if err == nil || err == util.ErrNotACdrom {
		return true
	} else {
		p.logger.Info("Drive is not online: %s", err)
		return false
	}
}

func (p provider) ShouldRetry() bool {
	return true
}

func (p *provider) BackoffDuration() time.Duration {
	return util.ExpBackoff(&p.backoff, maxBackoff)
}
