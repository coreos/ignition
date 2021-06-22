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
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func TestResourceGetSources(t *testing.T) {
	empty := ""
	src0 := "http://127.0.0.1:8080/abcd"
	src1 := Source("http://127.0.0.1:8081/abcd")
	src2 := Source("http://127.0.0.1:8082/abcd")
	src3 := Source("http://127.0.0.1:8083/abcd")

	tests := []struct {
		in  Resource
		out []Source
		err error
	}{
		{
			in: Resource{
				Source: &empty,
			},
			out: []Source{Source(empty)},
			err: nil,
		},
		{
			in: Resource{
				Sources: []Source{""},
			},
			out: []Source{Source(empty)},
			err: nil,
		},
		{
			in: Resource{
				Source: &src0,
			},
			out: []Source{Source(src0)},
			err: nil,
		},
		{
			in: Resource{
				Sources: []Source{src1, src2, src3},
			},
			out: []Source{src1, src2, src3},
			err: nil,
		},
		{
			in: Resource{
				Source:  &src0,
				Sources: []Source{src1, src2, src3},
			},
			out: []Source{},
			err: errors.ErrSourcesInvalid,
		},
	}

	for i, test := range tests {
		r := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(path.New("", "source"), test.err)

		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad report: want %v got %v", i, expected, r)
		}

		sources := test.in.GetSources()
		if test.err == nil && !reflect.DeepEqual(sources, test.out) {
			t.Errorf("#%d: bad report: want %v got %v", i, test.out, sources)
		}
	}
}
