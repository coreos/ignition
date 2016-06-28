// Copyright 2015 CoreOS, Inc.
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

package util

import (
	"fmt"
	"os"
	"syscall"
)

// These constants come from <cdrom.h>.
const (
	CDROM_DRIVE_STATUS = 0x5326
)

// These constants come from <cdrom.h>.
const (
	CDS_NO_INFO = iota
	CDS_NO_DISC
	CDS_TRAY_OPEN
	CDS_DRIVE_NOT_READY
	CDS_DISC_OK
)

func AssertCdromOnline(devicePath string) error {
	device, err := os.Open(devicePath)
	if err != nil {
		return fmt.Errorf("failed to open CD-ROM device %s: %v", devicePath, err)
	}
	defer device.Close()

	status, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(device.Fd()),
		uintptr(CDROM_DRIVE_STATUS),
		uintptr(0),
	)

	switch status {
	case CDS_NO_INFO:
		return fmt.Errorf("%s drive status: no info", devicePath)
	case CDS_NO_DISC:
		return fmt.Errorf("%s drive status: no disc", devicePath)
	case CDS_TRAY_OPEN:
		return fmt.Errorf("%s drive status: open", devicePath)
	case CDS_DRIVE_NOT_READY:
		return fmt.Errorf("%s drive status: not ready", devicePath)
	case CDS_DISC_OK:
		return nil
	default:
		return fmt.Errorf("%s failed to get drive status: %s", devicePath, errno.Error())
	}
}
