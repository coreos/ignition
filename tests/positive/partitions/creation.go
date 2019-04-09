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

package partitions

import (
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	// Tests that just create partitions with no 0s
	register.Register(register.PositiveTest, CreatePartition())
	register.Register(register.PositiveTest, WipeAndCreateNewPartitions())
	register.Register(register.PositiveTest, AppendPartitions())
	register.Register(register.PositiveTest, ResizeRoot())
}

func CreatePartition() types.Test {
	name := "Create a single partition on a blank disk"
	in := append(types.GetBaseDisk(), types.Disk{Alignment: types.IgnitionAlignment})
	out := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "create-partition",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid1",
			},
		},
	})
	config := `{
		"ignition": {
			"version": "$version"
		},
		"storage": {
			"disks": [
			{
				"device": "$disk1",
				"partitions": [
				{
					"number": 1,
					"sizeMiB": 32,
					"label": "create-partition",
					"typeGuid": "$uuid0",
					"guid": "$uuid1"
				}
				]
			}
			]
		}
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func WipeAndCreateNewPartitions() types.Test {
	name := "Wipe disk and create new partitions"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "$version"
		},
		"storage": {
			"disks": [
			{
				"device": "$disk1",
				"wipeTable": true,
				"partitions": [
				{
					"label": "important-data",
					"number": 1,
					"sizeMiB": 32,
					"typeGuid": "$uuid0",
					"guid": "$uuid1"
				},
				{
					"label": "ephemeral-data",
					"number": 2,
					"sizeMiB": 64,
					"typeGuid": "$uuid2",
					"guid": "$uuid3"
				}
				]
			}
			]
		}
	}`
	configMinVersion := "3.0.0"
	// Create dummy partitions. The UUIDs in the input partitions
	// are intentionally different so if Ignition doesn't do the right thing the
	// validation will fail.
	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid0",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "$uuid2",
				GUID:     "$uuid0",
			},
		},
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid1",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "$uuid2",
				GUID:     "$uuid3",
			},
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func AppendPartitions() types.Test {
	name := "Append partition to an existing partition table"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "$version"
		},
		"storage": {
			"disks": [{
				"device": "$disk1",
				"wipeTable": false,
				"partitions": [{
					"label": "additional-partition",
					"number": 3,
					"sizeMiB": 32,
					"typeGuid": "$uuid0",
					"guid": "$uuid1"
				},
				{
					"label": "additional-partition2",
					"number": 4,
					"sizeMiB": 32,
					"typeGuid": "$uuid0",
					"guid": "$uuid2"
				}]
			}]
		}
	}`
	configMinVersion := "3.0.0"

	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid3",
				GUID:     "$uuid4",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "$uuid5",
				GUID:     "$uuid3",
			},
		},
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid3",
				GUID:     "$uuid4",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "$uuid5",
				GUID:     "$uuid3",
			},
			{
				Label:    "additional-partition",
				Number:   3,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid1",
			},
			{
				Label:    "additional-partition2",
				Number:   4,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid2",
			},
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ResizeRoot() types.Test {
	name := "Resize the ROOT partition to be bigger"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	out[0].Partitions[9-2-1].Length = 12943360 + 65536
	config := `{
		"ignition": {
			"version": "$version"
		},
		"storage": {
			"disks": [{
				"device": "$disk0",
				"partitions": [{
					"label": "ROOT",
					"number": 9,
					"sizeMiB": 6352,
					"typeGuid": "3884DD41-8582-4404-B9A8-E9B84F2DF50E",
					"guid": "$uuid0",
					"wipePartitionEntry": true
				}
				]
			}]
		}
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
