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
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/src/exec/stages"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/providers"
	putil "github.com/coreos/ignition/src/providers/util"
	"github.com/coreos/ignition/src/util"
)

const (
	DefaultOnlineTimeout = time.Minute
)

var (
	ErrSchemeUnsupported = errors.New("unsupported url scheme")
	ErrNetworkFailure    = errors.New("network failure")
)

var (
	baseConfig = types.Config{
		Ignition: types.Ignition{Version: types.IgnitionVersion(types.MaxVersion)},
		Storage: types.Storage{
			Filesystems: []types.Filesystem{{
				Name: "root",
				Path: "/sysroot",
			}},
		},
	}
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
		return stages.Get(stageName).Create(e.Logger, e.Root).Run(config.Append(baseConfig, cfg))
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
func (e Engine) acquireConfig() (cfg types.Config, err error) {
	// First try read the config @ e.ConfigCache.
	b, err := ioutil.ReadFile(e.ConfigCache)
	if err == nil {
		if err = json.Unmarshal(b, &cfg); err != nil {
			e.Logger.Crit("failed to parse cached config: %v", err)
		}
		return
	}

	// (Re)Fetch the config if the cache is unreadable.
	cfg, err = e.fetchProviderConfig()
	if err != nil {
		e.Logger.Crit("failed to fetch config: %s", err)
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

// fetchProviderConfig returns the configuration from the engine's provider
// returning an error if the provider is unavailable. This will also render the
// config (see renderConfig) before returning.
func (e Engine) fetchProviderConfig() (types.Config, error) {
	if err := putil.WaitUntilOnline(e.Provider, e.OnlineTimeout); err != nil {
		return types.Config{}, err
	}

	cfg, err := e.Provider.FetchConfig()
	switch err {
	case config.ErrDeprecated:
		e.Logger.Warning("%v: the provided config format is deprecated and will not be supported in the future", err)
		fallthrough
	case nil:
		return e.renderConfig(cfg)
	default:
		return types.Config{}, err
	}
}

// renderConfig evaluates "ignition.config.replace" and "ignition.config.append"
// in the given config and returns the result. If "ignition.config.replace" is
// set, the referenced and evaluted config will be returned. Otherwise, if
// "ignition.config.append" is set, each of the referenced configs will be
// evaluated and appended to the provided config. If neither option is set, the
// provided config will be returned unmodified.
func (e Engine) renderConfig(cfg types.Config) (types.Config, error) {
	if cfgRef := cfg.Ignition.Config.Replace; cfgRef != nil {
		return e.fetchReferencedConfig(*cfgRef)
	}

	appendedCfg := cfg
	for _, cfgRef := range cfg.Ignition.Config.Append {
		newCfg, err := e.fetchReferencedConfig(cfgRef)
		if err != nil {
			return newCfg, err
		}

		appendedCfg = config.Append(appendedCfg, newCfg)
	}
	return appendedCfg, nil
}

// fetchReferencedConfig fetches, renders, and attempts to verify the requested
// config.
func (e Engine) fetchReferencedConfig(cfgRef types.ConfigReference) (types.Config, error) {
	var rawCfg []byte
	switch cfgRef.Source.Scheme {
	case "http":
		rawCfg = util.NewHttpClient(e.Logger).
			FetchConfig(cfgRef.Source.String(), http.StatusOK, http.StatusNoContent)
		if rawCfg == nil {
			return types.Config{}, ErrNetworkFailure
		}
	default:
		return types.Config{}, ErrSchemeUnsupported
	}

	if err := util.AssertValid(cfgRef.Verification, rawCfg); err != nil {
		return types.Config{}, err
	}

	cfg, err := config.Parse(rawCfg)
	if err != nil {
		return types.Config{}, err
	}

	return e.renderConfig(cfg)
}
