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
)

func TestFilesystemFormatAssertValid(t *testing.T) {
	type in struct {
		format FilesystemFormat
	}
	type out struct {
		err error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{format: FilesystemFormat("ext4")},
			out: out{},
		},
		{
			in:  in{format: FilesystemFormat("btrfs")},
			out: out{},
		},
		{
			in:  in{format: FilesystemFormat("")},
			out: out{err: ErrFilesystemInvalidFormat},
		},
	}

	for i, test := range tests {
		err := test.in.format.AssertValid()
		if !reflect.DeepEqual(test.out.err, err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}

func TestFilesystemAssertValid(t *testing.T) {
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
			in:  in{filesystem: Filesystem{Mount: &FilesystemMount{Device: "/foo", Format: "ext4"}}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Mount: &FilesystemMount{Device: "/foo"}}},
			out: out{err: ErrFilesystemInvalidFormat},
		},
		{
			in:  in{filesystem: Filesystem{Mount: &FilesystemMount{Format: "ext4"}}},
			out: out{err: ErrPathRelative},
		},
		{
			in:  in{filesystem: Filesystem{Path: func(p Path) *Path { return &p }("/mount")}},
			out: out{},
		},
		{
			in:  in{filesystem: Filesystem{Path: func(p Path) *Path { return &p }("mount")}},
			out: out{err: ErrPathRelative},
		},
		{
			in:  in{filesystem: Filesystem{Path: func(p Path) *Path { return &p }("/mount"), Mount: &FilesystemMount{Device: "/foo", Format: "ext4"}}},
			out: out{err: ErrFilesystemMountAndPath},
		},
		{
			in:  in{filesystem: Filesystem{}},
			out: out{err: ErrFilesystemNoMountPath},
		},
	}

	for i, test := range tests {
		err := test.in.filesystem.AssertValid()
		if !reflect.DeepEqual(test.out.err, err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
