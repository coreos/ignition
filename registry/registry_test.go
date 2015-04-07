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

package registry

import (
	"reflect"
	"testing"
)

// Minimally implement the Registrant interface
type rant struct {
	name string
}

func (t rant) Name() string {
	return t.name
}

func TestCreateRegister(t *testing.T) {
	type in struct {
		rants []rant
	}
	type out struct {
		rants Registry
	}

	a := rant{name: "a"}
	b := rant{name: "b"}
	c := rant{name: "c"}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "empty",
			in:   in{rants: []rant{}},
			out:  out{rants: Registry{name: "empty", registrants: map[string]Registrant{}}},
		},
		{
			name: "three abc ...",
			in:   in{rants: []rant{a, b, c}},
			out:  out{rants: Registry{name: "three abc ...", registrants: map[string]Registrant{"a": a, "b": b, "c": c}}},
		},
	}

	for i, test := range tests {
		tr := Create(test.name)
		for _, r := range test.in.rants {
			tr.Register(r)
		}
		if !reflect.DeepEqual(&test.out.rants, tr) {
			t.Errorf("#%d: bad registrants: want %#v, got %#v", i, &test.out.rants, tr)
		}
	}
}

func TestGet(t *testing.T) {
	type in struct {
		rants []rant
		get   string
	}
	type out struct {
		expect *rant
	}

	a := rant{name: "a"}
	b := rant{name: "b"}
	c := rant{name: "c"}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{rants: nil, get: "a"},
			out: out{expect: nil},
		},
		{
			in:  in{rants: []rant{a, b, c}, get: "a"},
			out: out{expect: &a},
		},
		{
			in:  in{rants: []rant{a, b, c}, get: "c"},
			out: out{expect: &c},
		},
	}

	for i, test := range tests {
		tr := Create("test")
		for _, r := range test.in.rants {
			tr.Register(r)
		}
		r := tr.Get(test.in.get)
		if r == nil {
			if test.out.expect != nil {
				t.Errorf("#%d: got nil expected %#v", i, r)
			}
		} else if !reflect.DeepEqual(*test.out.expect, r.(rant)) {
			t.Errorf("#%d: bad registrant: want %#v, got %#v", i, *test.out.expect, r.(rant))
		}
	}
}

func TestNames(t *testing.T) {
	type in struct {
		rants []rant
	}
	type out struct {
		names []string
	}

	a := rant{name: "a"}
	b := rant{name: "b"}
	c := rant{name: "c"}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{rants: nil},
			out: out{names: []string{}},
		},
		{
			in:  in{rants: []rant{a, b, c}},
			out: out{names: []string{"a", "b", "c"}},
		},
	}

	for i, test := range tests {
		tr := Create("test")
		for _, r := range test.in.rants {
			tr.Register(r)
		}
		names := tr.Names()
		if !reflect.DeepEqual(test.out.names, names) {
			t.Errorf("#%d: bad names: want %#v, got %#v", i, test.out.names, names)
		}
	}
}
