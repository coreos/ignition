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

package distro

// Distro-specific settings that can be overridden at link time with e.g.
// -X github.com/coreos/ignition/internal/distro.mdadmCmd=/opt/bin/mdadm
var (
	// Helper programs
	mdadmCmd   = "/usr/sbin/mdadm"
	mountCmd   = "/usr/bin/mount"
	sgdiskCmd  = "/usr/sbin/sgdisk"
	udevadmCmd = "/usr/bin/udevadm"

	// Filesystem tools
	btrfsMkfsCmd = "/usr/sbin/mkfs.btrfs"
	ext4MkfsCmd  = "/usr/sbin/mkfs.ext4"
	swapMkfsCmd  = "/usr/sbin/mkswap"
	vfatMkfsCmd  = "/usr/sbin/mkfs.vfat"
	xfsMkfsCmd   = "/usr/sbin/mkfs.xfs"
)

func MdadmCmd() string   { return mdadmCmd }
func MountCmd() string   { return mountCmd }
func SgdiskCmd() string  { return sgdiskCmd }
func UdevadmCmd() string { return udevadmCmd }

func BtrfsMkfsCmd() string { return btrfsMkfsCmd }
func Ext4MkfsCmd() string  { return ext4MkfsCmd }
func SwapMkfsCmd() string  { return swapMkfsCmd }
func VfatMkfsCmd() string  { return vfatMkfsCmd }
func XfsMkfsCmd() string   { return xfsMkfsCmd }
