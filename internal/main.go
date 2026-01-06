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
	flags := struct {
		configCache         string
		fetchTimeout        time.Duration
		generateCloudConfig bool
		needNet             string
		platform            platform.Name
		root                string
		stage               stages.Name
		stateFile           string
		version             bool
		logToStdout         bool
	}{}

	flag.StringVar(&flags.configCache, "config-cache", "/run/ignition.json", "where to cache the config")
	flag.DurationVar(&flags.fetchTimeout, "fetch-timeout", exec.DefaultFetchTimeout, "initial duration for which to wait for config")
	flag.StringVar(&flags.needNet, "neednet", "/run/ignition/neednet", "flag file to write from fetch-offline if networking is needed")
	flag.BoolVar(&flags.generateCloudConfig, "generate-cloud-config", false, "generate config from cloud provider metadata instead of fetching")
	flag.Var(&flags.platform, "platform", fmt.Sprintf("current platform. %v", platform.Names()))
	flag.StringVar(&flags.root, "root", "/", "root of the filesystem")
	flag.Var(&flags.stage, "stage", fmt.Sprintf("execution stage. %v", stages.Names()))
	flag.StringVar(&flags.stateFile, "state-file", "/run/ignition/state", "where to store internal state")
	flag.BoolVar(&flags.version, "version", false, "print the version and exit")
	flag.BoolVar(&flags.logToStdout, "log-to-stdout", false, "log to stdout instead of the system log when set")

	flag.Parse()

	// Never allow cloud config generation during fetch-offline stage (no networking)
	if flags.stage == "fetch" && flags.platform == "azure" {
		flags.generateCloudConfig = true
	}

	if flags.version {
		fmt.Printf("%s\n", version.String)
		return
	}

	if flags.platform == "" {
		fmt.Fprint(os.Stderr, "'--platform' must be provided\n")
		os.Exit(2)
	}

	if flags.stage == "" {
		fmt.Fprint(os.Stderr, "'--stage' must be provided\n")
		os.Exit(2)
	}

	logger := log.New(flags.logToStdout)
	defer logger.Close()

	logger.Info("%s", version.String)
	logger.Info("Stage: %v", flags.stage)
	logger.Info("Platform: %v", flags.platform)
	logger.Info("GenerateCloudConfig: %v", flags.generateCloudConfig)

	platformConfig := platform.MustGet(flags.platform.String())
	fetcher, err := platformConfig.NewFetcher(&logger)
	if err != nil {
		logger.Crit("failed to generate fetcher: %s", err)
		os.Exit(3)
	}
	state, err := state.Load(flags.stateFile)
	if err != nil {
		logger.Crit("reading state: %s", err)
		os.Exit(3)
	}
	engine := exec.Engine{
		Root:                flags.root,
		FetchTimeout:        flags.fetchTimeout,
		GenerateCloudConfig: flags.generateCloudConfig,
		Logger:              &logger,
		NeedNet:             flags.needNet,
		ConfigCache:         flags.configCache,
		PlatformConfig:      platformConfig,
		Fetcher:             &fetcher,
		State:               &state,
	}

	err = engine.Run(flags.stage.String())
	if statusErr := engine.PlatformConfig.Status(flags.stage.String(), *engine.Fetcher, err); statusErr != nil {
		logger.Err("POST Status error: %v", statusErr.Error())
	}
	if err != nil {
		logger.Crit("Ignition failed: %v", err.Error())
		os.Exit(1)
	}
	if err := engine.State.Save(flags.stateFile); err != nil {
		logger.Crit("writing state: %v", err)
		os.Exit(1)
	}
	logger.Info("Ignition finished successfully")
}

func ignitionApplyMain() {
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
		return
	}

	if pflag.NArg() != 1 {
		pflag.Usage()
		os.Exit(1)
	}
	cfgArg := pflag.Arg(0)

	logger := log.New(true)
	defer logger.Close()

	logger.Info("%s", version.String)

	var blob []byte
	var err error
	if cfgArg == "-" {
		blob, err = io.ReadAll(os.Stdin)
	} else {
		// XXX: could in the future support fetching directly from HTTP(S) + `-checksum|-insecure` ?
		blob, err = os.ReadFile(cfgArg)
	}
	if err != nil {
		logger.Crit("couldn't read config: %v", err)
		os.Exit(1)
	}

	cfg, rpt, err := config.Parse(blob)
	logger.LogReport(rpt)
	if rpt.IsFatal() || err != nil {
		logger.Crit("couldn't parse config: %v", err)
		os.Exit(1)
	}

	if err := apply.Run(cfg, flags, &logger); err != nil {
		logger.Crit("failed to apply: %v", err)
		os.Exit(1)
	}
}

func ignitionRmCfgMain() {
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
		return
	}

	if pflag.NArg() != 0 {
		pflag.Usage()
		os.Exit(2)
	}

	if flags.platform == "" {
		fmt.Fprint(os.Stderr, "'--platform' must be provided\n")
		os.Exit(2)
	}

	logger := log.New(flags.logToStdout)
	defer logger.Close()

	logger.Info("%s", version.String)

	platformConfig := platform.MustGet(flags.platform)
	fetcher, err := platformConfig.NewFetcher(&logger)
	if err != nil {
		logger.Crit("failed to generate fetcher: %s", err)
		os.Exit(3)
	}

	if err := platformConfig.DelConfig(&fetcher); err != nil {
		logger.Crit("couldn't delete config: %s", err)
		os.Exit(1)
	}

	logger.Info("Successfully deleted config")
}
