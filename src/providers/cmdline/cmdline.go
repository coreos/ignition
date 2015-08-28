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
// in the kernel boot option "coreos.config.url".

package cmdline

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	oemMountPath   = "/mnt/oem"               // Mountpoint where oem partition is mounted when present.
	oemDirPath     = "/sysroot/usr/share/oem" // OEM dir within root fs to consider for pxe scenarios.
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
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type provider struct {
	logger    log.Logger
	backoff   time.Duration
	path      string
	client    *http.Client
	configUrl string
	rawConfig []byte
}

func (provider) Name() string {
	return name
}

func (p provider) FetchConfig() (config.Config, error) {
	if p.rawConfig == nil {
		return config.Config{}, nil
	} else {
		return config.Parse(p.rawConfig)
	}
}

func (p *provider) IsOnline() bool {
	if p.configUrl == "" {
		args, err := ioutil.ReadFile(p.path)
		if err != nil {
			p.logger.Err("couldn't read cmdline")
			return false
		}

		p.configUrl = parseCmdline(args)
		p.logger.Debug("parsed url from cmdline: %q", p.configUrl)
		if p.configUrl == "" {
			// If the cmdline flag wasn't provided, just no-op.
			p.logger.Info("no config URL provided")
			return true
		}
	}

	return p.getRawConfig()

}

func (p provider) ShouldRetry() bool {
	return true
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

// getRawConfig gets the raw configuration data from p.configUrl.
// Supported URL schemes are:
// http://	remote resource accessed via http
// oem://	local file in /sysroot/usr/share/oem or /mnt/oem
func (p *provider) getRawConfig() bool {
	url, err := url.Parse(p.configUrl)
	if err != nil {
		p.logger.Err("failed to parse url: %v", err)
		return false
	}

	switch url.Scheme {
	case "http":
		if resp, err := p.client.Get(p.configUrl); err == nil {
			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK, http.StatusNoContent:
			default:
				p.logger.Debug("failed fetching: HTTP status: %s",
					resp.Status)
				return false
			}

			p.logger.Debug("successfully fetched")
			p.rawConfig, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				p.logger.Err("failed to read body: %v", err)
				return false
			}
		} else {
			p.logger.Warning("failed fetching: %v", err)
			return false
		}
	case "oem":
		path := filepath.Clean(url.Path)
		if !filepath.IsAbs(path) {
			p.logger.Err("oem path is not absolute: %q", url.Path)
			return false
		}

		absPath := filepath.Join(oemMountPath, path)
		p.rawConfig, err = ioutil.ReadFile(absPath)
		if os.IsNotExist(err) {
			p.logger.Info("oem config not found in %q, trying %q",
				oemMountPath, oemDirPath)
			absPath = filepath.Join(oemDirPath, path)
			p.rawConfig, err = ioutil.ReadFile(absPath)
		}

		if err != nil {
			p.logger.Err("failed to read oem config: %v", err)
			return false
		}
	default:
		p.logger.Err("unsupported url scheme: %q", url.Scheme)
		return false
	}

	return true
}
