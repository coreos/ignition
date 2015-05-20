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

package oem

import (
	"github.com/coreos/ignition/src/registry"
)

// Config represents a set of command line flags that map to a particular OEM.
type Config struct {
	name  string
	flags map[string]string
}

func (c Config) Name() string {
	return c.name
}

func (c Config) Flags() map[string]string {
	return c.flags
}

var configs = registry.Create("oem configs")

func init() {
	configs.Register(Config{
		name: "pxe",
		flags: map[string]string{
			"provider": "cmdline",
		},
	})
}

func Get(name string) (config Config, ok bool) {
	config, ok = configs.Get(name).(Config)
	return
}

func Names() (names []string) {
	return configs.Names()
}
