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
		name: "azure",
		flags: map[string]string{
			"provider": "azure",
		},
	})
	configs.Register(Config{
		name: "cloudsigma",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "cloudstack",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "digitalocean",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "brightbox",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "openstack",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "ec2",
		flags: map[string]string{
			"provider":       "ec2",
			"online-timeout": "0",
		},
	})
	configs.Register(Config{
		name: "exoscale",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "gce",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "hyperv",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "niftycloud",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "packet",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "pxe",
		flags: map[string]string{
			"provider": "cmdline",
		},
	})
	configs.Register(Config{
		name: "rackspace",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "rackspace-onmetal",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "vagrant",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "vmware",
		flags: map[string]string{
			"provider": "vmware",
		},
	})
	configs.Register(Config{
		name: "xendom0",
		flags: map[string]string{
			"provider": "noop",
		},
	})
	configs.Register(Config{
		name: "interoute",
		flags: map[string]string{
			"provider": "noop",
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
