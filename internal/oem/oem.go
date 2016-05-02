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
	"fmt"
	"time"

	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/providers"
	"github.com/coreos/ignition/internal/providers/azure"
	"github.com/coreos/ignition/internal/providers/cmdline"
	"github.com/coreos/ignition/internal/providers/ec2"
	"github.com/coreos/ignition/internal/providers/gce"
	"github.com/coreos/ignition/internal/providers/noop"
	"github.com/coreos/ignition/internal/providers/vmware"
	"github.com/coreos/ignition/internal/registry"

	"github.com/vincent-petithory/dataurl"
)

// Config represents a set of command line flags that map to a particular OEM.
type Config struct {
	name     string
	flags    map[string]string
	provider providers.ProviderCreator
	config   types.Config
}

func (c Config) Name() string {
	return c.name
}

func (c Config) Flags() map[string]string {
	return c.flags
}

func (c Config) Provider() providers.ProviderCreator {
	return c.provider
}

func (c Config) Config() types.Config {
	return c.config
}

var configs = registry.Create("oem configs")

func Get(name string) (config Config, ok bool) {
	config, ok = configs.Get(name).(Config)
	return
}

func MustGet(name string) Config {
	if config, ok := Get(name); ok {
		return config
	} else {
		panic(fmt.Sprintf("invalid OEM name %q provided", name))
	}
}

func Names() (names []string) {
	return configs.Names()
}

func init() {
	configs.Register(Config{
		name:     "azure",
		provider: azure.Creator{},
	})
	configs.Register(Config{
		name:     "cloudsigma",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "cloudstack",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "digitalocean",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "brightbox",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "openstack",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "ec2",
		provider: ec2.Creator{},
		flags: map[string]string{
			"online-timeout": "0",
		},
		config: types.Config{
			Systemd: types.Systemd{
				Units: flatten(
					sshKeys(),
					etcdConfigs(1200*time.Millisecond),
					userCloudInit("EC2", "ec2-compat"),
					configDrive(),
				),
			},
			Storage: types.Storage{
				Files: []types.File{oemId("ID=ec2\nNAME=Amazon EC2\nHOME_URL=http://aws.amazon.com/ec2/\nBUG_REPORT_URL=https://github.com/coreos/bugs/issues")},
			},
		},
	})
	configs.Register(Config{
		name:     "exoscale",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "gce",
		provider: gce.Creator{},
		config: types.Config{
			Systemd: types.Systemd{
				Units: flatten(
					sshKeys(),
				),
			},
		},
	})
	configs.Register(Config{
		name:     "hyperv",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "niftycloud",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "packet",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "pxe",
		provider: cmdline.Creator{},
	})
	configs.Register(Config{
		name:     "rackspace",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "rackspace-onmetal",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "vagrant",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "vmware",
		provider: vmware.Creator{},
	})
	configs.Register(Config{
		name:     "xendom0",
		provider: noop.Creator{},
	})
	configs.Register(Config{
		name:     "interoute",
		provider: noop.Creator{},
	})
}

func flatten(uss ...[]types.SystemdUnit) []types.SystemdUnit {
	var units []types.SystemdUnit
	for _, us := range uss {
		units = append(units, us...)
	}
	return units
}

func oemId(contents string) types.File {
	return types.File{
		Filesystem: "root",
		Path:       "/etc/oem-release",
		Mode:       0644,
		Contents: types.FileContents{Source: types.Url{
			Scheme: "data",
			Opaque: "," + dataurl.EscapeString(contents),
		}},
	}
}

func sshKeys() []types.SystemdUnit {
	return []types.SystemdUnit{{Name: "coreos-metadata-sshkeys@.service", Enable: true}}
}

func etcdConfigs(election_timeout time.Duration) []types.SystemdUnit {
	dropIn := types.SystemdUnitDropIn{
		Name:     "10-oem.conf",
		Contents: fmt.Sprintf("[Service]\nEnvironment=ETCD_PEER_ELECTION_TIMEOUT=%s", election_timeout/time.Millisecond),
	}
	return []types.SystemdUnit{
		{Name: "etcd.service", DropIns: []types.SystemdUnitDropIn{dropIn}},
		{Name: "etcd2.service", DropIns: []types.SystemdUnitDropIn{dropIn}},
	}
}

func userCloudInit(name string, oem string) []types.SystemdUnit {
	return []types.SystemdUnit{{
		Name:     "oem-cloudinit.service",
		Enable:   true,
		Contents: fmt.Sprintf("[Unit]\nDescription=Cloudinit from %s metadata\n\n[Service]\nType=oneshot\nExecStart=/usr/bin/coreos-cloudinit --oem=%s\n\n[Install]\nWantedBy=multi-user.target", name, oem),
	}}
}

func configDrive() []types.SystemdUnit {
	return []types.SystemdUnit{
		{Name: "user-configdrive.service", Mask: true},
		{Name: "user-configvirtfs.service", Mask: true},
	}
}
