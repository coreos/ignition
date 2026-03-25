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
	"testing"

	"github.com/coreos/ignition/v2/config/util"
	v3_2 "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"

	"github.com/coreos/vcontext/path"
	"github.com/stretchr/testify/assert"
)

var (
	configURL = "http://example.com/myconf.ign"
	caURL     = "http://example.com/myca.cert"
	fileURL   = "http://example.com/myfile.txt"
)

func TestMerge(t *testing.T) {
	type test struct {
		in1        types.Config
		in2        types.Config
		out        types.Config
		transcript Transcript
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
			transcript: Transcript{[]Mapping{
				{path.New(TAG_CHILD, "ignition", "version"), path.New(TAG_RESULT, "ignition", "version")},
				{path.New(TAG_PARENT, "ignition"), path.New(TAG_RESULT, "ignition")},
				{path.New(TAG_CHILD, "ignition"), path.New(TAG_RESULT, "ignition")},
			}},
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
			transcript: Transcript{[]Mapping{
				{path.New(TAG_CHILD, "passwd", "users", 0, "name"), path.New(TAG_RESULT, "passwd", "users", 0, "name")},
				{path.New(TAG_PARENT, "passwd", "users", 0, "sshAuthorizedKeys", 0), path.New(TAG_RESULT, "passwd", "users", 0, "sshAuthorizedKeys", 0)},
				{path.New(TAG_CHILD, "passwd", "users", 0, "sshAuthorizedKeys", 1), path.New(TAG_RESULT, "passwd", "users", 0, "sshAuthorizedKeys", 1)},
				{path.New(TAG_CHILD, "passwd", "users", 0, "sshAuthorizedKeys", 0), path.New(TAG_RESULT, "passwd", "users", 0, "sshAuthorizedKeys", 2)},
				{path.New(TAG_PARENT, "passwd", "users", 0, "sshAuthorizedKeys"), path.New(TAG_RESULT, "passwd", "users", 0, "sshAuthorizedKeys")},
				{path.New(TAG_CHILD, "passwd", "users", 0, "sshAuthorizedKeys"), path.New(TAG_RESULT, "passwd", "users", 0, "sshAuthorizedKeys")},
				{path.New(TAG_PARENT, "passwd", "users", 0), path.New(TAG_RESULT, "passwd", "users", 0)},
				{path.New(TAG_CHILD, "passwd", "users", 0), path.New(TAG_RESULT, "passwd", "users", 0)},
				{path.New(TAG_PARENT, "passwd", "users"), path.New(TAG_RESULT, "passwd", "users")},
				{path.New(TAG_CHILD, "passwd", "users"), path.New(TAG_RESULT, "passwd", "users")},
				{path.New(TAG_PARENT, "passwd"), path.New(TAG_RESULT, "passwd")},
				{path.New(TAG_CHILD, "passwd"), path.New(TAG_RESULT, "passwd")},
				{path.New(TAG_CHILD, "storage", "directories", 0, "path"), path.New(TAG_RESULT, "storage", "directories", 0, "path")},
				{path.New(TAG_CHILD, "storage", "directories", 0), path.New(TAG_RESULT, "storage", "directories", 0)},
				{path.New(TAG_CHILD, "storage", "directories"), path.New(TAG_RESULT, "storage", "directories")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "device"), path.New(TAG_RESULT, "storage", "disks", 0, "device")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 0, "label"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 0, "label")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 0, "number"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 0, "number")},
				{path.New(TAG_PARENT, "storage", "disks", 0, "partitions", 0, "startMiB"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 0, "startMiB")},
				{path.New(TAG_PARENT, "storage", "disks", 0, "partitions", 0), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 0)},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 0), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 0)},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 1, "label"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 1, "label")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 1, "number"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 1, "number")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 1), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 1)},
				{path.New(TAG_PARENT, "storage", "disks", 0, "partitions"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "wipeTable"), path.New(TAG_RESULT, "storage", "disks", 0, "wipeTable")},
				{path.New(TAG_PARENT, "storage", "disks", 0), path.New(TAG_RESULT, "storage", "disks", 0)},
				{path.New(TAG_CHILD, "storage", "disks", 0), path.New(TAG_RESULT, "storage", "disks", 0)},
				{path.New(TAG_CHILD, "storage", "disks", 1, "device"), path.New(TAG_RESULT, "storage", "disks", 1, "device")},
				{path.New(TAG_PARENT, "storage", "disks", 1, "wipeTable"), path.New(TAG_RESULT, "storage", "disks", 1, "wipeTable")},
				{path.New(TAG_PARENT, "storage", "disks", 1), path.New(TAG_RESULT, "storage", "disks", 1)},
				{path.New(TAG_CHILD, "storage", "disks", 1), path.New(TAG_RESULT, "storage", "disks", 1)},
				{path.New(TAG_CHILD, "storage", "disks", 2, "device"), path.New(TAG_RESULT, "storage", "disks", 2, "device")},
				{path.New(TAG_CHILD, "storage", "disks", 2, "wipeTable"), path.New(TAG_RESULT, "storage", "disks", 2, "wipeTable")},
				{path.New(TAG_CHILD, "storage", "disks", 2), path.New(TAG_RESULT, "storage", "disks", 2)},
				{path.New(TAG_PARENT, "storage", "disks"), path.New(TAG_RESULT, "storage", "disks")},
				{path.New(TAG_CHILD, "storage", "disks"), path.New(TAG_RESULT, "storage", "disks")},
				{path.New(TAG_CHILD, "storage", "files", 0, "path"), path.New(TAG_RESULT, "storage", "files", 0, "path")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "source")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "source")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 1)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 1, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 2, "source")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 1), path.New(TAG_RESULT, "storage", "files", 0, "append", 2)},
				{path.New(TAG_PARENT, "storage", "files", 0, "append"), path.New(TAG_RESULT, "storage", "files", 0, "append")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append"), path.New(TAG_RESULT, "storage", "files", 0, "append")},
				{path.New(TAG_PARENT, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_PARENT, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_CHILD, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_CHILD, "storage", "links", 0, "path"), path.New(TAG_RESULT, "storage", "links", 0, "path")},
				{path.New(TAG_CHILD, "storage", "links", 0), path.New(TAG_RESULT, "storage", "links", 0)},
				{path.New(TAG_CHILD, "storage", "links"), path.New(TAG_RESULT, "storage", "links")},
				{path.New(TAG_PARENT, "storage"), path.New(TAG_RESULT, "storage")},
				{path.New(TAG_CHILD, "storage"), path.New(TAG_RESULT, "storage")},
			}},
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
										Value: util.StrToPtr("old-value"),
									},
									{
										Name:  "same-header",
										Value: util.StrToPtr("old-value"),
									},
									{
										Name:  "to-remove-header",
										Value: util.StrToPtr("some-value"),
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
										Value: util.StrToPtr("new-value"),
									},
									{
										Name:  "same-header",
										Value: util.StrToPtr("new-value"),
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
										Value: util.StrToPtr("old-value"),
									},
									{
										Name:  "same-header",
										Value: util.StrToPtr("new-value"),
									},
									{
										Name:  "new-header",
										Value: util.StrToPtr("new-value"),
									},
								},
							},
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0, "httpHeaders", 0), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 0)},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 2, "name"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 2, "value"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 1, "value")},
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0, "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 1)},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 2), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 1)},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 1, "name"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 2, "name")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 1, "value"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 2, "value")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 2)},
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0, "httpHeaders"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "source"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "source")},
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0), path.New(TAG_RESULT, "ignition", "config", "merge", 0)},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0), path.New(TAG_RESULT, "ignition", "config", "merge", 0)},
				{path.New(TAG_PARENT, "ignition", "config", "merge"), path.New(TAG_RESULT, "ignition", "config", "merge")},
				{path.New(TAG_CHILD, "ignition", "config", "merge"), path.New(TAG_RESULT, "ignition", "config", "merge")},
				{path.New(TAG_PARENT, "ignition", "config"), path.New(TAG_RESULT, "ignition", "config")},
				{path.New(TAG_CHILD, "ignition", "config"), path.New(TAG_RESULT, "ignition", "config")},
				{path.New(TAG_PARENT, "ignition"), path.New(TAG_RESULT, "ignition")},
				{path.New(TAG_CHILD, "ignition"), path.New(TAG_RESULT, "ignition")},
			}},
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
									Value: util.StrToPtr("old-value"),
								},
								{
									Name:  "same-header",
									Value: util.StrToPtr("old-value"),
								},
								{
									Name:  "to-remove-header",
									Value: util.StrToPtr("some-value"),
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
									Value: util.StrToPtr("new-value"),
								},
								{
									Name:  "same-header",
									Value: util.StrToPtr("new-value"),
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
									Value: util.StrToPtr("old-value"),
								},
								{
									Name:  "same-header",
									Value: util.StrToPtr("new-value"),
								},
								{
									Name:  "new-header",
									Value: util.StrToPtr("new-value"),
								},
							},
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "ignition", "config", "replace", "httpHeaders", 0, "name"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "ignition", "config", "replace", "httpHeaders", 0, "value"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "ignition", "config", "replace", "httpHeaders", 0), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 0)},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 2, "name"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 2, "value"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 1, "value")},
				{path.New(TAG_PARENT, "ignition", "config", "replace", "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 1)},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 2), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 1)},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 1, "name"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 2, "name")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 1, "value"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 2, "value")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 2)},
				{path.New(TAG_PARENT, "ignition", "config", "replace", "httpHeaders"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "source"), path.New(TAG_RESULT, "ignition", "config", "replace", "source")},
				{path.New(TAG_PARENT, "ignition", "config", "replace"), path.New(TAG_RESULT, "ignition", "config", "replace")},
				{path.New(TAG_CHILD, "ignition", "config", "replace"), path.New(TAG_RESULT, "ignition", "config", "replace")},
				{path.New(TAG_PARENT, "ignition", "config"), path.New(TAG_RESULT, "ignition", "config")},
				{path.New(TAG_CHILD, "ignition", "config"), path.New(TAG_RESULT, "ignition", "config")},
				{path.New(TAG_PARENT, "ignition"), path.New(TAG_RESULT, "ignition")},
				{path.New(TAG_CHILD, "ignition"), path.New(TAG_RESULT, "ignition")},
			}},
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
											Value: util.StrToPtr("old-value"),
										},
										{
											Name:  "same-header",
											Value: util.StrToPtr("old-value"),
										},
										{
											Name:  "to-remove-header",
											Value: util.StrToPtr("some-value"),
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
											Value: util.StrToPtr("new-value"),
										},
										{
											Name:  "same-header",
											Value: util.StrToPtr("new-value"),
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
											Value: util.StrToPtr("old-value"),
										},
										{
											Name:  "same-header",
											Value: util.StrToPtr("new-value"),
										},
										{
											Name:  "new-header",
											Value: util.StrToPtr("new-value"),
										},
									},
								},
							},
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0)},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2, "name"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2, "value"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1, "value")},
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1)},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1)},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1, "name"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2, "name")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1, "value"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2, "value")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2)},
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "source"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "source")},
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0)},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0)},
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities")},
				{path.New(TAG_PARENT, "ignition", "security", "tls"), path.New(TAG_RESULT, "ignition", "security", "tls")},
				{path.New(TAG_CHILD, "ignition", "security", "tls"), path.New(TAG_RESULT, "ignition", "security", "tls")},
				{path.New(TAG_PARENT, "ignition", "security"), path.New(TAG_RESULT, "ignition", "security")},
				{path.New(TAG_CHILD, "ignition", "security"), path.New(TAG_RESULT, "ignition", "security")},
				{path.New(TAG_PARENT, "ignition"), path.New(TAG_RESULT, "ignition")},
				{path.New(TAG_CHILD, "ignition"), path.New(TAG_RESULT, "ignition")},
			}},
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
											Value: util.StrToPtr("old-value"),
										},
										{
											Name:  "same-header",
											Value: util.StrToPtr("old-value"),
										},
										{
											Name:  "to-remove-header",
											Value: util.StrToPtr("some-value"),
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
											Value: util.StrToPtr("new-value"),
										},
										{
											Name:  "same-header",
											Value: util.StrToPtr("new-value"),
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
											Value: util.StrToPtr("old-value"),
										},
										{
											Name:  "same-header",
											Value: util.StrToPtr("new-value"),
										},
										{
											Name:  "new-header",
											Value: util.StrToPtr("new-value"),
										},
									},
								},
							},
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 2, "name"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 2, "value"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 1, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 1), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 1)},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 2), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 1)},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 1, "name"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 2, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 1, "value"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 2, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 1), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 2)},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "source"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "source")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents"), path.New(TAG_RESULT, "storage", "files", 0, "contents")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents"), path.New(TAG_RESULT, "storage", "files", 0, "contents")},
				{path.New(TAG_PARENT, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_PARENT, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_CHILD, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_PARENT, "storage"), path.New(TAG_RESULT, "storage")},
				{path.New(TAG_CHILD, "storage"), path.New(TAG_RESULT, "storage")},
			}},
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
												Value: util.StrToPtr("old-value"),
											},
											{
												Name:  "same-header",
												Value: util.StrToPtr("old-value"),
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
												Value: util.StrToPtr("new-value"),
											},
											{
												Name:  "same-header",
												Value: util.StrToPtr("new-value"),
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
												Value: util.StrToPtr("old-value"),
											},
											{
												Name:  "same-header",
												Value: util.StrToPtr("old-value"),
											},
										},
									},
									{
										Source: &fileURL,
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "new-header",
												Value: util.StrToPtr("new-value"),
											},
											{
												Name:  "same-header",
												Value: util.StrToPtr("new-value"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0)},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 1, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 1, "name")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 1, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 1, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 1), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 1)},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "source")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 0, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 0, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 1, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 1, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 1, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 1), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 1)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "source")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 1)},
				{path.New(TAG_PARENT, "storage", "files", 0, "append"), path.New(TAG_RESULT, "storage", "files", 0, "append")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append"), path.New(TAG_RESULT, "storage", "files", 0, "append")},
				{path.New(TAG_PARENT, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_PARENT, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_CHILD, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_PARENT, "storage"), path.New(TAG_RESULT, "storage")},
				{path.New(TAG_CHILD, "storage"), path.New(TAG_RESULT, "storage")},
			}},
		},

		// strictly-parent or strictly-child subtrees
		{
			in1: types.Config{
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device:  "/dev/sda",
							Options: []types.FilesystemOption{"a", "b"},
						},
						{
							Device:  "/dev/sdb",
							Options: []types.FilesystemOption{"a", "b"},
						},
					},
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/a",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
								},
							},
						},
						{
							Node: types.Node{
								Path: "/b",
							},
							FileEmbedded1: types.FileEmbedded1{
								Mode: util.IntToPtr(0644),
							},
						},
						{
							Node: types.Node{
								Path: "/c",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
								},
							},
						},
					},
					Directories: []types.Directory{
						{
							Node: types.Node{
								Path: "/d",
							},
						},
					},
				},
			},
			in2: types.Config{
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device:       "/dev/sda",
							MountOptions: []types.MountOption{"c", "d"},
						},
						{
							Device:  "/dev/sdb",
							Options: []types.FilesystemOption{"c", "d"},
						},
					},
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/a",
							},
							FileEmbedded1: types.FileEmbedded1{
								Mode: util.IntToPtr(0644),
							},
						},
						{
							Node: types.Node{
								Path: "/b",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
								},
							},
						},
						{
							Node: types.Node{
								Path: "/c",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Compression: util.StrToPtr("gzip"),
								},
							},
						},
					},
					Links: []types.Link{
						{
							Node: types.Node{
								Path: "/e",
							},
						},
					},
				},
			},
			out: types.Config{
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device:       "/dev/sda",
							MountOptions: []types.MountOption{"c", "d"},
							Options:      []types.FilesystemOption{"a", "b"},
						},
						{
							Device:  "/dev/sdb",
							Options: []types.FilesystemOption{"a", "b", "c", "d"},
						},
					},
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/a",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
								},
								Mode: util.IntToPtr(0644),
							},
						},
						{
							Node: types.Node{
								Path: "/b",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
								},
								Mode: util.IntToPtr(0644),
							},
						},
						{
							Node: types.Node{
								Path: "/c",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Compression: util.StrToPtr("gzip"),
									Source:      util.StrToPtr("data:"),
								},
							},
						},
					},
					Directories: []types.Directory{
						{
							Node: types.Node{
								Path: "/d",
							},
						},
					},
					Links: []types.Link{
						{
							Node: types.Node{
								Path: "/e",
							},
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "storage", "directories", 0, "path"), path.New(TAG_RESULT, "storage", "directories", 0, "path")},
				{path.New(TAG_PARENT, "storage", "directories", 0), path.New(TAG_RESULT, "storage", "directories", 0)},
				{path.New(TAG_PARENT, "storage", "directories"), path.New(TAG_RESULT, "storage", "directories")},
				{path.New(TAG_CHILD, "storage", "files", 0, "path"), path.New(TAG_RESULT, "storage", "files", 0, "path")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "source"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "source")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents"), path.New(TAG_RESULT, "storage", "files", 0, "contents")},
				{path.New(TAG_CHILD, "storage", "files", 0, "mode"), path.New(TAG_RESULT, "storage", "files", 0, "mode")},
				{path.New(TAG_PARENT, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_CHILD, "storage", "files", 1, "path"), path.New(TAG_RESULT, "storage", "files", 1, "path")},
				{path.New(TAG_CHILD, "storage", "files", 1, "contents", "source"), path.New(TAG_RESULT, "storage", "files", 1, "contents", "source")},
				{path.New(TAG_CHILD, "storage", "files", 1, "contents"), path.New(TAG_RESULT, "storage", "files", 1, "contents")},
				{path.New(TAG_PARENT, "storage", "files", 1, "mode"), path.New(TAG_RESULT, "storage", "files", 1, "mode")},
				{path.New(TAG_PARENT, "storage", "files", 1), path.New(TAG_RESULT, "storage", "files", 1)},
				{path.New(TAG_CHILD, "storage", "files", 1), path.New(TAG_RESULT, "storage", "files", 1)},
				{path.New(TAG_CHILD, "storage", "files", 2, "path"), path.New(TAG_RESULT, "storage", "files", 2, "path")},
				{path.New(TAG_CHILD, "storage", "files", 2, "contents", "compression"), path.New(TAG_RESULT, "storage", "files", 2, "contents", "compression")},
				{path.New(TAG_PARENT, "storage", "files", 2, "contents", "source"), path.New(TAG_RESULT, "storage", "files", 2, "contents", "source")},
				{path.New(TAG_PARENT, "storage", "files", 2, "contents"), path.New(TAG_RESULT, "storage", "files", 2, "contents")},
				{path.New(TAG_CHILD, "storage", "files", 2, "contents"), path.New(TAG_RESULT, "storage", "files", 2, "contents")},
				{path.New(TAG_PARENT, "storage", "files", 2), path.New(TAG_RESULT, "storage", "files", 2)},
				{path.New(TAG_CHILD, "storage", "files", 2), path.New(TAG_RESULT, "storage", "files", 2)},
				{path.New(TAG_PARENT, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_CHILD, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_CHILD, "storage", "filesystems", 0, "device"), path.New(TAG_RESULT, "storage", "filesystems", 0, "device")},
				{path.New(TAG_CHILD, "storage", "filesystems", 0, "mountOptions", 0), path.New(TAG_RESULT, "storage", "filesystems", 0, "mountOptions", 0)},
				{path.New(TAG_CHILD, "storage", "filesystems", 0, "mountOptions", 1), path.New(TAG_RESULT, "storage", "filesystems", 0, "mountOptions", 1)},
				{path.New(TAG_CHILD, "storage", "filesystems", 0, "mountOptions"), path.New(TAG_RESULT, "storage", "filesystems", 0, "mountOptions")},
				{path.New(TAG_PARENT, "storage", "filesystems", 0, "options", 0), path.New(TAG_RESULT, "storage", "filesystems", 0, "options", 0)},
				{path.New(TAG_PARENT, "storage", "filesystems", 0, "options", 1), path.New(TAG_RESULT, "storage", "filesystems", 0, "options", 1)},
				{path.New(TAG_PARENT, "storage", "filesystems", 0, "options"), path.New(TAG_RESULT, "storage", "filesystems", 0, "options")},
				{path.New(TAG_PARENT, "storage", "filesystems", 0), path.New(TAG_RESULT, "storage", "filesystems", 0)},
				{path.New(TAG_CHILD, "storage", "filesystems", 0), path.New(TAG_RESULT, "storage", "filesystems", 0)},
				{path.New(TAG_CHILD, "storage", "filesystems", 1, "device"), path.New(TAG_RESULT, "storage", "filesystems", 1, "device")},
				{path.New(TAG_PARENT, "storage", "filesystems", 1, "options", 0), path.New(TAG_RESULT, "storage", "filesystems", 1, "options", 0)},
				{path.New(TAG_PARENT, "storage", "filesystems", 1, "options", 1), path.New(TAG_RESULT, "storage", "filesystems", 1, "options", 1)},
				{path.New(TAG_CHILD, "storage", "filesystems", 1, "options", 0), path.New(TAG_RESULT, "storage", "filesystems", 1, "options", 2)},
				{path.New(TAG_CHILD, "storage", "filesystems", 1, "options", 1), path.New(TAG_RESULT, "storage", "filesystems", 1, "options", 3)},
				{path.New(TAG_PARENT, "storage", "filesystems", 1, "options"), path.New(TAG_RESULT, "storage", "filesystems", 1, "options")},
				{path.New(TAG_CHILD, "storage", "filesystems", 1, "options"), path.New(TAG_RESULT, "storage", "filesystems", 1, "options")},
				{path.New(TAG_PARENT, "storage", "filesystems", 1), path.New(TAG_RESULT, "storage", "filesystems", 1)},
				{path.New(TAG_CHILD, "storage", "filesystems", 1), path.New(TAG_RESULT, "storage", "filesystems", 1)},
				{path.New(TAG_PARENT, "storage", "filesystems"), path.New(TAG_RESULT, "storage", "filesystems")},
				{path.New(TAG_CHILD, "storage", "filesystems"), path.New(TAG_RESULT, "storage", "filesystems")},
				{path.New(TAG_CHILD, "storage", "links", 0, "path"), path.New(TAG_RESULT, "storage", "links", 0, "path")},
				{path.New(TAG_CHILD, "storage", "links", 0), path.New(TAG_RESULT, "storage", "links", 0)},
				{path.New(TAG_CHILD, "storage", "links"), path.New(TAG_RESULT, "storage", "links")},
				{path.New(TAG_PARENT, "storage"), path.New(TAG_RESULT, "storage")},
				{path.New(TAG_CHILD, "storage"), path.New(TAG_RESULT, "storage")},
			}},
		},

		// completely parent subtree
		{
			in1: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/a",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
									HTTPHeaders: []types.HTTPHeader{
										{
											Name:  "header",
											Value: util.StrToPtr("value"),
										},
									},
								},
								Append: []types.Resource{
									{
										Source: util.StrToPtr("data:"),
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "header",
												Value: util.StrToPtr("value"),
											},
										},
									},
								},
							},
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/sda",
							Options: []types.FilesystemOption{
								"z",
							},
						},
					},
				},
			},
			in2: types.Config{},
			out: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/a",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
									HTTPHeaders: []types.HTTPHeader{
										{
											Name:  "header",
											Value: util.StrToPtr("value"),
										},
									},
								},
								Append: []types.Resource{
									{
										Source: util.StrToPtr("data:"),
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "header",
												Value: util.StrToPtr("value"),
											},
										},
									},
								},
							},
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/sda",
							Options: []types.FilesystemOption{
								"z",
							},
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "storage", "files", 0, "path"), path.New(TAG_RESULT, "storage", "files", 0, "path")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0)},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "source")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0)},
				{path.New(TAG_PARENT, "storage", "files", 0, "append"), path.New(TAG_RESULT, "storage", "files", 0, "append")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0)},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "source"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "source")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents"), path.New(TAG_RESULT, "storage", "files", 0, "contents")},
				{path.New(TAG_PARENT, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_PARENT, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_PARENT, "storage", "filesystems", 0, "device"), path.New(TAG_RESULT, "storage", "filesystems", 0, "device")},
				{path.New(TAG_PARENT, "storage", "filesystems", 0, "options", 0), path.New(TAG_RESULT, "storage", "filesystems", 0, "options", 0)},
				{path.New(TAG_PARENT, "storage", "filesystems", 0, "options"), path.New(TAG_RESULT, "storage", "filesystems", 0, "options")},
				{path.New(TAG_PARENT, "storage", "filesystems", 0), path.New(TAG_RESULT, "storage", "filesystems", 0)},
				{path.New(TAG_PARENT, "storage", "filesystems"), path.New(TAG_RESULT, "storage", "filesystems")},
				{path.New(TAG_PARENT, "storage"), path.New(TAG_RESULT, "storage")},
			}},
		},

		// completely child subtree
		{
			in1: types.Config{},
			in2: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/a",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
									HTTPHeaders: []types.HTTPHeader{
										{
											Name:  "header",
											Value: util.StrToPtr("value"),
										},
									},
								},
								Append: []types.Resource{
									{
										Source: util.StrToPtr("data:"),
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "header",
												Value: util.StrToPtr("value"),
											},
										},
									},
								},
							},
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/sda",
							Options: []types.FilesystemOption{
								"z",
							},
						},
					},
				},
			},
			out: types.Config{
				Storage: types.Storage{
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/a",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source: util.StrToPtr("data:"),
									HTTPHeaders: []types.HTTPHeader{
										{
											Name:  "header",
											Value: util.StrToPtr("value"),
										},
									},
								},
								Append: []types.Resource{
									{
										Source: util.StrToPtr("data:"),
										HTTPHeaders: []types.HTTPHeader{
											{
												Name:  "header",
												Value: util.StrToPtr("value"),
											},
										},
									},
								},
							},
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/sda",
							Options: []types.FilesystemOption{
								"z",
							},
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_CHILD, "storage", "files", 0, "path"), path.New(TAG_RESULT, "storage", "files", 0, "path")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "source")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append"), path.New(TAG_RESULT, "storage", "files", 0, "append")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "source"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "source")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents"), path.New(TAG_RESULT, "storage", "files", 0, "contents")},
				{path.New(TAG_CHILD, "storage", "files", 0), path.New(TAG_RESULT, "storage", "files", 0)},
				{path.New(TAG_CHILD, "storage", "files"), path.New(TAG_RESULT, "storage", "files")},
				{path.New(TAG_CHILD, "storage", "filesystems", 0, "device"), path.New(TAG_RESULT, "storage", "filesystems", 0, "device")},
				{path.New(TAG_CHILD, "storage", "filesystems", 0, "options", 0), path.New(TAG_RESULT, "storage", "filesystems", 0, "options", 0)},
				{path.New(TAG_CHILD, "storage", "filesystems", 0, "options"), path.New(TAG_RESULT, "storage", "filesystems", 0, "options")},
				{path.New(TAG_CHILD, "storage", "filesystems", 0), path.New(TAG_RESULT, "storage", "filesystems", 0)},
				{path.New(TAG_CHILD, "storage", "filesystems"), path.New(TAG_RESULT, "storage", "filesystems")},
				{path.New(TAG_CHILD, "storage"), path.New(TAG_RESULT, "storage")},
			}},
		},

		// kernel arguments MergedKeys test where child ShouldNotExist overrides parent ShouldExist
		{
			in1: types.Config{
				KernelArguments: types.KernelArguments{
					ShouldExist: []types.KernelArgument{
						"foo",
						"bar baz",
						"test",
					},
					ShouldNotExist: []types.KernelArgument{
						"brown fox",
					},
				},
			},
			in2: types.Config{
				KernelArguments: types.KernelArguments{
					ShouldNotExist: []types.KernelArgument{
						"test",
					},
				},
			},
			out: types.Config{
				KernelArguments: types.KernelArguments{
					ShouldExist: []types.KernelArgument{
						"foo",
						"bar baz",
					},
					ShouldNotExist: []types.KernelArgument{
						"brown fox",
						"test",
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "kernelArguments", "shouldExist", 0), path.New(TAG_RESULT, "kernelArguments", "shouldExist", 0)},
				{path.New(TAG_PARENT, "kernelArguments", "shouldExist", 1), path.New(TAG_RESULT, "kernelArguments", "shouldExist", 1)},
				{path.New(TAG_PARENT, "kernelArguments", "shouldExist"), path.New(TAG_RESULT, "kernelArguments", "shouldExist")},
				{path.New(TAG_PARENT, "kernelArguments", "shouldNotExist", 0), path.New(TAG_RESULT, "kernelArguments", "shouldNotExist", 0)},
				{path.New(TAG_CHILD, "kernelArguments", "shouldNotExist", 0), path.New(TAG_RESULT, "kernelArguments", "shouldNotExist", 1)},
				{path.New(TAG_PARENT, "kernelArguments", "shouldNotExist"), path.New(TAG_RESULT, "kernelArguments", "shouldNotExist")},
				{path.New(TAG_CHILD, "kernelArguments", "shouldNotExist"), path.New(TAG_RESULT, "kernelArguments", "shouldNotExist")},
				{path.New(TAG_PARENT, "kernelArguments"), path.New(TAG_RESULT, "kernelArguments")},
				{path.New(TAG_CHILD, "kernelArguments"), path.New(TAG_RESULT, "kernelArguments")},
			}},
		},
	}

	for i, test := range tests {
		outi, transcript := MergeStructTranscribe(test.in1, test.in2)
		out := outi.(types.Config)

		assert.Equal(t, test.out, out, "#%d bad merge", i)
		assert.Equal(t, test.transcript, transcript, "#%d bad transcript", i)
	}
}

// We are explicitly testing 3.2.0 because it mistakenly has struct
// pointers. These should not exist but ended up in the Clevis & Luks
// structs in spec 3.2.0.
// https://github.com/coreos/ignition/issues/1132
func TestMergeStructPointers(t *testing.T) {
	type test struct {
		in1        v3_2.Config
		in2        v3_2.Config
		out        v3_2.Config
		transcript Transcript
	}

	tests := []test{
		{
			in1: v3_2.Config{
				Storage: v3_2.Storage{
					Luks: []v3_2.Luks{
						// nested struct pointers, one override
						{
							Clevis: &v3_2.Clevis{
								Custom: &v3_2.Custom{
									Config: "cfg",
									Pin:    "pin",
								},
								Threshold: util.IntToPtr(1),
							},
							Device: util.StrToPtr("/dev/foo"),
							Name:   "bar",
						},
					},
				},
			},
			in2: v3_2.Config{
				Storage: v3_2.Storage{
					Luks: []v3_2.Luks{
						// nested struct pointers
						{
							Clevis: &v3_2.Clevis{
								Threshold: util.IntToPtr(2),
							},
							Name: "bar",
						},
						// struct pointer containing nil struct pointer
						{
							Clevis: &v3_2.Clevis{
								Tpm2: util.BoolToPtr(true),
							},
							Device: util.StrToPtr("/dev/baz"),
							Name:   "bleh",
						},
					},
				},
			},
			out: v3_2.Config{
				Storage: v3_2.Storage{
					Luks: []v3_2.Luks{
						{
							Clevis: &v3_2.Clevis{
								Custom: &v3_2.Custom{
									Config: "cfg",
									Pin:    "pin",
								},
								Threshold: util.IntToPtr(2),
							},
							Device: util.StrToPtr("/dev/foo"),
							Name:   "bar",
						},
						{
							Clevis: &v3_2.Clevis{
								Tpm2: util.BoolToPtr(true),
							},
							Device: util.StrToPtr("/dev/baz"),
							Name:   "bleh",
						},
					},
				},
			},
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "storage", "luks", 0, "clevis", "custom", "config"), path.New(TAG_RESULT, "storage", "luks", 0, "clevis", "custom", "config")},
				{path.New(TAG_PARENT, "storage", "luks", 0, "clevis", "custom", "pin"), path.New(TAG_RESULT, "storage", "luks", 0, "clevis", "custom", "pin")},
				{path.New(TAG_PARENT, "storage", "luks", 0, "clevis", "custom"), path.New(TAG_RESULT, "storage", "luks", 0, "clevis", "custom")},
				{path.New(TAG_CHILD, "storage", "luks", 0, "clevis", "threshold"), path.New(TAG_RESULT, "storage", "luks", 0, "clevis", "threshold")},
				{path.New(TAG_PARENT, "storage", "luks", 0, "clevis"), path.New(TAG_RESULT, "storage", "luks", 0, "clevis")},
				{path.New(TAG_CHILD, "storage", "luks", 0, "clevis"), path.New(TAG_RESULT, "storage", "luks", 0, "clevis")},
				{path.New(TAG_PARENT, "storage", "luks", 0, "device"), path.New(TAG_RESULT, "storage", "luks", 0, "device")},
				{path.New(TAG_CHILD, "storage", "luks", 0, "name"), path.New(TAG_RESULT, "storage", "luks", 0, "name")},
				{path.New(TAG_PARENT, "storage", "luks", 0), path.New(TAG_RESULT, "storage", "luks", 0)},
				{path.New(TAG_CHILD, "storage", "luks", 0), path.New(TAG_RESULT, "storage", "luks", 0)},
				{path.New(TAG_CHILD, "storage", "luks", 1, "clevis", "tpm2"), path.New(TAG_RESULT, "storage", "luks", 1, "clevis", "tpm2")},
				{path.New(TAG_CHILD, "storage", "luks", 1, "clevis"), path.New(TAG_RESULT, "storage", "luks", 1, "clevis")},
				{path.New(TAG_CHILD, "storage", "luks", 1, "device"), path.New(TAG_RESULT, "storage", "luks", 1, "device")},
				{path.New(TAG_CHILD, "storage", "luks", 1, "name"), path.New(TAG_RESULT, "storage", "luks", 1, "name")},
				{path.New(TAG_CHILD, "storage", "luks", 1), path.New(TAG_RESULT, "storage", "luks", 1)},
				{path.New(TAG_PARENT, "storage", "luks"), path.New(TAG_RESULT, "storage", "luks")},
				{path.New(TAG_CHILD, "storage", "luks"), path.New(TAG_RESULT, "storage", "luks")},
				{path.New(TAG_PARENT, "storage"), path.New(TAG_RESULT, "storage")},
				{path.New(TAG_CHILD, "storage"), path.New(TAG_RESULT, "storage")},
			}},
		},
	}

	for i, test := range tests {
		outi, transcript := MergeStructTranscribe(test.in1, test.in2)
		out := outi.(v3_2.Config)

		assert.Equal(t, test.out, out, "#%d bad merge", i)
		assert.Equal(t, test.transcript, transcript, "#%d bad transcript", i)
	}
}
