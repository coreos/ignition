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

	"github.com/coreos/ignition/config/shared/errors"
	"github.com/coreos/ignition/config/validate/report"
)

func TestFilesystemValidateFormat(t *testing.T) {
	type in struct {
		format string
	}
	type out struct {
		err error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{format: "ext4"},
			out: out{},
		},
		{
			in:  in{format: "btrfs"},
			out: out{},
		},
		{
			in:  in{format: ""},
			out: out{err: errors.ErrFilesystemInvalidFormat},
		},
	}

	for i, test := range tests {
		err := Filesystem{Format: test.in.format, Device: "/"}.ValidateFormat()
		if !reflect.DeepEqual(report.ReportFromError(test.out.err, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}

func TestFilesystemValidatePath(t *testing.T) {
	type in struct {
		filesystem Filesystem
	}
	type out struct {
		err error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{filesystem: Filesystem{Path: "/foo"}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Path: ""}},
			out: out{errors.ErrNoPath},
		},
		{
			in:  in{filesystem: Filesystem{Path: "foo"}},
			out: out{err: errors.ErrPathRelative},
		},
	}

	for i, test := range tests {
		err := test.in.filesystem.ValidatePath()
		if !reflect.DeepEqual(report.ReportFromError(test.out.err, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}

func TestLabelValidate(t *testing.T) {
	type in struct {
		filesystem Filesystem
	}
	type out struct {
		err error
	}

	strToPtr := func(p string) *string { return &p }

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{filesystem: Filesystem{Format: "ext4", Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "ext4", Label: strToPtr("data")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "ext4", Label: strToPtr("thislabelistoolong")}},
			out: out{err: errors.ErrExt4LabelTooLong},
		},
		{
			in:  in{filesystem: Filesystem{Format: "btrfs", Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "btrfs", Label: strToPtr("thislabelisnottoolong")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "btrfs", Label: strToPtr("thislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolong")}},
			out: out{err: errors.ErrBtrfsLabelTooLong},
		},
		{
			in:  in{filesystem: Filesystem{Format: "xfs", Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "xfs", Label: strToPtr("data")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "xfs", Label: strToPtr("thislabelistoolong")}},
			out: out{err: errors.ErrXfsLabelTooLong},
		},
		{
			in:  in{filesystem: Filesystem{Format: "swap", Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "swap", Label: strToPtr("data")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "swap", Label: strToPtr("thislabelistoolong")}},
			out: out{err: errors.ErrSwapLabelTooLong},
		},
		{
			in:  in{filesystem: Filesystem{Format: "vfat", Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "vfat", Label: strToPtr("data")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: "vfat", Label: strToPtr("thislabelistoolong")}},
			out: out{err: errors.ErrVfatLabelTooLong},
		},
	}

	for i, test := range tests {
		err := test.in.filesystem.ValidateLabel()
		if !reflect.DeepEqual(report.ReportFromError(test.out.err, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
