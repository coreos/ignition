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
	"errors"
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

// cResultToErr takes a result_t from the blkid c code and returns a golang
// error describing the result code.
func cResultToErr(res C.result_t) error {
	switch res {
	case C.RESULT_OK:
		return nil
	case C.RESULT_OPEN_FAILED:
		return errors.New("failed to open")
	case C.RESULT_PROBE_AMBIVALENT:
		return errors.New("found multiple filesystem types")
	case C.RESULT_PROBE_FAILED:
		return errors.New("failed to probe")
	case C.RESULT_LOOKUP_FAILED:
		return errors.New("failed to look up attribute")
	case C.RESULT_NO_PARTITION_TABLE:
		return errors.New("no partition table found")
	case C.RESULT_BAD_INDEX:
		return errors.New("bad partition index specified")
	case C.RESULT_GET_PARTLIST_FAILED:
		return errors.New("failed to get list of partitions")
	case C.RESULT_GET_CACHE_FAILED:
		return fmt.Errorf("failed to retrieve cache")
	case C.RESULT_DISK_HAS_NO_TYPE:
		return errors.New("disk has no type string, despite having a partition table")
	case C.RESULT_DISK_NOT_GPT:
		return errors.New("disk does not have a GPT")
	case C.RESULT_BAD_PARAMS:
		return errors.New("bad parameters passed")
	case C.RESULT_OVERFLOW:
		return errors.New("return value doesn't fit in buffer")
	case C.RESULT_MAX_BLOCK_DEVICES:
		return fmt.Errorf("found too many filesystems of the specified type")
	case C.RESULT_NO_TOPO:
		return errors.New("failed to get topology information")
	case C.RESULT_NO_SECTOR_SIZE:
		return errors.New("failed to get logical sector size")
	case C.RESULT_BAD_SECTOR_SIZE:
		return errors.New("logical sector size is not a multiple of 512")
	default:
		return fmt.Errorf("unknown error: %d", res)
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
	if err := cResultToErr(C.blkid_get_logical_sector_size(cDevice, &sectorSize)); err != nil {
		return DiskInfo{}, fmt.Errorf("getting sector size of %q: %w", device, err)
	}
	output.LogicalSectorSize = int(sectorSize)

	numParts := C.int(0)
	if err := cResultToErr(C.blkid_get_num_partitions(cDevice, &numParts)); err != nil {
		return DiskInfo{}, fmt.Errorf("getting partition count of %q: %w", device, err)
	}

	for i := 0; i < int(numParts); i++ {
		if err := cResultToErr(C.blkid_get_partition(cDevice, C.int(i), &cInfo)); err != nil {
			return DiskInfo{}, fmt.Errorf("querying partition %d of %q: %w", i, device, err)
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

	if err := cResultToErr(C.blkid_lookup(cDevice, C.bool(allowAmbivalent), cFieldName, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))); err != nil {
		return "", fmt.Errorf("querying filesystem field %q of %q: %w", fieldName, device, err)
	}
	return string(buf[:bytes.IndexByte(buf[:], 0)]), nil
}

// GetBlockDevices returns a slice of block devices with the given filesystem
func GetBlockDevices(fstype string) ([]string, error) {
	var dev C.struct_block_device_list
	res := C.blkid_get_block_devices(C.CString(fstype), &dev)

	if res != C.RESULT_OK {
		return nil, cResultToErr(res)
	}

	length := int(dev.count)
	blkDeviceList := make([]string, length)
	for i := 0; i < length; i++ {
		blkDeviceList[i] = C.GoString(&dev.path[i][0])
	}
	return blkDeviceList, nil
}
