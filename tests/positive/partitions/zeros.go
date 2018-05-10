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
	// Tests that just create partitions and involve 0s
	register.Register(register.PositiveTest, PartitionSizeStart0())
	register.Register(register.PositiveTest, PartitionStartNumber0())
	register.Register(register.PositiveTest, ResizeRootFillDisk())
	register.Register(register.PositiveTest, VerifyRootFillsDisk())
	register.Register(register.PositiveTest, VerifyUnspecifiedIsDoNotCare())
	register.Register(register.PositiveTest, NumberZeroHappensLast())
}

func PartitionSizeStart0() types.Test {
	name := "Create a partition with size and start 0"
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
					"label": "fills-disk",
					"number": 1,
					"start": 0,
					"size": 0,
					"typeGuid": "F39C522B-9966-4429-A8F8-417CD5D83E5E",
					"guid": "3ED3993F-0016-422B-B134-09FCBA6F66EF"
				}]
			}]
		}
	}`

	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "fills-disk",
				Number:   1,
				Length:   65536,
				TypeGUID: "F39C522B-9966-4429-A8F8-417CD5D83E5E",
				GUID:     "3ED3993F-0016-422B-B134-09FCBA6F66EF",
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

func PartitionStartNumber0() types.Test {
	name := "Create a partition with number and start 0"
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
					"label": "uno",
					"size": 65536,
					"typeGuid": "F39C522B-9966-4429-A8F8-417CD5D83E5E",
					"guid": "3ED3993F-0016-422B-B134-09FCBA6F66EF"
				},
				{
					"label": "dos",
					"size": 65536,
					"typeGuid": "F39C522B-9966-4429-A8F8-417CD5D83E5E",
					"guid": "6A6BD6B9-4345-4AFB-974E-08D5A343E8F8"
				},
				{
					"label": "tres",
					"size": 65536,
					"typeGuid": "F39C522B-9966-4429-A8F8-417CD5D83E5E",
					"guid": "FE6108DC-096A-4E62-83CB-A11CE9D8E633"
				}]
			}]
		}
	}`

	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:    "uno",
				Number:   1,
				Length:   65536,
				TypeGUID: "F39C522B-9966-4429-A8F8-417CD5D83E5E",
				GUID:     "3ED3993F-0016-422B-B134-09FCBA6F66EF",
			},
			{
				Label:    "dos",
				Number:   2,
				Length:   65536,
				TypeGUID: "F39C522B-9966-4429-A8F8-417CD5D83E5E",
				GUID:     "6A6BD6B9-4345-4AFB-974E-08D5A343E8F8",
			},
			{
				Label:    "tres",
				Number:   3,
				Length:   65536,
				TypeGUID: "F39C522B-9966-4429-A8F8-417CD5D83E5E",
				GUID:     "FE6108DC-096A-4E62-83CB-A11CE9D8E633",
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

func ResizeRootFillDisk() types.Test {
	name := "Resize the ROOT partition to fill the disk"
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
					"start": 0,
					"size": 0,
					"typeGuid": "3884DD41-8582-4404-B9A8-E9B84F2DF50E",
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

func VerifyRootFillsDisk() types.Test {
	name := "Verify the ROOT partition to fills the default disk"
	in := types.GetBaseDisk()
	out := in
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
					"start": 0,
					"size": 0,
					"typeGuid": "3884DD41-8582-4404-B9A8-E9B84F2DF50E"
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

func VerifyUnspecifiedIsDoNotCare() types.Test {
	name := "Verify unspecified size/start matches even when its not the max size"
	in := types.GetBaseDisk()
	in[0].Partitions = append(in[0].Partitions, &types.Partition{
		TypeCode: "blank",
		Length:   65536,
	})
	out := types.GetBaseDisk()
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
					"typeGuid": "3884DD41-8582-4404-B9A8-E9B84F2DF50E"
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

func NumberZeroHappensLast() types.Test {
	name := "Verify the partitions with number=0 happen are processed last"
	in := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Number:   1,
				Label:    "foobar",
				Length:   65536,
				TypeGUID: "1f4ce97c-10fc-4daf-8b2c-0075bd34df43",
				GUID:     "8426957e-e444-40c8-93ed-f6c0d69cccde",
			},
		},
	})
	out := append(types.GetBaseDisk(), types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Number:   1,
				Label:    "foobar",
				Length:   65536,
				TypeGUID: "1f4ce97c-10fc-4daf-8b2c-0075bd34df43",
				GUID:     "8426957e-e444-40c8-93ed-f6c0d69cccde",
			},
			{
				Number: 2,
				Label:  "newpart",
				Length: 65536,
			},
		},
	})
	config := `{
		"ignition": {
			"version": "2.3.0-experimental"
		},
		"storage": {
			"disks": [{
				"device": "$disk1",
				"partitions": [
				{
					"label": "newpart",
					"size": 65536
				},
				{
					"number": 1,
					"label": "foobar"
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
