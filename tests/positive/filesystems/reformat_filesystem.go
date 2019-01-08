// Copyright 2017 CoreOS, Inc.
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

package filesystems

import (
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.PositiveTest, ReformatToBTRFS())
	register.Register(register.PositiveTest, ReformatToXFS())
	register.Register(register.PositiveTest, ReformatToVFAT())
	register.Register(register.PositiveTest, ReformatToEXT4())
	register.Register(register.PositiveTest, ReformatToSWAP())
}

func ReformatToBTRFS() types.Test {
	name := "Reformat a Filesystem to Btrfs"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "filesystems": [{
	      "mount": {
	        "device": "$DEVICE",
	        "format": "btrfs",
	        "create": {
	          "force": true,
	          "options": [ "--label=OEM", "--uuid=$uuid0" ]
	        }
	      }
	    }]
	  }
	}`
	configMinVersion := "3.0.0-experimental"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "btrfs"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ReformatToXFS() types.Test {
	name := "Reformat a Filesystem to XFS"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "filesystems": [{
	      "mount": {
	        "device": "$DEVICE",
	        "format": "xfs",
	        "create": {
	          "force": true,
	          "options": [ "-L", "OEM", "-m", "uuid=$uuid0" ]
	        }
	      }
	    }]
	  }
	}`
	configMinVersion := "3.0.0-experimental"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "xfs"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ReformatToVFAT() types.Test {
	name := "Reformat a Filesystem to VFAT"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "filesystems": [{
	      "mount": {
	        "device": "$DEVICE",
	        "format": "vfat",
	        "label": "OEM",
		"uuid": "2e24ec82",
		"wipeFilesystem": true
	      }
	    }]
	  }
	}`
	configMinVersion := "3.0.0-experimental"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "vfat"
	out[0].Partitions.GetPartition("OEM").FilesystemUUID = "2e24ec82"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ReformatToEXT4() types.Test {
	name := "Reformat a Filesystem to EXT4"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "filesystems": [{
	      "mount": {
	        "device": "$DEVICE",
	        "format": "ext4",
	        "create": {
	          "force": true,
	          "options": [ "-L", "OEM", "-U", "$uuid0" ]
	        }
	      }
	    }]
	  }
	}`
	configMinVersion := "3.0.0-experimental"
	in[0].Partitions.GetPartition("OEM").FilesystemType = "ext2"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "ext4"
	out[0].Partitions.GetPartition("OEM").FilesystemUUID = "$uuid0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ReformatToSWAP() types.Test {
	name := "Reformat a Filesystem to SWAP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "filesystems": [{
	      "mount": {
	        "device": "$DEVICE",
	        "format": "swap",
	        "label": "OEM",
	        "uuid": "$uuid0",
		"wipeFilesystem": true
	      }
	    }]
	  }
	}`
	configMinVersion := "3.0.0-experimental"
	in[0].Partitions.GetPartition("OEM").FilesystemType = "ext2"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "swap"
	out[0].Partitions.GetPartition("OEM").FilesystemUUID = "$uuid0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
