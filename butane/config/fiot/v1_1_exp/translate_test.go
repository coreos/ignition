// Copyright 2022 Red Hat, Inc
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

package v1_1_exp

import (
	"fmt"
	"testing"

	baseutil "github.com/coreos/butane/base/util"
	base "github.com/coreos/butane/base/v0_8_exp"
	"github.com/coreos/butane/config/common"
	confutil "github.com/coreos/butane/config/util"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

// Test that we error on unsupported fields for fiot
func TestTranslateInvalid(t *testing.T) {
	type InvalidEntry struct {
		Kind report.EntryKind
		Err  error
		Path path.ContextPath
	}
	tests := []struct {
		In      Config
		Entries []InvalidEntry
	}{
		// we don't support setting kernel arguments
		{
			Config{
				Config: base.Config{
					KernelArguments: base.KernelArguments{
						ShouldExist: []base.KernelArgument{
							"test",
						},
					},
				},
			},
			[]InvalidEntry{
				{
					report.Error,
					common.ErrGeneralKernelArgumentSupport,
					path.New("yaml", "kernel_arguments"),
				},
			},
		},
		// we don't support unsetting kernel arguments either
		{
			Config{
				Config: base.Config{
					KernelArguments: base.KernelArguments{
						ShouldNotExist: []base.KernelArgument{
							"another-test",
						},
					},
				},
			},
			[]InvalidEntry{
				{
					report.Error,
					common.ErrGeneralKernelArgumentSupport,
					path.New("yaml", "kernel_arguments"),
				},
			},
		},
		// disk customizations are made in Image Builder, fiot doesn't support this via ignition
		{
			Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "some-device",
							},
						},
					},
				},
			},
			[]InvalidEntry{
				{
					report.Error,
					common.ErrDiskSupport,
					path.New("yaml", "storage", "disks"),
				},
			},
		},
		// filesystem customizations are made in Image Builder, fiot doesn't support this via ignition
		{
			Config{
				Config: base.Config{
					Storage: base.Storage{
						Filesystems: []base.Filesystem{
							{
								Device: "/dev/disk/by-label/TEST",
								Path:   util.StrToPtr("/var"),
							},
						},
					},
				},
			},
			[]InvalidEntry{
				{
					report.Error,
					common.ErrFilesystemSupport,
					path.New("yaml", "storage", "filesystems"),
				},
			},
		},
		// default luks configuration is made in Image Builder for fiot, we don't support this via ignition
		{
			Config{
				Config: base.Config{
					Storage: base.Storage{
						Luks: []base.Luks{
							{
								Label: util.StrToPtr("some-label"),
							},
						},
					},
				},
			},
			[]InvalidEntry{
				{
					report.Error,
					common.ErrLuksSupport,
					path.New("yaml", "storage", "luks"),
				},
			},
		},
		// we don't support configuring raid via ignition
		{
			Config{
				Config: base.Config{
					Storage: base.Storage{
						Raid: []base.Raid{
							{
								Name: "some-name",
							},
						},
					},
				},
			},
			[]InvalidEntry{
				{
					report.Error,
					common.ErrRaidSupport,
					path.New("yaml", "storage", "raid"),
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			var expectedReport report.Report
			for _, entry := range test.Entries {
				expectedReport.AddOnError(entry.Path, entry.Err)
			}
			actual, translations, r := test.In.ToIgn3_7Unvalidated(common.TranslateOptions{})
			r.Merge(fieldFilters.Verify(actual))
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.In, r)
			assert.Equal(t, expectedReport, r, "report mismatch")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}
