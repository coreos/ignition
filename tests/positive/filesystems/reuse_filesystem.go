// Copyright 2017 CoreOS, Inc.
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
	"github.com/coreos/ignition/v2/tests/fixtures"
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, ReuseExistingFilesystem())
	register.Register(register.PositiveTest, ReuseAmbivalentFilesystem())
}

func ReuseExistingFilesystem() types.Test {
	name := "filesystem.reuse"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "important-data",
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
				"format": "xfs",
				"label": "data",
				"uuid": "$uuid0"
			}]
		}
	}`
	configMinVersion := "3.0.0"
	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:           "important-data",
				Number:          1,
				Length:          2621440,
				FilesystemType:  "xfs",
				FilesystemLabel: "data",
				FilesystemUUID:  "$uuid0",
				Files: []types.File{
					{
						Node: types.Node{
							Name:      "bar",
							Directory: "foo",
						},
						Contents: "example file\n",
					},
				},
			},
		},
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:           "important-data",
				Number:          1,
				Length:          2621440,
				FilesystemType:  "xfs",
				FilesystemLabel: "data",
				FilesystemUUID:  "$uuid0",
				Files: []types.File{
					{
						Node: types.Node{
							Name:      "bar",
							Directory: "foo",
						},
						Contents: "example file\n",
					},
				},
			},
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

// Successfully reuse a filesystem which libblkid thinks has multiple
// type signatures.
func ReuseAmbivalentFilesystem() types.Test {
	name := "filesystem.reuse.ambivalent"
	in := []types.Disk{
		{
			Alignment: types.DefaultAlignment,
			Partitions: types.Partitions{
				{
					Number:         1,
					Label:          "ROOT",
					TypeCode:       "data",
					Length:         131072,
					FilesystemType: "ext4",
				},
				{
					Number:          2,
					Label:           "ZFS",
					TypeCode:        "data",
					Length:          131072,
					FilesystemType:  "image",
					FilesystemImage: fixtures.Ext4ZFS,
				},
			},
		},
	}
	out := []types.Disk{
		{
			Alignment: types.DefaultAlignment,
			Partitions: types.Partitions{
				{
					Number:         1,
					Label:          "ROOT",
					TypeCode:       "data",
					Length:         131072,
					FilesystemType: "ext4",
				},
				{
					Number:          2,
					Label:           "ZFS",
					TypeCode:        "data",
					Length:          131072,
					FilesystemType:  "ext4",
					FilesystemUUID:  "f63bf118-f6d7-40a3-b64c-a92b05a7f9ee",
					FilesystemLabel: "some-label",
					Ambivalent:      true,
					Files: []types.File{
						{
							Node: types.Node{
								Name:      "bar",
								Directory: "foo",
							},
							Contents: "example file\n",
						},
					},
				},
			},
		},
	}
	mntDevices := []types.MntDevice{
		{
			Label:        "ZFS",
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
				"format": "ext4",
				"label": "some-label",
				"uuid": "f63bf118-f6d7-40a3-b64c-a92b05a7f9ee"
			}]
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
