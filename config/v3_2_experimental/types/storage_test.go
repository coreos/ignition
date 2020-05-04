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

func TestStorageValidate(t *testing.T) {
	tests := []struct {
		in  Storage
		at  path.ContextPath
		out error
	}{
		{
			in:  Storage{},
			out: nil,
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{Path: "/foo"},
					},
					{
						Node: Node{Path: "/quux"},
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
			out: nil,
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{Path: "/foo"},
					},
				},
				Files: []File{
					{
						Node: Node{Path: "/foo/bar"},
					},
				},
			},
			out: errors.ErrFileUsedSymlink,
			at:  path.New("", "files", 0),
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{Path: "/foo"},
					},
				},
				Directories: []Directory{
					{
						Node: Node{Path: "/foo/bar"},
					},
				},
			},
			out: errors.ErrDirectoryUsedSymlink,
			at:  path.New("", "directories", 0),
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{Path: "/foo"},
					},
					{
						Node: Node{Path: "/foo/bar"},
					},
				},
			},
			out: errors.ErrLinkUsedSymlink,
			at:  path.New("", "links", 1),
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{Path: "/foo"},
					},
					{
						Node: Node{Path: "/foob/bar"},
					},
				},
			},
			out: nil,
		},
		{
			in: Storage{
				Links: []Link{
					{
						Node: Node{Path: "/quux"},
						LinkEmbedded1: LinkEmbedded1{
							Target: "/foo/bar",
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
			out: errors.ErrHardLinkToDirectory,
			at:  path.New("", "links", 0),
		},
	}

	for i, test := range tests {
		r := test.in.Validate(path.ContextPath{})
		expected := report.Report{}
		expected.AddOnError(test.at, test.out)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad report: want %v, got %v", i, expected, r)
		}
	}
}
