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

package v4_21

import (
	"fmt"
	"testing"

	baseutil "github.com/coreos/butane/base/util"
	base "github.com/coreos/butane/base/v0_6"
	"github.com/coreos/butane/config/common"
	fcos "github.com/coreos/butane/config/fcos/v1_6"
	"github.com/coreos/butane/config/openshift/v4_21/result"
	confutil "github.com/coreos/butane/config/util"
	"github.com/coreos/butane/translate"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_5/types"
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

	_, _, r := in.ToIgn3_5Unvalidated(common.TranslateOptions{})
	assert.Equal(t, expected, r, "report mismatch")
}

func TestTranslateConfig(t *testing.T) {
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
							Version: "3.5.0",
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
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := test.in.ToMachineConfig4_21Unvalidated(common.TranslateOptions{})
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
									PasswordHash:      util.StrToPtr("corned beef"),
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
								},
								{
									Path: "/h",
									Contents: base.Resource{
										Source: util.StrToPtr("https://example.com/"),
									},
									Mode: util.IntToPtr(04755),
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
								{
									Device: "/dev/vda5",
									Format: util.StrToPtr("none"),
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
									Target: util.StrToPtr("/t"),
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
									HomeDir:                util.StrToPtr("/home/drum"),
									NoCreateHome:           util.BoolToPtr(true),
									NoLogInit:              util.BoolToPtr(true),
									NoUserGroup:            util.BoolToPtr(true),
									PasswordHash:           util.StrToPtr("corned beef"),
									PrimaryGroup:           util.StrToPtr("wheel"),
									SSHAuthorizedKeys:      []base.SSHAuthorizedKey{"value"},
									SSHAuthorizedKeysLocal: []string{},
									Shell:                  util.StrToPtr("/bin/tcsh"),
									ShouldExist:            util.BoolToPtr(false),
									System:                 util.BoolToPtr(true),
									UID:                    util.IntToPtr(42),
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
						KernelArguments: base.KernelArguments{
							ShouldExist: []base.KernelArgument{
								"foo",
							},
							ShouldNotExist: []base.KernelArgument{
								"bar",
							},
						},
					},
				},
			},
			[]entry{
				// code
				{report.Error, common.ErrBtrfsSupport, path.New("yaml", "storage", "filesystems", 0, "format")},
				{report.Error, common.ErrFilesystemNoneSupport, path.New("yaml", "storage", "filesystems", 1, "format")},
				{report.Error, common.ErrFileSchemeSupport, path.New("yaml", "storage", "files", 2, "contents", "source")},
				{report.Error, common.ErrFileSpecialModeSupport, path.New("yaml", "storage", "files", 2, "mode")},
				{report.Error, common.ErrUserNameSupport, path.New("yaml", "passwd", "users", 1, "name")},
				// filters
				{report.Error, common.ErrKernelArgumentSupport, path.New("yaml", "kernel_arguments")},
				{report.Error, common.ErrGroupSupport, path.New("yaml", "passwd", "groups")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "gecos")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "groups")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "home_dir")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "no_create_home")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "no_log_init")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "no_user_group")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "primary_group")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "shell")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "should_exist")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "system")},
				{report.Error, common.ErrUserFieldSupport, path.New("yaml", "passwd", "users", 0, "uid")},
				{report.Error, common.ErrDirectorySupport, path.New("yaml", "storage", "directories")},
				{report.Error, common.ErrFileAppendSupport, path.New("yaml", "storage", "files", 1, "append")},
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
			actual, translations, r := test.in.ToMachineConfig4_21Unvalidated(common.TranslateOptions{})
			r.Merge(fieldFilters.Verify(actual))
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, expectedReport, r, "report mismatch")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}
