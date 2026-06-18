// Copyright 2021 Red Hat, Inc
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
// limitations under the License.)

package v4_8

import (
	"fmt"
	"testing"

	baseutil "github.com/coreos/butane/base/util"
	base "github.com/coreos/butane/base/v0_3"
	"github.com/coreos/butane/config/common"
	fcos "github.com/coreos/butane/config/fcos/v1_3"
	"github.com/coreos/butane/config/openshift/v4_8/result"
	confutil "github.com/coreos/butane/config/util"
	"github.com/coreos/butane/translate"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

// TestElidedFieldWarning tests that we warn when transpiling fields to an
// Ignition config that can't be represented in an Ignition config.
func TestElidedFieldWarning(t *testing.T) {
	in := Config{
		Metadata: Metadata{
			Name: "z",
		},
		OpenShift: OpenShift{
			KernelArguments: []string{"a", "b"},
			FIPS:            util.BoolToPtr(true),
			KernelType:      util.StrToPtr("realtime"),
		},
	}

	var expected report.Report
	expected.AddOnWarn(path.New("yaml", "openshift", "kernel_arguments"), common.ErrFieldElided)
	expected.AddOnWarn(path.New("yaml", "openshift", "fips"), common.ErrFieldElided)
	expected.AddOnWarn(path.New("yaml", "openshift", "kernel_type"), common.ErrFieldElided)

	_, _, r := in.ToIgn3_2Unvalidated(common.TranslateOptions{})
	assert.Equal(t, expected, r, "report mismatch")
}

func TestTranslateConfig(t *testing.T) {
	zzz := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	tests := []struct {
		in         Config
		out        result.MachineConfig
		exceptions []translate.Translation
	}{
		// empty-ish config
		{
			Config{
				Metadata: Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
			},
			result.MachineConfig{
				ApiVersion: result.MC_API_VERSION,
				Kind:       result.MC_KIND,
				Metadata: result.Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				Spec: result.Spec{
					Config: types.Config{
						Ignition: types.Ignition{
							Version: "3.2.0",
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "apiVersion")},
				{From: path.New("yaml", "version"), To: path.New("json", "kind")},
				{From: path.New("yaml", "version"), To: path.New("json", "spec")},
				{From: path.New("yaml"), To: path.New("json", "spec", "config")},
				{From: path.New("yaml", "ignition"), To: path.New("json", "spec", "config", "ignition")},
				{From: path.New("yaml", "version"), To: path.New("json", "spec", "config", "ignition", "version")},
			},
		},
		// ensure automatic compression is disabled
		{
			Config{
				Metadata: Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				Config: fcos.Config{
					Config: base.Config{
						Storage: base.Storage{
							Files: []base.File{
								{
									Path: "/z",
									Contents: base.Resource{
										Inline: util.StrToPtr(zzz),
									},
								},
							},
						},
					},
				},
			},
			result.MachineConfig{
				ApiVersion: result.MC_API_VERSION,
				Kind:       result.MC_KIND,
				Metadata: result.Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				Spec: result.Spec{
					Config: types.Config{
						Ignition: types.Ignition{
							Version: "3.2.0",
						},
						Storage: types.Storage{
							Files: []types.File{
								{
									Node: types.Node{
										Path: "/z",
									},
									FileEmbedded1: types.FileEmbedded1{
										Contents: types.Resource{
											Source:      util.StrToPtr("data:," + zzz),
											Compression: util.StrToPtr(""),
										},
									},
								},
							},
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "apiVersion")},
				{From: path.New("yaml", "version"), To: path.New("json", "kind")},
				{From: path.New("yaml", "version"), To: path.New("json", "spec")},
				{From: path.New("yaml"), To: path.New("json", "spec", "config")},
				{From: path.New("yaml", "ignition"), To: path.New("json", "spec", "config", "ignition")},
				{From: path.New("yaml", "version"), To: path.New("json", "spec", "config", "ignition", "version")},
				{From: path.New("yaml", "storage"), To: path.New("json", "spec", "config", "storage")},
				{From: path.New("yaml", "storage", "files"), To: path.New("json", "spec", "config", "storage", "files")},
				{From: path.New("yaml", "storage", "files", 0), To: path.New("json", "spec", "config", "storage", "files", 0)},
				{From: path.New("yaml", "storage", "files", 0, "path"), To: path.New("json", "spec", "config", "storage", "files", 0, "path")},
				{From: path.New("yaml", "storage", "files", 0, "contents"), To: path.New("json", "spec", "config", "storage", "files", 0, "contents")},
				{From: path.New("yaml", "storage", "files", 0, "contents", "inline"), To: path.New("json", "spec", "config", "storage", "files", 0, "contents", "source")},
				{From: path.New("yaml", "storage", "files", 0, "contents", "inline"), To: path.New("json", "spec", "config", "storage", "files", 0, "contents", "compression")},
			},
		},
		// FIPS
		{
			Config{
				Metadata: Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				OpenShift: OpenShift{
					FIPS: util.BoolToPtr(true),
				},
				Config: fcos.Config{
					Config: base.Config{
						Storage: base.Storage{
							Luks: []base.Luks{
								{
									Name: "a",
								},
								{
									Name:    "b",
									Options: []base.LuksOption{"b", "b"},
								},
								{
									Name:    "c",
									Options: []base.LuksOption{"c", "--cipher", "c"},
								},
								{
									Name:    "d",
									Options: []base.LuksOption{"--cipher=z"},
								},
								{
									Name:    "e",
									Options: []base.LuksOption{"-c", "z"},
								},
								{
									Name:    "f",
									Options: []base.LuksOption{"--ciphertext"},
								},
							},
						},
					},
					BootDevice: fcos.BootDevice{
						Luks: fcos.BootDeviceLuks{
							Tpm2: util.BoolToPtr(true),
						},
					},
				},
			},
			result.MachineConfig{
				ApiVersion: result.MC_API_VERSION,
				Kind:       result.MC_KIND,
				Metadata: result.Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				Spec: result.Spec{
					Config: types.Config{
						Ignition: types.Ignition{
							Version: "3.2.0",
						},
						Storage: types.Storage{
							Filesystems: []types.Filesystem{
								{
									Device:         "/dev/mapper/root",
									Format:         util.StrToPtr("xfs"),
									Label:          util.StrToPtr("root"),
									WipeFilesystem: util.BoolToPtr(true),
								},
							},
							Luks: []types.Luks{
								{
									Name:       "root",
									Device:     util.StrToPtr("/dev/disk/by-partlabel/root"),
									Label:      util.StrToPtr("luks-root"),
									WipeVolume: util.BoolToPtr(true),
									Options:    []types.LuksOption{fipsCipherOption, fipsCipherArgument},
									Clevis: &types.Clevis{
										Tpm2: util.BoolToPtr(true),
									},
								},
								{
									Name:    "a",
									Options: []types.LuksOption{fipsCipherOption, fipsCipherArgument},
								},
								{
									Name:    "b",
									Options: []types.LuksOption{"b", "b", fipsCipherOption, fipsCipherArgument},
								},
								{
									Name:    "c",
									Options: []types.LuksOption{"c", "--cipher", "c"},
								},
								{
									Name:    "d",
									Options: []types.LuksOption{"--cipher=z"},
								},
								{
									Name:    "e",
									Options: []types.LuksOption{"-c", "z"},
								},
								{
									Name:    "f",
									Options: []types.LuksOption{"--ciphertext", fipsCipherOption, fipsCipherArgument},
								},
							},
						},
					},
					FIPS: util.BoolToPtr(true),
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "apiVersion")},
				{From: path.New("yaml", "version"), To: path.New("json", "kind")},
				{From: path.New("yaml", "version"), To: path.New("json", "spec")},
				{From: path.New("yaml"), To: path.New("json", "spec", "config")},
				{From: path.New("yaml", "ignition"), To: path.New("json", "spec", "config", "ignition")},
				{From: path.New("yaml", "version"), To: path.New("json", "spec", "config", "ignition", "version")},
				{From: path.New("yaml", "boot_device", "luks", "tpm2"), To: path.New("json", "spec", "config", "storage", "luks", 0, "clevis", "tpm2")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "spec", "config", "storage", "luks", 0, "clevis")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "spec", "config", "storage", "luks", 0, "device")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "spec", "config", "storage", "luks", 0, "label")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "spec", "config", "storage", "luks", 0, "name")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "spec", "config", "storage", "luks", 0, "wipeVolume")},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 0, "options", 0)},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 0, "options", 1)},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 0, "options")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "spec", "config", "storage", "luks", 0)},
				{From: path.New("yaml", "storage", "luks", 0, "name"), To: path.New("json", "spec", "config", "storage", "luks", 1, "name")},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 1, "options", 0)},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 1, "options", 1)},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 1, "options")},
				{From: path.New("yaml", "storage", "luks", 0), To: path.New("json", "spec", "config", "storage", "luks", 1)},
				{From: path.New("yaml", "storage", "luks", 1, "name"), To: path.New("json", "spec", "config", "storage", "luks", 2, "name")},
				{From: path.New("yaml", "storage", "luks", 1, "options", 0), To: path.New("json", "spec", "config", "storage", "luks", 2, "options", 0)},
				{From: path.New("yaml", "storage", "luks", 1, "options", 1), To: path.New("json", "spec", "config", "storage", "luks", 2, "options", 1)},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 2, "options", 2)},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 2, "options", 3)},
				{From: path.New("yaml", "storage", "luks", 1, "options"), To: path.New("json", "spec", "config", "storage", "luks", 2, "options")},
				{From: path.New("yaml", "storage", "luks", 1), To: path.New("json", "spec", "config", "storage", "luks", 2)},
				{From: path.New("yaml", "storage", "luks", 2, "name"), To: path.New("json", "spec", "config", "storage", "luks", 3, "name")},
				{From: path.New("yaml", "storage", "luks", 2, "options", 0), To: path.New("json", "spec", "config", "storage", "luks", 3, "options", 0)},
				{From: path.New("yaml", "storage", "luks", 2, "options", 1), To: path.New("json", "spec", "config", "storage", "luks", 3, "options", 1)},
				{From: path.New("yaml", "storage", "luks", 2, "options", 2), To: path.New("json", "spec", "config", "storage", "luks", 3, "options", 2)},
				{From: path.New("yaml", "storage", "luks", 2, "options"), To: path.New("json", "spec", "config", "storage", "luks", 3, "options")},
				{From: path.New("yaml", "storage", "luks", 2), To: path.New("json", "spec", "config", "storage", "luks", 3)},
				{From: path.New("yaml", "storage", "luks", 3, "name"), To: path.New("json", "spec", "config", "storage", "luks", 4, "name")},
				{From: path.New("yaml", "storage", "luks", 3, "options", 0), To: path.New("json", "spec", "config", "storage", "luks", 4, "options", 0)},
				{From: path.New("yaml", "storage", "luks", 3, "options"), To: path.New("json", "spec", "config", "storage", "luks", 4, "options")},
				{From: path.New("yaml", "storage", "luks", 3), To: path.New("json", "spec", "config", "storage", "luks", 4)},
				{From: path.New("yaml", "storage", "luks", 4, "name"), To: path.New("json", "spec", "config", "storage", "luks", 5, "name")},
				{From: path.New("yaml", "storage", "luks", 4, "options", 0), To: path.New("json", "spec", "config", "storage", "luks", 5, "options", 0)},
				{From: path.New("yaml", "storage", "luks", 4, "options", 1), To: path.New("json", "spec", "config", "storage", "luks", 5, "options", 1)},
				{From: path.New("yaml", "storage", "luks", 4, "options"), To: path.New("json", "spec", "config", "storage", "luks", 5, "options")},
				{From: path.New("yaml", "storage", "luks", 4), To: path.New("json", "spec", "config", "storage", "luks", 5)},
				{From: path.New("yaml", "storage", "luks", 5, "name"), To: path.New("json", "spec", "config", "storage", "luks", 6, "name")},
				{From: path.New("yaml", "storage", "luks", 5, "options", 0), To: path.New("json", "spec", "config", "storage", "luks", 6, "options", 0)},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 6, "options", 1)},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "config", "storage", "luks", 6, "options", 2)},
				{From: path.New("yaml", "storage", "luks", 5, "options"), To: path.New("json", "spec", "config", "storage", "luks", 6, "options")},
				{From: path.New("yaml", "storage", "luks", 5), To: path.New("json", "spec", "config", "storage", "luks", 6)},
				{From: path.New("yaml", "storage", "luks"), To: path.New("json", "spec", "config", "storage", "luks")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "spec", "config", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "spec", "config", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "spec", "config", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "spec", "config", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "spec", "config", "storage", "filesystems", 0)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "spec", "config", "storage", "filesystems")},
				{From: path.New("yaml", "storage"), To: path.New("json", "spec", "config", "storage")},
				{From: path.New("yaml", "openshift", "fips"), To: path.New("json", "spec", "fips")},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := test.in.ToMachineConfig4_8Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, report.Report{}, r, "non-empty report")
			baseutil.VerifyTranslations(t, translations, test.exceptions)
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// Test post-translation validation of RHCOS/MCO support for Ignition config fields.
func TestValidateSupport(t *testing.T) {
	type entry struct {
		kind report.EntryKind
		err  error
		path path.ContextPath
	}
	tests := []struct {
		in      Config
		entries []entry
	}{
		// empty-ish config
		{
			Config{
				Metadata: Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
			},
			[]entry{},
		},
		// core user with only accepted fields
		{
			Config{
				Metadata: Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				Config: fcos.Config{
					Config: base.Config{
						Passwd: base.Passwd{
							Users: []base.PasswdUser{
								{
									Name:              "core",
									SSHAuthorizedKeys: []base.SSHAuthorizedKey{"value"},
								},
							},
						},
					},
				},
			},
			[]entry{},
		},
		// valid data URL
		{
			Config{
				Metadata: Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				Config: fcos.Config{
					Config: base.Config{
						Storage: base.Storage{
							Files: []base.File{
								{
									Path: "/f",
									Contents: base.Resource{
										Source: util.StrToPtr("data:,foo"),
									},
								},
							},
						},
					},
				},
			},
			[]entry{},
		},
		// config with uncompressed inline contents
		// (shouldn't reject Compression: StrToPtr(""))
		{
			Config{
				Metadata: Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				Config: fcos.Config{
					Config: base.Config{
						Storage: base.Storage{
							Files: []base.File{
								{
									Path: "/a",
									Contents: base.Resource{
										Inline: util.StrToPtr("foo"),
									},
								},
							},
						},
					},
				},
			},
			[]entry{},
		},
		// all the warnings/errors
		{
			Config{
				Metadata: Metadata{
					Name: "z",
					Labels: map[string]string{
						ROLE_LABEL_KEY: "z",
					},
				},
				Config: fcos.Config{
					Config: base.Config{
						Storage: base.Storage{
							Files: []base.File{
								{
									Path: "/f",
								},
								{
									Path: "/g",
									Append: []base.Resource{
										{
											Inline: util.StrToPtr("z"),
										},
									},
									Contents: base.Resource{
										Inline:      util.StrToPtr("z"),
										Compression: util.StrToPtr("gzip"),
									},
								},
								{
									Path: "/h",
									Contents: base.Resource{
										Source: util.StrToPtr("https://example.com/"),
									},
								},
								{
									Path: "/i",
									Contents: base.Resource{
										Source: util.StrToPtr("data:,z"),
										HTTPHeaders: base.HTTPHeaders{
											{
												Name:  "foo",
												Value: util.StrToPtr("bar"),
											},
										},
									},
								},
							},
							Filesystems: []base.Filesystem{
								{
									Device: "/dev/vda4",
									Format: util.StrToPtr("btrfs"),
								},
							},
							Directories: []base.Directory{
								{
									Path: "/d",
								},
							},
							Links: []base.Link{
								{
									Path:   "/l",
									Target: "/t",
								},
							},
						},
						Passwd: base.Passwd{
							Users: []base.PasswdUser{
								{
									Name:  "core",
									Gecos: util.StrToPtr("mercury delay line"),
									Groups: []base.Group{
										"z",
									},
									HomeDir:           util.StrToPtr("/home/drum"),
									NoCreateHome:      util.BoolToPtr(true),
									NoLogInit:         util.BoolToPtr(true),
									NoUserGroup:       util.BoolToPtr(true),
									PasswordHash:      util.StrToPtr("corned beef"),
									PrimaryGroup:      util.StrToPtr("wheel"),
									SSHAuthorizedKeys: []base.SSHAuthorizedKey{"value"},
									Shell:             util.StrToPtr("/bin/tcsh"),
									ShouldExist:       util.BoolToPtr(false),
									System:            util.BoolToPtr(true),
									UID:               util.IntToPtr(42),
								},
								{
									Name: "bovik",
								},
							},
							Groups: []base.PasswdGroup{
								{
									Name: "mock",
								},
							},
						},
					},
				},
			},
			[]entry{
				// code
				{report.Error, common.ErrBtrfsSupport, path.New("yaml", "storage", "filesystems", 0, "format")},
				{report.Error, common.ErrFileSchemeSupport, path.New("yaml", "storage", "files", 2, "contents", "source")},
				{report.Error, common.ErrUserNameSupport, path.New("yaml", "passwd", "users", 1, "name")},
				// filters
				{report.Error, common.ErrGroupSupport, path.New("yaml", "passwd", "groups")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "gecos")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "groups")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "home_dir")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "no_create_home")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "no_log_init")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "no_user_group")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "password_hash")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "primary_group")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "shell")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "should_exist")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "system")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "uid")},
				{report.Error, common.ErrDirectorySupport, path.New("yaml", "storage", "directories")},
				{report.Error, common.ErrFileAppendSupport, path.New("yaml", "storage", "files", 1, "append")},
				{report.Error, common.ErrFileCompressionSupport, path.New("yaml", "storage", "files", 1, "contents", "compression")},
				{report.Error, common.ErrFileHeaderSupport, path.New("yaml", "storage", "files", 3, "contents", "http_headers")},
				{report.Error, common.ErrLinkSupport, path.New("yaml", "storage", "links")},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			var expectedReport report.Report
			for _, entry := range test.entries {
				expectedReport.AddOn(entry.path, entry.err, entry.kind)
			}
			actual, translations, r := test.in.ToMachineConfig4_8Unvalidated(common.TranslateOptions{})
			r.Merge(fieldFilters.Verify(actual))
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, expectedReport, r, "report mismatch")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}
