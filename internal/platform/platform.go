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
	"errors"
	"fmt"

	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/providers"
	"github.com/coreos/ignition/v2/internal/registry"
	"github.com/coreos/ignition/v2/internal/resource"
)

var (
	ErrCannotDelete = errors.New("cannot delete config on this platform")
)

// Config defines the capabilities of a particular platform, for use by the
// rest of Ignition.
type Config struct {
	// don't allow direct access to fields
	p Provider
}

// Provider is the struct that platform implementations use to define their
// capabilities for use by this package.
type Provider struct {
	Name       string
	NewFetcher providers.FuncNewFetcher
	Fetch      providers.FuncFetchConfig
	Init       providers.FuncInit
	Status     providers.FuncPostStatus
	DelConfig  providers.FuncDelConfig
}

func (c Config) Name() string {
	return c.p.Name
}

func (c Config) FetchFunc() providers.FuncFetchConfig {
	return c.p.Fetch
}

func (c Config) NewFetcher(l *log.Logger) (resource.Fetcher, error) {
	if c.p.NewFetcher != nil {
		return c.p.NewFetcher(l)
	} else {
		return resource.Fetcher{
			Logger: l,
		}, nil
	}
}

// Init performs additional fetcher configuration post-config fetch.  This
// ensures that networking is already available if a platform needs to reach
// out to the metadata service to fetch additional options / data.
func (c Config) Init(f *resource.Fetcher) error {
	if c.p.Init != nil {
		return c.p.Init(f)
	}
	return nil
}

// Status takes a Fetcher and the error from Run (from engine)
func (c Config) Status(stageName string, f resource.Fetcher, statusErr error) error {
	if c.p.Status != nil {
		return c.p.Status(stageName, f, statusErr)
	}
	return nil
}

func (c Config) DelConfig(f *resource.Fetcher) error {
	if c.p.DelConfig != nil {
		return c.p.DelConfig(f)
	} else {
		return ErrCannotDelete
	}
}

var configs = registry.Create("platform configs")

func Register(provider Provider) {
	configs.Register(Config{
		p: provider,
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
