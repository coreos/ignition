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

import (
	"fmt"
	"os"
)

// Distro-specific settings that can be overridden at link time with e.g.
// -X github.com/coreos/ignition/v2/internal/distro.mdadmCmd=/opt/bin/mdadm
var (
	// Device node directories and paths
	diskByLabelDir = "/dev/disk/by-label"

	// initrd file paths
	kernelCmdlinePath = "/proc/cmdline"
	bootIDPath        = "/proc/sys/kernel/random/boot_id"
	// initramfs directory containing distro-provided base config
	systemConfigDir = "/usr/lib/ignition"

	// Helper programs
	groupaddCmd  = "groupadd"
	groupdelCmd  = "groupdel"
	mdadmCmd     = "mdadm"
	mountCmd     = "mount"
	partxCmd     = "partx"
	sgdiskCmd    = "sgdisk"
	sfdiskCmd    = "sfdisk"
	modprobeCmd  = "modprobe"
	udevadmCmd   = "udevadm"
	usermodCmd   = "usermod"
	useraddCmd   = "useradd"
	userdelCmd   = "userdel"
	setfilesCmd  = "setfiles"
	wipefsCmd    = "wipefs"
	systemctlCmd = "systemctl"

	// Filesystem tools
	btrfsMkfsCmd = "mkfs.btrfs"
	ext4MkfsCmd  = "mkfs.ext4"
	swapMkfsCmd  = "mkswap"
	vfatMkfsCmd  = "mkfs.fat"
	xfsMkfsCmd   = "mkfs.xfs"

	// z/VM programs
	vmurCmd           = "vmur"
	chccwdevCmd       = "chccwdev"
	cioIgnoreCmd      = "cio_ignore"
	zkeycryptsetupCmd = "zkey-cryptsetup"
	zkeyCmd           = "zkey"

	// LUKS programs
	clevisCmd     = "clevis"
	cryptsetupCmd = "cryptsetup"

	// kargs programs
	kargsCmd = "ignition-kargs-helper"

	// Flags
	selinuxRelabel  = "true"
	blackboxTesting = "false"
	// writeAuthorizedKeysFragment indicates whether to write SSH keys
	// specified in the Ignition config as a fragment to
	// ".ssh/authorized_keys.d/ignition" ("true"), or to
	// ".ssh/authorized_keys" ("false").
	writeAuthorizedKeysFragment = "true"

	// Special file paths in the real root
	luksRealRootKeyFilePath = "/etc/luks/"
	resultFilePath          = "/etc/.ignition-result.json"
	luksCexSecureKeyRepo    = "/etc/zkey/repository/"
)

func DiskByLabelDir() string { return diskByLabelDir }

func KernelCmdlinePath() string { return kernelCmdlinePath }
func BootIDPath() string        { return bootIDPath }
func SystemConfigDir() string   { return fromEnv("SYSTEM_CONFIG_DIR", systemConfigDir) }

func GroupaddCmd() string  { return groupaddCmd }
func GroupdelCmd() string  { return groupdelCmd }
func MdadmCmd() string     { return mdadmCmd }
func MountCmd() string     { return mountCmd }
func PartxCmd() string     { return partxCmd }
func SgdiskCmd() string    { return sgdiskCmd }
func SfdiskCmd() string    { return sfdiskCmd }
func ModprobeCmd() string  { return modprobeCmd }
func UdevadmCmd() string   { return udevadmCmd }
func UsermodCmd() string   { return usermodCmd }
func UseraddCmd() string   { return useraddCmd }
func UserdelCmd() string   { return userdelCmd }
func SetfilesCmd() string  { return setfilesCmd }
func WipefsCmd() string    { return wipefsCmd }
func SystemctlCmd() string { return systemctlCmd }

func BtrfsMkfsCmd() string { return btrfsMkfsCmd }
func Ext4MkfsCmd() string  { return ext4MkfsCmd }
func SwapMkfsCmd() string  { return swapMkfsCmd }
func VfatMkfsCmd() string  { return vfatMkfsCmd }
func XfsMkfsCmd() string   { return xfsMkfsCmd }

func VmurCmd() string      { return vmurCmd }
func ChccwdevCmd() string  { return chccwdevCmd }
func CioIgnoreCmd() string { return cioIgnoreCmd }
func ZkeyCryptCmd() string { return zkeycryptsetupCmd }
func ZkeyCmd() string      { return zkeyCmd }

func ClevisCmd() string     { return clevisCmd }
func CryptsetupCmd() string { return cryptsetupCmd }

func KargsCmd() string { return kargsCmd }

func LuksRealRootKeyFilePath() string   { return luksRealRootKeyFilePath }
func ResultFilePath() string            { return resultFilePath }
func LuksRealVolumeKeyFilePath() string { return luksCexSecureKeyRepo }

func SelinuxRelabel() bool  { return bakedStringToBool(selinuxRelabel) && !BlackboxTesting() }
func BlackboxTesting() bool { return bakedStringToBool(blackboxTesting) }
func WriteAuthorizedKeysFragment() bool {
	return bakedStringToBool(fromEnv("WRITE_AUTHORIZED_KEYS_FRAGMENT", writeAuthorizedKeysFragment))
}

func fromEnv(nameSuffix, defaultValue string) string {
	value := os.Getenv("IGNITION_" + nameSuffix)
	if value != "" {
		return value
	}
	return defaultValue
}

func bakedStringToBool(s string) bool {
	// the linker only supports string args, so do some basic bool sensing
	switch s {
	case "true", "1":
		return true
	case "false", "0":
		return false
	default:
		// if we got a bad compile flag, just crash and burn rather than assume
		panic(fmt.Sprintf("value '%s' cannot be interpreted as a boolean", s))
	}
}
