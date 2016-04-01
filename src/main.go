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
	_ "github.com/coreos/ignition/src/exec/stages/disks"
	_ "github.com/coreos/ignition/src/exec/stages/files"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/oem"
	"github.com/coreos/ignition/src/version"
)

func main() {
	flags := struct {
		clearCache    bool
		configCache   string
		onlineTimeout time.Duration
		oem           oem.Name
		root          string
		stage         stages.Name
		version       bool
	}{}

	flag.BoolVar(&flags.clearCache, "clear-cache", false, "clear any cached config")
	flag.StringVar(&flags.configCache, "config-cache", "/run/ignition.json", "where to cache the config")
	flag.DurationVar(&flags.onlineTimeout, "online-timeout", exec.DefaultOnlineTimeout, "how long to wait for a provider to come online")
	flag.Var(&flags.oem, "oem", fmt.Sprintf("current oem. %v", oem.Names()))
	flag.StringVar(&flags.root, "root", "/", "root of the filesystem")
	flag.Var(&flags.stage, "stage", fmt.Sprintf("execution stage. %v", stages.Names()))
	flag.BoolVar(&flags.version, "version", false, "print the version and exit")

	flag.Parse()

	if flags.version {
		fmt.Printf("%s\n", version.String)
		return
	}

	if flags.oem == "" {
		fmt.Fprint(os.Stderr, "'--oem' must be provided\n")
		os.Exit(2)
	}

	if flags.stage == "" {
		fmt.Fprint(os.Stderr, "'--stage' must be provided\n")
		os.Exit(2)
	}

	for k, v := range oem.MustGet(flags.oem.String()).Flags() {
		if err := flag.Set(k, v); err != nil {
			panic(err)
		}
	}

	logger := log.New()
	defer logger.Close()

	logger.Info(version.String)

	if flags.clearCache {
		if err := os.Remove(flags.configCache); err != nil {
			logger.Err("unable to clear cache: %v", err)
		}
	}

	engine := exec.Engine{
		Root:          flags.root,
		OnlineTimeout: flags.onlineTimeout,
		Logger:        &logger,
		ConfigCache:   flags.configCache,
		Provider:      oem.MustGet(flags.oem.String()).Provider().Create(&logger),
	}

	if !engine.Run(flags.stage.String()) {
		os.Exit(1)
	}
}
