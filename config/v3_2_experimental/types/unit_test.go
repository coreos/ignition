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
	"github.com/vincent-petithory/dataurl"
)

func TestSystemdUnitValidateContents(t *testing.T) {
	tests := []struct {
		in  Unit
		out error
	}{
		{
			Unit{
				Name: "test.service",
				Contents: Resource{
					Source: util.StrToPtr(dataurl.EncodeBytes([]byte("[Foo]\nQux=Bar"))),
				},
			},
			nil,
		},
		{
			Unit{Name: "test.service", Dropins: []Dropin{{}}},
			nil,
		},
	}

	for i, test := range tests {
		err := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(path.New("", "contents"), test.out)
		if !reflect.DeepEqual(expected, err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, expected, err)
		}
	}
}

func TestSystemdUnitValidateName(t *testing.T) {
	tests := []struct {
		in  string
		out error
	}{
		{
			"test.service",
			nil,
		},
		{
			"test.socket",
			nil,
		},
		{
			"test.blah",
			errors.ErrInvalidSystemdExt,
		},
	}

	for i, test := range tests {
		err := validateName(test.in)
		if test.out != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, err)
		}
	}
}

func TestSystemdUnitDropInValidate(t *testing.T) {
	tests := []struct {
		in  Dropin
		out error
	}{
		{
			Dropin{
				Name: "test.conf",
				Contents: Resource{
					Source: util.StrToPtr(dataurl.EncodeBytes([]byte("[Foo]\nQux=Bar"))),
				},
			},
			nil,
		},
	}

	for i, test := range tests {
		err := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(path.New("", "contents"), test.out)
		if !reflect.DeepEqual(expected, err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, expected, err)
		}
	}
}
