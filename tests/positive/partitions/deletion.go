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
	// Tests that deletes partition(s)
	register.Register(register.PositiveTest, DeleteOne())
	register.Register(register.PositiveTest, DeleteAll())
}

func DeleteOne() types.Test {
	name := "Delete single partition"
	in := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:  "deleteme",
				Number: 1,
				Length: 65536,
			},
		},
	})
	out := append(types.GetBaseDisk(), types.Disk{Alignment: types.IgnitionAlignment})
	config := `{
		"ignition": {
			"version": "2.3.0-experimental"
		},
		"storage": {
			"disks": [
			{
				"device": "$disk1",
				"partitions": [
				{
					"number": 1,
					"shouldExist": false,
					"wipePartitionEntry": true
				}
				]
			}
			]
		}
	}`
	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func DeleteAll() types.Test {
	name := "Delete all partitions on a disk"
	in := append(types.GetBaseDisk(), types.GetBaseDisk()...)
	out := append(types.GetBaseDisk(), types.Disk{Alignment: types.IgnitionAlignment})
	config := `{
		"ignition": {
			"version": "2.3.0-experimental"
		},
		"storage": {
			"disks": [
			{
				"device": "$disk1",
				"partitions": [
				{
					"number": 1,
					"shouldExist": false,
					"wipePartitionEntry": true
				},
				{
					"number": 2,
					"shouldExist": false,
					"wipePartitionEntry": true
				},
				{
					"number": 3,
					"shouldExist": false,
					"wipePartitionEntry": true
				},
				{
					"number": 4,
					"shouldExist": false,
					"wipePartitionEntry": true
				},
				{
					"number": 6,
					"shouldExist": false,
					"wipePartitionEntry": true
				},
				{
					"number": 7,
					"shouldExist": false,
					"wipePartitionEntry": true
				},
				{
					"number": 9,
					"shouldExist": false,
					"wipePartitionEntry": true
				}
				]
			}
			]
		}
	}`
	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}
