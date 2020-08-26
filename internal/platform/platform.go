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

package platform

import (
	"fmt"

	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/providers"
	"github.com/coreos/ignition/v2/internal/providers/aliyun"
	"github.com/coreos/ignition/v2/internal/providers/aws"
	"github.com/coreos/ignition/v2/internal/providers/azure"
	"github.com/coreos/ignition/v2/internal/providers/azurestack"
	"github.com/coreos/ignition/v2/internal/providers/cloudstack"
	"github.com/coreos/ignition/v2/internal/providers/digitalocean"
	"github.com/coreos/ignition/v2/internal/providers/exoscale"
	"github.com/coreos/ignition/v2/internal/providers/file"
	"github.com/coreos/ignition/v2/internal/providers/gcp"
	"github.com/coreos/ignition/v2/internal/providers/ibmcloud"
	"github.com/coreos/ignition/v2/internal/providers/noop"
	"github.com/coreos/ignition/v2/internal/providers/openstack"
	"github.com/coreos/ignition/v2/internal/providers/packet"
	"github.com/coreos/ignition/v2/internal/providers/qemu"
	"github.com/coreos/ignition/v2/internal/providers/virtualbox"
	"github.com/coreos/ignition/v2/internal/providers/vmware"
	"github.com/coreos/ignition/v2/internal/providers/vultr"
	"github.com/coreos/ignition/v2/internal/providers/zvm"
	"github.com/coreos/ignition/v2/internal/registry"
	"github.com/coreos/ignition/v2/internal/resource"
)

// Config represents a set of options that map to a particular platform.
type Config struct {
	name       string
	fetch      providers.FuncFetchConfig
	init       providers.FuncInit
	newFetcher providers.FuncNewFetcher
	status     providers.FuncPostStatus
}

func (c Config) Name() string {
	return c.name
}

func (c Config) FetchFunc() providers.FuncFetchConfig {
	return c.fetch
}

func (c Config) NewFetcherFunc() providers.FuncNewFetcher {
	if c.newFetcher != nil {
		return c.newFetcher
	}
	return func(l *log.Logger) (resource.Fetcher, error) {
		return resource.Fetcher{
			Logger: l,
		}, nil
	}
}

// InitFunc returns a function that performs additional fetcher
// configuration post-config fetch. This ensures that networking
// is already available if a platform needs to reach out to the
// metadata service to fetch additional options / data.
func (c Config) InitFunc() providers.FuncInit {
	if c.init != nil {
		return c.init
	}
	return func(f *resource.Fetcher) error {
		return nil
	}
}

// Status takes a Fetcher and the error from Run (from engine)
func (c Config) Status(stageName string, f resource.Fetcher, statusErr error) error {
	if c.status != nil {
		return c.status(stageName, f, statusErr)
	}
	return nil
}

var configs = registry.Create("platform configs")

func init() {
	configs.Register(Config{
		name:  "aliyun",
		fetch: aliyun.FetchConfig,
	})
	configs.Register(Config{
		name:       "aws",
		fetch:      aws.FetchConfig,
		init:       aws.Init,
		newFetcher: aws.NewFetcher,
	})
	configs.Register(Config{
		name:  "azure",
		fetch: azure.FetchConfig,
	})
	configs.Register(Config{
		name:  "azurestack",
		fetch: azurestack.FetchConfig,
	})
	configs.Register(Config{
		name:  "brightbox",
		fetch: openstack.FetchConfig,
	})
	configs.Register(Config{
		name:  "cloudstack",
		fetch: cloudstack.FetchConfig,
	})
	configs.Register(Config{
		name:  "digitalocean",
		fetch: digitalocean.FetchConfig,
	})
	configs.Register(Config{
		name:  "exoscale",
		fetch: exoscale.FetchConfig,
	})
	configs.Register(Config{
		name:  "file",
		fetch: file.FetchConfig,
	})
	configs.Register(Config{
		name:  "gcp",
		fetch: gcp.FetchConfig,
	})
	configs.Register(Config{
		name:  "ibmcloud",
		fetch: ibmcloud.FetchConfig,
	})
	configs.Register(Config{
		name:  "metal",
		fetch: noop.FetchConfig,
	})
	configs.Register(Config{
		name:  "openstack",
		fetch: openstack.FetchConfig,
	})
	configs.Register(Config{
		name:   "packet",
		fetch:  packet.FetchConfig,
		status: packet.PostStatus,
	})
	configs.Register(Config{
		name:  "qemu",
		fetch: qemu.FetchConfig,
	})
	configs.Register(Config{
		name:  "virtualbox",
		fetch: virtualbox.FetchConfig,
	})
	configs.Register(Config{
		name:  "vmware",
		fetch: vmware.FetchConfig,
	})
	configs.Register(Config{
		name:  "vultr",
		fetch: vultr.FetchConfig,
	})
	configs.Register(Config{
		name:  "zvm",
		fetch: zvm.FetchConfig,
	})
}

func Get(name string) (config Config, ok bool) {
	config, ok = configs.Get(name).(Config)
	return
}

func MustGet(name string) Config {
	if config, ok := Get(name); ok {
		return config
	} else {
		panic(fmt.Sprintf("invalid platform name %q provided", name))
	}
}

func Names() (names []string) {
	return configs.Names()
}
