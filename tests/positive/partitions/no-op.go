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
	// Tests that do nothing
	register.Register(register.PositiveTest, DoNothing())
	register.Register(register.PositiveTest, SpecifiedNonexistent())
}

func DoNothing() types.Test {
	name := "Do nothing when told to add no partitions"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "2.0.0"
		},
		"storage": {
			"disks": [
			{
				"device": "$disk0"
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

func SpecifiedNonexistent() types.Test {
	name := "Verify partitions with shouldexist=false do not get created"
	in := append(types.GetBaseDisk(), types.Disk{Alignment: types.IgnitionAlignment})
	out := append(types.GetBaseDisk(), types.Disk{Alignment: types.IgnitionAlignment})
	config := `{
		"ignition": {
			"version": "2.3.0-experimental"
		},
		"storage": {
			"disks": [
			{
				"device": "$disk0",
				"partitions": [
				{
					"number": 10,
					"shouldExist": false
				},
				{
					"number": 11,
					"shouldExist": false
				},
				{
					"number": 999,
					"shouldExist": false
				}
				]
			},
			{
				"device": "$disk1",
				"partitions": [
				{
					"number": 1,
					"shouldExist": false
				},
				{
					"number": 11,
					"shouldExist": false
				},
				{
					"number": 999,
					"shouldExist": false
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
