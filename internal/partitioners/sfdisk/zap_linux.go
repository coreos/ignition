// Copyright 2024 Red Hat, Inc.
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

//go:build linux

package sfdisk

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// diskGeometry returns the device size in bytes and its logical sector
// size via block ioctls.
func diskGeometry(f *os.File) (int64, int64, error) {
	var size uint64
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKGETSIZE64, uintptr(unsafe.Pointer(&size))); errno != 0 {
		return 0, 0, errno
	}
	ssize, err := unix.IoctlGetInt(int(f.Fd()), unix.BLKSSZGET)
	if err != nil {
		return 0, 0, err
	}
	return int64(size), int64(ssize), nil
}

// rereadPartitionTable asks the kernel to reread the partition table
// (BLKRRPART), best effort like sgdisk: the kernel rejects it while any
// partition of the device is mounted, and ignition's partx follow-up
// keeps the kernel view in sync regardless.
func rereadPartitionTable(f *os.File) {
	// best effort, see above
	_, _, _ = unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKRRPART, 0)
}
