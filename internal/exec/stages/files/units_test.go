// Copyright 2020 CoreOS, Inc.
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

package files

import (
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
)

func TestParseInstanceUnit(t *testing.T) {
	type in struct {
		unit types.Unit
	}
	type out struct {
		unitName string
		instance string
		parseErr error
	}
	tests := []struct {
		in  in
		out out
	}{
		{in: in{types.Unit{Name: "echo@bar.service"}},
			out: out{unitName: "echo@.service", instance: "bar",
				parseErr: nil},
		},

		{in: in{types.Unit{Name: "echo@foo.service"}},
			out: out{unitName: "echo@.service", instance: "foo",
				parseErr: nil},
		},
		{in: in{types.Unit{Name: "echo.service"}},
			out: out{unitName: "", instance: "",
				parseErr: errors.ErrInvalidInstantiatedUnit},
		},
		{in: in{types.Unit{Name: "echo@fooservice"}},
			out: out{unitName: "", instance: "",
				parseErr: errors.ErrNoSystemdExt},
		},
		{in: in{types.Unit{Name: "echo@.service"}},
			out: out{unitName: "echo@.service", instance: "",
				parseErr: nil},
		},
		{in: in{types.Unit{Name: "postgresql@9.3-main.service"}},
			out: out{unitName: "postgresql@.service", instance: "9.3-main",
				parseErr: nil},
		},
	}
	for i, test := range tests {
		unitName, instance, err := parseInstanceUnit(test.in.unit)
		if test.out.parseErr != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.parseErr, err)
		}
		if !reflect.DeepEqual(test.out.unitName, unitName) {
			t.Errorf("#%d: bad unitName: want %v, got %v", i, test.out.unitName, unitName)
		}
		if !reflect.DeepEqual(test.out.instance, instance) {
			t.Errorf("#%d: bad instance: want %v, got %v", i, test.out.instance, instance)
		}
	}
}
