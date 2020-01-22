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

	"github.com/coreos/ignition/config/shared/errors"
	"github.com/coreos/ignition/config/validate/report"
)

func TestHeadersValidate(t *testing.T) {
	type in struct {
		headers HTTPHeaders
	}
	type out struct {
		err error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			// Valid headers
			in: in{
				headers: HTTPHeaders{
					HTTPHeader{Name: "header1", Value: "header1value"},
					HTTPHeader{Name: "header2", Value: ""},
				},
			},
			out: out{},
		},
		{
			// Duplicate headers
			in: in{
				headers: HTTPHeaders{
					HTTPHeader{Name: "header1", Value: "header1value"},
					HTTPHeader{Name: "header1", Value: "header2value"},
				},
			},
			out: out{err: fmt.Errorf("Found duplicate HTTP header: \"header1\"")},
		},
		{
			// Empty headers
			in: in{
				headers: HTTPHeaders{
					HTTPHeader{Name: "header1", Value: "header1value"},
					HTTPHeader{Name: "", Value: "header2value"},
				},
			},
			out: out{err: errors.ErrEmptyHTTPHeaderName},
		},
	}

	for i, test := range tests {
		err := test.in.headers.Validate()
		if !reflect.DeepEqual(report.ReportFromError(test.out.err, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
