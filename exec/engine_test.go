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

package exec

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/providers"
	"github.com/coreos/ignition/providers/test"
)

func TestAddProvider(t *testing.T) {
	type in struct {
		engine    Engine
		providers []providers.Provider
	}
	type out struct {
		providers map[string]providers.Provider
	}

	a := test.MockProvider{name: "a"}
	b := test.MockProvider{name: "b"}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{engine: Engine{}, providers: nil},
			out: out{providers: nil},
		},
		{
			in:  in{engine: Engine{}, providers: []providers.Provider{a}},
			out: out{providers: map[string]providers.Provider{"a": a}},
		},
		{
			in:  in{engine: Engine{}, providers: []providers.Provider{a, b}},
			out: out{providers: map[string]providers.Provider{"a": a, "b": b}},
		},
		{
			in:  in{engine: Engine{}, providers: []providers.Provider{a, a}},
			out: out{providers: map[string]providers.Provider{"a": a}},
		},
		{
			in:  in{engine: Engine{}, providers: []providers.Provider{a, b, a}},
			out: out{providers: map[string]providers.Provider{"a": a, "b": b}},
		},
	}

	for i, test := range tests {
		for _, p := range test.in.providers {
			test.in.engine.AddProvider(p)
		}
		if !reflect.DeepEqual(test.out.providers, test.in.engine.providers) {
			t.Errorf("#%d: bad providers: want %#v, got %#v", i, test.out.providers, test.in.engine.providers)
		}
	}
}

func TestProviders(t *testing.T) {
	type in struct {
		engine Engine
	}
	type out struct {
		providers []providers.Provider
	}

	a := test.MockProvider{name: "a"}
	b := test.MockProvider{name: "b"}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{engine: Engine{}},
			out: out{providers: []providers.Provider{}},
		},
		{
			in:  in{engine: Engine{providers: map[string]providers.Provider{"a": a}}},
			out: out{providers: []providers.Provider{a}},
		},
		{
			in:  in{engine: Engine{providers: map[string]providers.Provider{"a": a, "b": b}}},
			out: out{providers: []providers.Provider{a, b}},
		},
	}

	for i, test := range tests {
		providers := test.in.engine.Providers()
		if !reflect.DeepEqual(test.out.providers, providers) {
			t.Errorf("#%d: bad providers: want %#v, got %#v", i, test.out.providers, providers)
		}
	}
}

// TODO
func TestRun(t *testing.T) {
}

func TestFetchConfigs(t *testing.T) {
	type in struct {
		providers []providers.Provider
		timeout   time.Duration
	}
	type out struct {
		config config.Config
		err    error
	}

	online := test.MockProvider{
		Online: true,
		Err:    errors.New("test error"),
		Config: config.Config{
			Systemd: config.Systemd{
				Units: []config.Unit{},
			},
		},
	}
	offline := test.MockProvider{Online: false}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{providers: nil, timeout: time.Second},
			out: out{config: config.Config{}, err: ErrNoProviders},
		},
		{
			in:  in{providers: []providers.Provider{online}, timeout: time.Second},
			out: out{config: online.Config, err: online.Err},
		},
		{
			in:  in{providers: []providers.Provider{offline}, timeout: time.Second},
			out: out{config: config.Config{}, err: ErrNoProviders},
		},
		{
			in:  in{providers: []providers.Provider{online, offline}, timeout: time.Second},
			out: out{config: online.Config, err: online.Err},
		},
	}

	for i, test := range tests {
		config, err := fetchConfig(test.in.providers, test.in.timeout)
		if !reflect.DeepEqual(test.out.config, config) {
			t.Errorf("#%d: bad provider: want %+v, got %+v", i, test.out.config, config)
		}
		if test.out.err != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}

func TestSelectProvider(t *testing.T) {
	type in struct {
		providers []providers.Provider
		timeout   time.Duration
	}
	type out struct {
		provider providers.Provider
		err      error
	}

	online := test.MockProvider{Online: true}
	offline := test.MockProvider{Online: false}
	offlineRetry := test.MockProvider{Online: false, Retry: true}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{providers: nil, timeout: time.Second},
			out: out{provider: nil, err: ErrNoProviders},
		},
		{
			in:  in{providers: []providers.Provider{online}, timeout: time.Second},
			out: out{provider: online, err: nil},
		},
		{
			in:  in{providers: []providers.Provider{offline}, timeout: time.Second},
			out: out{provider: nil, err: ErrNoProviders},
		},
		{
			in:  in{providers: []providers.Provider{offlineRetry}, timeout: time.Second},
			out: out{provider: nil, err: ErrTimeout},
		},
		{
			in:  in{providers: []providers.Provider{online, offline}, timeout: time.Second},
			out: out{provider: online, err: nil},
		},
		{
			in:  in{providers: []providers.Provider{online, offlineRetry}, timeout: time.Second},
			out: out{provider: online, err: nil},
		},
	}

	for i, test := range tests {
		provider, err := selectProvider(test.in.providers, test.in.timeout)
		if !reflect.DeepEqual(test.out.provider, provider) {
			t.Errorf("#%d: bad provider: want %+v, got %+v", i, test.out.provider, provider)
		}
		if test.out.err != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
