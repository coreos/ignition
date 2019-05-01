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
	// Tests that mix verification, wipePartitionEntry, deletion, and creation, but
	// do not use any zero semantics. See complex.go if you want to enter that madness
	register.Register(register.PositiveTest, Match1Recreate1Delete1Create1())
	register.Register(register.PositiveTest, NothingMatches())
}

func Match1Recreate1Delete1Create1() types.Test {
	name := "partition.match.recreate.delete.add"
	in := append(types.GetBaseDisk(), types.Disk{
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
				Length:   65536,
				TypeGUID: "$uuid2",
				GUID:     "$uuid0",
			},
			{
				Label:    "bunch-of-junk",
				Number:   3,
				Length:   131072,
				TypeGUID: "$uuid2",
				GUID:     "$uuid0",
			},
		},
	})
	out := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:  "important-data",
				Number: 1,
				Length: 65536,
			},
			{
				Label:  "ephemeral-data",
				Number: 2,
				Length: 131072,
			},
			{
				Label:  "even-more-data",
				Number: 4,
				Length: 65536,
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
					"label": "important-data",
					"number": 1,
					"startMiB": 1,
					"sizeMiB": 32
				},
				{
					"label": "ephemeral-data",
					"number": 2,
					"startMiB": 33,
					"sizeMiB": 64,
					"wipePartitionEntry": true
				},
				{
					"number": 3,
					"shouldExist": false,
					"wipePartitionEntry": true
				},
				{
					"label": "even-more-data",
					"number": 4,
					"startMiB": 97,
					"sizeMiB": 32
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

func NothingMatches() types.Test {
	name := "partition.match.recreate"
	// partition 1 has the wrong type guid, 2 has the wrong guid and 3 has the wrong size and label
	// there's a test in complex.go that is similar, but 1 has the wrong size and thus everything
	// gets moved around (with start/size 0)
	in := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid0",
			},
			{
				Label:  "ephemeral-data",
				Number: 2,
				Length: 65536,
				GUID:   "$uuid0",
			},
			{
				Label:  "bunch-of-junk",
				Number: 3,
				Length: 131072,
			},
		},
	})
	out := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid1",
			},
			{
				Label:  "ephemeral-data",
				Number: 2,
				Length: 65536,
				GUID:   "$uuid1",
			},
			{
				Label:  "even-more-data",
				Number: 3,
				Length: 65536,
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
					"label": "important-data",
					"number": 1,
					"startMiB": 1,
					"sizeMiB": 32,
					"wipePartitionEntry": true,
					"typeGuid": "$uuid1"
				},
				{
					"label": "ephemeral-data",
					"number": 2,
					"startMiB": 33,
					"sizeMiB": 32,
					"wipePartitionEntry": true,
					"guid": "$uuid1"
				},
				{
					"label": "even-more-data",
					"number": 3,
					"startMiB": 65,
					"sizeMiB": 32,
					"wipePartitionEntry": true
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
