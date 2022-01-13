// Copyright 2021 Red Hat, Inc.
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
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"net/url"

	"github.com/coreos/ignition/v2/config"
	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/state"
	"github.com/coreos/ignition/v2/internal/util"

	latest "github.com/coreos/ignition/v2/config/v3_4_experimental"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
)

type ConfigFetcher struct {
	Logger  *log.Logger
	Fetcher *resource.Fetcher
	State   *state.State
}

// RenderConfig evaluates "ignition.config.replace" and "ignition.config.merge"
// in the given config and returns the result. If "ignition.config.replace" is
// set, the referenced and evaluted config will be returned. Otherwise, if
// "ignition.config.merge" is set, each of the referenced configs will be
// evaluated and merged into the provided config. If neither option is set, the
// provided config will be returned unmodified. An updated fetcher will be
// returned with any new timeouts set.
func (f *ConfigFetcher) RenderConfig(cfg types.Config) (types.Config, error) {
	if cfgRef := cfg.Ignition.Config.Replace; cfgRef.Source != nil {
		newCfg, err := f.fetchReferencedConfig(cfgRef)
		if err != nil {
			return types.Config{}, err
		}

		// Replace the HTTP client in the fetcher to be configured with the
		// timeouts of the new config
		err = f.Fetcher.UpdateHttpTimeoutsAndCAs(newCfg.Ignition.Timeouts, newCfg.Ignition.Security.TLS.CertificateAuthorities, newCfg.Ignition.Proxy)
		if err != nil {
			return types.Config{}, err
		}

		return f.RenderConfig(newCfg)
	}

	mergedCfg := cfg
	for _, cfgRef := range cfg.Ignition.Config.Merge {
		newCfg, err := f.fetchReferencedConfig(cfgRef)
		if err != nil {
			return types.Config{}, err
		}

		// Merge the old config with the new config before the new config has
		// been rendered, so we can use the new config's timeouts and CAs when
		// fetching more configs.
		cfgForFetcherSettings := latest.Merge(mergedCfg, newCfg)
		err = f.Fetcher.UpdateHttpTimeoutsAndCAs(cfgForFetcherSettings.Ignition.Timeouts, cfgForFetcherSettings.Ignition.Security.TLS.CertificateAuthorities, cfgForFetcherSettings.Ignition.Proxy)
		if err != nil {
			return types.Config{}, err
		}

		newCfg, err = f.RenderConfig(newCfg)
		if err != nil {
			return types.Config{}, err
		}

		mergedCfg = latest.Merge(mergedCfg, newCfg)
	}
	return mergedCfg, nil
}

// fetchReferencedConfig fetches and parses the requested config.
// cfgRef.Source must not be nil
func (f *ConfigFetcher) fetchReferencedConfig(cfgRef types.Resource) (types.Config, error) {
	// this is also already checked at validation time
	if cfgRef.Source == nil {
		f.Logger.Crit("invalid referenced config: %v", errors.ErrSourceRequired)
		return types.Config{}, errors.ErrSourceRequired
	}
	u, err := url.Parse(*cfgRef.Source)
	if err != nil {
		return types.Config{}, err
	}
	var headers http.Header
	if cfgRef.HTTPHeaders != nil && len(cfgRef.HTTPHeaders) > 0 {
		headers, err = cfgRef.HTTPHeaders.Parse()
		if err != nil {
			return types.Config{}, err
		}
	}
	compression := ""
	if cfgRef.Compression != nil {
		compression = *cfgRef.Compression
	}
	rawCfg, err := f.Fetcher.FetchToBuffer(*u, resource.FetchOptions{
		Headers:     headers,
		Compression: compression,
	})
	if err != nil {
		return types.Config{}, err
	}

	hash := sha512.Sum512(rawCfg)
	if u.Scheme != "data" {
		f.Logger.Debug("fetched referenced config at %s with SHA512: %s", *cfgRef.Source, hex.EncodeToString(hash[:]))
	} else {
		// data url's might contain secrets
		f.Logger.Debug("fetched referenced config from data url with SHA512: %s", hex.EncodeToString(hash[:]))
	}

	if err := util.AssertValid(cfgRef.Verification, rawCfg); err != nil {
		return types.Config{}, err
	}

	cfg, r, err := config.Parse(rawCfg)
	f.Logger.LogReport(r)
	if err != nil {
		return types.Config{}, err
	}

	f.State.FetchedConfigs = append(f.State.FetchedConfigs, state.FetchedConfig{
		Kind:       "user",
		Source:     u.Path,
		Referenced: true,
	})

	return cfg, nil
}
