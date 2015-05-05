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

package file

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/providers"
	"github.com/coreos/ignition/src/providers/util"
)

const (
	name           = "file"
	fileName       = "config.json"
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
	}
}

type provider struct {
	backoff     time.Duration
	logger      log.Logger
	rawConfig   []byte
	shouldRetry bool
}

func (provider) Name() string {
	return name
}

func (p provider) FetchConfig() (config.Config, error) {
	return config.Parse(p.rawConfig)
}

func (p *provider) IsOnline() bool {
	var err error
	p.rawConfig, err = ioutil.ReadFile(fileName)
	if err != nil {
		p.logger.Err(fmt.Sprintf("couldn't read config %q: %v", fileName, err))
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
