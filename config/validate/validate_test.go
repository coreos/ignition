// Copyright 2019 Red Hat, Inc.
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

package validate

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	ignerrors "github.com/coreos/ignition/v2/config/shared/errors"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/coreos/vcontext/tree"
)

var (
	errDummy = errors.New("dummy error")
	empty    = path.New("json")
)

// mkReport generates reports with a single error with the specified context, line, col and kind
func mkReport(err error, c path.ContextPath, k report.Kind, line, col int64) (r report.Report) {
	r.AddOn(c, err, k)
	if line == 0 {
		return
	}
	r.Entries[0].Marker.StartP = &tree.Pos{Line: line, Column: col}
	return
}

// mangleReport wipes the fields we don't populate in our tests so the tests don't need to be as verbose
func mangleReport(r *report.Report) {
	for i := range r.Entries {
		if sp := r.Entries[i].Marker.StartP; sp != nil {
			sp.Index = 0
		}
		r.Entries[i].Marker.EndP = nil
	}
}

type test struct{}

func (t test) Validate(c path.ContextPath) (r report.Report) {
	r.AddOnError(c, errDummy)
	return
}

type testDup struct{}

func (t testDup) Key() string {
	return "same"
}

type test2 struct {
	Test test `json:"foobar"`
}

type test3 struct {
	NoDups []testDup `json:"dups"`
}

type test4 struct {
	Ignored []testDup `json:"dups"`
}

func (t test4) IgnoreDuplicates() map[string]struct{} {
	return map[string]struct{}{
		"Ignored": {},
	}
}

type test5 struct {
	NoDups1 []testDup `json:"dups1"`
	NoDups2 []testDup `json:"dups2"`
}

func (t test5) MergedKeys() map[string]string {
	return map[string]string{
		"NoDups1": "dups",
		"NoDups2": "dups",
	}
}

func TestValidateWithContext(t *testing.T) {
	tests := []struct {
		in    interface{}
		inRaw string
		out   report.Report
	}{
		{
			in: struct{}{},
		},
		{
			in:  test{},
			out: mkReport(errDummy, empty, report.Error, 0, 0),
		},
		{
			in:    test{},
			inRaw: "{   }",
			out:   mkReport(errDummy, empty, report.Error, 1, 2),
		},
		{
			in:    struct{}{},
			inRaw: `{"foo":"bar"}`,
			out:   mkReport(fmt.Errorf("unused key foo"), path.New("json", tree.Key("foo")), report.Warn, 1, 2),
		},
		{
			in:    test2{},
			inRaw: `{"foobar": {}}`,
			out:   mkReport(errDummy, path.New("json", "foobar"), report.Error, 1, 13),
		},
		{
			in: test3{},
		},
		{
			in: test3{
				NoDups: make([]testDup, 1),
			},
		},
		{
			in: test3{
				NoDups: make([]testDup, 2),
			},
			out: mkReport(ignerrors.ErrDuplicate, path.New("json", "dups", 1), report.Error, 0, 0),
		},
		{
			in: test4{
				Ignored: make([]testDup, 2),
			},
		},
		{
			in: test5{
				NoDups1: make([]testDup, 1),
			},
		},
		{
			in: test5{
				NoDups1: make([]testDup, 1),
				NoDups2: make([]testDup, 1),
			},
			out: mkReport(ignerrors.ErrDuplicate, path.New("json", "dups2", 0), report.Error, 0, 0),
		},
	}

	for i, test := range tests {
		r := ValidateWithContext(test.in, []byte(test.inRaw))
		mangleReport(&r)
		if !reflect.DeepEqual(test.out, r) {
			t.Errorf("#%d: bad report: want %+v got %+v", i, test.out, r)
		}
	}
}
