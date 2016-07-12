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
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/providers"
	putil "github.com/coreos/ignition/internal/providers/util"
	"github.com/coreos/ignition/internal/util"

	"github.com/packethost/packngo/metadata"
)

const (
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 30 * time.Second
)

type Creator struct{}

func (Creator) Create(logger *log.Logger) providers.Provider {
	return &provider{
		logger:  logger,
		backoff: initialBackoff,
		client:  util.NewHttpClient(logger),
	}
}

type provider struct {
	logger    *log.Logger
	backoff   time.Duration
	client    util.HttpClient
	rawConfig []byte
}

func (p provider) FetchConfig() (types.Config, error) {
	return config.Parse(p.rawConfig)
}

func (p *provider) IsOnline() bool {
	return (p.logger.LogOp(func() error {
		data, err := metadata.GetUserData()
		if err != nil {
			return err
		}

		p.rawConfig = data
		return nil
	}, "fetching config from packet metadata") == nil)
}

func (p provider) ShouldRetry() bool {
	return true
}

func (p *provider) BackoffDuration() time.Duration {
	return putil.ExpBackoff(&p.backoff, maxBackoff)
}
