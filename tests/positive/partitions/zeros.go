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
	// Tests that just create partitions and involve 0s
	register.Register(register.PositiveTest, PartitionSizeStart0())
	register.Register(register.PositiveTest, PartitionStartNumber0())
	register.Register(register.PositiveTest, ResizeRootFillDisk())
	register.Register(register.PositiveTest, VerifyRootFillsDisk())
	register.Register(register.PositiveTest, VerifyUnspecifiedIsDoNotCare())
	register.Register(register.PositiveTest, NumberZeroHappensLast())
}

func PartitionSizeStart0() types.Test {
	name := "partition.create.startsize0"
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
					"label": "fills-disk",
					"number": 1,
					"startMiB": 0,
					"sizeMiB": 0,
					"typeGuid": "$uuid0",
					"guid": "$uuid1"
				}]
			}]
		}
	}`
	configMinVersion := "3.0.0"

	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "fills-disk",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid1",
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

func PartitionStartNumber0() types.Test {
	name := "partition.create.numbersize0"
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
					"label": "uno",
					"sizeMiB": 32,
					"typeGuid": "$uuid0",
					"guid": "$uuid1"
				},
				{
					"label": "dos",
					"sizeMiB": 32,
					"typeGuid": "$uuid0",
					"guid": "$uuid2"
				},
				{
					"label": "tres",
					"sizeMiB": 32,
					"typeGuid": "$uuid0",
					"guid": "$uuid3"
				}]
			}]
		}
	}`
	configMinVersion := "3.0.0"

	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "uno",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid1",
			},
			{
				Label:    "dos",
				Number:   2,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid2",
			},
			{
				Label:    "tres",
				Number:   3,
				Length:   65536,
				TypeGUID: "$uuid0",
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

func ResizeRootFillDisk() types.Test {
	name := "partition.resizeroot.withzeros"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	out[0].Partitions[9-6-1].Length = 12943360 + 65536
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
					"startMiB": 0,
					"sizeMiB": 0,
					"typeGuid": "3884DD41-8582-4404-B9A8-E9B84F2DF50E",
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

func VerifyRootFillsDisk() types.Test {
	name := "partition.match.rootfillsdisk"
	in := types.GetBaseDisk()
	out := in
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
					"startMiB": 0,
					"sizeMiB": 0,
					"typeGuid": "3884DD41-8582-4404-B9A8-E9B84F2DF50E"
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

func VerifyUnspecifiedIsDoNotCare() types.Test {
	name := "partition.match.sizestart.unspecd"
	in := types.GetBaseDisk()
	in[0].Partitions = append(in[0].Partitions, &types.Partition{
		TypeCode: "blank",
		Length:   65536,
	})
	out := types.GetBaseDisk()
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
					"typeGuid": "3884DD41-8582-4404-B9A8-E9B84F2DF50E"
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

func NumberZeroHappensLast() types.Test {
	name := "partition.number0"
	in := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Number:   1,
				Label:    "foobar",
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid1",
			},
		},
	})
	out := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Number:   1,
				Label:    "foobar",
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid1",
			},
			{
				Number: 2,
				Label:  "newpart",
				Length: 65536,
			},
		},
	})
	config := `{
		"ignition": {
			"version": "$version"
		},
		"storage": {
			"disks": [{
				"device": "$disk1",
				"partitions": [
				{
					"label": "newpart",
					"sizeMiB": 32
				},
				{
					"number": 1,
					"label": "foobar"
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
