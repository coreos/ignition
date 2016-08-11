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

	"github.com/coreos/ignition/config/validate/report"
)

func TestNodeValidate(t *testing.T) {
	type in struct {
		node Node
	}
	type out struct {
		report report.Report
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{node: Node{}},
			out: out{report: report.ReportFromError(ErrNoFilesystem, report.EntryError)},
		},
		{
			in:  in{node: Node{Filesystem: "foo"}},
			out: out{},
		},
	}

	for i, test := range tests {
		report := test.in.node.Validate()
		if !reflect.DeepEqual(test.out.report, report) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.report, report)
		}
	}
}

func TestNodeModeValidate(t *testing.T) {
	type in struct {
		mode NodeMode
	}
	type out struct {
		report report.Report
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{mode: NodeMode(0)},
			out: out{},
		},
		{
			in:  in{mode: NodeMode(0644)},
			out: out{},
		},
		{
			in:  in{mode: NodeMode(01755)},
			out: out{},
		},
		{
			in:  in{mode: NodeMode(07777)},
			out: out{},
		},
		{
			in:  in{mode: NodeMode(010000)},
			out: out{report: report.ReportFromError(ErrFileIllegalMode, report.EntryError)},
		},
	}

	for i, test := range tests {
		report := test.in.mode.Validate()
		if !reflect.DeepEqual(test.out.report, report) {
			t.Errorf("#%d: bad report: want %v, got %v", i, test.out.report, report)
		}
	}
}
