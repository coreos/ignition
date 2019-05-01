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
					"label": "BIOS-BOOT",
					"number": 2,
					"startMiB": 130,
					"sizeMiB": 2,
					"typeGuid": "21686148-6449-6E6F-744E-656564454649"
				},
				{
					"label": "USR-A",
					"number": 3,
					"startMiB": 132,
					"sizeMiB": 1024,
					"typeGuid": "5DFBF5F4-2848-4BAC-AA5E-0D9A20B745A6",
					"guid": "7130c94a-213a-4e5a-8e26-6cce9662f132"
				},
				{
					"label": "USR-B",
					"number": 4,
					"startMiB": 1156,
					"sizeMiB": 1024,
					"typeGuid": "5DFBF5F4-2848-4BAC-AA5E-0D9A20B745A6",
					"guid": "e03dd35c-7c2d-4a47-b3fe-27f15780a57c"
				},
				{
					"label": "OEM",
					"number": 6,
					"startMiB": 2180,
					"sizeMiB": 128,
					"typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
				},
				{
					"label": "OEM-CONFIG",
					"number": 7,
					"startMiB": 2308,
					"sizeMiB": 64,
					"typeGuid": "c95dc21a-df0e-4340-8d7b-26cbfa9a03e0"
				},
				{
					"label": "ROOT",
					"number": 9,
					"startMiB": 2372,
					"sizeMiB": 6320,
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
					"label": "BIOS-BOOT",
					"number": 2,
					"startMiB": 130,
					"sizeMiB": 2,
					"wipePartitionEntry": true,
					"typeGuid": "21686148-6449-6E6F-744E-656564454649"
				},
				{
					"label": "USR-A",
					"number": 3,
					"startMiB": 132,
					"sizeMiB": 1024,
					"typeGuid": "5DFBF5F4-2848-4BAC-AA5E-0D9A20B745A6",
					"wipePartitionEntry": true,
					"guid": "7130c94a-213a-4e5a-8e26-6cce9662f132"
				},
				{
					"label": "USR-B",
					"number": 4,
					"startMiB": 1156,
					"sizeMiB": 1024,
					"typeGuid": "5DFBF5F4-2848-4BAC-AA5E-0D9A20B745A6",
					"wipePartitionEntry": true,
					"guid": "e03dd35c-7c2d-4a47-b3fe-27f15780a57c"
				},
				{
					"label": "OEM",
					"number": 6,
					"startMiB": 2180,
					"sizeMiB": 128,
					"wipePartitionEntry": true,
					"typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
				},
				{
					"label": "OEM-CONFIG",
					"number": 7,
					"startMiB": 2308,
					"sizeMiB": 64,
					"wipePartitionEntry": true,
					"typeGuid": "c95dc21a-df0e-4340-8d7b-26cbfa9a03e0"
				},
				{
					"label": "ROOT",
					"number": 9,
					"startMiB": 2372,
					"sizeMiB": 6320,
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
