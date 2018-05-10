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
	// Tests that just create partitions with no 0s
	register.Register(register.PositiveTest, CreatePartition())
	register.Register(register.PositiveTest, WipeAndCreateNewPartitions())
	register.Register(register.PositiveTest, AppendPartitions())
	register.Register(register.PositiveTest, ResizeRoot())
}

func CreatePartition() types.Test {
	name := "Create a single partition on a blank disk"
	in := append(types.GetBaseDisk(), types.Disk{Alignment: types.IgnitionAlignment})
	out := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "create-partition",
				Number:   1,
				Length:   65536,
				TypeGUID: "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
				GUID:     "05AE8178-224E-4744-862A-4F4B042662D0",
			},
		},
	})
	config := `{
		"ignition": {
			"version": "2.1.0"
		},
		"storage": {
			"disks": [
			{
				"device": "$disk1",
				"partitions": [
				{
					"number": 1,
					"size": 65536,
					"label": "create-partition",
					"typeGuid": "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
					"guid": "05AE8178-224E-4744-862A-4F4B042662D0"
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

func WipeAndCreateNewPartitions() types.Test {
	name := "Wipe disk and create new partitions"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "2.1.0"
		},
		"storage": {
			"disks": [
			{
				"device": "$disk1",
				"wipeTable": true,
				"partitions": [
				{
					"label": "important-data",
					"number": 1,
					"size": 65536,
					"typeGuid": "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
					"guid": "8A7A6E26-5E8F-4CCA-A654-46215D4696AC"
				},
				{
					"label": "ephemeral-data",
					"number": 2,
					"size": 131072,
					"typeGuid": "CA7D7CCB-63ED-4C53-861C-1742536059CC",
					"guid": "A51034E6-26B3-48DF-BEED-220562AC7AD1"
				}
				]
			}
			]
		}
	}`
	// Create dummy partitions. The UUIDs in the input partitions
	// are intentionally different so if Ignition doesn't do the right thing the
	// validation will fail.
	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
				GUID:     "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "CA7D7CCB-63ED-4C53-861C-1742536059CC",
				GUID:     "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
			},
		},
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
				GUID:     "8A7A6E26-5E8F-4CCA-A654-46215D4696AC",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "CA7D7CCB-63ED-4C53-861C-1742536059CC",
				GUID:     "A51034E6-26B3-48DF-BEED-220562AC7AD1",
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

func AppendPartitions() types.Test {
	name := "Append partition to an existing partition table"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "2.1.0"
		},
		"storage": {
			"disks": [{
				"device": "$disk1",
				"wipeTable": false,
				"partitions": [{
					"label": "additional-partition",
					"number": 3,
					"size": 65536,
					"typeGuid": "F39C522B-9966-4429-A8F8-417CD5D83E5E",
					"guid": "3ED3993F-0016-422B-B134-09FCBA6F66EF"
				},
				{
					"label": "additional-partition2",
					"number": 4,
					"size": 65536,
					"typeGuid": "F39C522B-9966-4429-A8F8-417CD5D83E5E",
					"guid": "accedd09-76c2-4363-9893-f5689a78c47f"
				}]
			}]
		}
	}`

	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
				GUID:     "8A7A6E26-5E8F-4CCA-A654-46215D4696AC",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "CA7D7CCB-63ED-4C53-861C-1742536059CC",
				GUID:     "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
			},
		},
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
				GUID:     "8A7A6E26-5E8F-4CCA-A654-46215D4696AC",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "CA7D7CCB-63ED-4C53-861C-1742536059CC",
				GUID:     "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
			},
			{
				Label:    "additional-partition",
				Number:   3,
				Length:   65536,
				TypeGUID: "F39C522B-9966-4429-A8F8-417CD5D83E5E",
				GUID:     "3ED3993F-0016-422B-B134-09FCBA6F66EF",
			},
			{
				Label:    "additional-partition2",
				Number:   4,
				Length:   65536,
				TypeGUID: "F39C522B-9966-4429-A8F8-417CD5D83E5E",
				GUID:     "accedd09-76c2-4363-9893-f5689a78c47f",
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

func ResizeRoot() types.Test {
	name := "Resize the ROOT partition to be bigger"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	out[0].Partitions[9-2-1].Length = 12943360 + 65536
	config := `{
		"ignition": {
			"version": "2.3.0-experimental"
		},
		"storage": {
			"disks": [{
				"device": "$disk0",
				"partitions": [{
					"label": "ROOT",
					"number": 9,
					"size": 13008896,
					"typeGuid": "3884DD41-8582-4404-B9A8-E9B84F2DF50E",
					"guid": "3ED3993F-0016-422B-B134-09FCBA6F66EF",
					"wipePartitionEntry": true
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
