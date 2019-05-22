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

	"github.com/ajeddeloh/vcontext/path"
	"github.com/ajeddeloh/vcontext/report"
	"github.com/ajeddeloh/vcontext/tree"
)

var (
	dummy = errors.New("dummy error")
	empty = path.New("json")
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
	r.AddOnError(c, dummy)
	return
}

type test2 struct {
	Test test `json:"foobar"`
}

func TestValidateWithContext(t *testing.T) {
	tests := []struct {
		in    interface{}
		inRaw string
		out   report.Report
	}{
		{
			in:  struct{}{},
			out: report.Report{},
		},
		{
			in:  test{},
			out: mkReport(dummy, empty, report.Error, 0, 0),
		},
		{
			in:    test{},
			inRaw: "{   }",
			out:   mkReport(dummy, empty, report.Error, 1, 2),
		},
		{
			in:    struct{}{},
			inRaw: `{"foo":"bar"}`,
			out:   mkReport(fmt.Errorf("Unused key foo"), path.New("json", tree.Key("foo")), report.Warn, 1, 2),
		},
		{
			in:    test2{},
			inRaw: `{"foobar": {}}`,
			out:   mkReport(dummy, path.New("json", "foobar"), report.Error, 1, 13),
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
