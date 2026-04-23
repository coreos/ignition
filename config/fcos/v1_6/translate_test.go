// Copyright 2020 Red Hat, Inc
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

package v1_6

import (
	"fmt"
	"testing"

	baseutil "github.com/coreos/butane/base/util"
	base "github.com/coreos/butane/base/v0_6"
	"github.com/coreos/butane/config/common"
	confutil "github.com/coreos/butane/config/util"
	"github.com/coreos/butane/translate"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_5/types"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

// Most of this is covered by the Ignition translator generic tests, so just test the custom bits

// TestTranslateBootDevice tests translating the Butane config boot_device section.
func TestTranslateBootDevice(t *testing.T) {
	tests := []struct {
		in         Config
		out        types.Config
		exceptions []translate.Translation
		report     report.Report
	}{
		// empty config
		{
			Config{},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
			},
			report.Report{},
		},
		// partition number for the `root` label is incorrect
		{
			Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:   util.StrToPtr("root"),
										SizeMiB: util.IntToPtr(12000),
										Resize:  util.BoolToPtr(true),
									},
									{
										Label:   util.StrToPtr("var-home"),
										SizeMiB: util.IntToPtr(10240),
									},
								},
							},
						},
						Filesystems: []base.Filesystem{
							{
								Device:         "/dev/disk/by-partlabel/var-home",
								Format:         util.StrToPtr("xfs"),
								Path:           util.StrToPtr("/var/home"),
								Label:          util.StrToPtr("var-home"),
								WipeFilesystem: util.BoolToPtr(false),
							},
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device: "/dev/vda",
							Partitions: []types.Partition{
								{
									Label:   util.StrToPtr("root"),
									SizeMiB: util.IntToPtr(12000),
									Resize:  util.BoolToPtr(true),
								},
								{
									Label:   util.StrToPtr("var-home"),
									SizeMiB: util.IntToPtr(10240),
								},
							},
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device:         "/dev/disk/by-partlabel/var-home",
							Format:         util.StrToPtr("xfs"),
							Path:           util.StrToPtr("/var/home"),
							Label:          util.StrToPtr("var-home"),
							WipeFilesystem: util.BoolToPtr(false),
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 0, "label"), To: path.New("json", "storage", "disks", 0, "partitions", 0, "label")},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 0, "size_mib"), To: path.New("json", "storage", "disks", 0, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 0, "resize"), To: path.New("json", "storage", "disks", 0, "partitions", 0, "resize")},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 1, "label"), To: path.New("json", "storage", "disks", 0, "partitions", 1, "label")},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 1, "size_mib"), To: path.New("json", "storage", "disks", 0, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0)},
				{From: path.New("yaml", "storage", "disks", 0), To: path.New("json", "storage", "disks", 0)},
				{From: path.New("yaml", "storage", "filesystems", 0, "device"), To: path.New("json", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "storage", "filesystems", 0, "format"), To: path.New("json", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "storage", "filesystems", 0, "path"), To: path.New("json", "storage", "filesystems", 0, "path")},
				{From: path.New("yaml", "storage", "filesystems", 0, "label"), To: path.New("json", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "storage", "filesystems", 0, "wipe_filesystem"), To: path.New("json", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "storage", "filesystems", 0), To: path.New("json", "storage", "filesystems", 0)},
				{From: path.New("yaml", "storage", "filesystems"), To: path.New("json", "storage", "filesystems")},
				{From: path.New("yaml", "storage"), To: path.New("json", "storage")},
			},
			report.Report{
				Entries: []report.Entry{
					{
						Kind:    report.Warn,
						Message: common.ErrWrongPartitionNumber.Error(),
						Context: path.New("yaml", "storage", "disks", 0, "partitions", 0, "label"),
					},
				},
			},
		},
		// LUKS, x86_64
		{
			Config{
				BootDevice: BootDevice{
					Luks: BootDeviceLuks{
						Discard: util.BoolToPtr(true),
						Tang: []base.Tang{{
							URL:        "https://example.com/",
							Thumbprint: util.StrToPtr("z"),
						}},
						Threshold: util.IntToPtr(2),
						Tpm2:      util.BoolToPtr(true),
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Luks: []types.Luks{
						{
							Clevis: types.Clevis{
								Tang: []types.Tang{{
									URL:        "https://example.com/",
									Thumbprint: util.StrToPtr("z"),
								}},
								Threshold: util.IntToPtr(2),
								Tpm2:      util.BoolToPtr(true),
							},
							Device:     util.StrToPtr("/dev/disk/by-partlabel/root"),
							Discard:    util.BoolToPtr(true),
							Label:      util.StrToPtr("luks-root"),
							Name:       "root",
							WipeVolume: util.BoolToPtr(true),
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device:         "/dev/mapper/root",
							Format:         util.StrToPtr("xfs"),
							Label:          util.StrToPtr("root"),
							WipeFilesystem: util.BoolToPtr(true),
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "url"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "url")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "thumbprint"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "thumbprint")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0)},
				{From: path.New("yaml", "boot_device", "luks", "tang"), To: path.New("json", "storage", "luks", 0, "clevis", "tang")},
				{From: path.New("yaml", "boot_device", "luks", "threshold"), To: path.New("json", "storage", "luks", 0, "clevis", "threshold")},
				{From: path.New("yaml", "boot_device", "luks", "tpm2"), To: path.New("json", "storage", "luks", 0, "clevis", "tpm2")},
				{From: path.New("yaml", "boot_device", "luks", "discard"), To: path.New("json", "storage", "luks", 0, "discard")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "clevis")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "device")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "label")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "name")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "wipeVolume")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0)},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage")},
			},
			report.Report{},
		},
		// LUKS, x86_64, with Tang set for offline provisioning
		{
			Config{
				BootDevice: BootDevice{
					Luks: BootDeviceLuks{
						Tang: []base.Tang{{
							URL:           "https://example.com/",
							Thumbprint:    util.StrToPtr("z"),
							Advertisement: util.StrToPtr("{\"payload\": \"xyzzy\"}"),
						}},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Luks: []types.Luks{
						{
							Clevis: types.Clevis{
								Tang: []types.Tang{{
									URL:           "https://example.com/",
									Thumbprint:    util.StrToPtr("z"),
									Advertisement: util.StrToPtr("{\"payload\": \"xyzzy\"}"),
								}},
							},
							Device:     util.StrToPtr("/dev/disk/by-partlabel/root"),
							Label:      util.StrToPtr("luks-root"),
							Name:       "root",
							WipeVolume: util.BoolToPtr(true),
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device:         "/dev/mapper/root",
							Format:         util.StrToPtr("xfs"),
							Label:          util.StrToPtr("root"),
							WipeFilesystem: util.BoolToPtr(true),
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "url"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "url")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "thumbprint"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "thumbprint")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "advertisement"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "advertisement")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0)},
				{From: path.New("yaml", "boot_device", "luks", "tang"), To: path.New("json", "storage", "luks", 0, "clevis", "tang")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "clevis")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "device")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "label")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "name")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "wipeVolume")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0)},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 0)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage")},
			},
			report.Report{},
		},
		// 3-disk mirror, x86_64
		{
			Config{
				BootDevice: BootDevice{
					Mirror: BootDeviceMirror{
						Devices: []string{"/dev/vda", "/dev/vdb", "/dev/vdc"},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device: "/dev/vda",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("bios-1"),
									SizeMiB:  util.IntToPtr(biosV1SizeMiB),
									TypeGUID: util.StrToPtr(biosTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-1"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-1"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-1"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
						{
							Device: "/dev/vdb",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("bios-2"),
									SizeMiB:  util.IntToPtr(biosV1SizeMiB),
									TypeGUID: util.StrToPtr(biosTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-2"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-2"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-2"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
						{
							Device: "/dev/vdc",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("bios-3"),
									SizeMiB:  util.IntToPtr(biosV1SizeMiB),
									TypeGUID: util.StrToPtr(biosTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-3"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-3"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-3"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
					},
					Raid: []types.Raid{
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/boot-1",
								"/dev/disk/by-partlabel/boot-2",
								"/dev/disk/by-partlabel/boot-3",
							},
							Level:   util.StrToPtr("raid1"),
							Name:    "md-boot",
							Options: []types.RaidOption{"--metadata=1.0"},
						},
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/root-1",
								"/dev/disk/by-partlabel/root-2",
								"/dev/disk/by-partlabel/root-3",
							},
							Level: util.StrToPtr("raid1"),
							Name:  "md-root",
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device:         "/dev/disk/by-partlabel/esp-1",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-1"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/disk/by-partlabel/esp-2",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-2"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/disk/by-partlabel/esp-3",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-3"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/md/md-boot",
							Format:         util.StrToPtr("ext4"),
							Label:          util.StrToPtr("boot"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/md/md-root",
							Format:         util.StrToPtr("xfs"),
							Label:          util.StrToPtr("root"),
							WipeFilesystem: util.BoolToPtr(true),
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices"), To: path.New("json", "storage", "disks")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 2)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 2)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3, "device")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3, "format")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4, "device")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4, "format")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4, "label")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage")},
			},
			report.Report{},
		},
		// 3-disk mirror + LUKS, x86_64
		{
			Config{
				BootDevice: BootDevice{
					Luks: BootDeviceLuks{
						Discard: util.BoolToPtr(true),
						Tang: []base.Tang{{
							URL:        "https://example.com/",
							Thumbprint: util.StrToPtr("z"),
						}},
						Threshold: util.IntToPtr(2),
						Tpm2:      util.BoolToPtr(true),
					},
					Mirror: BootDeviceMirror{
						Devices: []string{"/dev/vda", "/dev/vdb", "/dev/vdc"},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device: "/dev/vda",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("bios-1"),
									SizeMiB:  util.IntToPtr(biosV1SizeMiB),
									TypeGUID: util.StrToPtr(biosTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-1"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-1"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-1"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
						{
							Device: "/dev/vdb",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("bios-2"),
									SizeMiB:  util.IntToPtr(biosV1SizeMiB),
									TypeGUID: util.StrToPtr(biosTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-2"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-2"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-2"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
						{
							Device: "/dev/vdc",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("bios-3"),
									SizeMiB:  util.IntToPtr(biosV1SizeMiB),
									TypeGUID: util.StrToPtr(biosTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-3"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-3"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-3"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
					},
					Raid: []types.Raid{
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/boot-1",
								"/dev/disk/by-partlabel/boot-2",
								"/dev/disk/by-partlabel/boot-3",
							},
							Level:   util.StrToPtr("raid1"),
							Name:    "md-boot",
							Options: []types.RaidOption{"--metadata=1.0"},
						},
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/root-1",
								"/dev/disk/by-partlabel/root-2",
								"/dev/disk/by-partlabel/root-3",
							},
							Level: util.StrToPtr("raid1"),
							Name:  "md-root",
						},
					},
					Luks: []types.Luks{
						{
							Clevis: types.Clevis{
								Tang: []types.Tang{{
									URL:        "https://example.com/",
									Thumbprint: util.StrToPtr("z"),
								}},
								Threshold: util.IntToPtr(2),
								Tpm2:      util.BoolToPtr(true),
							},
							Device:     util.StrToPtr("/dev/md/md-root"),
							Discard:    util.BoolToPtr(true),
							Label:      util.StrToPtr("luks-root"),
							Name:       "root",
							WipeVolume: util.BoolToPtr(true),
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device:         "/dev/disk/by-partlabel/esp-1",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-1"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/disk/by-partlabel/esp-2",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-2"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/disk/by-partlabel/esp-3",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-3"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/md/md-boot",
							Format:         util.StrToPtr("ext4"),
							Label:          util.StrToPtr("boot"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/mapper/root",
							Format:         util.StrToPtr("xfs"),
							Label:          util.StrToPtr("root"),
							WipeFilesystem: util.BoolToPtr(true),
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "disks", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 2), To: path.New("json", "storage", "filesystems", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices"), To: path.New("json", "storage", "disks")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 2)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 2)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "url"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "url")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "thumbprint"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "thumbprint")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0)},
				{From: path.New("yaml", "boot_device", "luks", "tang"), To: path.New("json", "storage", "luks", 0, "clevis", "tang")},
				{From: path.New("yaml", "boot_device", "luks", "threshold"), To: path.New("json", "storage", "luks", 0, "clevis", "threshold")},
				{From: path.New("yaml", "boot_device", "luks", "tpm2"), To: path.New("json", "storage", "luks", 0, "clevis", "tpm2")},
				{From: path.New("yaml", "boot_device", "luks", "discard"), To: path.New("json", "storage", "luks", 0, "discard")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "clevis")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "device")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "label")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "name")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "wipeVolume")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0)},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3, "device")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3, "format")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 3)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4, "device")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4, "format")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4, "label")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 4)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage")},
			},
			report.Report{},
		},
		// 2-disk mirror + LUKS, aarch64
		{
			Config{
				BootDevice: BootDevice{
					Layout: util.StrToPtr("aarch64"),
					Luks: BootDeviceLuks{
						Discard: util.BoolToPtr(true),
						Tang: []base.Tang{{
							URL:        "https://example.com/",
							Thumbprint: util.StrToPtr("z"),
						}},
						Threshold: util.IntToPtr(2),
						Tpm2:      util.BoolToPtr(true),
					},
					Mirror: BootDeviceMirror{
						Devices: []string{"/dev/vda", "/dev/vdb"},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device: "/dev/vda",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("reserved-1"),
									SizeMiB:  util.IntToPtr(reservedV1SizeMiB),
									TypeGUID: util.StrToPtr(reservedTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-1"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-1"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-1"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
						{
							Device: "/dev/vdb",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("reserved-2"),
									SizeMiB:  util.IntToPtr(reservedV1SizeMiB),
									TypeGUID: util.StrToPtr(reservedTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-2"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-2"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-2"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
					},
					Raid: []types.Raid{
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/boot-1",
								"/dev/disk/by-partlabel/boot-2",
							},
							Level:   util.StrToPtr("raid1"),
							Name:    "md-boot",
							Options: []types.RaidOption{"--metadata=1.0"},
						},
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/root-1",
								"/dev/disk/by-partlabel/root-2",
							},
							Level: util.StrToPtr("raid1"),
							Name:  "md-root",
						},
					},
					Luks: []types.Luks{
						{
							Clevis: types.Clevis{
								Tang: []types.Tang{{
									URL:        "https://example.com/",
									Thumbprint: util.StrToPtr("z"),
								}},
								Threshold: util.IntToPtr(2),
								Tpm2:      util.BoolToPtr(true),
							},
							Device:     util.StrToPtr("/dev/md/md-root"),
							Discard:    util.BoolToPtr(true),
							Label:      util.StrToPtr("luks-root"),
							Name:       "root",
							WipeVolume: util.BoolToPtr(true),
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device:         "/dev/disk/by-partlabel/esp-1",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-1"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/disk/by-partlabel/esp-2",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-2"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/md/md-boot",
							Format:         util.StrToPtr("ext4"),
							Label:          util.StrToPtr("boot"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/mapper/root",
							Format:         util.StrToPtr("xfs"),
							Label:          util.StrToPtr("root"),
							WipeFilesystem: util.BoolToPtr(true),
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices"), To: path.New("json", "storage", "disks")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "url"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "url")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "thumbprint"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "thumbprint")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0)},
				{From: path.New("yaml", "boot_device", "luks", "tang"), To: path.New("json", "storage", "luks", 0, "clevis", "tang")},
				{From: path.New("yaml", "boot_device", "luks", "threshold"), To: path.New("json", "storage", "luks", 0, "clevis", "threshold")},
				{From: path.New("yaml", "boot_device", "luks", "tpm2"), To: path.New("json", "storage", "luks", 0, "clevis", "tpm2")},
				{From: path.New("yaml", "boot_device", "luks", "discard"), To: path.New("json", "storage", "luks", 0, "discard")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "clevis")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "device")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "label")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "name")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "wipeVolume")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0)},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2, "device")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2, "format")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 3, "device")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 3, "format")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 3, "label")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 3, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 3)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage")},
			},
			report.Report{},
		},
		// 2-disk mirror + LUKS, ppc64le
		{
			Config{
				BootDevice: BootDevice{
					Layout: util.StrToPtr("ppc64le"),
					Luks: BootDeviceLuks{
						Discard: util.BoolToPtr(true),
						Tang: []base.Tang{{
							URL:        "https://example.com/",
							Thumbprint: util.StrToPtr("z"),
						}},
						Threshold: util.IntToPtr(2),
						Tpm2:      util.BoolToPtr(true),
					},
					Mirror: BootDeviceMirror{
						Devices: []string{"/dev/vda", "/dev/vdb"},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device: "/dev/vda",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("prep-1"),
									SizeMiB:  util.IntToPtr(prepV1SizeMiB),
									TypeGUID: util.StrToPtr(prepTypeGuid),
								},
								{
									Label:    util.StrToPtr("reserved-1"),
									SizeMiB:  util.IntToPtr(reservedV1SizeMiB),
									TypeGUID: util.StrToPtr(reservedTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-1"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-1"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
						{
							Device: "/dev/vdb",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("prep-2"),
									SizeMiB:  util.IntToPtr(prepV1SizeMiB),
									TypeGUID: util.StrToPtr(prepTypeGuid),
								},
								{
									Label:    util.StrToPtr("reserved-2"),
									SizeMiB:  util.IntToPtr(reservedV1SizeMiB),
									TypeGUID: util.StrToPtr(reservedTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-2"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label: util.StrToPtr("root-2"),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
					},
					Raid: []types.Raid{
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/boot-1",
								"/dev/disk/by-partlabel/boot-2",
							},
							Level:   util.StrToPtr("raid1"),
							Name:    "md-boot",
							Options: []types.RaidOption{"--metadata=1.0"},
						},
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/root-1",
								"/dev/disk/by-partlabel/root-2",
							},
							Level: util.StrToPtr("raid1"),
							Name:  "md-root",
						},
					},
					Luks: []types.Luks{
						{
							Clevis: types.Clevis{
								Tang: []types.Tang{{
									URL:        "https://example.com/",
									Thumbprint: util.StrToPtr("z"),
								}},
								Threshold: util.IntToPtr(2),
								Tpm2:      util.BoolToPtr(true),
							},
							Device:     util.StrToPtr("/dev/md/md-root"),
							Discard:    util.BoolToPtr(true),
							Label:      util.StrToPtr("luks-root"),
							Name:       "root",
							WipeVolume: util.BoolToPtr(true),
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device:         "/dev/md/md-boot",
							Format:         util.StrToPtr("ext4"),
							Label:          util.StrToPtr("boot"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/mapper/root",
							Format:         util.StrToPtr("xfs"),
							Label:          util.StrToPtr("root"),
							WipeFilesystem: util.BoolToPtr(true),
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 3, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "wipeTable")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices"), To: path.New("json", "storage", "disks")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "url"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "url")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "thumbprint"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "thumbprint")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0)},
				{From: path.New("yaml", "boot_device", "luks", "tang"), To: path.New("json", "storage", "luks", 0, "clevis", "tang")},
				{From: path.New("yaml", "boot_device", "luks", "threshold"), To: path.New("json", "storage", "luks", 0, "clevis", "threshold")},
				{From: path.New("yaml", "boot_device", "luks", "tpm2"), To: path.New("json", "storage", "luks", 0, "clevis", "tpm2")},
				{From: path.New("yaml", "boot_device", "luks", "discard"), To: path.New("json", "storage", "luks", 0, "discard")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "clevis")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "device")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "label")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "name")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "wipeVolume")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0)},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 0)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 1, "device")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 1, "format")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 1, "label")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 1, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 1)},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage")},
			},
			report.Report{},
		},
		// 2-disk mirror + LUKS with overridden root partition size
		// and filesystem type, x86_64
		{
			Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:   util.StrToPtr("root-1"),
										SizeMiB: util.IntToPtr(8192),
									},
								},
							},
							{
								Device: "/dev/vdb",
								Partitions: []base.Partition{
									{
										Label:   util.StrToPtr("root-2"),
										SizeMiB: util.IntToPtr(8192),
									},
								},
							},
						},
						Filesystems: []base.Filesystem{
							{
								Device: "/dev/mapper/root",
								Format: util.StrToPtr("ext4"),
							},
						},
					},
				},
				BootDevice: BootDevice{
					Luks: BootDeviceLuks{
						Discard: util.BoolToPtr(true),
						Tang: []base.Tang{{
							URL:        "https://example.com/",
							Thumbprint: util.StrToPtr("z"),
						}},
						Threshold: util.IntToPtr(2),
						Tpm2:      util.BoolToPtr(true),
					},
					Mirror: BootDeviceMirror{
						Devices: []string{"/dev/vda", "/dev/vdb"},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Disks: []types.Disk{
						{
							Device: "/dev/vda",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("bios-1"),
									SizeMiB:  util.IntToPtr(biosV1SizeMiB),
									TypeGUID: util.StrToPtr(biosTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-1"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-1"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label:   util.StrToPtr("root-1"),
									SizeMiB: util.IntToPtr(8192),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
						{
							Device: "/dev/vdb",
							Partitions: []types.Partition{
								{
									Label:    util.StrToPtr("bios-2"),
									SizeMiB:  util.IntToPtr(biosV1SizeMiB),
									TypeGUID: util.StrToPtr(biosTypeGuid),
								},
								{
									Label:    util.StrToPtr("esp-2"),
									SizeMiB:  util.IntToPtr(espV1SizeMiB),
									TypeGUID: util.StrToPtr(espTypeGuid),
								},
								{
									Label:   util.StrToPtr("boot-2"),
									SizeMiB: util.IntToPtr(bootV1SizeMiB),
								},
								{
									Label:   util.StrToPtr("root-2"),
									SizeMiB: util.IntToPtr(8192),
								},
							},
							WipeTable: util.BoolToPtr(true),
						},
					},
					Raid: []types.Raid{
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/boot-1",
								"/dev/disk/by-partlabel/boot-2",
							},
							Level:   util.StrToPtr("raid1"),
							Name:    "md-boot",
							Options: []types.RaidOption{"--metadata=1.0"},
						},
						{
							Devices: []types.Device{
								"/dev/disk/by-partlabel/root-1",
								"/dev/disk/by-partlabel/root-2",
							},
							Level: util.StrToPtr("raid1"),
							Name:  "md-root",
						},
					},
					Luks: []types.Luks{
						{
							Clevis: types.Clevis{
								Tang: []types.Tang{{
									URL:        "https://example.com/",
									Thumbprint: util.StrToPtr("z"),
								}},
								Threshold: util.IntToPtr(2),
								Tpm2:      util.BoolToPtr(true),
							},
							Device:     util.StrToPtr("/dev/md/md-root"),
							Discard:    util.BoolToPtr(true),
							Label:      util.StrToPtr("luks-root"),
							Name:       "root",
							WipeVolume: util.BoolToPtr(true),
						},
					},
					Filesystems: []types.Filesystem{
						{
							Device:         "/dev/disk/by-partlabel/esp-1",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-1"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/disk/by-partlabel/esp-2",
							Format:         util.StrToPtr("vfat"),
							Label:          util.StrToPtr("esp-2"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/md/md-boot",
							Format:         util.StrToPtr("ext4"),
							Label:          util.StrToPtr("boot"),
							WipeFilesystem: util.BoolToPtr(true),
						}, {
							Device:         "/dev/mapper/root",
							Format:         util.StrToPtr("ext4"),
							Label:          util.StrToPtr("root"),
							WipeFilesystem: util.BoolToPtr(true),
						},
					},
				},
			},
			[]translate.Translation{
				{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "partitions", 2)},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 0, "label"), To: path.New("json", "storage", "disks", 0, "partitions", 3, "label")},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 0, "size_mib"), To: path.New("json", "storage", "disks", 0, "partitions", 3, "sizeMiB")},
				{From: path.New("yaml", "storage", "disks", 0, "partitions", 0), To: path.New("json", "storage", "disks", 0, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "disks", 0, "wipeTable")},
				{From: path.New("yaml", "storage", "disks", 0), To: path.New("json", "storage", "disks", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 0), To: path.New("json", "storage", "filesystems", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 0)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1, "typeGuid")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2, "sizeMiB")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "partitions", 2)},
				{From: path.New("yaml", "storage", "disks", 1, "partitions", 0, "label"), To: path.New("json", "storage", "disks", 1, "partitions", 3, "label")},
				{From: path.New("yaml", "storage", "disks", 1, "partitions", 0, "size_mib"), To: path.New("json", "storage", "disks", 1, "partitions", 3, "sizeMiB")},
				{From: path.New("yaml", "storage", "disks", 1, "partitions", 0), To: path.New("json", "storage", "disks", 1, "partitions", 3)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "disks", 1, "wipeTable")},
				{From: path.New("yaml", "storage", "disks", 1), To: path.New("json", "storage", "disks", 1)},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "device")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "format")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "label")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror", "devices", 1), To: path.New("json", "storage", "filesystems", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0, "options")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 0)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "devices")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "level")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1, "name")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid", 1)},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "raid")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "url"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "url")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0, "thumbprint"), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0, "thumbprint")},
				{From: path.New("yaml", "boot_device", "luks", "tang", 0), To: path.New("json", "storage", "luks", 0, "clevis", "tang", 0)},
				{From: path.New("yaml", "boot_device", "luks", "tang"), To: path.New("json", "storage", "luks", 0, "clevis", "tang")},
				{From: path.New("yaml", "boot_device", "luks", "threshold"), To: path.New("json", "storage", "luks", 0, "clevis", "threshold")},
				{From: path.New("yaml", "boot_device", "luks", "tpm2"), To: path.New("json", "storage", "luks", 0, "clevis", "tpm2")},
				{From: path.New("yaml", "boot_device", "luks", "discard"), To: path.New("json", "storage", "luks", 0, "discard")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "clevis")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "device")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "label")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "name")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0, "wipeVolume")},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks", 0)},
				{From: path.New("yaml", "boot_device", "luks"), To: path.New("json", "storage", "luks")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2, "device")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2, "format")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2, "label")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2, "wipeFilesystem")},
				{From: path.New("yaml", "boot_device", "mirror"), To: path.New("json", "storage", "filesystems", 2)},
				{From: path.New("yaml", "storage", "filesystems", 0, "device"), To: path.New("json", "storage", "filesystems", 3, "device")},
				{From: path.New("yaml", "storage", "filesystems", 0, "format"), To: path.New("json", "storage", "filesystems", 3, "format")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 3, "label")},
				{From: path.New("yaml", "boot_device"), To: path.New("json", "storage", "filesystems", 3, "wipeFilesystem")},
				{From: path.New("yaml", "storage", "filesystems", 0), To: path.New("json", "storage", "filesystems", 3)},
				{From: path.New("yaml", "storage", "filesystems"), To: path.New("json", "storage", "filesystems")},
				{From: path.New("yaml", "storage"), To: path.New("json", "storage")},
			},
			report.Report{},
		},
	}

	// The partition sizes of existing layouts must never change, but
	// we use the constants in tests for clarity.  Ensure no one has
	// changed them.
	assert.Equal(t, reservedV1SizeMiB, 1)
	assert.Equal(t, biosV1SizeMiB, 1)
	assert.Equal(t, prepV1SizeMiB, 4)
	assert.Equal(t, espV1SizeMiB, 127)
	assert.Equal(t, bootV1SizeMiB, 384)

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := test.in.ToIgn3_5Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, test.report, r, "report mismatch")
			baseutil.VerifyTranslations(t, translations, test.exceptions)
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

func TestRootPartitionConstraints(t *testing.T) {
	tests := []struct {
		name   string
		in     Config
		report report.Report
	}{
		{
			name: "root constrained by auto-positioned partition",
			in: Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:   util.StrToPtr("root"),
										Number:  4,
										SizeMiB: util.IntToPtr(0), // fill available
									},
									{
										Label:    util.StrToPtr("data"),
										StartMiB: util.IntToPtr(0), // auto-positioned - will be placed after root
									},
								},
							},
						},
					},
				},
			},
			report: report.Report{
				Entries: []report.Entry{
					{
						Kind:    report.Warn,
						Message: common.ErrRootConstrained.Error(),
						Context: path.New("yaml", "storage", "disks", 0, "partitions", 0, "label"),
					},
				},
			},
		},
		{
			name: "root constrained by auto-positioned partition with explicit root start",
			in: Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:    util.StrToPtr("root"),
										Number:   4,
										SizeMiB:  util.IntToPtr(0), // fill available
										StartMiB: util.IntToPtr(2048),
									},
									{
										Label:    util.StrToPtr("var"),
										StartMiB: util.IntToPtr(0), // auto-positioned - constrains root
									},
								},
							},
						},
					},
				},
			},
			report: report.Report{
				Entries: []report.Entry{
					{
						Kind:    report.Warn,
						Message: common.ErrRootConstrained.Error(),
						Context: path.New("yaml", "storage", "disks", 0, "partitions", 0, "label"),
					},
				},
			},
		},
		// Root partition NOT constrained because next partition has explicit StartMiB
		{
			name: "root not constrained with explicit StartMiB after",
			in: Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:    util.StrToPtr("root"),
										Number:   4,
										SizeMiB:  util.IntToPtr(0), // fill available
										StartMiB: util.IntToPtr(2048),
									},
									{
										Label:    util.StrToPtr("data"),
										StartMiB: util.IntToPtr(10240), // explicit position - does NOT constrain root
									},
								},
							},
						},
					},
				},
			},
			report: report.Report{},
		},
		// Root partition constrained by auto-positioned partition even when
		// an explicit partition is also present
		{
			name: "root constrained by auto-positioned partition with explicit also present",
			in: Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:    util.StrToPtr("root"),
										Number:   4,
										SizeMiB:  util.IntToPtr(0), // fill available
										StartMiB: util.IntToPtr(2048),
									},
									{
										Label:    util.StrToPtr("var"),
										StartMiB: util.IntToPtr(0), // auto-positioned - constrains root
									},
									{
										Label:    util.StrToPtr("data"),
										StartMiB: util.IntToPtr(10240), // explicit position
									},
								},
							},
						},
					},
				},
			},
			report: report.Report{
				Entries: []report.Entry{
					{
						Kind:    report.Warn,
						Message: common.ErrRootConstrained.Error(),
						Context: path.New("yaml", "storage", "disks", 0, "partitions", 0, "label"),
					},
				},
			},
		},
		{
			name: "root partition too small",
			in: Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:   util.StrToPtr("root"),
										Number:  4,
										SizeMiB: util.IntToPtr(4096),
									},
								},
							},
						},
					},
				},
			},
			report: report.Report{
				Entries: []report.Entry{
					{
						Kind:    report.Warn,
						Message: common.ErrRootTooSmall.Error(),
						Context: path.New("json", "storage", "disks", 0, "partitions", 0, "size_mib"),
					},
				},
			},
		},
		{
			name: "root partition exactly 8GiB",
			in: Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:   util.StrToPtr("root"),
										Number:  4,
										SizeMiB: util.IntToPtr(8192),
									},
								},
							},
						},
					},
				},
			},
			report: report.Report{},
		},
		{
			name: "root constrained with nil sizeMiB and nil startMiB",
			in: Config{
				Config: base.Config{
					Storage: base.Storage{
						Disks: []base.Disk{
							{
								Device: "/dev/vda",
								Partitions: []base.Partition{
									{
										Label:  util.StrToPtr("root"),
										Number: 4,
									},
									{
										Label: util.StrToPtr("data"),
									},
								},
							},
						},
					},
				},
			},
			report: report.Report{
				Entries: []report.Entry{
					{
						Kind:    report.Warn,
						Message: common.ErrRootConstrained.Error(),
						Context: path.New("yaml", "storage", "disks", 0, "partitions", 0, "label"),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, translations, r := test.in.ToIgn3_5Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			assert.Equal(t, test.report, r, "report mismatch")
		})
	}
}

// TestTranslateGrub tests translating the Butane config Grub section.
func TestTranslateGrub(t *testing.T) {
	// Some tests below have the same translations
	translations := []translate.Translation{
		{From: path.New("yaml", "version"), To: path.New("json", "ignition", "version")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "filesystems")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "filesystems", 0)},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "filesystems", 0, "path")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "filesystems", 0, "device")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "filesystems", 0, "format")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "files")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "files", 0)},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "files", 0, "path")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "files", 0, "append")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "files", 0, "append", 0)},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "files", 0, "append", 0, "source")},
		{From: path.New("yaml", "grub", "users"), To: path.New("json", "storage", "files", 0, "append", 0, "compression")},
	}
	tests := []struct {
		in         Config
		out        types.Config
		exceptions []translate.Translation
		report     report.Report
	}{
		// config with 1 user
		{
			Config{
				Grub: Grub{
					Users: []GrubUser{
						{
							Name:         "root",
							PasswordHash: util.StrToPtr("grub.pbkdf2.sha512.10000.874A958E526409..."),
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/disk/by-label/boot",
							Format: util.StrToPtr("ext4"),
							Path:   util.StrToPtr("/boot"),
						},
					},
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/boot/grub2/user.cfg",
							},
							FileEmbedded1: types.FileEmbedded1{
								Append: []types.Resource{
									{
										Source:      util.StrToPtr("data:,%23%20Generated%20by%20Butane%0A%0Aset%20superusers%3D%22root%22%0Apassword_pbkdf2%20root%20grub.pbkdf2.sha512.10000.874A958E526409...%0A"),
										Compression: util.StrToPtr(""),
									},
								},
							},
						},
					},
				},
			},
			translations,
			report.Report{},
		},
		// config with 2 users (and 2 different hashes)
		{
			Config{
				Grub: Grub{
					Users: []GrubUser{
						{
							Name:         "root1",
							PasswordHash: util.StrToPtr("grub.pbkdf2.sha512.10000.874A958E526409..."),
						},
						{
							Name:         "root2",
							PasswordHash: util.StrToPtr("grub.pbkdf2.sha512.10000.874B829D126209..."),
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.5.0",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/disk/by-label/boot",
							Format: util.StrToPtr("ext4"),
							Path:   util.StrToPtr("/boot"),
						},
					},
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/boot/grub2/user.cfg",
							},
							FileEmbedded1: types.FileEmbedded1{
								Append: []types.Resource{
									{
										Source:      util.StrToPtr("data:;base64,H4sIAAAAAAAC/3zMsQrCMBDG8b1PcdT9SI62JoODRfExJCGngtCEuwTx7UWyiss3fH/47eDCG0uonCC+YW01bDwMyhW0FZamLHoYJedq4bs0DiWovrKka4nPdCPo8S4tYn9QH2G2hNYYY9Dtp6Of3XmmZTIeEX8C9BdYHfmTpYU68AkAAP//Mp8bt7YAAAA="),
										Compression: util.StrToPtr("gzip"),
									},
								},
							},
						},
					},
				},
			},
			translations,
			report.Report{},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := test.in.ToIgn3_5Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, test.report, r, "report mismatch")
			baseutil.VerifyTranslations(t, translations, test.exceptions)
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}
