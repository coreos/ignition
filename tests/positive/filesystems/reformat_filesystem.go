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
	register.Register(register.PositiveTest, ReformatToBTRFS_2_0_0())
	register.Register(register.PositiveTest, ReformatToXFS_2_0_0())
	register.Register(register.PositiveTest, ReformatToEXT4_2_0_0())
	register.Register(register.PositiveTest, ReformatToBTRFS_2_1_0())
	register.Register(register.PositiveTest, ReformatToXFS_2_1_0())
	register.Register(register.PositiveTest, ReformatToVFAT_2_1_0())
	register.Register(register.PositiveTest, ReformatToEXT4_2_1_0())
	register.Register(register.PositiveTest, ReformatToSWAP_2_1_0())
}

func ReformatToBTRFS_2_0_0() types.Test {
	name := "Reformat a Filesystem to Btrfs - 2.0.0"
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
	configMinVersion := "2.0.0"
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

func ReformatToXFS_2_0_0() types.Test {
	name := "Reformat a Filesystem to XFS - 2.0.0"
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
	configMinVersion := "2.0.0"
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

func ReformatToVFAT_2_0_0() types.Test {
	name := "Reformat a Filesystem to VFAT - 2.0.0"
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
	        "create": {
	          "force": true,
	          "options": [ "-n", "OEM", "-i", "$uuid0" ]
	        }
	      }
	    }]
	  }
	}`
	configMinVersion := "2.0.0"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "vfat"
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

func ReformatToEXT4_2_0_0() types.Test {
	name := "Reformat a Filesystem to EXT4 - 2.0.0"
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
	configMinVersion := "2.0.0"
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

func ReformatToBTRFS_2_1_0() types.Test {
	name := "Reformat a Filesystem to Btrfs - 2.1.0"
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
	        "label": "OEM",
		"uuid": "$uuid0",
		"wipeFilesystem": true
	      }
	    }]
	  }
	}`
	configMinVersion := "2.1.0"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "btrfs"
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

func ReformatToXFS_2_1_0() types.Test {
	name := "Reformat a Filesystem to XFS - 2.1.0"
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
	        "label": "OEM",
		"uuid": "$uuid0",
		"wipeFilesystem": true
	      }
	    }]
	  }
	}`
	configMinVersion := "2.1.0"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "xfs"
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

func ReformatToVFAT_2_1_0() types.Test {
	name := "Reformat a Filesystem to VFAT - 2.1.0"
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
	configMinVersion := "2.1.0"
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

func ReformatToEXT4_2_1_0() types.Test {
	name := "Reformat a Filesystem to EXT4 - 2.1.0"
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
	        "label": "OEM",
		"uuid": "$uuid0",
		"wipeFilesystem": true
	      }
	    }]
	  }
	}`
	configMinVersion := "2.1.0"
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

func ReformatToSWAP_2_1_0() types.Test {
	name := "Reformat a Filesystem to SWAP - 2.1.0"
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
	configMinVersion := "2.1.0"
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
