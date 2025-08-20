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
	"fmt"
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		in  Config
		out error
		at  path.ContextPath
	}{
		// test 0: file conflicts with systemd dropin file, error
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{
							Node: Node{Path: "/etc/systemd/system/foo.service.d/bar.conf"},
						},
					},
				},
				Systemd: Systemd{
					Units: []Unit{
						{
							Name: "foo.service",
							Dropins: []Dropin{
								{
									Name:     "bar.conf",
									Contents: util.StrToPtr("[Foo]\nQux=Bar"),
								},
							},
						},
					},
				},
			},
			out: errors.ErrPathConflictsSystemd,
			at:  path.New("json", "storage", "files", 0, "path"),
		},
		// test 1: file conflicts with systemd unit, error
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{
							Node: Node{Path: "/etc/systemd/system/foo.service"},
						},
					},
				},
				Systemd: Systemd{
					Units: []Unit{
						{
							Name:     "foo.service",
							Contents: util.StrToPtr("[Foo]\nQux=Bar"),
							Enabled:  util.BoolToPtr(true),
						},
					},
				},
			},
			out: errors.ErrPathConflictsSystemd,
			at:  path.New("json", "storage", "files", 0, "path"),
		},
		// test 2: directory conflicts with systemd dropin file, error
		{
			in: Config{
				Storage: Storage{
					Directories: []Directory{
						{
							Node: Node{Path: "/etc/systemd/system/foo.service.d/bar.conf"},
						},
					},
				},
				Systemd: Systemd{
					[]Unit{
						{
							Name: "foo.service",
							Dropins: []Dropin{
								{
									Name:     "bar.conf",
									Contents: util.StrToPtr("[Foo]\nQux=Bar"),
								},
							},
						},
					},
				},
			},
			out: errors.ErrPathConflictsSystemd,
			at:  path.New("json", "storage", "directories", 0, "path"),
		},
		// test 3: directory conflicts with systemd unit, error
		{
			in: Config{
				Storage: Storage{
					Directories: []Directory{
						{
							Node: Node{Path: "/etc/systemd/system/foo.service"},
						},
					},
				},
				Systemd: Systemd{
					[]Unit{
						{
							Name:     "foo.service",
							Contents: util.StrToPtr("[foo]\nQux=Baz"),
							Enabled:  util.BoolToPtr(true),
						},
					},
				},
			},
			out: errors.ErrPathConflictsSystemd,
			at:  path.New("json", "storage", "directories", 0, "path"),
		},
		// test 4: link conflicts with systemd dropin file, error
		{
			in: Config{
				Storage: Storage{
					Links: []Link{
						{
							Node:          Node{Path: "/etc/systemd/system/foo.service.d/bar.conf"},
							LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/qux.conf")},
						},
					},
				},
				Systemd: Systemd{
					[]Unit{
						{
							Name: "foo.service",
							Dropins: []Dropin{
								{
									Name:     "bar.conf",
									Contents: util.StrToPtr("[Foo]\nQux=Bar"),
								},
							},
						},
					},
				},
			},
			out: errors.ErrPathConflictsSystemd,
			at:  path.New("json", "storage", "links", 0, "path"),
		},
		// test 5: link conflicts with systemd unit, error
		{
			in: Config{
				Storage: Storage{
					Links: []Link{
						{
							Node:          Node{Path: "/etc/systemd/system/foo.service"},
							LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/qux.conf")},
						},
					},
				},
				Systemd: Systemd{
					[]Unit{
						{
							Name:     "foo.service",
							Contents: util.StrToPtr("[foo]\nQux=Baz"),
							Enabled:  util.BoolToPtr(true),
						},
					},
				},
			},
			out: errors.ErrPathConflictsSystemd,
			at:  path.New("json", "storage", "links", 0, "path"),
		},
		// test 6: non-conflicting scenarios
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{
							Node: Node{Path: "/etc/systemd/system/bar.service.d/baz.conf"},
						},
						{
							Node: Node{Path: "/etc/systemd/system/bar.service"},
						},
						{
							Node: Node{Path: "/etc/systemd/system/foo.service.d/qux.conf"},
						},
					},
					Links: []Link{
						{
							Node:          Node{Path: "/etc/systemd/system/qux.service"},
							LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/qux.conf")},
						},
						{
							Node:          Node{Path: "/etc/systemd/system/quux.service.d/foo.conf"},
							LinkEmbedded1: LinkEmbedded1{Target: util.StrToPtr("/foo.conf")},
						},
					},
					Directories: []Directory{
						{
							Node: Node{Path: "/etc/systemd/system/quux.service.d"},
						},
					},
				},
				Systemd: Systemd{
					Units: []Unit{
						{
							Name:     "foo.service",
							Contents: util.StrToPtr("[Foo]\nQux=Baz"),
							Enabled:  util.BoolToPtr(true),
						},
						{
							Name: "bar.service",
							Dropins: []Dropin{
								{
									Name: "baz.conf",
								},
							},
							Enabled: util.BoolToPtr(true),
						},
						{
							Name: "qux.service",
							Dropins: []Dropin{
								{
									Name:     "bar.conf",
									Contents: util.StrToPtr("[Foo]\nQux=Baz"),
								},
							},
						},
						{
							Name:     "quux.service",
							Contents: util.StrToPtr("[Foo]\nQux=Baz"),
							Enabled:  util.BoolToPtr(true),
						},
					},
				},
			},
		},

		// test 7: file path conflicts with another file path, should error
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{Node: Node{Path: "/foo/bar"}},
						{Node: Node{Path: "/foo/bar/baz"}},
					},
				},
			},
			out: fmt.Errorf("invalid entry at path /foo/bar/baz: %w", errors.ErrMissLabeledDir),
			at:  path.New("json", "storage", "files", 1, "path"),
		},
		// test 8: file path conflicts with link path, should error
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{Node: Node{Path: "/foo/bar"}},
					},
					Links: []Link{
						{Node: Node{Path: "/foo/bar/baz"}},
					},
				},
			},
			out: fmt.Errorf("invalid entry at path /foo/bar/baz: %w", errors.ErrMissLabeledDir),
			at:  path.New("json", "storage", "links", 0, "path"),
		},

		// test 9: file path conflicts with directory path, should error
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{Node: Node{Path: "/foo/bar"}},
					},
					Directories: []Directory{
						{Node: Node{Path: "/foo/bar/baz"}},
					},
				},
			},
			out: fmt.Errorf("invalid entry at path /foo/bar/baz: %w", errors.ErrMissLabeledDir),
			at:  path.New("json", "storage", "directories", 0, "path"),
		},

		// test 10: non-conflicting scenarios with same parent directory, should not error
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{Node: Node{Path: "/foo/bar/baz"}},
					},
					Directories: []Directory{
						{Node: Node{Path: "/foo/bar"}},
					},
				},
			},
		},
		// test 11: non-conflicting scenarios with a link, should not error
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{Node: Node{Path: "/foo/bar"}},
					},
					Links: []Link{
						{Node: Node{Path: "/baz/qux"}},
					},
				},
			},
		},
		// test 12: deep path conflict - file conflicts with deeper nested path, should error
		{
			in: Config{
				Storage: Storage{
					Files: []File{
						{Node: Node{Path: "/foo/bar"}},
						{Node: Node{Path: "/foo/bar/baz/deep/file"}},
					},
				},
			},
			out: fmt.Errorf("invalid entry at path /foo/bar/baz/deep/file: %w", errors.ErrMissLabeledDir),
			at:  path.New("json", "storage", "files", 1, "path"),
		},
		// test 13: multiple levels with directories, should not error
		{
			in: Config{
				Storage: Storage{
					Directories: []Directory{
						{Node: Node{Path: "/foo"}},
						{Node: Node{Path: "/foo/bar"}},
					},
					Files: []File{
						{Node: Node{Path: "/foo/bar/baz"}},
					},
				},
			},
		},
		// test 14: link conflicts with file parent, should error
		{
			in: Config{
				Storage: Storage{
					Links: []Link{
						{Node: Node{Path: "/foo/bar"}},
					},
					Files: []File{
						{Node: Node{Path: "/foo/bar/child"}},
					},
				},
			},
			out: fmt.Errorf("invalid entry at path /foo/bar/child: %w", errors.ErrMissLabeledDir),
			at:  path.New("json", "storage", "files", 0, "path"),
		},
	}
	for i, test := range tests {
		r := test.in.Validate(path.New("json"))
		expected := report.Report{}
		expected.AddOnError(test.at, test.out)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad error: expected : %v, got %v", i, expected, r)
		}
	}
}
