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
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/journal"
	"github.com/coreos/ignition/v2/config"
	"github.com/coreos/ignition/v2/config/shared/errors"
	latest "github.com/coreos/ignition/v2/config/v3_3_experimental"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	executil "github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers"
	"github.com/coreos/ignition/v2/internal/providers/cmdline"
	"github.com/coreos/ignition/v2/internal/providers/system"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/util"

	"github.com/coreos/vcontext/report"
	"github.com/coreos/vcontext/validate"
	"github.com/google/renameio"
)

const (
	DefaultFetchTimeout = 2 * time.Minute
	// This variable will help to identify ignition journal messages
	// related to the user/base config.
	ignitionFetchedConfigMsgId = "57124006b5c94805b77ce473e92a8aeb"

	needNetPath = "/run/ignition/neednet"
)

var (
	emptyConfig = types.Config{
		Ignition: types.Ignition{Version: types.MaxVersion.String()},
	}
)

// Engine represents the entity that fetches and executes a configuration.
type Engine struct {
	ConfigCache    string
	FetchTimeout   time.Duration
	Logger         *log.Logger
	Root           string
	PlatformConfig platform.Config
	Fetcher        *resource.Fetcher
	fetchedConfigs []fetchedConfig
}

type fetchedConfig struct {
	kind       string
	source     string
	referenced bool
}

// Run executes the stage of the given name. It returns true if the stage
// successfully ran and false if there were any errors.
func (e Engine) Run(stageName string) error {
	if e.Fetcher == nil || e.Logger == nil {
		fmt.Fprintf(os.Stderr, "engine incorrectly configured\n")
		return errors.ErrEngineConfiguration
	}
	baseConfig := emptyConfig

	systemBaseConfig, r, err := system.FetchBaseConfig(e.Logger, e.PlatformConfig.Name())
	e.logReport(r)
	if err != nil && err != providers.ErrNoProvider {
		e.Logger.Crit("failed to acquire system base config: %v", err)
		return err
	} else if err == nil {
		e.fetchedConfigs = append(e.fetchedConfigs, fetchedConfig{
			kind:       "base",
			source:     "system",
			referenced: false,
		})
	}

	// We special-case the fetch-offline stage a bit here: we want to be able
	// to handle the case where the provider itself requires networking.
	if stageName == "fetch-offline" {
		e.Fetcher.Offline = true
	}

	cfg, err := e.acquireConfig(stageName)
	if err == resource.ErrNeedNet && stageName == "fetch-offline" {
		err = SignalNeedNet()
		if err != nil {
			e.Logger.Crit("failed to signal neednet: %v", err)
		}
		return err
	} else if err == errors.ErrEmpty {
		e.Logger.Info("%v: ignoring user-provided config", err)
	} else if err != nil {
		e.Logger.Crit("failed to acquire config: %v", err)
		return err
	}

	e.Logger.PushPrefix(stageName)
	defer e.Logger.PopPrefix()

	// Run the platform config's Init function post-config fetch
	// to perform any additional fetcher configuration that might
	// need networking to already be set up. e.x. configuring the
	// S3RegionHint when running on AWS
	if err := e.PlatformConfig.InitFunc()(e.Fetcher); err != nil {
		return fmt.Errorf("initializing platform config: %v", err)
	}

	fullConfig := latest.Merge(baseConfig, latest.Merge(systemBaseConfig, cfg))
	if err = stages.Get(stageName).Create(e.Logger, e.Root, *e.Fetcher).Run(fullConfig); err != nil {
		// e.Logger could be nil
		fmt.Fprintf(os.Stderr, "%s failed", stageName)
		tmp, jsonerr := json.MarshalIndent(fullConfig, "", "  ")
		if jsonerr != nil {
			// Nothing else to do with this error
			fmt.Fprintf(os.Stderr, "Could not marshal full config: %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "Full config:\n%s", string(tmp))
		}
		return err
	}
	e.Logger.Info("%s passed", stageName)
	return nil
}

// logStructuredJournalEntry logs information related to
// a user/base config into the systemd journal log.
func logStructuredJournalEntry(cfgInfo fetchedConfig) error {
	ignitionInfo := map[string]string{
		"IGNITION_CONFIG_TYPE":       cfgInfo.kind,
		"IGNITION_CONFIG_SRC":        cfgInfo.source,
		"IGNITION_CONFIG_REFERENCED": strconv.FormatBool(cfgInfo.referenced),
		"MESSAGE_ID":                 ignitionFetchedConfigMsgId,
	}
	referenced := ""
	if cfgInfo.referenced {
		referenced = "referenced "
	}
	msg := fmt.Sprintf("fetched %s%s config from %q", referenced, cfgInfo.kind, cfgInfo.source)
	if err := journal.Send(msg, journal.PriInfo, ignitionInfo); err != nil {
		return err
	}
	return nil
}

// acquireConfig will perform differently based on the stage it is being
// called from. In fetch stages it will attempt to fetch the provider
// config (writing an empty provider config if it is empty). In all other
// stages it will attempt to fetch from the local cache only.
func (e *Engine) acquireConfig(stageName string) (cfg types.Config, err error) {
	switch {
	case strings.HasPrefix(stageName, "fetch"):
		cfg, err = e.acquireProviderConfig()

		// if we've successfully fetched and cached the configs, log about them
		if err == nil {
			for _, cfgInfo := range e.fetchedConfigs {
				if logerr := logStructuredJournalEntry(cfgInfo); logerr != nil {
					e.Logger.Info("failed to log systemd journal entry: %v", logerr)
				}
			}
		}
	default:
		cfg, err = e.acquireCachedConfig()
	}
	return
}

// acquireCachedConfig returns the configuration from a local cache if
// available
func (e *Engine) acquireCachedConfig() (cfg types.Config, err error) {
	var b []byte
	b, err = ioutil.ReadFile(e.ConfigCache)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		e.Logger.Crit("failed to parse cached config: %v", err)
		return
	}
	// Create an http client and fetcher with the timeouts from the cached
	// config
	err = e.Fetcher.UpdateHttpTimeoutsAndCAs(cfg.Ignition.Timeouts, cfg.Ignition.Security.TLS.CertificateAuthorities, cfg.Ignition.Proxy)
	if err != nil {
		e.Logger.Crit("failed to update timeouts and CAs for fetcher: %v", err)
		return
	}
	return
}

// acquireProviderConfig attempts to fetch the configuration from the
// provider.
func (e *Engine) acquireProviderConfig() (cfg types.Config, err error) {
	// Create a new http client and fetcher with the timeouts set via the flags,
	// since we don't have a config with timeout values we can use
	timeout := int(e.FetchTimeout.Seconds())
	emptyProxy := types.Proxy{}
	err = e.Fetcher.UpdateHttpTimeoutsAndCAs(types.Timeouts{HTTPTotal: &timeout}, nil, emptyProxy)
	if err != nil {
		e.Logger.Crit("failed to update timeouts and CAs for fetcher: %v", err)
		return
	}

	// (Re)Fetch the config if the cache is unreadable.
	cfg, err = e.fetchProviderConfig()
	if err == errors.ErrEmpty {
		// Continue if the provider config was empty as we want to write an empty
		// cache config for use by other stages.
		cfg = emptyConfig
		e.Logger.Info("%v: provider config was empty, continuing with empty cache config", err)
	} else if err != nil {
		e.Logger.Warning("failed to fetch config: %s", err)
		return
	}

	// Update the http client to use the timeouts and CAs from the newly fetched
	// config
	err = e.Fetcher.UpdateHttpTimeoutsAndCAs(cfg.Ignition.Timeouts, cfg.Ignition.Security.TLS.CertificateAuthorities, cfg.Ignition.Proxy)
	if err != nil {
		e.Logger.Crit("failed to update timeouts and CAs for fetcher: %v", err)
		return
	}

	err = e.Fetcher.RewriteCAsWithDataUrls(cfg.Ignition.Security.TLS.CertificateAuthorities)
	if err != nil {
		e.Logger.Crit("error handling CAs: %v", err)
		return
	}

	rpt := validate.Validate(cfg, "json")
	e.logReport(rpt)
	if rpt.IsFatal() {
		err = errors.ErrInvalid
		e.Logger.Crit("merging configs resulted in an invalid config")
		return
	}

	// Populate the config cache.
	b, err := json.Marshal(cfg)
	if err != nil {
		e.Logger.Crit("failed to marshal cached config: %v", err)
		return
	}
	if err = renameio.WriteFile(e.ConfigCache, b, 0640); err != nil {
		e.Logger.Crit("failed to write cached config: %v", err)
		return
	}

	return
}

// fetchProviderConfig returns the externally-provided configuration. It first
// checks to see if the command-line option is present. If so, it uses that
// source for the configuration. If the command-line option is not present, it
// checks for a user config in the system config dir. If that is also missing,
// it checks the config engine's provider. An error is returned if the provider
// is unavailable. This will also render the config (see renderConfig) before
// returning.
func (e *Engine) fetchProviderConfig() (types.Config, error) {
	// note this is an array because iteration order is important; see comment
	// block just above
	fetchers := []struct {
		name      string
		fetchFunc providers.FuncFetchConfig
	}{
		{"cmdline", cmdline.FetchConfig},
		{"system", system.FetchConfig},
		{e.PlatformConfig.Name(), e.PlatformConfig.FetchFunc()},
	}
	var cfg types.Config
	var r report.Report
	var err error
	var providerKey string
	for _, fetcher := range fetchers {
		cfg, r, err = fetcher.fetchFunc(e.Fetcher)
		if err != providers.ErrNoProvider {
			// successful, or failed on another error
			providerKey = fetcher.name
			break
		}
	}

	e.logReport(r)
	if err != nil {
		return types.Config{}, err
	}

	e.fetchedConfigs = append(e.fetchedConfigs, fetchedConfig{
		kind:       "user",
		source:     providerKey,
		referenced: false,
	})

	// Replace the HTTP client in the fetcher to be configured with the
	// timeouts of the config
	err = e.Fetcher.UpdateHttpTimeoutsAndCAs(cfg.Ignition.Timeouts, cfg.Ignition.Security.TLS.CertificateAuthorities, cfg.Ignition.Proxy)
	if err != nil {
		return types.Config{}, err
	}

	return e.renderConfig(cfg)
}

// renderConfig evaluates "ignition.config.replace" and "ignition.config.append"
// in the given config and returns the result. If "ignition.config.replace" is
// set, the referenced and evaluted config will be returned. Otherwise, if
// "ignition.config.append" is set, each of the referenced configs will be
// evaluated and appended to the provided config. If neither option is set, the
// provided config will be returned unmodified. An updated fetcher will be
// returned with any new timeouts set.
func (e *Engine) renderConfig(cfg types.Config) (types.Config, error) {
	if cfgRef := cfg.Ignition.Config.Replace; cfgRef.Source != nil {
		newCfg, err := e.fetchReferencedConfig(cfgRef)
		if err != nil {
			return types.Config{}, err
		}

		// Replace the HTTP client in the fetcher to be configured with the
		// timeouts of the new config
		err = e.Fetcher.UpdateHttpTimeoutsAndCAs(newCfg.Ignition.Timeouts, newCfg.Ignition.Security.TLS.CertificateAuthorities, newCfg.Ignition.Proxy)
		if err != nil {
			return types.Config{}, err
		}

		return e.renderConfig(newCfg)
	}

	appendedCfg := cfg
	for _, cfgRef := range cfg.Ignition.Config.Merge {
		newCfg, err := e.fetchReferencedConfig(cfgRef)
		if err != nil {
			return types.Config{}, err
		}

		// Merge the old config with the new config before the new config has
		// been rendered, so we can use the new config's timeouts and CAs when
		// fetching more configs.
		cfgForFetcherSettings := latest.Merge(appendedCfg, newCfg)
		err = e.Fetcher.UpdateHttpTimeoutsAndCAs(cfgForFetcherSettings.Ignition.Timeouts, cfgForFetcherSettings.Ignition.Security.TLS.CertificateAuthorities, cfgForFetcherSettings.Ignition.Proxy)
		if err != nil {
			return types.Config{}, err
		}

		newCfg, err = e.renderConfig(newCfg)
		if err != nil {
			return types.Config{}, err
		}

		appendedCfg = latest.Merge(appendedCfg, newCfg)
	}
	return appendedCfg, nil
}

// fetchReferencedConfig fetches and parses the requested config.
// cfgRef.Source must not be nil
func (e *Engine) fetchReferencedConfig(cfgRef types.Resource) (types.Config, error) {
	// this is also already checked at validation time
	if cfgRef.Source == nil {
		e.Logger.Crit("invalid referenced config: %v", errors.ErrSourceRequired)
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
	rawCfg, err := e.Fetcher.FetchToBuffer(*u, resource.FetchOptions{
		Headers:     headers,
		Compression: compression,
	})
	if err != nil {
		return types.Config{}, err
	}

	hash := sha512.Sum512(rawCfg)
	if u.Scheme != "data" {
		e.Logger.Debug("fetched referenced config at %s with SHA512: %s", *cfgRef.Source, hex.EncodeToString(hash[:]))
	} else {
		// data url's might contain secrets
		e.Logger.Debug("fetched referenced config from data url with SHA512: %s", hex.EncodeToString(hash[:]))
	}

	e.fetchedConfigs = append(e.fetchedConfigs, fetchedConfig{
		kind:       "user",
		source:     u.Path,
		referenced: true,
	})

	if err := util.AssertValid(cfgRef.Verification, rawCfg); err != nil {
		return types.Config{}, err
	}

	cfg, r, err := config.Parse(rawCfg)
	e.logReport(r)
	if err != nil {
		return types.Config{}, err
	}

	return cfg, nil
}

func SignalNeedNet() error {
	if err := executil.MkdirForFile(needNetPath); err != nil {
		return err
	}
	if f, err := os.Create(needNetPath); err != nil {
		return err
	} else {
		f.Close()
	}
	return nil
}

func (e Engine) logReport(r report.Report) {
	for _, entry := range r.Entries {
		switch entry.Kind {
		case report.Error:
			e.Logger.Crit("%v", entry)
		case report.Warn:
			e.Logger.Warning("%v", entry)
		case report.Info:
			e.Logger.Info("%v", entry)
		}
	}
}
