// Copyright 2015 CoreOS, Inc.
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

package files

import (
	"reflect"
	"sort"
	"testing"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec/util"
)

func TestEntrySort(t *testing.T) {
	type in struct {
		data []types.Directory
	}

	type out struct {
		data []types.Directory
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in: in{data: []types.Directory{
				{
					Node: types.Node{
						Path: "/a/b/c/d/e/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c/d/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/",
					},
				},
			}},
			out: out{data: []types.Directory{
				{
					Node: types.Node{
						Path: "/a/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c/d/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c/d/e/",
					},
				},
			}},
		},
		{
			in: in{data: []types.Directory{
				{
					Node: types.Node{
						Path: "/a////b/c/d/e/",
					},
				},
				{
					Node: types.Node{
						Path: "/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c//d/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/",
					},
				},
			}},
			out: out{data: []types.Directory{
				{
					Node: types.Node{
						Path: "/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/b/c//d/",
					},
				},
				{
					Node: types.Node{
						Path: "/a////b/c/d/e/",
					},
				},
			}},
		},
		{
			in: in{data: []types.Directory{
				{
					Node: types.Node{
						Path: "/a/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/../a/b",
					},
				},
				{
					Node: types.Node{
						Path: "/",
					},
				},
			}},
			out: out{data: []types.Directory{
				{
					Node: types.Node{
						Path: "/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/",
					},
				},
				{
					Node: types.Node{
						Path: "/a/../a/b",
					},
				},
			}},
		},
	}

	for i, test := range tests {
		entries := []filesystemEntry{}
		for _, entry := range test.in.data {
			entries = append(entries, dirEntry(entry))
		}
		sort.Slice(entries, func(i, j int) bool { return util.Depth(entries[i].node().Path) < util.Depth(entries[j].node().Path) })
		outpaths := make([]types.Directory, len(test.in.data))
		for j, dir := range entries {
			outpaths[j].Node.Path = dir.node().Path
		}
		if !reflect.DeepEqual(test.out.data, outpaths) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.data, outpaths)
		}
	}
}
