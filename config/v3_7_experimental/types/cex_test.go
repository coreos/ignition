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

	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func TestCexValidate(t *testing.T) {
	tests := []struct {
		in  Cex
		at  path.ContextPath
		out error
	}{
		{
			in:  Cex{},
			out: nil,
		},
		{
			in: Cex{
				Enabled: util.BoolToPtr(true),
			},
			out: nil,
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
