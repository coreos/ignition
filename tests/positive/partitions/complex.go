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
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	// Everything and the kitchen sink
	register.Register(register.PositiveTest, KitchenSink())
}

func KitchenSink() types.Test {
	name := "Complex partitioning case"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	// Ignition should not clobber by default, so omit the partition 5 entry
	config := `{
		"ignition": {
			"version": "$version"
		},
		"storage": {
			"disks": [{
				"device": "$disk1",
				"wipeTable": false,
				"partitions": [
				{
					"label": "p1",
					"number": 1,
					"start": 2048,
					"size": 65536,
					"typeGuid": "$uuid0",
					"guid": "$uuid1",
					"wipePartitionEntry": true
				},
				{
					"label": "dont-delete",
					"number": 2,
					"size": 65536
				},
				{
					"label": "new-biggest",
					"number": 3,
					"size": 0,
					"start": 0,
					"typeGuid": "$uuid2",
					"guid": "$uuid3",
					"wipePartitionEntry": true
				},
				{
					"number": 4,
					"shouldExist": false,
					"wipePartitionEntry": true
				},
				{
					"number": 9,
					"shouldExist": false
				}
				]
			}]
		}
	}`
	configMinVersion := "3.0.0-experimental"

	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "p1",
				Number:   1,
				Length:   131072,
				TypeGUID: "$uuid0",
				GUID:     "$uuid4",
			},
			{
				Label:    "dont-delete",
				Number:   2,
				Length:   65536,
				TypeGUID: "$uuid5",
				GUID:     "$uuid6",
			},
			{
				Label:    "garbo-town",
				Number:   3,
				Length:   65536,
				TypeGUID: "$uuid7",
				GUID:     "$uuid8",
			},
			{
				TypeCode: "blank",
				Length:   131072,
			},
			{
				Label:    "more-junk",
				Number:   4,
				Length:   65536,
				TypeGUID: "$uuid9",
				GUID:     "$uuid10",
			},
			{
				Label:    "dont-delete2",
				Number:   5,
				Length:   65536,
				TypeGUID: "$uuid11",
				GUID:     "$uuid12",
			},
			{
				TypeCode: "blank",
				Length:   131072,
			},
		},
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "p1",
				Number:   1,
				Length:   65536,
				TypeGUID: "$uuid0",
				GUID:     "$uuid1",
			},
			{
				Label:    "dont-delete",
				Number:   2,
				Length:   65536,
				TypeGUID: "$uuid5",
				GUID:     "$uuid6",
			},
			{
				Label:    "new-biggest",
				Number:   3,
				Length:   262144,
				TypeGUID: "$uuid2",
				GUID:     "$uuid3",
			},
			{
				Label:    "dont-delete2",
				Number:   5,
				Length:   65536,
				TypeGUID: "$uuid11",
				GUID:     "$uuid12",
			},
			{
				TypeCode: "blank",
				Length:   131072,
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
