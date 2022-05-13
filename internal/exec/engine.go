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
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/journal"
	"github.com/coreos/ignition/v2/config/shared/errors"
	latest "github.com/coreos/ignition/v2/config/v3_4_experimental"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	executil "github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers"
	"github.com/coreos/ignition/v2/internal/providers/cmdline"
	"github.com/coreos/ignition/v2/internal/providers/system"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/state"

	"github.com/coreos/vcontext/report"
	"github.com/coreos/vcontext/validate"
	"github.com/google/renameio/v2"
)

const (
	DefaultFetchTimeout = 2 * time.Minute
	// This variable will help to identify ignition journal messages
	// related to the user/base config.
	ignitionFetchedConfigMsgId = "57124006b5c94805b77ce473e92a8aeb"
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
	NeedNet        string
	Root           string
	PlatformConfig platform.Config
	Fetcher        *resource.Fetcher
	State          *state.State
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
	e.Logger.LogReport(r)
	if err != nil && err != providers.ErrNoProvider {
		e.Logger.Crit("failed to acquire system base config: %v", err)
		return err
	} else if err == nil {
		e.State.FetchedConfigs = append(e.State.FetchedConfigs, state.FetchedConfig{
			Kind:       "base",
			Source:     "system",
			Referenced: false,
		})
	}

	// We special-case the fetch-offline stage a bit here: we want to be able
	// to handle the case where the provider itself requires networking.
	if stageName == "fetch-offline" {
		e.Fetcher.Offline = true
	}

	// Run the platform config's Init function pre-config fetch
	// to perform any additional fetcher configuration  e.x.
	// configuring the S3RegionHint when running on AWS.
	err = e.PlatformConfig.InitFunc()(e.Fetcher)
	if err == resource.ErrNeedNet && stageName == "fetch-offline" {
		err = e.signalNeedNet()
		if err != nil {
			e.Logger.Crit("failed to signal neednet: %v", err)
		}
		return err
	} else if err != nil {
		return fmt.Errorf("initializing platform config: %v", err)
	}

	cfg, err := e.acquireConfig(stageName)
	if err == resource.ErrNeedNet && stageName == "fetch-offline" {
		err = e.signalNeedNet()
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

	fullConfig := latest.Merge(baseConfig, latest.Merge(systemBaseConfig, cfg))
	err = stages.Get(stageName).Create(e.Logger, e.Root, *e.Fetcher, e.State).Run(fullConfig)
	if err == resource.ErrNeedNet && stageName == "fetch-offline" {
		err = e.signalNeedNet()
		if err != nil {
			e.Logger.Crit("failed to signal neednet: %v", err)
		}
		// fall through
	}
	if err != nil {
		// e.Logger could be nil
		fmt.Fprintf(os.Stderr, "%s failed\n", stageName)
		tmp, jsonerr := json.MarshalIndent(fullConfig, "", "  ")
		if jsonerr != nil {
			// Nothing else to do with this error
			fmt.Fprintf(os.Stderr, "Could not marshal full config: %v\n", jsonerr)
		} else {
			fmt.Fprintf(os.Stderr, "Full config:\n%s\n", string(tmp))
		}
		return err
	}
	e.Logger.Info("%s passed", stageName)
	return nil
}

// logStructuredJournalEntry logs information related to
// a user/base config into the systemd journal log.
func logStructuredJournalEntry(cfgInfo state.FetchedConfig) error {
	ignitionInfo := map[string]string{
		"IGNITION_CONFIG_TYPE":       cfgInfo.Kind,
		"IGNITION_CONFIG_SRC":        cfgInfo.Source,
		"IGNITION_CONFIG_REFERENCED": strconv.FormatBool(cfgInfo.Referenced),
		"MESSAGE_ID":                 ignitionFetchedConfigMsgId,
	}
	referenced := ""
	if cfgInfo.Referenced {
		referenced = "referenced "
	}
	msg := fmt.Sprintf("fetched %s%s config from %q", referenced, cfgInfo.Kind, cfgInfo.Source)
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
			for _, cfgInfo := range e.State.FetchedConfigs {
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
	} else if err == resource.ErrNeedNet {
		e.Logger.Info("failed to fetch config: %s", err)
		return
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
	e.Logger.LogReport(rpt)
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

	e.Logger.LogReport(r)
	if err != nil {
		return types.Config{}, err
	}

	e.State.FetchedConfigs = append(e.State.FetchedConfigs, state.FetchedConfig{
		Kind:       "user",
		Source:     providerKey,
		Referenced: false,
	})

	// Replace the HTTP client in the fetcher to be configured with the
	// timeouts of the config
	err = e.Fetcher.UpdateHttpTimeoutsAndCAs(cfg.Ignition.Timeouts, cfg.Ignition.Security.TLS.CertificateAuthorities, cfg.Ignition.Proxy)
	if err != nil {
		return types.Config{}, err
	}

	configFetcher := ConfigFetcher{
		Logger:  e.Logger,
		Fetcher: e.Fetcher,
		State:   e.State,
	}

	return configFetcher.RenderConfig(cfg)
}

func (e *Engine) signalNeedNet() error {
	if err := executil.MkdirForFile(e.NeedNet); err != nil {
		return err
	}
	if f, err := os.Create(e.NeedNet); err != nil {
		return err
	} else {
		f.Close()
	}
	return nil
}
