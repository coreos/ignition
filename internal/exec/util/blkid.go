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

// +build linux

package util

// We want at least this warning, since the default C behavior of
// assuming int foo(int) is totally broken.

// #cgo CFLAGS: -Werror=implicit-function-declaration
// #cgo LDFLAGS: -lblkid
// #include <stdlib.h>
// #include "blkid.h"
import "C"

import (
	"bytes"
	"fmt"
	"strings"
	"unsafe"

	"github.com/coreos/ignition/v2/config/util"
)

const (
	field_name_type  = "TYPE"
	field_name_uuid  = "UUID"
	field_name_label = "LABEL"
)

type DiskInfo struct {
	LogicalSectorSize int // 4k or 512
	Partitions        []PartitionInfo
}

func (d DiskInfo) GetPartition(n int) (PartitionInfo, bool) {
	for _, part := range d.Partitions {
		if part.Number == n {
			return part, true
		}
	}
	return PartitionInfo{}, false
}

type PartitionInfo struct {
	Label         string
	GUID          string
	TypeGUID      string
	StartSector   int64
	SizeInSectors int64
	Number        int
}

type FilesystemInfo struct {
	Type  string
	UUID  string
	Label string
}

// If allowAmbivalent is false, fail if we find traces of more than one
// filesystem on the device.
func GetFilesystemInfo(device string, allowAmbivalent bool) (FilesystemInfo, error) {
	var info FilesystemInfo
	var err error
	info.Type, err = filesystemLookup(device, allowAmbivalent, field_name_type)
	if err != nil {
		return FilesystemInfo{}, err
	}
	info.UUID, err = filesystemLookup(device, allowAmbivalent, field_name_uuid)
	if err != nil {
		return FilesystemInfo{}, err
	}
	info.Label, err = filesystemLookup(device, allowAmbivalent, field_name_label)
	if err != nil {
		return FilesystemInfo{}, err
	}
	return info, nil
}

// cResultToErr takes a result_t from the blkid c code and a device it was operating on
// and returns a golang error describing the result code.
func cResultToErr(res C.result_t, device string) error {
	switch res {
	case C.RESULT_OK:
		return nil
	case C.RESULT_OPEN_FAILED:
		return fmt.Errorf("failed to open %q", device)
	case C.RESULT_PROBE_AMBIVALENT:
		return fmt.Errorf("found multiple filesystem types on %q", device)
	case C.RESULT_PROBE_FAILED:
		return fmt.Errorf("failed to probe %q", device)
	case C.RESULT_LOOKUP_FAILED:
		return fmt.Errorf("failed to lookup attribute on %q", device)
	case C.RESULT_NO_PARTITION_TABLE:
		return fmt.Errorf("no partition table found on %q", device)
	case C.RESULT_BAD_INDEX:
		return fmt.Errorf("bad partition index specified for device %q", device)
	case C.RESULT_GET_PARTLIST_FAILED:
		return fmt.Errorf("failed to get list of partitions on %q", device)
	case C.RESULT_DISK_HAS_NO_TYPE:
		return fmt.Errorf("%q has no type string, despite having a partition table", device)
	case C.RESULT_DISK_NOT_GPT:
		return fmt.Errorf("%q is not a gpt disk", device)
	case C.RESULT_BAD_PARAMS:
		return fmt.Errorf("internal error. bad params passed while handling %q", device)
	case C.RESULT_OVERFLOW:
		return fmt.Errorf("internal error. libblkid returned impossibly large value when handling %q", device)
	case C.RESULT_NO_TOPO:
		return fmt.Errorf("failed to get topology information for %q", device)
	case C.RESULT_NO_SECTOR_SIZE:
		return fmt.Errorf("failed to get logical sector size for %q", device)
	case C.RESULT_BAD_SECTOR_SIZE:
		return fmt.Errorf("logical sector size for %q was not a multiple of 512", device)
	default:
		return fmt.Errorf("Unknown error while handling %q. err code: %d", device, res)
	}
}

func CBufToGoStr(s [C.PART_INFO_BUF_SIZE]C.char) string {
	return C.GoString(&s[0])
}

func CBufToGoPtr(s [C.PART_INFO_BUF_SIZE]C.char) *string {
	return util.StrToPtr(CBufToGoStr(s))
}

// DumpPartitionTable returns a list of all partitions on device (e.g. /dev/vda). The list
// of partitions returned is unordered.
func DumpDisk(device string) (DiskInfo, error) {
	output := DiskInfo{}

	var cInfo C.struct_partition_info

	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	var sectorSize C.int
	if err := cResultToErr(C.blkid_get_logical_sector_size(cDevice, &sectorSize), device); err != nil {
		return DiskInfo{}, err
	}
	output.LogicalSectorSize = int(sectorSize)

	numParts := C.int(0)
	if err := cResultToErr(C.blkid_get_num_partitions(cDevice, &numParts), device); err != nil {
		return DiskInfo{}, err
	}

	for i := 0; i < int(numParts); i++ {
		if err := cResultToErr(C.blkid_get_partition(cDevice, C.int(i), &cInfo), device); err != nil {
			return DiskInfo{}, err
		}

		current := PartitionInfo{
			Label:         CBufToGoStr(cInfo.label),
			GUID:          strings.ToUpper(CBufToGoStr(cInfo.uuid)),
			TypeGUID:      strings.ToUpper(CBufToGoStr(cInfo.type_guid)),
			Number:        int(cInfo.number),
			StartSector:   int64(cInfo.start),
			SizeInSectors: int64(cInfo.size),
		}

		output.Partitions = append(output.Partitions, current)
	}
	return output, nil
}

func filesystemLookup(device string, allowAmbivalent bool, fieldName string) (string, error) {
	var buf [256]byte

	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))
	cFieldName := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cFieldName))

	if err := cResultToErr(C.blkid_lookup(cDevice, C.bool(allowAmbivalent), cFieldName, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf))), device); err != nil {
		return "", err
	}
	return string(buf[:bytes.IndexByte(buf[:], 0)]), nil
}
