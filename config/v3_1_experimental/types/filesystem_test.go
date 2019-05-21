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
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
)

func TestFilesystemValidateFormat(t *testing.T) {
	tests := []struct {
		in  Filesystem
		out error
	}{
		{
			Filesystem{Format: util.StrToPtr("ext4")},
			nil,
		},
		{
			Filesystem{Format: util.StrToPtr("btrfs")},
			nil,
		},
		{
			Filesystem{Format: util.StrToPtr("")},
			nil,
		},
		{
			Filesystem{Format: nil},
			nil,
		},
		{
			Filesystem{Format: util.StrToPtr(""), Path: util.StrToPtr("/")},
			errors.ErrFormatNilWithOthers,
		},
		{
			Filesystem{Format: nil, Path: util.StrToPtr("/")},
			errors.ErrFormatNilWithOthers,
		},
	}

	for i, test := range tests {
		err := test.in.validateFormat()
		if test.out != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, err)
		}
	}
}

func TestFilesystemValidatePath(t *testing.T) {
	tests := []struct {
		in  Filesystem
		out error
	}{
		{
			Filesystem{Path: util.StrToPtr("/foo")},
			nil,
		},
		{
			Filesystem{Path: util.StrToPtr("")},
			nil,
		},
		{
			Filesystem{Path: nil},
			nil,
		},
		{
			Filesystem{Path: util.StrToPtr("foo")},
			errors.ErrPathRelative,
		},
	}

	for i, test := range tests {
		err := test.in.validatePath()
		if test.out != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, err)
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

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("ext4"), Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("ext4"), Label: util.StrToPtr("data")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("ext4"), Label: util.StrToPtr("thislabelistoolong")}},
			out: out{err: errors.ErrExt4LabelTooLong},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("btrfs"), Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("btrfs"), Label: util.StrToPtr("thislabelisnottoolong")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("btrfs"), Label: util.StrToPtr("thislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolongthislabelistoolong")}},
			out: out{err: errors.ErrBtrfsLabelTooLong},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("xfs"), Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("xfs"), Label: util.StrToPtr("data")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("xfs"), Label: util.StrToPtr("thislabelistoolong")}},
			out: out{err: errors.ErrXfsLabelTooLong},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("swap"), Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("swap"), Label: util.StrToPtr("data")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("swap"), Label: util.StrToPtr("thislabelistoolong")}},
			out: out{err: errors.ErrSwapLabelTooLong},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("vfat"), Label: nil}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("vfat"), Label: util.StrToPtr("data")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Format: util.StrToPtr("vfat"), Label: util.StrToPtr("thislabelistoolong")}},
			out: out{err: errors.ErrVfatLabelTooLong},
		},
	}

	for i, test := range tests {
		err := test.in.filesystem.validateLabel()
		if test.out.err != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
