// Copyright 2018 CoreOS, Inc.
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

package filesystems

import (
	"fmt"

	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, WipeFilesystemWithSameType())
	register.Register(register.PositiveTest, FilesystemCreationOnMultipleDisks())
}

func WipeFilesystemWithSameType() types.Test {
	name := "filesystem.create.wipe.sametype"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "EFI-SYSTEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
		"ignition": { "version": "$version" },
		"storage": {
			"filesystems": [{
				"device": "$DEVICE",
				"format": "ext4",
				"path": "/boot",
				"wipeFilesystem": true
			}]
		}
	}`
	configMinVersion := "3.0.0"

	in[0].Partitions.GetPartition("EFI-SYSTEM").FilesystemType = "ext4"
	out[0].Partitions.GetPartition("EFI-SYSTEM").FilesystemType = "ext4"
	out[0].Partitions.GetPartition("EFI-SYSTEM").Files = []types.File{}
	out[0].Partitions.AddRemovedNodes("EFI-SYSTEM", []types.Node{
		{
			Name:      "multiLine",
			Directory: "path/example",
		}, {
			Name:      "singleLine",
			Directory: "another/path/example",
		}, {
			Name:      "emptyFile",
			Directory: "empty",
		}, {
			Name:      "noPath",
			Directory: "",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func FilesystemCreationOnMultipleDisks() types.Test {
	name := "filesystem.create.multipledisks"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()

	mntDevices := []types.MntDevice{}

	for i := 0; i < 4; i++ {
		label := fmt.Sprintf("data-%d", i)
		in = append(in, types.Disk{
			Alignment: types.IgnitionAlignment,
			Partitions: types.Partitions{
				{
					Label:          label,
					Number:         1,
					Length:         65536,
					FilesystemType: "blank",
				},
			},
		})

		out = append(out, types.Disk{
			Alignment: types.IgnitionAlignment,
			Partitions: types.Partitions{
				{
					Label:          label,
					Number:         1,
					Length:         65536,
					FilesystemType: "xfs",
				},
			},
		})

		mntDevices = append(mntDevices, types.MntDevice{
			Label:        label,
			Substitution: fmt.Sprintf("$dev%d", i),
		})
	}

	config := `{
		"ignition": {"version": "$version"},
		"storage": {
			"filesystems": [
				{
					"path": "/tmp-0",
					"device": "$dev0",
					"format": "xfs",
					"label": "data-0"
				},
				{
					"path": "/tmp-1",
					"device": "$dev1",
					"format": "xfs",
					"label": "data-1"
				},
				{
					"path": "/tmp-2",
					"device": "$dev2",
					"format": "xfs",
					"label": "data-2"
				},
				{
					"path": "/tmp-3",
					"device": "$dev3",
					"format": "xfs",
					"label": "data-3"
				}
			]
		}
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
