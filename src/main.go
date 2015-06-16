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
	"os"
	"time"

	"github.com/coreos/ignition/src/exec"
	"github.com/coreos/ignition/src/exec/stages"
	_ "github.com/coreos/ignition/src/exec/stages/prepivot"
	_ "github.com/coreos/ignition/src/exec/stages/storage"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/oem"
	"github.com/coreos/ignition/src/providers"
	_ "github.com/coreos/ignition/src/providers/cmdline"
	_ "github.com/coreos/ignition/src/providers/file"

	"github.com/coreos/ignition/Godeps/_workspace/src/github.com/coreos/go-semver/semver"
)

const versionString = "0.0.0+git"

var version = *semver.Must(semver.NewVersion(versionString))

func main() {
	flags := struct {
		clearCache   bool
		configCache  string
		fetchTimeout time.Duration
		oem          oem.Name
		providers    providers.List
		root         string
		stage        stages.Name
		version      bool
	}{}

	flag.BoolVar(&flags.clearCache, "clear-cache", false, "clear any cached config")
	flag.StringVar(&flags.configCache, "config-cache", "/tmp/ignition.json", "where to cache the config")
	flag.DurationVar(&flags.fetchTimeout, "fetchtimeout", exec.DefaultFetchTimeout, "")
	flag.Var(&flags.oem, "oem", fmt.Sprintf("current oem. %v", oem.Names()))
	flag.Var(&flags.providers, "provider", fmt.Sprintf("provider of config. can be specified multiple times. %v", providers.Names()))
	flag.StringVar(&flags.root, "root", "/", "root of the filesystem")
	flag.Var(&flags.stage, "stage", fmt.Sprintf("execution stage. %v", stages.Names()))
	flag.BoolVar(&flags.version, "version", false, "print the version and exit")

	flag.Parse()

	if config, ok := oem.Get(flags.oem.String()); ok {
		for k, v := range config.Flags() {
			flag.Set(k, v)
		}
	}

	if flags.version {
		fmt.Printf("ignition %s\n", versionString)
		return
	}

	if flags.stage == "" {
		fmt.Fprint(os.Stderr, "'--stage' must be provided\n")
		os.Exit(2)
	}

	logger := log.New()
	defer logger.Close()

	if flags.clearCache {
		if err := os.Remove(flags.configCache); err != nil {
			logger.Err("unable to clear cache: %v", err)
		}
	}

	engine := exec.Engine{
		Root:         flags.root,
		FetchTimeout: flags.fetchTimeout,
		Logger:       logger,
		ConfigCache:  flags.configCache,
	}.Init()
	for _, name := range flags.providers {
		engine.AddProvider(providers.Get(name).Create(logger))
	}

	if !engine.Run(flags.stage.String()) {
		os.Exit(1)
	}
}
