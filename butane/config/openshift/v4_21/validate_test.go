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

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		in      Metadata
		out     error
		errPath path.ContextPath
	}{
		// missing name
		{
			Metadata{
				Labels: map[string]string{
					ROLE_LABEL_KEY: "r",
				},
			},
			common.ErrNameRequired,
			path.New("yaml", "name"),
		},
		// missing role
		{
			Metadata{
				Name: "n",
			},
			common.ErrRoleRequired,
			path.New("yaml", "labels"),
		},
		// empty role
		{
			Metadata{
				Name: "n",
				Labels: map[string]string{
					ROLE_LABEL_KEY: "",
				},
			},
			common.ErrRoleRequired,
			path.New("yaml", "labels"),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnError(test.errPath, test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

func TestValidateOpenShift(t *testing.T) {
	tests := []struct {
		in      OpenShift
		out     error
		errPath path.ContextPath
	}{
		// empty struct
		{
			OpenShift{},
			nil,
			path.New("yaml"),
		},
		// bad kernel type
		{
			OpenShift{
				KernelType: util.StrToPtr("hurd"),
			},
			common.ErrInvalidKernelType,
			path.New("yaml", "kernel_type"),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnError(test.errPath, test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		in      Config
		out     error
		errPath path.ContextPath
	}{
		// missing kargs for CEX support
		{
			Config{
				Config: fcos.Config{
					BootDevice: fcos.BootDevice{
						Layout: util.StrToPtr("s390x-eckd"),
						Luks: fcos.BootDeviceLuks{
							Device: util.StrToPtr("/dev/dasda"),
							Cex: base.Cex{
								Enabled: util.BoolToPtr(true),
							},
						},
					},
				},
				OpenShift: OpenShift{
					// explicitly empty kernel argument list
					KernelArguments: []string{},
				},
			},
			common.ErrMissingKernelArgumentCex,
			path.New("yaml", "openshift", "kernel_arguments"),
		},
		{
			Config{
				Config: fcos.Config{
					BootDevice: fcos.BootDevice{
						Layout: util.StrToPtr("s390x-zfcp"),
						Luks: fcos.BootDeviceLuks{
							Device: util.StrToPtr("/dev/sda"),
							Cex: base.Cex{
								Enabled: util.BoolToPtr(true),
							},
						},
					},
				},
				OpenShift: OpenShift{
					// explicitly empty kernel argument list
					KernelArguments: []string{},
				},
			},
			common.ErrMissingKernelArgumentCex,
			path.New("yaml", "openshift", "kernel_arguments"),
		},
		{
			Config{
				Config: fcos.Config{
					BootDevice: fcos.BootDevice{
						Layout: util.StrToPtr("s390x-virt"),
						Luks: fcos.BootDeviceLuks{
							Cex: base.Cex{
								Enabled: util.BoolToPtr(true),
							},
						},
					},
				},
				OpenShift: OpenShift{
					// explicitly empty kernel argument list
					KernelArguments: []string{},
				},
			},
			common.ErrMissingKernelArgumentCex,
			path.New("yaml", "openshift", "kernel_arguments"),
		},
		{
			Config{
				Config: fcos.Config{
					Config: base.Config{
						Storage: base.Storage{
							Filesystems: []base.Filesystem{
								{
									Device:         "/dev/mapper/root",
									Format:         util.StrToPtr("xfs"),
									Label:          util.StrToPtr("root"),
									WipeFilesystem: util.BoolToPtr(true),
								},
								{
									Device:         "/dev/mapper/foo",
									Format:         util.StrToPtr("xfs"),
									Label:          util.StrToPtr("foo"),
									WipeFilesystem: util.BoolToPtr(true),
								},
							},
							Luks: []base.Luks{
								{
									Name:       "root",
									Label:      util.StrToPtr("luks-root"),
									Device:     util.StrToPtr("/dev/disk/by-label/root"),
									WipeVolume: util.BoolToPtr(true),
									Cex: base.Cex{
										Enabled: util.BoolToPtr(true),
									},
								},
								{
									Name:       "foo",
									Label:      util.StrToPtr("luks-foo"),
									Device:     util.StrToPtr("/dev/disk/by-label/foo"),
									WipeVolume: util.BoolToPtr(true),
									Cex: base.Cex{
										Enabled: util.BoolToPtr(true),
									},
								},
							},
						},
					},
				},
				OpenShift: OpenShift{
					// explicitly empty kernel argument list
					KernelArguments: []string{},
				},
			},
			common.ErrMissingKernelArgumentCex,
			path.New("yaml", "openshift", "kernel_arguments"),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnError(test.errPath, test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

// TestReportCorrelation tests that errors are correctly correlated to their source lines
func TestReportCorrelation(t *testing.T) {
	tests := []struct {
		in      string
		message string
		line    int64
	}{
		// Butane unused key check
		{
			`
                         metadata:
                           name: something
                           labels:
                             machineconfiguration.openshift.io/role: r
                         storage:
                           files:
                           - path: /z
                             q: z`,
			"unused key q",
			9,
		},
		// Butane YAML validation error
		{
			`
                         metadata:
                           name: something
                           labels:
                             machineconfiguration.openshift.io/role: r
                         storage:
                           files:
                           - path: /z
                             contents:
                               source: https://example.com
                               inline: z`,
			common.ErrTooManyResourceSources.Error(),
			10,
		},
		// Butane YAML validation warning
		{
			`
                         metadata:
                           name: something
                           labels:
                             machineconfiguration.openshift.io/role: r
                         storage:
                           files:
                           - path: /z
                             mode: 444`,
			common.ErrDecimalMode.Error(),
			9,
		},
		// Butane translation error
		{
			`
                         metadata:
                           name: something
                           labels:
                             machineconfiguration.openshift.io/role: r
                         storage:
                           files:
                           - path: /z
                             contents:
                               local: z`,
			common.ErrNoFilesDir.Error(),
			10,
		},
		// Ignition validation error, leaf node
		{
			`
                         metadata:
                           name: something
                           labels:
                             machineconfiguration.openshift.io/role: r
                         storage:
                           files:
                           - path: z`,
			errors.ErrPathRelative.Error(),
			8,
		},
		// Ignition validation error, partition
		{
			`
                         metadata:
                           name: something
                           labels:
                             machineconfiguration.openshift.io/role: r
                         storage:
                           disks:
                           - device: /dev/z
                             wipe_table: true
                             partitions:
                               - start_mib: 5`,
			errors.ErrNeedLabelOrNumber.Error(),
			11,
		},
		// Ignition validation error, partition list
		{
			`
                         metadata:
                           name: something
                           labels:
                             machineconfiguration.openshift.io/role: r
                         storage:
                           disks:
                           - device: /dev/z
                             wipe_table: true
                             partitions:
                               - number: 1
                                 should_exist: false
                               - label: z`,
			errors.ErrZeroesWithShouldNotExist.Error(),
			11,
		},
		// Ignition duplicate key check, paths
		{
			`
                         metadata:
                           name: something
                           labels:
                             machineconfiguration.openshift.io/role: r
                         storage:
                           files:
                           - path: /z
                           - path: /z`,
			errors.ErrDuplicate.Error(),
			9,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			for _, raw := range []bool{false, true} {
				_, r, _ := ToConfigBytes([]byte(test.in), common.TranslateBytesOptions{
					Raw: raw,
				})
				assert.Len(t, r.Entries, 1, "unexpected report length, raw %v", raw)
				assert.Equal(t, test.message, r.Entries[0].Message, "bad error, raw %v", raw)
				assert.NotNil(t, r.Entries[0].Marker.StartP, "marker start is nil, raw %v", raw)
				assert.Equal(t, test.line, r.Entries[0].Marker.StartP.Line, "incorrect error line, raw %v", raw)
			}
		})
	}
}
