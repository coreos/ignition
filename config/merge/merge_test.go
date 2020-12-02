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
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"

	"github.com/coreos/vcontext/path"
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
				{path.New(TAG_CHILD, "storage", "directories", 0, "path"), path.New(TAG_RESULT, "storage", "directories", 0, "path")},
				{path.New(TAG_CHILD, "storage", "directories", 0), path.New(TAG_RESULT, "storage", "directories", 0)},
				{path.New(TAG_CHILD, "storage", "disks", 0, "device"), path.New(TAG_RESULT, "storage", "disks", 0, "device")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 0, "label"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 0, "label")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 0, "number"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 0, "number")},
				{path.New(TAG_PARENT, "storage", "disks", 0, "partitions", 0, "startMiB"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 0, "startMiB")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 1, "label"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 1, "label")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 1, "number"), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 1, "number")},
				{path.New(TAG_CHILD, "storage", "disks", 0, "partitions", 1), path.New(TAG_RESULT, "storage", "disks", 0, "partitions", 1)},
				{path.New(TAG_CHILD, "storage", "disks", 0, "wipeTable"), path.New(TAG_RESULT, "storage", "disks", 0, "wipeTable")},
				{path.New(TAG_CHILD, "storage", "disks", 1, "device"), path.New(TAG_RESULT, "storage", "disks", 1, "device")},
				{path.New(TAG_PARENT, "storage", "disks", 1, "wipeTable"), path.New(TAG_RESULT, "storage", "disks", 1, "wipeTable")},
				{path.New(TAG_CHILD, "storage", "disks", 2, "device"), path.New(TAG_RESULT, "storage", "disks", 2, "device")},
				{path.New(TAG_CHILD, "storage", "disks", 2, "wipeTable"), path.New(TAG_RESULT, "storage", "disks", 2, "wipeTable")},
				{path.New(TAG_CHILD, "storage", "disks", 2), path.New(TAG_RESULT, "storage", "disks", 2)},
				{path.New(TAG_CHILD, "storage", "files", 0, "path"), path.New(TAG_RESULT, "storage", "files", 0, "path")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "source")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "source")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 1)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 1, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 2, "source")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 1), path.New(TAG_RESULT, "storage", "files", 0, "append", 2)},
				{path.New(TAG_CHILD, "storage", "links", 0, "path"), path.New(TAG_RESULT, "storage", "links", 0, "path")},
				{path.New(TAG_CHILD, "storage", "links", 0), path.New(TAG_RESULT, "storage", "links", 0)},
			}},
		},

		// struct pointers
		// we're not supposed to have any, but some ended up in the
		// Clevis and Luks structs in spec 3.2.0
		// https://github.com/coreos/ignition/issues/1132
		{
			in1: types.Config{
				Storage: types.Storage{
					Luks: []types.Luks{
						// nested struct pointers, one override
						{
							Clevis: &types.Clevis{
								Custom: &types.Custom{
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
			in2: types.Config{
				Storage: types.Storage{
					Luks: []types.Luks{
						// nested struct pointers
						{
							Clevis: &types.Clevis{
								Threshold: util.IntToPtr(2),
							},
							Name: "bar",
						},
						// struct pointer containing nil struct pointer
						{
							Clevis: &types.Clevis{
								Tpm2: util.BoolToPtr(true),
							},
							Device: util.StrToPtr("/dev/baz"),
							Name:   "bleh",
						},
					},
				},
			},
			out: types.Config{
				Storage: types.Storage{
					Luks: []types.Luks{
						{
							Clevis: &types.Clevis{
								Custom: &types.Custom{
									Config: "cfg",
									Pin:    "pin",
								},
								Threshold: util.IntToPtr(2),
							},
							Device: util.StrToPtr("/dev/foo"),
							Name:   "bar",
						},
						{
							Clevis: &types.Clevis{
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
				{path.New(TAG_PARENT, "storage", "luks", 0, "device"), path.New(TAG_RESULT, "storage", "luks", 0, "device")},
				{path.New(TAG_CHILD, "storage", "luks", 0, "name"), path.New(TAG_RESULT, "storage", "luks", 0, "name")},
				{path.New(TAG_CHILD, "storage", "luks", 1, "clevis", "tpm2"), path.New(TAG_RESULT, "storage", "luks", 1, "clevis", "tpm2")},
				{path.New(TAG_CHILD, "storage", "luks", 1, "clevis"), path.New(TAG_RESULT, "storage", "luks", 1, "clevis")},
				{path.New(TAG_CHILD, "storage", "luks", 1, "device"), path.New(TAG_RESULT, "storage", "luks", 1, "device")},
				{path.New(TAG_CHILD, "storage", "luks", 1, "name"), path.New(TAG_RESULT, "storage", "luks", 1, "name")},
				{path.New(TAG_CHILD, "storage", "luks", 1), path.New(TAG_RESULT, "storage", "luks", 1)},
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
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "ignition", "config", "merge", 0, "httpHeaders", 0), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 0)},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 2, "name"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 2, "value"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 1, "value")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 1, "name"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 2, "name")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 1, "value"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 2, "value")},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "httpHeaders", 2)},
				{path.New(TAG_CHILD, "ignition", "config", "merge", 0, "source"), path.New(TAG_RESULT, "ignition", "config", "merge", 0, "source")},
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
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "ignition", "config", "replace", "httpHeaders", 0, "name"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "ignition", "config", "replace", "httpHeaders", 0, "value"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "ignition", "config", "replace", "httpHeaders", 0), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 0)},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 2, "name"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 2, "value"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 1, "value")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 1, "name"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 2, "name")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 1, "value"), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 2, "value")},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "config", "replace", "httpHeaders", 2)},
				{path.New(TAG_CHILD, "ignition", "config", "replace", "source"), path.New(TAG_RESULT, "ignition", "config", "replace", "source")},
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
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 0)},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2, "name"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2, "value"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1, "value")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1, "name"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2, "name")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1, "value"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2, "value")},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 1), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "httpHeaders", 2)},
				{path.New(TAG_CHILD, "ignition", "security", "tls", "certificateAuthorities", 0, "source"), path.New(TAG_RESULT, "ignition", "security", "tls", "certificateAuthorities", 0, "source")},
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
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "contents", "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 2, "name"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 2, "value"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 1, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 1, "name"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 2, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 1, "value"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 2, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "httpHeaders", 1), path.New(TAG_RESULT, "storage", "files", 0, "contents", "httpHeaders", 2)},
				{path.New(TAG_CHILD, "storage", "files", 0, "contents", "source"), path.New(TAG_RESULT, "storage", "files", 0, "contents", "source")},
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
			transcript: Transcript{[]Mapping{
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 0)},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 1, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 1, "name")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 1, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 1, "value")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "httpHeaders", 1), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "httpHeaders", 1)},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 0, "source")},
				{path.New(TAG_PARENT, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 0, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 0, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 0)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 1, "name"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 1, "name")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 1, "value"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 1, "value")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "httpHeaders", 1), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "httpHeaders", 1)},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0, "source"), path.New(TAG_RESULT, "storage", "files", 0, "append", 1, "source")},
				{path.New(TAG_CHILD, "storage", "files", 0, "append", 0), path.New(TAG_RESULT, "storage", "files", 0, "append", 1)},
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
