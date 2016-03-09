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

package exec

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/exec/stages"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/providers"
	"github.com/coreos/ignition/src/providers/util"
)

const (
	DefaultOnlineTimeout = time.Minute
)

// Engine represents the entity that fetches and executes a configuration.
type Engine struct {
	ConfigCache   string
	OnlineTimeout time.Duration
	Logger        *log.Logger
	Root          string
	Provider      providers.Provider
}

// Run executes the stage of the given name. It returns true if the stage
// successfully ran and false if there were any errors.
func (e Engine) Run(stageName string) bool {
	cfg, err := e.acquireConfig()
	switch err {
	case nil:
		e.Logger.PushPrefix(stageName)
		defer e.Logger.PopPrefix()
		return stages.Get(stageName).Create(e.Logger, e.Root).Run(cfg)
	case config.ErrCloudConfig, config.ErrScript, config.ErrEmpty:
		e.Logger.Info("%v: ignoring and exiting...", err)
		return true
	default:
		e.Logger.Crit("failed to acquire config: %v", err)
		return false
	}
}

// acquireConfig returns the configuration, first checking a local cache
// before attempting to fetch it from the provider.
func (e Engine) acquireConfig() (cfg config.Config, err error) {
	// First try read the config @ e.ConfigCache.
	b, err := ioutil.ReadFile(e.ConfigCache)
	if err == nil {
		if err = json.Unmarshal(b, &cfg); err != nil {
			e.Logger.Crit("failed to parse cached config: %v", err)
		}
		return
	}

	// (Re)Fetch the config if the cache is unreadable.
	cfg, err = fetchConfig(e.Provider, e.OnlineTimeout)
	if err != nil {
		e.Logger.Crit("failed to fetch config: %v", err)
		return
	}
	e.Logger.Debug("fetched config: %+v", cfg)

	// Populate the config cache.
	b, err = json.Marshal(cfg)
	if err != nil {
		e.Logger.Crit("failed to marshal cached config: %v", err)
		return
	}
	if err = ioutil.WriteFile(e.ConfigCache, b, 0640); err != nil {
		e.Logger.Crit("failed to write cached config: %v", err)
		return
	}

	return
}

// fetchConfig returns the configuration from the provider or returns an error
// if the provider is unavailable.
func fetchConfig(provider providers.Provider, timeout time.Duration) (config.Config, error) {
	if err := util.WaitUntilOnline(provider, timeout); err != nil {
		return config.Config{}, err
	}

	return provider.FetchConfig()
}
