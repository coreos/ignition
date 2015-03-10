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

package providers

import (
	"fmt"
	"sort"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/log"
)

type Provider interface {
	Name() string
	FetchConfig() (config.Config, error)
	IsOnline() bool
	ShouldRetry() bool
	BackoffDuration() time.Duration
}

type ProviderCreator interface {
	Name() string
	Create(logger log.Logger) Provider
}

var providers map[string]ProviderCreator

func Register(provider ProviderCreator) {
	if providers == nil {
		providers = map[string]ProviderCreator{}
	}
	if _, ok := providers[provider.Name()]; ok {
		panic(fmt.Sprintf("provider %q already registered", provider.Name()))
	}
	providers[provider.Name()] = provider
}

func Get(name string) ProviderCreator {
	if provider, ok := providers[name]; ok {
		return provider
	}

	return nil
}

func Names() []string {
	keys := []string{}
	for key := range providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	names := make([]string, 0, len(providers))
	for _, name := range keys {
		names = append(names, name)
	}
	return names
}
