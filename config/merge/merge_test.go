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

package merge

import (
	"reflect"
	"testing"

	"github.com/coreos/ignition/config/util"
	"github.com/coreos/ignition/config/v3_0/types"

	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	type test struct {
		in1 types.Config
		in2 types.Config
		out types.Config
	}

	tests := []test{
		{
			// case 1: merging empty configs is empty
		},
		{
			in1: types.Config{
				Ignition: types.Ignition{Version: "1234"},
			},
			in2: types.Config{
				Ignition: types.Ignition{Version: "haha this isn't validated"},
			},
			out: types.Config{
				Ignition: types.Ignition{Version: "haha this isn't validated"},
			},
		},
		{
			in1: types.Config{
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device:    "foo",
							WipeTable: util.BoolToPtr(true),
							Partitions: []types.Partition{
								{
									Number:   1,
									Label:    util.StrToPtr("label"),
									StartMiB: util.IntToPtr(4),
								},
							},
						},
						{
							Device:    "bar",
							WipeTable: util.BoolToPtr(true),
						},
					},
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/foo",
							},
							FileEmbedded1: types.FileEmbedded1{
								Append: []types.FileContents{
									{
										Source: util.StrToPtr("source1"),
									},
								},
							},
						},
						{
							Node: types.Node{
								Path: "/bar",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.FileContents{
									Compression: util.StrToPtr("gzip"),
								},
							},
						},
					},
				},
			},
			in2: types.Config{
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device:    "foo",
							WipeTable: util.BoolToPtr(false),
							Partitions: []types.Partition{
								{
									Number: 1,
									Label:  util.StrToPtr("labelchanged"),
								},
								{
									Number: 2,
									Label:  util.StrToPtr("label2"),
								},
							},
						},
						{
							Device: "bar",
						},
						{
							Device:    "baz",
							WipeTable: util.BoolToPtr(true),
						},
					},
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/foo",
							},
							FileEmbedded1: types.FileEmbedded1{
								Append: []types.FileContents{
									{
										Source: util.StrToPtr("source1"),
									},
									{
										Source: util.StrToPtr("source2"),
									},
								},
							},
						},
					},
					Directories: []types.Directory{
						{
							Node: types.Node{
								Path: "/bar",
							},
						},
					},
					Links: []types.Link{
						{
							Node: types.Node{
								Path: "/baz",
							},
						},
					},
				},
			},
			out: types.Config{
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device:    "foo",
							WipeTable: util.BoolToPtr(false),
							Partitions: []types.Partition{
								{
									Number:   1,
									Label:    util.StrToPtr("labelchanged"),
									StartMiB: util.IntToPtr(4),
								},
								{
									Number: 2,
									Label:  util.StrToPtr("label2"),
								},
							},
						},
						{
							Device:    "bar",
							WipeTable: util.BoolToPtr(true),
						},
						{
							Device:    "baz",
							WipeTable: util.BoolToPtr(true),
						},
					},
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/foo",
							},
							FileEmbedded1: types.FileEmbedded1{
								Append: []types.FileContents{
									{
										Source: util.StrToPtr("source1"),
									},
									{
										Source: util.StrToPtr("source1"),
									},
									{
										Source: util.StrToPtr("source2"),
									},
								},
							},
						},
					},
					Directories: []types.Directory{
						{
							Node: types.Node{
								Path: "/bar",
							},
						},
					},
					Links: []types.Link{
						{
							Node: types.Node{
								Path: "/baz",
							},
						},
					},
				},
			},
		},
	}

	for i, test := range tests {
		in1v := reflect.ValueOf(test.in1)
		in2v := reflect.ValueOf(test.in2)
		out := MergeStruct(in1v, in2v).Interface().(types.Config)

		assert.Equal(t, test.out, out, "#%d bas merge", i)
	}
}
