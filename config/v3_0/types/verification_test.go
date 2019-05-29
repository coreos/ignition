// Copyright 2016 CoreOS, Inc.
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

package types

import (
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func TestHashParts(t *testing.T) {
	tests := []struct {
		in  string
		out error
	}{
		{
			`"sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`,
			nil,
		},
		{
			`"sha512:01234567"`,
			errors.ErrHashMalformed,
		},
	}

	for i, test := range tests {
		fun, sum, err := Verification{Hash: &test.in}.HashParts()
		if err != test.out {
			t.Fatalf("#%d: bad error: want %+v, got %+v", i, test.out, err)
		}
		if err == nil && fun+"-"+sum != test.in {
			t.Fatalf("#%d: bad hash: want %+v, got %+v", i, test.in, fun+"-"+sum)
		}
	}
}

func TestHashValidate(t *testing.T) {
	h1 := "xor-abcdef"
	h2 := "sha512-123"
	h3 := "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		in  Verification
		out error
	}{
		{
			Verification{Hash: &h1},
			errors.ErrHashUnrecognized,
		},
		{
			Verification{Hash: &h2},
			errors.ErrHashWrongSize,
		},
		{
			Verification{Hash: &h3},
			nil,
		},
	}

	for i, test := range tests {
		err := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(path.New("", "hash"), test.out)
		if !reflect.DeepEqual(expected, err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, expected, err)
		}
	}
}
