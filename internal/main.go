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

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/ignition/v2/config"
	"github.com/coreos/ignition/v2/internal/apply"
	"github.com/coreos/ignition/v2/internal/exec"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	_ "github.com/coreos/ignition/v2/internal/register"
	"github.com/coreos/ignition/v2/internal/state"
	"github.com/coreos/ignition/v2/internal/version"
	"github.com/spf13/pflag"
)

func main() {
	switch filepath.Base(os.Args[0]) {
	case "ignition-apply":
		ignitionApplyMain()
	case "ignition-rmcfg":
		ignitionRmCfgMain()
	default:
		// assume regular Ignition
		ignitionMain()
	}
}

func ignitionMain() {
	err := func() (err error) {
		flags := struct {
			configCache  string
			fetchTimeout time.Duration
			needNet      string
			platform     platform.Name
			root         string
			stage        stages.Name
			stateFile    string
			version      bool
			logToStdout  bool
		}{}

		flag.StringVar(&flags.configCache, "config-cache", "/run/ignition.json", "where to cache the config")
		flag.DurationVar(&flags.fetchTimeout, "fetch-timeout", exec.DefaultFetchTimeout, "initial duration for which to wait for config")
		flag.StringVar(&flags.needNet, "neednet", "/run/ignition/neednet", "flag file to write from fetch-offline if networking is needed")
		flag.Var(&flags.platform, "platform", fmt.Sprintf("current platform. %v", platform.Names()))
		flag.StringVar(&flags.root, "root", "/", "root of the filesystem")
		flag.Var(&flags.stage, "stage", fmt.Sprintf("execution stage. %v", stages.Names()))
		flag.StringVar(&flags.stateFile, "state-file", "/run/ignition/state", "where to store internal state")
		flag.BoolVar(&flags.version, "version", false, "print the version and exit")
		flag.BoolVar(&flags.logToStdout, "log-to-stdout", false, "log to stdout instead of the system log when set")

		flag.Parse()

		if flags.version {
			fmt.Printf("%s\n", version.String)
			return nil
		}

		if flags.platform == "" {
			_, _ = fmt.Fprint(os.Stderr, "'--platform' must be provided\n")
			return errors.New("platform not provided")
		}

		if flags.stage == "" {
			_, _ = fmt.Fprint(os.Stderr, "'--stage' must be provided\n")
			return errors.New("stage not provided")
		}

		logger := log.New(flags.logToStdout)
		defer func() {
			err = errors.Join(err, logger.Close())
		}()

		logger.Info("%s", version.String)
		logger.Info("Stage: %v", flags.stage)

		platformConfig := platform.MustGet(flags.platform.String())
		fetcher, fetcherErr := platformConfig.NewFetcher(&logger)
		if fetcherErr != nil {
			logger.Crit("failed to generate fetcher: %s", fetcherErr)
			return fetcherErr
		}
		st, stateErr := state.Load(flags.stateFile)
		if stateErr != nil {
			logger.Crit("reading state: %s", stateErr)
			return stateErr
		}
		engine := exec.Engine{
			Root:           flags.root,
			FetchTimeout:   flags.fetchTimeout,
			Logger:         &logger,
			NeedNet:        flags.needNet,
			ConfigCache:    flags.configCache,
			PlatformConfig: platformConfig,
			Fetcher:        &fetcher,
			State:          &st,
		}

		runErr := engine.Run(flags.stage.String())
		if statusErr := engine.PlatformConfig.Status(flags.stage.String(), *engine.Fetcher, runErr); statusErr != nil {
			logger.Err("POST Status error: %v", statusErr.Error())
		}
		if runErr != nil {
			logger.Crit("Ignition failed: %v", runErr.Error())
			return runErr
		}
		if err := engine.State.Save(flags.stateFile); err != nil {
			logger.Crit("writing state: %v", err)
			return err
		}
		logger.Info("Ignition finished successfully")
		return nil
	}()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func ignitionApplyMain() {
	err := func() (err error) {
		printVersion := false
		flags := apply.Flags{}
		pflag.BoolVar(&printVersion, "version", false, "print the version of ignition-apply")
		pflag.StringVar(&flags.Root, "root", "/", "root of the filesystem")
		pflag.BoolVar(&flags.IgnoreUnsupported, "ignore-unsupported", false, "ignore unsupported config sections")
		pflag.BoolVar(&flags.Offline, "offline", false, "error out if config references remote resources")
		pflag.Usage = func() {
			_, _ = fmt.Fprintf(pflag.CommandLine.Output(), "Usage: %s [options] config.ign\n", os.Args[0])
			_, _ = fmt.Fprintf(pflag.CommandLine.Output(), "Options:\n")
			pflag.PrintDefaults()
		}
		pflag.Parse()

		if printVersion {
			fmt.Printf("%s\n", version.String)
			return nil
		}

		if pflag.NArg() != 1 {
			pflag.Usage()
			return errors.New("no config provided")
		}
		cfgArg := pflag.Arg(0)

		logger := log.New(true)
		defer func() {
			err = errors.Join(err, logger.Close())
		}()

		logger.Info("%s", version.String)

		var blob []byte
		var readErr error
		if cfgArg == "-" {
			blob, readErr = io.ReadAll(os.Stdin)
		} else {
			// XXX: could in the future support fetching directly from HTTP(S) + `-checksum|-insecure` ?
			blob, readErr = os.ReadFile(cfgArg)
		}
		if readErr != nil {
			logger.Crit("couldn't read config: %v", readErr)
			return readErr
		}

		cfg, rpt, parseErr := config.Parse(blob)
		logger.LogReport(rpt)
		if rpt.IsFatal() || parseErr != nil {
			logger.Crit("couldn't parse config: %v", parseErr)
			return parseErr
		}

		if applyErr := apply.Run(cfg, flags, &logger); applyErr != nil {
			logger.Crit("failed to apply: %v", applyErr)
			return applyErr
		}
		return nil
	}()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func ignitionRmCfgMain() {
	err := func() (err error) {
		flags := struct {
			logToStdout bool
			platform    string
			version     bool
		}{}
		pflag.StringVar(&flags.platform, "platform", "", fmt.Sprintf("current platform. %v", platform.Names()))
		pflag.BoolVar(&flags.logToStdout, "log-to-stdout", false, "log to stdout instead of the system log")
		pflag.BoolVar(&flags.version, "version", false, "print the version and exit")
		pflag.Usage = func() {
			_, _ = fmt.Fprintf(pflag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
			_, _ = fmt.Fprintf(pflag.CommandLine.Output(), "Options:\n")
			pflag.PrintDefaults()
		}
		pflag.Parse()

		if flags.version {
			fmt.Printf("%s\n", version.String)
			return nil
		}

		if pflag.NArg() != 0 {
			pflag.Usage()
			return errors.New("command takes no arguments")
		}

		if flags.platform == "" {
			_, _ = fmt.Fprint(os.Stderr, "'--platform' must be provided\n")
			return errors.New("platform not provided")
		}

		logger := log.New(flags.logToStdout)
		defer func() {
			err = errors.Join(err, logger.Close())
		}()

		logger.Info("%s", version.String)

		platformConfig := platform.MustGet(flags.platform)
		fetcher, fetcherErr := platformConfig.NewFetcher(&logger)
		if fetcherErr != nil {
			logger.Crit("failed to generate fetcher: %s", fetcherErr)
			return fetcherErr
		}

		if delErr := platformConfig.DelConfig(&fetcher); delErr != nil {
			logger.Crit("couldn't delete config: %s", delErr)
			return delErr
		}

		logger.Info("Successfully deleted config")
		return nil
	}()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
