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
	"reflect"
	"testing"

	"github.com/coreos/ignition/log"
	"github.com/coreos/ignition/registry"
)

type mockProviderCreator struct {
	name     string
	provider Provider
}

func (c mockProviderCreator) Name() string               { return c.name }
func (c mockProviderCreator) Create(log.Logger) Provider { return c.provider }

func TestGet(t *testing.T) {
	type in struct {
		providers []ProviderCreator
		name      string
	}
	type out struct {
		creator ProviderCreator
	}

	a := mockProviderCreator{name: "a"}
	b := mockProviderCreator{name: "b"}
	c := mockProviderCreator{name: "c"}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{providers: []ProviderCreator{}, name: "a"},
			out: out{creator: nil},
		},
		{
			in:  in{providers: []ProviderCreator{a, b, c}, name: "a"},
			out: out{creator: a},
		},
		{
			in:  in{providers: []ProviderCreator{a, b, c}, name: "c"},
			out: out{creator: c},
		},
	}

	for i, test := range tests {
		providers = registry.Create(fmt.Sprintf("test %d", i))
		for _, p := range test.in.providers {
			providers.Register(p)
		}

		p := Get(test.in.name)
		if !reflect.DeepEqual(test.out.creator, p) {
			t.Errorf("#%d: bad creator: want %#v, got %#v", i, test.out.creator, p)
		}
	}
}

func TestNames(t *testing.T) {
	type in struct {
		providers []ProviderCreator
	}
	type out struct {
		names []string
	}

	a := mockProviderCreator{name: "a"}
	b := mockProviderCreator{name: "b"}
	c := mockProviderCreator{name: "c"}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{providers: nil},
			out: out{names: []string{}},
		},
		{
			in:  in{providers: []ProviderCreator{a, b, c}},
			out: out{names: []string{"a", "b", "c"}},
		},
	}

	for i, test := range tests {
		providers = registry.Create(fmt.Sprintf("test %d", i))
		for _, p := range test.in.providers {
			providers.Register(p)
		}
		names := providers.Names()
		if !reflect.DeepEqual(test.out.names, names) {
			t.Errorf("#%d: bad names: want %#v, got %#v", i, test.out.names, names)
		}
	}
}
