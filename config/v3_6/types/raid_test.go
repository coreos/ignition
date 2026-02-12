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

func TestRaidValidate(t *testing.T) {
	tests := []struct {
		in  Raid
		at  path.ContextPath
		out error
	}{
		{
			in: Raid{
				Name:    "name",
				Level:   util.StrToPtr("0"),
				Devices: []Device{"/dev/fd0"},
				Spares:  util.IntToPtr(0),
			},
			out: nil,
		},
		{
			in: Raid{
				Name:    "name",
				Devices: []Device{"/dev/fd0"},
			},
			at:  path.New("", "level"),
			out: errors.ErrRaidLevelRequired,
		},
		{
			in: Raid{
				Name:    "name",
				Level:   util.StrToPtr("0"),
				Devices: []Device{"/dev/fd0"},
				Spares:  util.IntToPtr(1),
			},
			at:  path.New("", "level"),
			out: errors.ErrSparesUnsupportedForLevel,
		},
		{
			in: Raid{
				Name:    "name",
				Devices: []Device{"/dev/fd0"},
				Level:   util.StrToPtr("zzz"),
			},
			at:  path.New("", "level"),
			out: errors.ErrUnrecognizedRaidLevel,
		},
		{
			in: Raid{
				Name:  "name",
				Level: util.StrToPtr("0"),
			},
			at:  path.New("", "devices"),
			out: errors.ErrRaidDevicesRequired,
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
