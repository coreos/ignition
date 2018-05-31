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

// #cgo LDFLAGS: -lblkid
// #include <stdlib.h>
// #include "blkid.h"
import "C"

import (
	"bytes"
	"fmt"
	"strings"
	"unsafe"

	"github.com/coreos/ignition/config/util"
	"github.com/coreos/ignition/internal/config/types"
)

const (
	field_name_type  = "TYPE"
	field_name_uuid  = "UUID"
	field_name_label = "LABEL"
)

func FilesystemType(device string) (string, error) {
	return filesystemLookup(device, field_name_type)
}

func FilesystemUUID(device string) (string, error) {
	return filesystemLookup(device, field_name_uuid)
}

func FilesystemLabel(device string) (string, error) {
	return filesystemLookup(device, field_name_label)
}

// cResultToErr takes a result_t from the blkid c code and a device it was operating on
// and returns a golang error describing the result code.
func cResultToErr(res C.result_t, device string) error {
	switch res {
	case C.RESULT_OK:
		return nil
	case C.RESULT_OPEN_FAILED:
		return fmt.Errorf("failed to open %q", device)
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
	default:
		return fmt.Errorf("Unknown error while handling %q. err code: %d", device, res)
	}
}

func CBufToGoStr(s [C.PART_INFO_BUF_SIZE]C.char) string {
	return C.GoString(&s[0])
}

func CBufToGoPtr(s [C.PART_INFO_BUF_SIZE]C.char) *string {
	return util.StrToPtrStrict(CBufToGoStr(s))
}

// DumpPartitionTable returns a list of all partitions on device (e.g. /dev/vda). The list
// of partitions returned is unordered.
func DumpPartitionTable(device string) ([]types.Partition, error) {
	output := []types.Partition{}

	var cInfo C.struct_partition_info
	cInfoRef := (*C.struct_partition_info)(unsafe.Pointer(&cInfo))

	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	numParts := 0
	cNumPartsRef := (*C.int)(unsafe.Pointer(&numParts))
	if err := cResultToErr(C.blkid_get_num_partitions(cDevice, cNumPartsRef), device); err != nil {
		return []types.Partition{}, err
	}

	for i := 0; i < numParts; i++ {
		if err := cResultToErr(C.blkid_get_partition(cDevice, C.int(i), cInfoRef), device); err != nil {
			return []types.Partition{}, err
		}
		current := types.Partition{
			Label:    CBufToGoPtr(cInfo.label),
			GUID:     strings.ToUpper(CBufToGoStr(cInfo.uuid)),
			TypeGUID: strings.ToUpper(CBufToGoStr(cInfo.type_guid)),
			Number:   int(cInfo.number),
			Start:    util.IntToPtr(int(cInfo.start)),
			Size:     util.IntToPtr(int(cInfo.size)),
		}

		output = append(output, current)
	}
	return output, nil
}

func filesystemLookup(device string, fieldName string) (string, error) {
	var buf [256]byte

	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))
	cFieldName := C.CString(fieldName)
	defer C.free(unsafe.Pointer(cFieldName))

	if err := cResultToErr(C.blkid_lookup(cDevice, cFieldName, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf))), device); err != nil {
		return "", err
	}
	return string(buf[:bytes.IndexByte(buf[:], 0)]), nil
}
