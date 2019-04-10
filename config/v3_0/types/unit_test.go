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
	"fmt"
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/validate/report"
)

func TestSystemdUnitValidateContents(t *testing.T) {
	type in struct {
		unit Unit
	}
	type out struct {
		err error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{unit: Unit{Name: "test.service", Contents: util.StrToPtr("[Foo]\nQux=Bar")}},
			out: out{err: nil},
		},
		{
			in:  in{unit: Unit{Name: "test.service", Contents: util.StrToPtr("[Foo")}},
			out: out{err: fmt.Errorf("invalid unit content: unable to find end of section")},
		},
		{
			in:  in{unit: Unit{Name: "test.service", Contents: util.StrToPtr(""), Dropins: []Dropin{{}}}},
			out: out{err: nil},
		},
	}

	for i, test := range tests {
		err := test.in.unit.ValidateContents()
		if !reflect.DeepEqual(report.ReportFromError(test.out.err, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}

func TestSystemdUnitValidateName(t *testing.T) {
	type in struct {
		unit string
	}
	type out struct {
		err error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{unit: "test.service"},
			out: out{err: nil},
		},
		{
			in:  in{unit: "test.socket"},
			out: out{err: nil},
		},
		{
			in:  in{unit: "test.blah"},
			out: out{err: errors.ErrInvalidSystemdExt},
		},
	}

	for i, test := range tests {
		err := Unit{Name: test.in.unit, Contents: util.StrToPtr("[Foo]\nQux=Bar")}.ValidateName()
		if !reflect.DeepEqual(report.ReportFromError(test.out.err, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}

func TestSystemdUnitDropInValidate(t *testing.T) {
	type in struct {
		unit Dropin
	}
	type out struct {
		err error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{unit: Dropin{Name: "test.conf", Contents: util.StrToPtr("[Foo]\nQux=Bar")}},
			out: out{err: nil},
		},
		{
			in:  in{unit: Dropin{Name: "test.conf", Contents: util.StrToPtr("[Foo")}},
			out: out{err: fmt.Errorf("invalid unit content: unable to find end of section")},
		},
	}

	for i, test := range tests {
		err := test.in.unit.Validate()
		if !reflect.DeepEqual(report.ReportFromError(test.out.err, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
