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

package cmdline

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/providers"
	"github.com/coreos/ignition/src/providers/util"
)

const (
	name           = "cmdline"
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 30 * time.Second
	cmdlinePath    = "/proc/cmdline"
	cmdlineUrlFlag = "coreos.config.url"
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
		path:    cmdlinePath,
		client:  &http.Client{},
	}
}

type provider struct {
	logger      log.Logger
	backoff     time.Duration
	path        string
	shouldRetry bool
	client      *http.Client
	configUrl   string
	rawConfig   []byte
}

func (provider) Name() string {
	return name
}

func (p provider) FetchConfig() (config.Config, error) {
	return config.Parse(p.rawConfig)
}

func (p *provider) IsOnline() bool {
	if p.configUrl == "" {
		p.shouldRetry = true

		args, err := ioutil.ReadFile(p.path)
		if err != nil {
			p.logger.Err("couldn't read cmdline")
			return false
		}

		p.configUrl = parseCmdline(args)
		p.logger.Debug("parsed url from cmdline: %q", p.configUrl)
		if p.configUrl == "" {
			p.shouldRetry = false
			return false
		}
	}

	if resp, err := p.client.Get(p.configUrl); err == nil {
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK, http.StatusNoContent:
		default:
			p.logger.Debug("failed fetching: HTTP status: %s", resp.Status)
			return false
		}

		p.logger.Debug("successfully fetched")
		if p.rawConfig, err = ioutil.ReadAll(resp.Body); err != nil {
			p.logger.Err("failed to read body: %v", err)
			return false
		}
		return true
	} else {
		p.logger.Warning("failed fetching: %v", err)
	}

	return false
}

func (p provider) ShouldRetry() bool {
	return p.shouldRetry
}

func (p *provider) BackoffDuration() time.Duration {
	return util.ExpBackoff(&p.backoff, maxBackoff)
}

func parseCmdline(cmdline []byte) (url string) {
	for _, arg := range strings.Split(string(cmdline), " ") {
		parts := strings.SplitN(strings.TrimSpace(arg), "=", 2)
		key := parts[0]

		if key != cmdlineUrlFlag {
			continue
		}

		if len(parts) == 2 {
			url = parts[1]
		}
	}

	return
}
