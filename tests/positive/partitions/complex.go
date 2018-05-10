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
			"version": "2.3.0-experimental"
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
					"typeGuid": "316f19f9-9e0f-431e-859e-ae6908dbe8ca",
					"guid": "53f2e871-f468-437c-b90d-f3c6409df81a",
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
					"typeGuid": "6050e8fc-1e31-473b-bcc0-714e32fcb09d",
					"guid": "f14984bc-6f08-4885-b668-526263469a00",
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

	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "p1",
				Number:   1,
				Length:   131072,
				TypeGUID: "316f19f9-9e0f-431e-859e-ae6908dbe8ca",
				GUID:     "3ED3993F-0016-422B-B134-09FCBA6F66EF",
			},
			{
				Label:    "dont-delete",
				Number:   2,
				Length:   65536,
				TypeGUID: "F39C522B-9966-4429-A8F8-417CD5D83E5E",
				GUID:     "26cc1e6a-39a2-4502-a957-28f8e0ac00e7",
			},
			{
				Label:    "garbo-town",
				Number:   3,
				Length:   65536,
				TypeGUID: "9dfde6db-a308-497b-9f8c-55db12d4d9a1",
				GUID:     "115ee54b-00bd-4afe-be9b-bf1912dec92d",
			},
			{
				TypeCode: "blank",
				Length:   131072,
			},
			{
				Label:    "more-junk",
				Number:   4,
				Length:   65536,
				TypeGUID: "ee378f79-6b7a-4f7a-8029-7a5736d12bbf",
				GUID:     "5e89fa40-183c-4346-b2e1-2f10fa2190e1",
			},
			{
				Label:    "dont-delete2",
				Number:   5,
				Length:   65536,
				TypeGUID: "9dd1a91a-a39a-4594-b431-60a7fb630bcb",
				GUID:     "39baf7ef-cb3e-4343-8bfc-c2391b8b5607",
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
				TypeGUID: "316f19f9-9e0f-431e-859e-ae6908dbe8ca",
				GUID:     "53f2e871-f468-437c-b90d-f3c6409df81a",
			},
			{
				Label:    "dont-delete",
				Number:   2,
				Length:   65536,
				TypeGUID: "F39C522B-9966-4429-A8F8-417CD5D83E5E",
				GUID:     "26cc1e6a-39a2-4502-a957-28f8e0ac00e7",
			},
			{
				Label:    "new-biggest",
				Number:   3,
				Length:   262144,
				TypeGUID: "6050e8fc-1e31-473b-bcc0-714e32fcb09d",
				GUID:     "f14984bc-6f08-4885-b668-526263469a00",
			},
			{
				Label:    "dont-delete2",
				Number:   5,
				Length:   65536,
				TypeGUID: "9dd1a91a-a39a-4594-b431-60a7fb630bcb",
				GUID:     "39baf7ef-cb3e-4343-8bfc-c2391b8b5607",
			},
			{
				TypeCode: "blank",
				Length:   131072,
			},
		},
	})

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}
