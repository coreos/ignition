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
		in   Storage
		at   path.ContextPath
		err  error
		warn error
	}{
		{
			in: Storage{},
		},
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
