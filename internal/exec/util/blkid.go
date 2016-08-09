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
	"fmt"
	"unsafe"

	"github.com/coreos/ignition/config/types"
)

func FilesystemType(device types.Path) (string, error) {
	var fsType [16]byte

	cDevice := C.CString(string(device))
	defer C.free(unsafe.Pointer(cDevice))

	switch C.filesystem_type(cDevice, (*C.char)(unsafe.Pointer(&fsType[0])), C.size_t(len(fsType))) {
	case C.RESULT_OK:
		return string(fsType[:]), nil
	case C.RESULT_OPEN_FAILED:
		return "", fmt.Errorf("failed to open %q", device)
	case C.RESULT_PROBE_FAILED:
		return "", fmt.Errorf("failed to perform probe on %q", device)
	case C.RESULT_LOOKUP_FAILED:
		return "", fmt.Errorf("failed to lookup filesystem type of %q", device)
	default:
		return "", fmt.Errorf("unknown error")
	}
}
