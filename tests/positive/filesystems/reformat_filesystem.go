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
	"github.com/coreos/ignition/v2/tests/fixtures"
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, ReformatToBTRFS())
	register.Register(register.PositiveTest, ReformatToXFS())
	register.Register(register.PositiveTest, ReformatToVFAT())
	register.Register(register.PositiveTest, ReformatToEXT4())
	register.Register(register.PositiveTest, ReformatToSWAP())
	register.Register(register.PositiveTest, TestCannedZFSImage())
	register.Register(register.PositiveTest, TestEXT4ClobberZFS())
	register.Register(register.PositiveTest, TestXFSClobberZFS())
	register.Register(register.PositiveTest, TestVFATClobberZFS())
}

func ReformatToBTRFS() types.Test {
	name := "filesystem.create.btrfs.wipe"
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
	      "path": "/tmp0",
	      "device": "$DEVICE",
	      "format": "btrfs",
	      "wipeFilesystem": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
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
	name := "filesystem.create.xfs.wipe"
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
	      "path": "/tmp0",
	      "device": "$DEVICE",
	      "format": "xfs",
	      "label": "OEM",
	      "uuid": "$uuid0",
	      "wipeFilesystem": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
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
	name := "filesystem.create.vfat.wipe"
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
	      "path": "/tmp0",
	      "device": "$DEVICE",
	      "format": "vfat",
	      "label": "OEM",
	      "uuid": "2e24ec82",
	      "wipeFilesystem": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
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
	name := "filesystem.create.ext4.wipe"
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
	      "path": "/tmp0",
	      "device": "$DEVICE",
	      "format": "ext4",
	      "label": "OEM",
	      "uuid": "$uuid0",
	      "wipeFilesystem": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
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
	name := "filesystem.create.swap.wipe"
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
	      "path": "/tmp0",
	      "device": "$DEVICE",
	      "format": "swap",
	      "label": "OEM",
	      "uuid": "$uuid0",
	      "wipeFilesystem": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
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

func makeZFSDisk() []types.Disk {
	return []types.Disk{
		{
			Alignment: types.DefaultAlignment,
			Partitions: types.Partitions{
				{
					Number:         1,
					Label:          "ROOT",
					TypeCode:       "data",
					Length:         131072,
					FilesystemType: "ext4",
				},
				{
					Number:          2,
					Label:           "ZFS",
					TypeCode:        "data",
					Length:          131072,
					FilesystemType:  "image",
					FilesystemImage: fixtures.ZFS,
				},
			},
		},
	}
}

// We don't support creating ZFS filesystems, and doing so with zfs-fuse
// requires the zfs-fuse daemon to be running.  But ZFS also has an unusual
// property: it has multiple disk labels distributed throughout the disk,
// and none of mkfs.ext4, mkfs.xfs, or mkfs.vfat clobber them all.  If
// blkid or lsblk discover labels of both ZFS and one of those other
// filesystems, they won't report a filesystem type at all (and blkid will
// ignore the entire partition), and mount(8) will refuse to mount the FS
// without an explicit -t argument.  So we need to wipefs a partition before
// creating a new one.  In order to test this, we start from a canned ZFS
// image fixture.
//
// This test just copies in the ZFS fixture and confirms that it detects as
// ZFS.
func TestCannedZFSImage() types.Test {
	name := "filesystem.create.zfs.canned"
	in := makeZFSDisk()
	out := makeZFSDisk()
	config := `{
	  "ignition": { "version": "$version" }
	}`
	configMinVersion := "3.0.0"
	out[0].Partitions.GetPartition("ZFS").FilesystemType = "zfs_member"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func TestEXT4ClobberZFS() types.Test {
	name := "filesystem.create.zfs.ext4"
	in := makeZFSDisk()
	out := makeZFSDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "ZFS",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "filesystems": [{
	      "path": "/tmp0",
	      "device": "$DEVICE",
	      "format": "ext4",
	      "options": ["-E", "nodiscard"],
	      "wipeFilesystem": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	out[0].Partitions.GetPartition("ZFS").FilesystemType = "ext4"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func TestXFSClobberZFS() types.Test {
	name := "filesystem.create.zfs.xfs"
	in := makeZFSDisk()
	out := makeZFSDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "ZFS",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "filesystems": [{
	      "path": "/tmp0",
	      "device": "$DEVICE",
	      "format": "xfs",
	      "options": ["-K"],
	      "wipeFilesystem": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	out[0].Partitions.GetPartition("ZFS").FilesystemType = "xfs"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func TestVFATClobberZFS() types.Test {
	name := "filesystem.create.zfs.vfat"
	in := makeZFSDisk()
	out := makeZFSDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "ZFS",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "filesystems": [{
	      "path": "/tmp0",
	      "device": "$DEVICE",
	      "format": "vfat",
	      "wipeFilesystem": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	out[0].Partitions.GetPartition("ZFS").FilesystemType = "vfat"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
