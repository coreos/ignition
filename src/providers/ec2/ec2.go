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
	"bufio"
	"bytes"
	"fmt"
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
	name           = "ec2"
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 30 * time.Second
	baseUrl        = "http://169.254.169.254/2009-04-04/"
	userdataUrl    = baseUrl + "user-data"
	metadataUrl    = baseUrl + "meta-data"
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
	cfg, err := config.Parse(p.rawConfig)
	if err == nil || err == config.ErrEmpty {
		err = p.fetchSSHKeys(&cfg)
	}

	return cfg, err
}

func (p *provider) IsOnline() bool {

	data, status, err := p.getData(userdataUrl)
	if err != nil {
		return false
	}

	switch status {
	case http.StatusOK, http.StatusNoContent:
		p.logger.Debug("config successfully fetched")
		p.rawConfig = data
	case http.StatusNotFound:
		p.logger.Debug("no config to fetch")
	default:
		p.logger.Debug("failed fetching: HTTP status: %s", http.StatusText(status))
		return false
	}

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
	keynames, err := p.getAttributes("/public-keys")
	if err != nil {
		return fmt.Errorf("error reading keys: %v", err)
	}

	keyIDs := make(map[string]string)
	for _, keyname := range keynames {
		tokens := strings.SplitN(keyname, "=", 2)
		if len(tokens) != 2 {
			return fmt.Errorf("malformed public key: %q", keyname)
		}
		keyIDs[tokens[1]] = tokens[0]
	}

	keys := []string{}
	for _, id := range keyIDs {
		sshkey, err := p.getAttribute("/public-keys/%s/openssh-key", id)
		if err != nil {
			return err
		}
		keys = append(keys, sshkey)
		p.logger.Info("found SSH public key for %q", id)
	}

	for i, user := range cfg.Passwd.Users {
		if user.Name == "core" {
			cfg.Passwd.Users[i].SSHAuthorizedKeys =
				append(cfg.Passwd.Users[i].SSHAuthorizedKeys,
					keys...)
			return nil
		}
	}

	cfg.Passwd.Users = append(cfg.Passwd.Users, config.User{
		Name:              "core",
		SSHAuthorizedKeys: keys,
	})

	return nil
}

// getData gets a url and reads the body.
func (p *provider) getData(url string) (data []byte, status int, err error) {
	err = p.logger.LogOp(func() error {
		resp, err := p.client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		status = resp.StatusCode
		data, err = ioutil.ReadAll(resp.Body)
		p.logger.Debug("got data %q", data)

		return err
	}, "GET %q", url)

	return
}

// getAttributes gets a list of metadata attributes from the format string.
func (p *provider) getAttributes(format string, a ...interface{}) ([]string, error) {
	path := fmt.Sprintf(format, a...)
	data, status, err := p.getData(metadataUrl + path)
	if err != nil {
		return nil, err
	}

	switch status {
	case http.StatusOK:
		scanner := bufio.NewScanner(bytes.NewBuffer(data))
		data := []string{}
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
		return data, scanner.Err()
	case http.StatusNotFound:
		return []string{}, nil
	default:
		return nil, fmt.Errorf("bad response: HTTP status code: %d", status)
	}
}

// getAttribute gets a singleton metadata attribute from the format string.
func (p *provider) getAttribute(format string, a ...interface{}) (string, error) {
	if data, err := p.getAttributes(format, a...); err == nil && len(data) > 0 {
		return data[0], nil
	} else {
		return "", err
	}
}
