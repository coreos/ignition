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

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/registry"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/state"

	"github.com/coreos/vcontext/report"
)

var (
	ErrCannotDelete = errors.New("cannot delete config on this platform")
	ErrNoProvider   = errors.New("config provider was not online")
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
	NewFetcher func(logger *log.Logger) (resource.Fetcher, error)
	Fetch      func(f *resource.Fetcher) (types.Config, report.Report, error)
	Init       func(f *resource.Fetcher) error
	Status     func(stageName string, f resource.Fetcher, e error) error
	DelConfig  func(f *resource.Fetcher) error

	// Fetch, and also save output files to be written during files stage.
	// Avoid, unless you're certain you need it.
	FetchWithFiles func(f *resource.Fetcher) ([]types.File, types.Config, report.Report, error)
}

func (c Config) Name() string {
	return c.p.Name
}

func (c Config) Fetch(f *resource.Fetcher, state *state.State) (types.Config, report.Report, error) {
	if c.p.FetchWithFiles != nil {
		files, config, report, err := c.p.FetchWithFiles(f)
		state.ProviderOutputFiles = files
		return config, report, err
	} else {
		return c.p.Fetch(f)
	}
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
	configs.Register(NewConfig(provider))
}

// Helper function for wrapping a Provider, for use by specialized providers
// that don't want to add themselves to the registry.
func NewConfig(provider Provider) Config {
	return Config{
		p: provider,
	}
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
