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

package cmdline

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
	userdataUrl    = "http://169.254.169.254/2009-04-04/user-data"
	keyBaseUrl     = "http://169.254.169.254/2009-04-04/meta-data/public-keys/"
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
	switch err {
	case config.ErrCloudConfig, config.ErrScript, config.ErrEmpty:
	default:
		return config.Config{}, err
	}

	keynames, err := fetchAttributes(p.client, keyBaseUrl)
	if err != nil {
		return config.Config{}, err
	}

	keyIDs := []string{}
	for _, keyname := range keynames {
		tokens := strings.SplitN(keyname, "=", 2)
		if len(tokens) != 2 {
			return config.Config{}, fmt.Errorf("malformed public key: %q", keyname)
		}
		keyIDs = append(keyIDs, tokens[0])
		p.logger.Info("found SSH public key for %q\n", tokens[1])
	}

	sshKeys := []string{}
	for _, id := range keyIDs {
		sshkey, _, err := fetchBody(p.client, fmt.Sprintf("%s%s/openssh-key", keyBaseUrl, id), http.StatusOK)
		if err != nil {
			return config.Config{}, err
		}
		sshKeys = append(sshKeys, string(sshkey))
	}

	exists := false
	for i, user := range cfg.Passwd.Users {
		if user.Name == "core" {
			cfg.Passwd.Users[i].SSHAuthorizedKeys = append(cfg.Passwd.Users[i].SSHAuthorizedKeys, sshKeys...)
			exists = true
			break
		}
	}

	if !exists {
		cfg.Passwd.Users = append(cfg.Passwd.Users, config.User{
			Name:              "core",
			SSHAuthorizedKeys: sshKeys,
		})
	}

	return cfg, nil
}

func (p *provider) IsOnline() bool {
	body, status, err := fetchBody(p.client, userdataUrl, http.StatusOK, http.StatusNotFound)
	if err != nil {
		p.logger.Warning("failed fetching: %v", err)
		return false
	}

	switch status {
	case http.StatusOK:
		p.rawConfig = body
		p.logger.Debug("successfully fetched")
	case http.StatusNotFound:
		p.logger.Debug("no config to fetch")
	}

	return true
}

func (p provider) ShouldRetry() bool {
	return true
}

func (p *provider) BackoffDuration() time.Duration {
	return util.ExpBackoff(&p.backoff, maxBackoff)
}

func fetchBody(client *http.Client, url string, acceptedStatuses ...int) ([]byte, int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	defer resp.Body.Close()

	accepted := false
	for _, status := range acceptedStatuses {
		if status == resp.StatusCode {
			accepted = true
			break
		}
	}
	if !accepted {
		return nil, resp.StatusCode, fmt.Errorf("bad HTTP status: %s", http.StatusText(resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func fetchAttributes(client *http.Client, url string) ([]string, error) {
	body, status, err := fetchBody(client, url, http.StatusOK, http.StatusNotFound)
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
		scanner := bufio.NewScanner(bytes.NewBuffer(body))
		data := make([]string, 0)
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
		return data, scanner.Err()
	case http.StatusNotFound:
		return []string{}, nil
	default:
		panic("unexpected HTTP status")
	}
}
