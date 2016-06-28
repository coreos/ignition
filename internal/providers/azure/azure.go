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
	configDevice   = "/dev/disk/by-id/ata-Virtual_CD"
	configPath     = "/CustomData.bin"
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
	mnt, err := ioutil.TempDir("", "ignition-azure")
	if err != nil {
		return types.Config{}, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.Remove(mnt)

	p.logger.Debug("mounting config device")
	if err := p.logger.LogOp(
		func() error { return syscall.Mount(configDevice, mnt, "udf", syscall.MS_RDONLY, "") },
		"mounting %q at %q", configDevice, mnt,
	); err != nil {
		return types.Config{}, fmt.Errorf("failed to mount device %q at %q: %v", configDevice, mnt, err)
	}
	defer p.logger.LogOp(
		func() error { return syscall.Unmount(mnt, 0) },
		"unmounting %q at %q", configDevice, mnt,
	)

	p.logger.Debug("reading config")
	rawConfig, err := ioutil.ReadFile(filepath.Join(mnt, configPath))
	if err != nil && !os.IsNotExist(err) {
		return types.Config{}, fmt.Errorf("failed to read config: %v", err)
	}

	return config.Parse(rawConfig)
}

func (p provider) IsOnline() bool {
	p.logger.Debug("opening config device")
	err := util.AssertCdromOnline(configDevice)
	if err == nil {
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
