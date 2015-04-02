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
	"reflect"
	"testing"
)

func TestRegister(t *testing.T) {
	type in struct {
		providers []ProviderCreator
	}
	type out struct {
		providers map[string]ProviderCreator
	}

	a := MockProviderCreator{name: "a"}
	b := MockProviderCreator{name: "b"}
	c := MockProviderCreator{name: "c"}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{providers: []ProviderCreator{}},
			out: out{providers: map[string]ProviderCreator{}},
		},
		{
			in:  in{providers: []ProviderCreator{a, b, c}},
			out: out{providers: map[string]ProviderCreator{"a": a, "b": b, "c": c}},
		},
	}

	for i, test := range tests {
		providers = map[string]ProviderCreator{}
		for _, p := range test.in.providers {
			Register(p)
		}
		if !reflect.DeepEqual(test.out.providers, providers) {
			t.Errorf("#%d: bad providers: want %#v, got %#v", i, test.out.providers, providers)
		}
	}
}

func TestGet(t *testing.T) {
	type in struct {
		providers map[string]ProviderCreator
		name      string
	}
	type out struct {
		creator ProviderCreator
	}

	a := MockProviderCreator{name: "a"}
	b := MockProviderCreator{name: "b"}
	c := MockProviderCreator{name: "c"}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{providers: nil, name: "a"},
			out: out{creator: nil},
		},
		{
			in:  in{providers: map[string]ProviderCreator{}, name: "a"},
			out: out{creator: nil},
		},
		{
			in:  in{providers: map[string]ProviderCreator{"a": a, "b": b, "c": c}, name: "a"},
			out: out{creator: a},
		},
		{
			in:  in{providers: map[string]ProviderCreator{"a": a, "b": b, "c": c}, name: "c"},
			out: out{creator: c},
		},
	}

	for i, test := range tests {
		providers = test.in.providers
		creator := Get(test.in.name)
		if !reflect.DeepEqual(test.out.creator, creator) {
			t.Errorf("#%d: bad creator: want %#v, got %#v", i, test.out.creator, creator)
		}
	}
}

func TestNames(t *testing.T) {
	type in struct {
		providers map[string]ProviderCreator
	}
	type out struct {
		names []string
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{providers: nil},
			out: out{names: []string{}},
		},
		{
			in:  in{providers: map[string]ProviderCreator{}},
			out: out{names: []string{}},
		},
		{
			in:  in{providers: map[string]ProviderCreator{"a": MockProviderCreator{}, "b": MockProviderCreator{}, "c": MockProviderCreator{}}},
			out: out{names: []string{"a", "b", "c"}},
		},
	}

	for i, test := range tests {
		providers = test.in.providers
		names := Names()
		if !reflect.DeepEqual(test.out.names, names) {
			t.Errorf("#%d: bad names: want %#v, got %#v", i, test.out.names, names)
		}
	}
}
