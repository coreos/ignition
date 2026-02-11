// Copyright 2022 Red Hat, Inc.
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
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

func TestSystemdValidate(t *testing.T) {
	tests := []struct {
		in  Systemd
		out error
	}{
		{
			Systemd{
				[]Unit{
					{Name: "test@.service", Contents: util.StrToPtr("[Foo]\nQux=Bar")},
					{Name: "test@foo.service", Enabled: util.BoolToPtr(true)},
				},
			},
			errors.NewNoInstallSectionForInstantiableUnitError("test@.service", "test@foo.service"),
		},
		{
			Systemd{
				[]Unit{
					{Name: "test2@.service", Contents: util.StrToPtr("[Foo]\nQux=Bar")},
				},
			},
			nil,
		},
		{
			Systemd{
				[]Unit{
					{Name: "test@.service", Contents: util.StrToPtr("[Foo]\nQux=Bar")},
					{Name: "test@foo.service", Enabled: util.BoolToPtr(false)},
				},
			},
			nil,
		},
		{
			Systemd{
				[]Unit{
					{Name: "test2@.service", Contents: util.StrToPtr("[Unit]\nDescription=echo service template\n[Service]\nType=oneshot\nExecStart=/bin/echo %i\n[Install]\nWantedBy=multi-user.target\n")},
					{Name: "test2@foo.service", Enabled: util.BoolToPtr(false)},
				},
			},
			nil,
		},
		{
			Systemd{
				[]Unit{
					{Name: "test2@.service", Contents: util.StrToPtr("[Unit]\nDescription=echo service template\n[Service]\nType=oneshot\nExecStart=/bin/echo %i\n[Install]\nWantedBy=multi-user.target\n")},
					{Name: "test2@bar.service", Enabled: util.BoolToPtr(true)},
				},
			},
			nil,
		},
		{
			Systemd{
				[]Unit{
					{Name: "test@.service", Contents: util.StrToPtr("[Unit]\nDescription=echo service template\n[Service]\nType=oneshot\nExecStart=/bin/echo %i\n[Install]\nWantedBy=multi-user.target\n")},
					{Name: "test2@foo.service", Enabled: util.BoolToPtr(true)},
				},
			},
			nil,
		},
		{
			Systemd{
				[]Unit{
					{Name: "test@.service"},
					{Name: "test@bar.service", Enabled: util.BoolToPtr(true)},
				},
			},
			nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.ContextPath{})
			expected := report.Report{}
			expected.AddOnWarn(path.ContextPath{}.Append("units", 1, "contents"), test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}
