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
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func TestNodeValidatePath(t *testing.T) {
	node := Node{Path: "not/absolute"}
	rep := report.Report{}
	rep.AddOnError(path.ContextPath{}.Append("path"), errors.ErrPathRelative)
	if receivedRep := node.Validate(path.ContextPath{}); !reflect.DeepEqual(rep, receivedRep) {
		t.Errorf("bad error: want %v, got %v", rep, receivedRep)
	}
}

func TestNodeValidateUser(t *testing.T) {
	tests := []struct {
		in  NodeUser
		out error
	}{
		{
			NodeUser{util.IntToPtr(0), util.StrToPtr("")},
			nil,
		},
		{
			NodeUser{util.IntToPtr(1000), util.StrToPtr("")},
			nil,
		},
		{
			NodeUser{nil, util.StrToPtr("core")},
			nil,
		},
		{
			NodeUser{util.IntToPtr(1000), util.StrToPtr("core")},
			errors.ErrBothIDAndNameSet,
		},
	}

	for i, test := range tests {
		r := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(path.New(""), test.out)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad report: want %v got %v", i, test.out, r)
		}
	}
}

func TestNodeValidateGroup(t *testing.T) {
	tests := []struct {
		in  NodeGroup
		out error
	}{
		{
			NodeGroup{util.IntToPtr(0), util.StrToPtr("")},
			nil,
		},
		{
			NodeGroup{util.IntToPtr(1000), util.StrToPtr("")},
			nil,
		},
		{
			NodeGroup{nil, util.StrToPtr("core")},
			nil,
		},
		{
			NodeGroup{util.IntToPtr(1000), util.StrToPtr("core")},
			errors.ErrBothIDAndNameSet,
		},
	}

	for i, test := range tests {
		r := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(path.New(""), test.out)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad report: want %v got %v", i, test.out, r)
		}
	}
}
