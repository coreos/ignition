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

package providers

import (
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/log"
)

type MockProvider struct {
	name    string
	config  config.Config
	err     error
	online  bool
	retry   bool
	backoff time.Duration
}

func (p MockProvider) Name() string                        { return p.name }
func (p MockProvider) FetchConfig() (config.Config, error) { return p.config, p.err }
func (p MockProvider) IsOnline() bool                      { return p.online }
func (p MockProvider) ShouldRetry() bool                   { return p.retry }
func (p MockProvider) BackoffDuration() time.Duration      { return p.backoff }

type MockProviderCreator struct {
	name     string
	provider Provider
}

func (c MockProviderCreator) Name() string                 { return c.name }
func (c MockProviderCreator) Create(_ log.Logger) Provider { return c.provider }
