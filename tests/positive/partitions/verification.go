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
	// Tests the verify existing partitions but do not create new ones
	register.Register(register.PositiveTest, VerifyBaseDisk())
	register.Register(register.PositiveTest, VerifyBaseDiskWithWipe())
}

func VerifyBaseDisk() types.Test {
	name := "Verify the base disk does not change with a matching Ignition spec"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "2.1.0"
		},
		"storage": {
			"disks": [{
				"device": "$disk0",
				"partitions": [
				{
					"label": "EFI-SYSTEM",
					"number": 1,
					"start": 4096,
					"size": 262144,
					"typeGuid": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
				},
				{
					"label": "BIOS-BOOT",
					"number": 2,
					"start": 266240,
					"size": 4096,
					"typeGuid": "21686148-6449-6E6F-744E-656564454649"
				},
				{
					"label": "USR-A",
					"number": 3,
					"start": 270336,
					"size": 2097152,
					"typeGuid": "5dfbf5f4-2848-4bac-aa5e-0d9a20b745a6",
					"guid": "7130c94a-213a-4e5a-8e26-6cce9662f132"
				},
				{
					"label": "USR-B",
					"number": 4,
					"start": 2367488,
					"size": 2097152,
					"typeGuid": "5dfbf5f4-2848-4bac-aa5e-0d9a20b745a6",
					"guid": "e03dd35c-7c2d-4a47-b3fe-27f15780a57c"
				},
				{
					"label": "OEM",
					"number": 6,
					"start": 4464640,
					"size": 262144,
					"typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
				},
				{
					"label": "OEM-CONFIG",
					"number": 7,
					"start": 4726784,
					"size": 131072,
					"typeGuid": "c95dc21a-df0e-4340-8d7b-26cbfa9a03e0"
				},
				{
					"label": "ROOT",
					"number": 9,
					"start": 4857856,
					"size": 12943360,
					"typeGuid": "3884dd41-8582-4404-b9a8-e9b84f2df50e"
				}
				]
			}]
		}
	}`

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func VerifyBaseDiskWithWipe() types.Test {
	name := "Verify the base disk does not change with a matching Ignition spec with wipePartitionEntry as true"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	// guid from USR-A is removed so if it does try to recreate partitions, it will assign a random
	// one and fail the test.
	config := `{
		"ignition": {
			"version": "2.3.0-experimental"
		},
		"storage": {
			"disks": [{
				"device": "$disk0",
				"partitions": [
				{
					"label": "EFI-SYSTEM",
					"number": 1,
					"start": 4096,
					"size": 262144,
					"wipePartitionEntry": true,
					"typeGuid": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
				},
				{
					"label": "BIOS-BOOT",
					"number": 2,
					"start": 266240,
					"size": 4096,
					"wipePartitionEntry": true,
					"typeGuid": "21686148-6449-6E6F-744E-656564454649"
				},
				{
					"label": "USR-A",
					"number": 3,
					"start": 270336,
					"size": 2097152,
					"wipePartitionEntry": true,
					"typeGuid": "5dfbf5f4-2848-4bac-aa5e-0d9a20b745a6"
				},
				{
					"label": "USR-B",
					"number": 4,
					"start": 2367488,
					"size": 2097152,
					"wipePartitionEntry": true,
					"typeGuid": "5dfbf5f4-2848-4bac-aa5e-0d9a20b745a6",
					"guid": "e03dd35c-7c2d-4a47-b3fe-27f15780a57c"
				},
				{
					"label": "OEM",
					"number": 6,
					"start": 4464640,
					"size": 262144,
					"wipePartitionEntry": true,
					"typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
				},
				{
					"label": "OEM-CONFIG",
					"number": 7,
					"start": 4726784,
					"size": 131072,
					"wipePartitionEntry": true,
					"typeGuid": "c95dc21a-df0e-4340-8d7b-26cbfa9a03e0"
				},
				{
					"label": "ROOT",
					"number": 9,
					"start": 4857856,
					"size": 12943360,
					"wipePartitionEntry": true,
					"typeGuid": "3884dd41-8582-4404-b9a8-e9b84f2df50e"
				}
				]
			}]
		}
	}`

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}
