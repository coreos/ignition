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
	"fmt"
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/stretchr/testify/assert"
)

func TestModeValidate(t *testing.T) {
	tests := []struct {
		in  *int
		out error
	}{
		{
			nil,
			nil,
		},
		{
			util.IntToPtr(0),
			nil,
		},
		{
			util.IntToPtr(0644),
			nil,
		},
		{
			util.IntToPtr(01755),
			nil,
		},
		{
			util.IntToPtr(07777),
			nil,
		},
		{
			util.IntToPtr(010000),
			errors.ErrFileIllegalMode,
		},
	}

	for i, test := range tests {
		err := validateMode(test.in)
		if !reflect.DeepEqual(test.out, err) {
			t.Errorf("#%d: bad err: want %v, got %v", i, test.out, err)
		}
	}
}

func TestPermissionBitsValidate(t *testing.T) {
	tests := []struct {
		in  *int
		out error
	}{
		{
			nil,
			nil,
		},
		{
			util.IntToPtr(0),
			nil,
		},
		{
			util.IntToPtr(0644),
			nil,
		},
		{
			util.IntToPtr(0755),
			nil,
		},
		{
			util.IntToPtr(0777),
			nil,
		},
		{
			util.IntToPtr(01755),
			errors.ErrModeSpecialBits,
		},
		{
			util.IntToPtr(02755),
			errors.ErrModeSpecialBits,
		},
		{
			util.IntToPtr(04755),
			errors.ErrModeSpecialBits,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := validateModeSpecialBits(test.in)
			expected := test.out
			assert.Equal(t, actual, expected, "bad validation for special bits")
		})
	}
}
