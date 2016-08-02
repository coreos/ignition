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
	"strings"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/providers"
	"github.com/coreos/ignition/internal/util"
)

const (
	cmdlinePath    = "/proc/cmdline"
	cmdlineUrlFlag = "coreos.config.url"
)

type Creator struct{}

func (Creator) Create(logger *log.Logger) providers.Provider {
	return &provider{
		client: util.NewHttpClient(logger),
		logger: logger,
		path:   cmdlinePath,
	}
}

type provider struct {
	client util.HttpClient
	logger *log.Logger
	path   string
}

func (p provider) FetchConfig() (types.Config, error) {
	url, err := p.readCmdline()
	if err != nil || url == nil {
		return types.Config{}, err
	}

	data := p.client.FetchConfig(url.String(), http.StatusOK)
	if data == nil {
		return types.Config{}, providers.ErrNoProvider
	}

	return config.Parse(data)
}

func (p provider) readCmdline() (*url.URL, error) {
	args, err := ioutil.ReadFile(p.path)
	if err != nil {
		p.logger.Err("couldn't read cmdline: %v", err)
		return nil, err
	}

	rawUrl := parseCmdline(args)
	p.logger.Debug("parsed url from cmdline: %q", rawUrl)
	if rawUrl == "" {
		p.logger.Info("no config URL provided")
		return nil, nil
	}

	url, err := url.Parse(rawUrl)
	if err != nil {
		p.logger.Err("failed to parse url: %v", err)
		return nil, err
	}

	return url, err
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
