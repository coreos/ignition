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
	"strings"
	"testing"

	"github.com/coreos/ignition/config/shared/errors"
)

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestHeadersParse(t *testing.T) {
	tests := []struct {
		in  HTTPHeaders
		out error
	}{
		{
			// Valid headers
			HTTPHeaders{
				HTTPHeader{Name: "header1", Value: "header1value"},
				HTTPHeader{Name: "header2", Value: ""},
			},
			nil,
		},
		{
			// Duplicate headers
			HTTPHeaders{
				HTTPHeader{Name: "header1", Value: "value1"},
				HTTPHeader{Name: "header1", Value: "value2"},
			},
			errors.ErrDuplicateHTTPHeaders,
		},
		{
			// Empty headers
			HTTPHeaders{
				HTTPHeader{Name: "header1", Value: "header1value"},
				HTTPHeader{Name: "", Value: "emptyheadervalue"},
			},
			errors.ErrEmptyHTTPHeaderName,
		},
	}

	for i, test := range tests {
		_, err := test.in.Parse()
		if test.out != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, err)
		}
	}
}

func TestValidHeadersParse(t *testing.T) {
	// Valid headers
	headers := HTTPHeaders{
		HTTPHeader{Name: "header1", Value: "header1value"},
		HTTPHeader{Name: "header2", Value: "header2value"},
	}
	parseHeaders, err := headers.Parse()
	if err != nil {
		t.Errorf("error during parsing valid headers: %v", err)
	}
	if !equal(parseHeaders[strings.Title("header1")], []string{"header1value"}) || !equal(parseHeaders[strings.Title("header2")], []string{"header2value"}) {
		t.Errorf("parsed HTTP headers values are wrong")
	}
}
