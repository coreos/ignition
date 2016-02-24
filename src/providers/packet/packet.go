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

// The packet provider fetches a remote configuration from the packet.net
// userdata metadata service URL.

package packet

import (
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/providers"
	"github.com/coreos/ignition/src/providers/util"

	"github.com/coreos/ignition/third_party/github.com/packethost/packngo/metadata"
)

const (
	name           = "packet"
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 30 * time.Second
)

func init() {
	providers.Register(creator{})
}

type creator struct{}

func (creator) Name() string {
	return name
}

func (creator) Create(logger log.Logger) providers.Provider {
	return &provider{
		logger:  logger,
		backoff: initialBackoff,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type provider struct {
	logger    log.Logger
	backoff   time.Duration
	client    *http.Client
	rawConfig []byte
}

func (provider) Name() string {
	return name
}

func (p provider) FetchConfig() (config.Config, error) {
	cfg, err := config.Parse(p.rawConfig)
	if err == nil || err == config.ErrEmpty {
		err = p.fetchSSHKeys(&cfg)
	}

	return cfg, err
}

func (p *provider) IsOnline() bool {
	data, err := metadata.GetUserData()
	if err != nil {
		p.logger.Debug("failed fetching: %v", err)
		return false
	}

	p.logger.Debug("config successfully fetched")
	p.rawConfig = data

	return true
}

func (p provider) ShouldRetry() bool {
	return true
}

func (p *provider) BackoffDuration() time.Duration {
	return util.ExpBackoff(&p.backoff, maxBackoff)
}

// fetchSSHKeys fetches and appends ssh keys to the config.
func (p *provider) fetchSSHKeys(cfg *config.Config) error {
	metadata, err := p.getMetadata()
	if err != nil {
		return fmt.Errorf("error reading metadata: %v", err)
	}

	for i, user := range cfg.Passwd.Users {
		if user.Name == "core" {
			cfg.Passwd.Users[i].SSHAuthorizedKeys =
				append(cfg.Passwd.Users[i].SSHAuthorizedKeys,
					metadata.SSHKeys...)
			return nil
		}
	}

	cfg.Passwd.Users = append(cfg.Passwd.Users, config.User{
		Name:              "core",
		SSHAuthorizedKeys: metadata.SSHKeys,
	})

	return nil
}

func (p *provider) getMetadata() (cur *metadata.CurrentDevice, err error) {
	err = p.logger.LogOp(func() error {
		cd, err := metadata.GetMetadata()
		if err != nil {
			return err
		}

		cur = cd
		p.logger.Debug("got metadata %#v", cd)

		return err
	}, "PACKET METADATA %q", metadata.BaseURL)

	return
}
