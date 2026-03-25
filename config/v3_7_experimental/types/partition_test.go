// Copyright 2020 Red Hat, Inc.
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
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
)

func TestValidateLabel(t *testing.T) {
	tests := []struct {
		in  *string
		out error
	}{
		{
			util.StrToPtr("root"),
			nil,
		},
		{
			util.StrToPtr(""),
			nil,
		},
		{
			nil,
			nil,
		},
		{
			util.StrToPtr("111111111111111111111111111111111111"),
			nil,
		},
		{
			util.StrToPtr("1111111111111111111111111111111111111"),
			errors.ErrLabelTooLong,
		},
		{
			util.StrToPtr("test:"),
			errors.ErrLabelContainsColon,
		},
	}
	for i, test := range tests {
		err := Partition{Label: test.in}.validateLabel()
		if err != test.out {
			t.Errorf("#%d: wanted %v, got %v", i, test.out, err)
		}
	}
}

func TestValidateGUID(t *testing.T) {
	tests := []struct {
		in  *string
		out error
	}{
		{
			util.StrToPtr("5DFBF5F4-2848-4BAC-AA5E-0D9A20B745A6"),
			nil,
		},
		{
			util.StrToPtr("5dfbf5f4-2848-4bac-aa5e-0d9a20b745a6"),
			nil,
		},
		{
			util.StrToPtr(""),
			nil,
		},
		{
			nil,
			nil,
		},
		{
			util.StrToPtr("not-a-valid-typeguid"),
			errors.ErrDoesntMatchGUIDRegex,
		},
	}
	for i, test := range tests {
		err := validateGUID(test.in)
		if err != test.out {
			t.Errorf("#%d: wanted %v, got %v", i, test.out, err)
		}
	}
}
