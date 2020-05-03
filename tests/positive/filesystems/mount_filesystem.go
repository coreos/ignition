// Copyright 2019 Red Hat
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
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, MountFilesystemWithOptions())
}

func MountFilesystemWithOptions() types.Test {
	name := "filesystem.mount.options"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
		"ignition": {"version": "$version"},
		"storage": {
			"filesystems": [
			{
				"path": "/tmp0",
				"device": "$DEVICE",
				"wipeFilesystem": false,
				"format": "btrfs",
				"mountOptions": [
					"noexec",
					"subvolid=5"
				]
			}]
		}
	}`
	in[0].Partitions.GetPartition("OEM").FilesystemType = "btrfs"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "btrfs"
	configMinVersion := "3.1.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
