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

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"

	"github.com/stretchr/testify/assert"
)

var (
	configURL = "http://example.com/myconf.ign"
	caURL     = "http://example.com/myca.cert"
	fileURL   = "http://example.com/myfile.txt"
)

func toPointer(val string) *string {
	return &val
}

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
								Append: []types.Resource{
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
								Contents: types.Resource{
									Compression: util.StrToPtr("gzip"),
								},
							},
						},
					},
				},
				Passwd: types.Passwd{
					Users: []types.PasswdUser{
						{
							Name: "bovik",
							SSHAuthorizedKeys: []types.SSHAuthorizedKey{
								"one",
								"two",
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
								Append: []types.Resource{
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
				Passwd: types.Passwd{
					Users: []types.PasswdUser{
						{
							Name: "bovik",
							SSHAuthorizedKeys: []types.SSHAuthorizedKey{
								"three",
								"two",
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
								Append: []types.Resource{
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
				Passwd: types.Passwd{
					Users: []types.PasswdUser{
						{
							Name: "bovik",
							SSHAuthorizedKeys: []types.SSHAuthorizedKey{
								"one",
								"two",
								"three",
							},
						},
					},
				},
			},
		},

		// merge config reference that contains HTTP headers
		{
			in1: types.Config{
				Ignition: types.Ignition{
					Config: types.IgnitionConfig{
						Merge: []types.Resource{
							{
								Source: &configURL,
								HTTPHeaders: []types.HTTPHeader{
									{
										Name:  "old-header",
										Value: toPointer("old-value"),
									},
									{
										Name:  "same-header",
										Value: toPointer("old-value"),
									},
									{
										Name:  "to-remove-header",
										Value: toPointer("some-value"),
									},
								},
							},
						},
					},
				},
			},
			in2: types.Config{
				Ignition: types.Ignition{
					Config: types.IgnitionConfig{
						Merge: []types.Resource{
							{
								Source: &configURL,
								HTTPHeaders: []types.HTTPHeader{
									{
										Name: "to-remove-header",
									},
									{
										Name:  "new-header",
										Value: toPointer("new-value"),
									},
									{
										Name:  "same-header",
										Value: toPointer("new-value"),
									},
								},
							},
						},
					},
				},
			},
			out: types.Config{
				Ignition: types.Ignition{
					Config: types.IgnitionConfig{
						Merge: []types.Resource{
							{
								Source: &configURL,
								HTTPHeaders: []types.HTTPHeader{
									{
										Name:  "old-header",
										Value: toPointer("old-value"),
									},
									{
										Name:  "same-header",
										Value: toPointer("new-value"),
									},
									{
										Name:  "new-header",
										Value: toPointer("new-value"),
									},
								},
							},
						},
					},
				},
			},
		},

		// replace config reference that contains HTTP headers
		{
			in1: types.Config{
				Ignition: types.Ignition{
					Config: types.IgnitionConfig{
						Replace: types.Resource{
							Source: &configURL,
							HTTPHeaders: []types.HTTPHeader{
								{
									Name:  "old-header",
									Value: toPointer("old-value"),
								},
								{
									Name:  "same-header",
									Value: toPointer("old-value"),
								},
								{
									Name:  "to-remove-header",
									Value: toPointer("some-value"),
								},
							},
						},
					},
				},
			},
			in2: types.Config{
				Ignition: types.Ignition{
					Config: types.IgnitionConfig{
						Replace: types.Resource{
							Source: &configURL,
							HTTPHeaders: []types.HTTPHeader{
								{
									Name: "to-remove-header",
								},
								{
									Name:  "new-header",
									Value: toPointer("new-value"),
								},
								{
									Name:  "same-header",
									Value: toPointer("new-value"),
								},
							},
						},
					},
				},
			},
			out: types.Config{
				Ignition: types.Ignition{
					Config: types.IgnitionConfig{
						Replace: types.Resource{
							Source: &configURL,
							HTTPHeaders: []types.HTTPHeader{
								{
									Name:  "old-header",
									Value: toPointer("old-value"),
								},
								{
									Name:  "same-header",
									Value: toPointer("new-value"),
								},
								{
									Name:  "new-header",
									Value: toPointer("new-value"),
								},
							},
						},
					},
				},
			},
		},

		// CA reference that contains HTTP headers
		{
			in1: types.Config{
				Ignition: types.Ignition{
					Security: types.Security{
						TLS: types.TLS{
							CertificateAuthorities: []types.Resource{
								{
									Source: util.StrToPtr(caURL),
									HTTPHeaders: []types.HTTPHeader{
										{
											Name:  "old-header",
											Value: toPointer("old-value"),
										},
										{
											Name:  "same-header",
											Value: toPointer("old-value"),
										},
										{
											Name:  "to-remove-header",
											Value: toPointer("some-value"),
										},
									},
								},
							},
						},
					},
				},
			},
			in2: types.Config{
				Ignition: types.Ignition{
					Security: types.Security{
						TLS: types.TLS{
							CertificateAuthorities: []types.Resource{
								{
									Source: util.StrToPtr(caURL),
									HTTPHeaders: []types.HTTPHeader{
										{
											Name: "to-remove-header",
										},
										{
											Name:  "new-header",
											Value: toPointer("new-value"),
										},
										{
											Name:  "same-header",
											Value: toPointer("new-value"),
										},
									},
								},
							},
						},
					},
				},
			},
			out: types.Config{
				Ignition: types.Ignition{
					Security: types.Security{
						TLS: types.TLS{
							CertificateAuthorities: []types.Resource{
								{
									Source: util.StrToPtr(caURL),
									HTTPHeaders: []types.HTTPHeader{
										{
											Name:  "old-header",
											Value: toPointer("old-value"),
										},
										{
											Name:  "same-header",
											Value: toPointer("new-value"),
										},
										{
											Name:  "new-header",
											Value: toPointer("new-value"),
										},
									},
								},
							},
						},
					},
				},
			},
		},

		// file contents that contain HTTP headers
		{
			in1: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: &fileURL,
									HTTPHeaders: []types.HTTPHeader{
										{
											Name:  "old-header",
											Value: toPointer("old-value"),
										},
										{
											Name:  "same-header",
											Value: toPointer("old-value"),
										},
										{
											Name:  "to-remove-header",
											Value: toPointer("some-value"),
										},
									},
								},
							},
						},
					},
				},
			},
			in2: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: &fileURL,
									HTTPHeaders: []types.HTTPHeader{
										{
											Name: "to-remove-header",
										},
										{
											Name:  "new-header",
											Value: toPointer("new-value"),
										},
										{
											Name:  "same-header",
											Value: toPointer("new-value"),
										},
									},
								},
							},
						},
					},
				},
			},
			out: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: &fileURL,
									HTTPHeaders: []types.HTTPHeader{
										{
											Name:  "old-header",
											Value: toPointer("old-value"),
										},
										{
											Name:  "same-header",
											Value: toPointer("new-value"),
										},
										{
											Name:  "new-header",
											Value: toPointer("new-value"),
										},
									},
								},
							},
						},
					},
				},
			},
		},

		// file contents that contain HTTP headers
		{
			in1: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							FileEmbedded1: types.FileEmbedded1{
								Append: []types.Resource{
									{
										Source: &fileURL,
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "old-header",
												Value: toPointer("old-value"),
											},
											{
												Name:  "same-header",
												Value: toPointer("old-value"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			in2: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							FileEmbedded1: types.FileEmbedded1{
								Append: []types.Resource{
									{
										Source: &fileURL,
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "new-header",
												Value: toPointer("new-value"),
											},
											{
												Name:  "same-header",
												Value: toPointer("new-value"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			out: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							FileEmbedded1: types.FileEmbedded1{
								Append: []types.Resource{
									{
										Source: &fileURL,
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "old-header",
												Value: toPointer("old-value"),
											},
											{
												Name:  "same-header",
												Value: toPointer("old-value"),
											},
										},
									},
									{
										Source: &fileURL,
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "new-header",
												Value: toPointer("new-value"),
											},
											{
												Name:  "same-header",
												Value: toPointer("new-value"),
											},
										},
									},
								},
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

		assert.Equal(t, test.out, out, "#%d bad merge", i)
	}
}
