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

func TestStorageValidateErrors(t *testing.T) {
	tests := []struct {
		in   Storage
		at   path.ContextPath
		err  error
		warn error
	}{
		// test empty storage config returns nil
		{
			in: Storage{},
		},
		// test a storage config with no conflicting paths returns nil
		{
			in: Storage{
				Links: []Link{
					{
						Node:          Node{Path: "/foo"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/foo-t")},
					},
					{
						Node:          Node{Path: "/quux"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/quux-t")},
					},
				},
				Files: []File{
					{
						Node: Node{Path: "/bar"},
					},
				},
				Directories: []Directory{
					{
						Node: Node{Path: "/baz"},
					},
				},
			},
		},
		// test when a file uses a configured symlink path returns ErrFileUsedSymlink
		{
			in: Storage{
				Links: []Link{
					{
						Node:          Node{Path: "/foo"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/foo-t")},
					},
				},
				Files: []File{
					{
						Node: Node{Path: "/foo/bar"},
					},
				},
			},
			err: errors.ErrFileUsedSymlink,
			at:  path.New("", "files", 0),
		},
		// test when a directory uses a configured symlink path returns ErrDirectoryUsedSymlink
		{
			in: Storage{
				Links: []Link{
					{
						Node:          Node{Path: "/foo"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/foo-t")},
					},
				},
				Directories: []Directory{
					{
						Node: Node{Path: "/foo/bar"},
					},
				},
			},
			err: errors.ErrDirectoryUsedSymlink,
			at:  path.New("", "directories", 0),
		},
		// test the same path listed for two separate symlinks returns ErrLinkUsedSymlink
		{
			in: Storage{
				Links: []Link{
					{
						Node:          Node{Path: "/foo"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/foo-t")},
					},
					{
						Node:          Node{Path: "/foo/bar"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/foo-bar-t")},
					},
				},
			},
			err: errors.ErrLinkUsedSymlink,
			at:  path.New("", "links", 1),
		},
		// test a configured symlink with no target returns ErrLinkTargetRequired
		{
			in: Storage{
				Links: []Link{
					{
						Node:          Node{Path: "/foo"},
						LinkEmbedded1: LinkEmbedded1{Hard: util.BoolToPtr(true)},
					},
				},
			},
			err: errors.ErrLinkTargetRequired,
			at:  path.New("", "links", 0, "target"),
		},
		// test a configured symlink with a nil target returns ErrLinkTargetRequired
		{
			in: Storage{
				Links: []Link{
					{
						Node:          Node{Path: "/foo"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("")},
					},
				},
			},
			err: errors.ErrLinkTargetRequired,
			at:  path.New("", "links", 0, "target"),
		},
		// test that two symlinks can be configured at a time
		{
			in: Storage{
				Links: []Link{
					{
						Node:          Node{Path: "/foo"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/foo-t")},
					},
					{
						Node:          Node{Path: "/foob/bar"},
						LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/foob-bar-t")},
					},
				},
			},
		},
		// test when a directory uses a configured symlink with the 'Hard:= true' returns ErrHardLinkToDirectory
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{Path: "/quux"},
						LinkEmbedded1: LinkEmbedded1{
							Target: util.StrToPtr("/foo/bar"),
							Hard:   util.BoolToPtr(true),
						},
					},
				},
				Directories: []Directory{
					{
						Node: Node{Path: "/foo/bar"},
					},
				},
			},
			err: errors.ErrHardLinkToDirectory,
			at:  path.New("", "links", 0),
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{
							Path: "/quux",
							User: NodeUser{
								ID: util.IntToPtr(10),
							},
						},
						LinkEmbedded1: LinkEmbedded1{
							Target: util.StrToPtr("/foo/bar"),
							Hard:   util.BoolToPtr(true),
						},
					},
				},
			},
			warn: errors.ErrHardLinkSpecifiesOwner,
			at:   path.New("", "links", 0, "user", "id"),
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{
							Path: "/quux",
							User: NodeUser{
								Name: util.StrToPtr("bovik"),
							},
						},
						LinkEmbedded1: LinkEmbedded1{
							Target: util.StrToPtr("/foo/bar"),
							Hard:   util.BoolToPtr(true),
						},
					},
				},
			},
			warn: errors.ErrHardLinkSpecifiesOwner,
			at:   path.New("", "links", 0, "user", "name"),
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{
							Path: "/quux",
							Group: NodeGroup{
								ID: util.IntToPtr(10),
							},
						},
						LinkEmbedded1: LinkEmbedded1{
							Target: util.StrToPtr("/foo/bar"),
							Hard:   util.BoolToPtr(true),
						},
					},
				},
			},
			warn: errors.ErrHardLinkSpecifiesOwner,
			at:   path.New("", "links", 0, "group", "id"),
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{
							Path: "/quux",
							Group: NodeGroup{
								Name: util.StrToPtr("bovik"),
							},
						},
						LinkEmbedded1: LinkEmbedded1{
							Target: util.StrToPtr("/foo/bar"),
							Hard:   util.BoolToPtr(true),
						},
					},
				},
			},
			warn: errors.ErrHardLinkSpecifiesOwner,
			at:   path.New("", "links", 0, "group", "name"),
		},
	}

	for i, test := range tests {
		r := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(test.at, test.err)
		expected.AddOnWarn(test.at, test.warn)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad report: want %v, got %v", i, expected, r)
		}
	}
}

func TestStorageValidateWarnings(t *testing.T) {
	tests := []struct {
		in  Storage
		at  path.ContextPath
		out error
	}{
		// test a disk with partitions with the same 'device' as a filesystem returns ErrPartitionsOverwritten
		{
			in: Storage{
				Disks: []Disk{
					{
						Device: "/dev/sda",
						Partitions: []Partition{
							{}, {},
						},
					},
				},
				Filesystems: []Filesystem{
					{
						Device: "/dev/sda",
					},
				},
			},
			out: errors.ErrPartitionsOverwritten,
			at:  path.New("", "filesystems", 0, "device"),
		},
		// test a disk with the same 'device' and 'WipeTable:=true' as a configured filesystem returns ErrFilesystemImplicitWipe
		{
			in: Storage{
				Disks: []Disk{
					{
						Device:    "/dev/sda",
						WipeTable: util.BoolToPtr(true),
					},
				},
				Filesystems: []Filesystem{
					{
						Device: "/dev/sda",
					},
				},
			},
			out: errors.ErrFilesystemImplicitWipe,
			at:  path.New("", "filesystems", 0, "device"),
		},
		// test a disk with the same 'device' and 'WipeTable:=false' as a configured filesystem returns nil
		{
			in: Storage{
				Disks: []Disk{
					{
						Device:    "/dev/sda",
						WipeTable: util.BoolToPtr(false),
					},
				},
				Filesystems: []Filesystem{
					{
						Device: "/dev/sda",
					},
				},
			},
			out: nil,
		},
		// test a disk with no partitions with the same 'device' as a filesystem returns nil
		{
			in: Storage{
				Disks: []Disk{
					{
						Device: "/dev/sdb",
					},
				},
				Filesystems: []Filesystem{
					{
						Device: "/dev/sdb",
					},
				},
			},
			out: nil,
		},
	}

	for i, test := range tests {
		r := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnWarn(test.at, test.out)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad report: want %v, got %v", i, expected, r)
		}
	}
}
