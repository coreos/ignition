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
					HTTPHeader{HTTPHeaderItem("header1"), HTTPHeaderItem("header1value")},
					HTTPHeader{HTTPHeaderItem("header2"), HTTPHeaderItem("header2value")},
				},
			},
			out: out{},
		},
		{
			// Duplicate headers
			in: in{
				headers: HTTPHeaders{
					HTTPHeader{HTTPHeaderItem("header1"), HTTPHeaderItem("header1value")},
					HTTPHeader{HTTPHeaderItem("header1"), HTTPHeaderItem("header2value")},
				},
			},
			out: out{err: errors.ErrDuplicateHTTPHeaders},
		},
		{
			// Empty headers
			in: in{
				headers: HTTPHeaders{
					HTTPHeader{HTTPHeaderItem("header1"), HTTPHeaderItem("header1value")},
					HTTPHeader{HTTPHeaderItem(""), HTTPHeaderItem("header2value")},
				},
			},
			out: out{err: errors.ErrEmptyHTTPHeaderName},
		},
		{
			// Invalid headers with 3 elements
			in: in{
				headers: HTTPHeaders{
					HTTPHeader{HTTPHeaderItem("header1"), HTTPHeaderItem("header1value")},
					HTTPHeader{HTTPHeaderItem("invalid"), HTTPHeaderItem("value1"), HTTPHeaderItem("value2")},
				},
			},
			out: out{err: errors.ErrInvalidHTTPHeader},
		},
		{
			// Invalid header with 1 element
			in: in{
				headers: HTTPHeaders{
					HTTPHeader{HTTPHeaderItem("header1"), HTTPHeaderItem("header1value")},
					HTTPHeader{HTTPHeaderItem("invalid")},
				},
			},
			out: out{err: errors.ErrInvalidHTTPHeader},
		},
		{
			// Invalid header without elements
			in: in{
				headers: HTTPHeaders{
					HTTPHeader{HTTPHeaderItem("header1"), HTTPHeaderItem("header1value")},
					HTTPHeader{},
				},
			},
			out: out{err: errors.ErrInvalidHTTPHeader},
		},
	}

	for i, test := range tests {
		err := test.in.headers.Validate()
		if !reflect.DeepEqual(report.ReportFromError(test.out.err, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
