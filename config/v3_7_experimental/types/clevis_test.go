// Copyright 2021 Red Hat, Inc.
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
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func TestClevisCustomValidate(t *testing.T) {
	tests := []struct {
		in  ClevisCustom
		at  path.ContextPath
		out error
	}{
		{
			in:  ClevisCustom{},
			out: nil,
		},
		{
			in: ClevisCustom{
				Config:       util.StrToPtr("z"),
				NeedsNetwork: util.BoolToPtr(true),
				Pin:          util.StrToPtr("sss"),
			},
			out: nil,
		},
		{
			in: ClevisCustom{
				Config: util.StrToPtr("z"),
			},
			at:  path.New("", "pin"),
			out: errors.ErrClevisPinRequired,
		},
		{
			in: ClevisCustom{
				Config: util.StrToPtr("z"),
				Pin:    util.StrToPtr("z"),
			},
			at:  path.New("", "pin"),
			out: nil,
		},
		{
			in: ClevisCustom{
				Pin: util.StrToPtr("tpm2"),
			},
			at:  path.New("", "config"),
			out: errors.ErrClevisConfigRequired,
		},
	}

	for i, test := range tests {
		r := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(test.at, test.out)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad report: want %v, got %v", i, expected, r)
		}
	}
}
