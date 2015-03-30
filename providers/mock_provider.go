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
	// Go is a poorly thought-out language, resulting in hacks like 'Name_'
	// littered throughout the codebase.
	Name_   string
	Config  config.Config
	Err     error
	Online  bool
	Retry   bool
	Backoff time.Duration
}

func (p MockProvider) Name() string                        { return p.Name_ }
func (p MockProvider) FetchConfig() (config.Config, error) { return p.Config, p.Err }
func (p MockProvider) IsOnline() bool                      { return p.Online }
func (p MockProvider) ShouldRetry() bool                   { return p.Retry }
func (p MockProvider) BackoffDuration() time.Duration      { return p.Backoff }

type MockProviderCreator struct {
	Name_    string
	Provider Provider
}

func (c MockProviderCreator) Name() string                 { return c.Name_ }
func (c MockProviderCreator) Create(_ log.Logger) Provider { return c.Provider }
