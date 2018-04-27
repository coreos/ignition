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
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.PositiveTest, ForceNewFilesystemOfSameType())
	register.Register(register.PositiveTest, WipeFilesystemWithSameType())
}

func ForceNewFilesystemOfSameType() types.Test {
	name := "Force new Filesystem Creation of same type"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "EFI-SYSTEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
		"ignition": {"version": "2.0.0"},
		"storage": {
			"filesystems": [{
				"mount": {
					"device": "$DEVICE",
					"format": "ext4",
					"create": {
						"force": true
					}}
				 }]
			}
	}`

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
		Name:       name,
		In:         in,
		Out:        out,
		MntDevices: mntDevices,
		Config:     config,
	}
}

func WipeFilesystemWithSameType() types.Test {
	name := "Wipe Filesystem with Filesystem of same type"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "EFI-SYSTEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
		"ignition": { "version": "2.1.0" },
		"storage": {
			"filesystems": [{
				"mount": {
					"device": "$DEVICE",
					"format": "ext4",
					"wipeFilesystem": true
				}}]
			}
	}`

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
		Name:       name,
		In:         in,
		Out:        out,
		MntDevices: mntDevices,
		Config:     config,
	}
}
