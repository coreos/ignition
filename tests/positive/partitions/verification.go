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
	// Tests the verify existing partitions but do not create new ones
	register.Register(register.PositiveTest, VerifyBaseDisk())
	register.Register(register.PositiveTest, VerifyBaseDiskWithWipe())
}

func VerifyBaseDisk() types.Test {
	name := "partition.match.all"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "$version"
		},
		"storage": {
			"disks": [{
				"device": "$disk0",
				"partitions": [
				{
					"label": "EFI-SYSTEM",
					"number": 1,
					"startMiB": 2,
					"sizeMiB": 128,
					"typeGuid": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
				},
				{
					"label": "OEM",
					"number": 6,
					"startMiB": 130,
					"sizeMiB": 128,
					"typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
				},
				{
					"label": "ROOT",
					"number": 9,
					"startMiB": 258,
					"sizeMiB": 128,
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

func VerifyBaseDiskWithWipe() types.Test {
	name := "partition.match.all.withwipe"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	// guid from USR-A is removed so if it does try to recreate partitions, it will assign a random
	// one and fail the test.
	config := `{
		"ignition": {
			"version": "$version"
		},
		"storage": {
			"disks": [{
				"device": "$disk0",
				"partitions": [
				{
					"label": "EFI-SYSTEM",
					"number": 1,
					"startMiB": 2,
					"sizeMiB": 128,
					"wipePartitionEntry": true,
					"typeGuid": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
				},
				{
					"label": "OEM",
					"number": 6,
					"startMiB": 130,
					"sizeMiB": 128,
					"wipePartitionEntry": true,
					"typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
				},
				{
					"label": "ROOT",
					"number": 9,
					"startMiB": 258,
					"sizeMiB": 128,
					"wipePartitionEntry": true,
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
