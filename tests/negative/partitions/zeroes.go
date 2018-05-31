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
	// Tests that involve zeros
	register.Register(register.NegativeTest, Partition9DoesNotFillDisk())
	register.Register(register.NegativeTest, Partition9DoesNotStartCorrectly())
}

func Partition9DoesNotFillDisk() types.Test {
	name := "Partition 9 is size 0 but does not fill the disk"
	in := types.GetBaseDisk()
	in[0].Partitions = append(in[0].Partitions, &types.Partition{
		Number: 10,
		Length: 65536,
	})
	out := in
	config := `{
		"ignition": {"version": "2.3.0-experimental"},
		"storage": {
			"disks": [
			{
				"device": "$disk0",
				"partitions": [
				{
					"number": 9,
					"size": 0
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

func Partition9DoesNotStartCorrectly() types.Test {
	name := "Partition 9 does not start at the largest chunk"
	in := types.GetBaseDisk()
	//insert a gap before 9
	tmp := in[0].Partitions[9-2-1]
	in[0].Partitions[9-2-1] = &types.Partition{
		Number: 10,
		Length: 65536,
	}
	in[0].Partitions = append(in[0].Partitions, tmp)
	out := in
	config := `{
		"ignition": {"version": "2.3.0-experimental"},
		"storage": {
			"disks": [
			{
				"device": "$disk0",
				"partitions": [
				{
					"number": 9,
					"start": 0
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
