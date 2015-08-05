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

// The ec2 provider fetches a remote configuration from the ec2 user-data
// metadata service URL.

package ec2

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/providers"
	"github.com/coreos/ignition/src/providers/util"
)

const (
	name           = "ec2"
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 30 * time.Second
	userdataUrl    = "http://169.254.169.254/2009-04-04/user-data"
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
		client:  &http.Client{},
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
	return config.Parse(p.rawConfig)
}

func (p *provider) IsOnline() bool {
	if resp, err := p.client.Get(userdataUrl); err == nil {
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK, http.StatusNoContent:
			p.logger.Debug("successfully fetched")
			if p.rawConfig, err = ioutil.ReadAll(resp.Body); err != nil {
				p.logger.Err("failed to read body: %v", err)
				return false
			}
		case http.StatusNotFound:
			p.logger.Debug("no config to fetch")
		default:
			p.logger.Debug("failed fetching: HTTP status: %s", resp.Status)
			return false
		}

		return true
	} else {
		p.logger.Warning("failed fetching: %v", err)
	}

	return false
}

func (p provider) ShouldRetry() bool {
	return true
}

func (p *provider) BackoffDuration() time.Duration {
	return util.ExpBackoff(&p.backoff, maxBackoff)
}
