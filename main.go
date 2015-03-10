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
	"time"

	"github.com/coreos/ignition/exec"
	"github.com/coreos/ignition/providers"

	"github.com/coreos/ignition/Godeps/_workspace/src/github.com/coreos/go-semver/semver"
)

const versionString = "0.0.0+git"

var version = *semver.Must(semver.NewVersion(versionString))

func main() {
	flags := struct {
		fetchTimeout time.Duration
		providers    providers.List
		root         string
		version      bool
	}{}

	flag.DurationVar(&flags.fetchTimeout, "fetchtimeout", exec.DefaultFetchTimeout, "")
	flag.Var(&flags.providers, "provider", fmt.Sprintf("provider of config. can be specified multiple times. %v", providers.Names()))
	flag.StringVar(&flags.root, "root", "/", "root of the filesystem")
	flag.BoolVar(&flags.version, "version", false, "print the version and exit")

	flag.Parse()

	if flags.version {
		fmt.Printf("ignition %s\n", versionString)
		return
	}
}
