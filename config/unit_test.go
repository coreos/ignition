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

package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/coreos/ignition/Godeps/_workspace/src/github.com/go-yaml/yaml"
)

func TestUnitNameUnmarshalJSON(t *testing.T) {
	type in struct {
		data string
	}
	type out struct {
		unit UnitName
		err  error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{data: `"test.service"`},
			out: out{unit: UnitName("test.service")},
		},
		{
			in:  in{data: `"test.socket"`},
			out: out{unit: UnitName("test.socket")},
		},
	}

	for i, test := range tests {
		var unit UnitName
		err := json.Unmarshal([]byte(test.in.data), &unit)
		if !reflect.DeepEqual(test.out.err, err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
		if !reflect.DeepEqual(test.out.unit, unit) {
			t.Errorf("#%d: bad unit: want %#v, got %#v", i, test.out.unit, unit)
		}
	}
}

func TestUnitNameUnmarshalYAML(t *testing.T) {
	type in struct {
		data string
	}
	type out struct {
		unit UnitName
		err  error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{data: `"test.service"`},
			out: out{unit: UnitName("test.service")},
		},
		{
			in:  in{data: `"test.socket"`},
			out: out{unit: UnitName("test.socket")},
		},
	}

	for i, test := range tests {
		var unit UnitName
		err := yaml.Unmarshal([]byte(test.in.data), &unit)
		if !reflect.DeepEqual(test.out.err, err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
		if !reflect.DeepEqual(test.out.unit, unit) {
			t.Errorf("#%d: bad unit: want %#v, got %#v", i, test.out.unit, unit)
		}
	}
}
