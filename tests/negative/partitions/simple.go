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
	// Tests that have do not involve zeros
	register.Register(register.NegativeTest, ShouldNotExistNoWipeEntry())
	register.Register(register.NegativeTest, DoesNotMatchNoWipeEntry())
	register.Register(register.NegativeTest, ValidAndDoesNotMatchNoWipeEntry())
	register.Register(register.NegativeTest, NotThereAndDoesNotMatchNoWipeEntry())
}

func ShouldNotExistNoWipeEntry() types.Test {
	name := "Partition should not exist but wipePartitionEntry is false"
	in := types.GetBaseDisk()
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

func DoesNotMatchNoWipeEntry() types.Test {
	name := "Partition does not match and wipePartitionEntry is false"
	in := types.GetBaseDisk()
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
					"size": 4096
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

func ValidAndDoesNotMatchNoWipeEntry() types.Test {
	name := "Partition does not match and wipePartitionEntry is false but the first partition matches"
	in := types.GetBaseDisk()
	out := in
	config := `{
		"ignition": {"version": "2.3.0-experimental"},
		"storage": {
			"disks": [
			{
				"device": "$disk0",
				"partitions": [
				{
					"number": 1
				},
				{
					"number": 9,
					"size": 4096
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

func NotThereAndDoesNotMatchNoWipeEntry() types.Test {
	name := "Partition does not match and wipePartitionEntry is false but a partition matches not existing"
	in := types.GetBaseDisk()
	out := in
	config := `{
		"ignition": {"version": "2.3.0-experimental"},
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
					"number": 9,
					"size": 4096
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
