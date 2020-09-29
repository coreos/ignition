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

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func TestValidateProxyURL(t *testing.T) {
	tests := []struct {
		in     *string
		httpOk bool
		out    report.Entry
	}{
		{
			nil,
			false,
			report.Entry{},
		},
		{
			nil,
			true,
			report.Entry{},
		},
		{
			util.StrToPtr("https://example.com"),
			false,
			report.Entry{},
		},
		{
			util.StrToPtr("https://example.com"),
			true,
			report.Entry{},
		},
		{
			util.StrToPtr("http://example.com"),
			false,
			report.Entry{
				Kind:    report.Warn,
				Message: errors.ErrInsecureProxy.Error(),
			},
		},
		{
			util.StrToPtr("http://example.com"),
			true,
			report.Entry{},
		},
		{
			util.StrToPtr("ftp://example.com"),
			false,
			report.Entry{
				Kind:    report.Error,
				Message: errors.ErrInvalidProxy.Error(),
			},
		},
		{
			util.StrToPtr("ftp://example.com"),
			true,
			report.Entry{
				Kind:    report.Error,
				Message: errors.ErrInvalidProxy.Error(),
			},
		},
		{
			util.StrToPtr("http://[::1]a"),
			false,
			report.Entry{
				Kind:    report.Error,
				Message: errors.ErrInvalidUrl.Error(),
			},
		},
		{
			util.StrToPtr("http://[::1]a"),
			true,
			report.Entry{
				Kind:    report.Error,
				Message: errors.ErrInvalidUrl.Error(),
			},
		},
	}

	for i, test := range tests {
		r := report.Report{}
		validateProxyURL(test.in, path.New(""), &r, test.httpOk)
		e := report.Entry{}
		if len(r.Entries) > 0 {
			e = r.Entries[0]
		}
		if !reflect.DeepEqual(test.out, e) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, e)
		}
	}
}
