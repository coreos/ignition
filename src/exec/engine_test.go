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
	"github.com/coreos/ignition/src/providers"
)

type mockProvider struct {
	name    string
	config  config.Config
	err     error
	online  bool
	retry   bool
	backoff time.Duration
}

func (p mockProvider) Name() string                        { return p.name }
func (p mockProvider) FetchConfig() (config.Config, error) { return p.config, p.err }
func (p mockProvider) IsOnline() bool                      { return p.online }
func (p mockProvider) ShouldRetry() bool                   { return p.retry }
func (p mockProvider) BackoffDuration() time.Duration      { return p.backoff }

// TODO
func TestRun(t *testing.T) {
}

func TestFetchConfigs(t *testing.T) {
	type in struct {
		provider providers.Provider
		timeout  time.Duration
	}
	type out struct {
		config config.Config
		err    error
	}

	online := mockProvider{
		online: true,
		err:    errors.New("test error"),
		config: config.Config{
			Systemd: config.Systemd{
				Units: []config.SystemdUnit{},
			},
		},
	}
	offline := mockProvider{online: false}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{provider: online, timeout: time.Second},
			out: out{config: online.config, err: online.err},
		},
		{
			in:  in{provider: offline, timeout: time.Second},
			out: out{config: config.Config{}, err: ErrNoProvider},
		},
	}

	for i, test := range tests {
		config, err := fetchConfig(test.in.provider, test.in.timeout)
		if !reflect.DeepEqual(test.out.config, config) {
			t.Errorf("#%d: bad provider: want %+v, got %+v", i, test.out.config, config)
		}
		if test.out.err != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}

func TestWaitForProvider(t *testing.T) {
	type in struct {
		provider providers.Provider
		timeout  time.Duration
	}
	type out struct {
		err error
	}

	online := mockProvider{online: true}
	offline := mockProvider{online: false}
	offlineRetry := mockProvider{online: false, retry: true}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{provider: online, timeout: time.Second},
			out: out{err: nil},
		},
		{
			in:  in{provider: offline, timeout: time.Second},
			out: out{err: ErrNoProvider},
		},
		{
			in:  in{provider: offlineRetry, timeout: time.Second},
			out: out{err: ErrTimeout},
		},
	}

	for i, test := range tests {
		err := waitForProvider(test.in.provider, test.in.timeout)
		if test.out.err != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
