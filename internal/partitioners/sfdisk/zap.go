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

package sfdisk

import (
	"encoding/binary"
	"fmt"
	"os"
)

// zapTable removes all partition table structures from the device,
// mirroring `sgdisk --zap-all`: the protective/legacy MBR partition
// entries (bytes 446..511 of the first sector, including the 0x55AA
// signature so blkid no longer detects any table), and the primary
// and backup GPT headers plus their partition entry arrays. Filesystem
// data inside former partitions is deliberately left alone — erasing
// signatures is the filesystems stage's job (wipeFilesystem), and the
// whole-disk-filesystem use case (systemd-mkfs on a wipedTable disk
// with no partitions) requires the disk to look unpartitioned, not to
// carry an empty GPT label. sfdisk has no zap mode whose semantics
// match (--zap also wipes filesystem signatures), so the zeroing is
// done directly.
func (op *Operation) zapTable() error {
	f, err := os.OpenFile(op.dev, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	size, sectorSize, err := diskGeometry(f)
	if err != nil {
		return err
	}
	if size < sectorSize {
		return fmt.Errorf("device %q too small for an MBR sector", op.dev)
	}
	usb := uint64(sectorSize)
	usize := uint64(size)

	zero := make([]byte, sectorSize)

	// 1. MBR partition entries + boot signature (fixed 512-byte
	// offsets inside the first sector, independent of sector size).
	if _, err := f.WriteAt(zero[:66], 446); err != nil {
		return fmt.Errorf("zeroing MBR entries: %v", err)
	}

	// 2. GPT primary structures, per the on-disk header (entry arrays
	// can live outside the default LBA 2..33 window).
	hdr, err := readGPTHeader(f, sectorSize)
	if err != nil {
		return err
	}
	if hdr != nil {
		entriesBytes := hdr.entryCount * hdr.entrySize
		entriesSectors := (entriesBytes + usb - 1) / usb
		// primary: header at LBA 1, entry array at entryLBA ..
		// entryLBA+entriesSectors-1
		primEnd := (hdr.entryLBA + entriesSectors) * usb
		if primEnd > usize {
			primEnd = usize
		}
		if err := zeroRange(f, usb, primEnd, sectorSize); err != nil {
			return fmt.Errorf("zeroing primary GPT: %v", err)
		}
		// backup: entry array immediately precedes the backup header
		// at backupLBA. Clamp to the real device end so a header that
		// claims an off-device backup LBA cannot drive us backwards
		// into the primary area.
		if hdr.backupLBA > entriesSectors+33 {
			bkEnd := (hdr.backupLBA + 1) * usb
			if bkEnd > usize {
				bkEnd = usize
			}
			bkStart := bkEnd - entriesSectors*usb
			if bkStart > primEnd {
				if err := zeroRange(f, bkStart, bkEnd, sectorSize); err != nil {
					return fmt.Errorf("zeroing backup GPT: %v", err)
				}
			}
		}
	}

	// 3. Best-effort kernel reread. Like sgdisk, a failure here is not
	// fatal (the kernel refuses while any partition is mounted, and
	// the stage's partx handling keeps the kernel view in sync for
	// the whole-table backend).
	rereadPartitionTable(f)

	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}

// gptHeader is the subset of the GPT header we need to locate the
// entry arrays and backup structures.
type gptHeader struct {
	entryLBA   uint64
	backupLBA  uint64
	entryCount uint64
	entrySize  uint64
}

// readGPTHeader parses the primary GPT header at LBA 1. Returns
// (nil, nil) when there is no GPT there (blank or MBR-only disk),
// which zapTable treats as "nothing further to clear".
func readGPTHeader(f *os.File, sectorSize int64) (*gptHeader, error) {
	buf := make([]byte, 92)
	if _, err := f.ReadAt(buf, sectorSize); err != nil {
		// short device: nothing to zap beyond the MBR
		return nil, nil
	}
	if string(buf[0:8]) != "EFI PART" {
		return nil, nil
	}
	h := &gptHeader{}
	h.backupLBA = binary.LittleEndian.Uint64(buf[32:40])
	h.entryLBA = binary.LittleEndian.Uint64(buf[72:80])
	h.entryCount = uint64(binary.LittleEndian.Uint32(buf[80:84]))
	h.entrySize = uint64(binary.LittleEndian.Uint32(buf[84:88]))
	// sanity-clamp: a corrupt header must not make us zero the world
	if h.entrySize < 128 || h.entrySize > 1024 || h.entryCount == 0 || h.entryCount > 65536 {
		h.entryCount = 128
		h.entrySize = 128
	}
	return h, nil
}

func zeroRange(f *os.File, start, end uint64, chunk int64) error {
	zeros := make([]byte, chunk)
	for off := start; off < end; off += uint64(chunk) {
		n := uint64(chunk)
		if end-off < n {
			n = end - off
		}
		if _, err := f.WriteAt(zeros[:n], int64(off)); err != nil {
			return err
		}
	}
	return nil
}
