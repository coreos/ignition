// Copyright 2021 Red Hat, Inc.
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

// The virtualbox provider fetches the configuration from the /Ignition/Config
// guest property.

package virtualbox

// We want at least this warning, since the default C behavior of
// assuming int foo(int) is totally broken.

// #cgo CFLAGS: -Werror=implicit-function-declaration
// #include <linux/vbox_err.h>
// #include <stdlib.h>
// #include "virtualbox.h"
import "C"

import (
	"bytes"
	"fmt"
	"unsafe"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

const (
	configProperty         = "/Ignition/Config"
	configEncodingProperty = "/Ignition/Config/Encoding"
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	f.Logger.Debug("reading Ignition config from VirtualBox guest property")

	// for forward compatibility, check an encoding property analogous
	// to vmware's ignition.config.data.encoding, and fail if it's
	// present and non-empty
	encoding, err := fetchProperty(configEncodingProperty)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}
	if len(encoding) > 0 {
		return types.Config{}, report.Report{}, fmt.Errorf("unsupported %q value %q", configEncodingProperty, encoding)
	}

	config, err := fetchProperty(configProperty)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}
	if config == nil {
		f.Logger.Info("VirtualBox guest property %q does not exist; assuming no config", configProperty)
		return types.Config{}, report.Report{}, errors.ErrEmpty
	}
	return util.ParseConfig(f.Logger, config)
}

func DelConfig(f *resource.Fetcher) error {
	f.Logger.Info("deleting Ignition config from VirtualBox guest property")
	err := deleteProperty(configEncodingProperty)
	if err != nil {
		return err
	}
	err = deleteProperty(configProperty)
	if err != nil {
		return err
	}
	return nil
}

func fetchProperty(name string) ([]byte, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	buf := unsafe.Pointer(nil)
	defer C.free(buf)
	var size C.size_t

	ret, errno := C.virtualbox_get_guest_property(cName, &buf, &size)
	if ret != C.VINF_SUCCESS {
		if ret == C.VERR_GENERAL_FAILURE && errno != nil {
			return nil, fmt.Errorf("fetching VirtualBox guest property %q: %w", name, errno)
		}
		// see <linux/vbox_err.h>
		return nil, fmt.Errorf("fetching VirtualBox guest property %q: error %d", name, ret)
	}
	if buf == nil {
		return nil, nil
	}
	// property format: <data> NUL <flags> NUL
	// return only the data; ignore the flags
	s := C.GoBytes(buf, C.int(size))
	len := bytes.IndexByte(s, 0)
	if len == -1 {
		return nil, fmt.Errorf("VirtualBox guest property %q is not NUL-terminated", name)
	}
	return s[0:len], nil
}

func deleteProperty(name string) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	ret, errno := C.virtualbox_delete_guest_property(cName)
	if ret != C.VINF_SUCCESS {
		if ret == C.VERR_GENERAL_FAILURE && errno != nil {
			return fmt.Errorf("deleting VirtualBox guest property %q: %w", name, errno)
		}
		if ret == C.VERR_PERMISSION_DENIED {
			return fmt.Errorf("deleting VirtualBox guest property %q: permission denied; is the property read-only?", name)
		}
		// see <linux/vbox_err.h>
		return fmt.Errorf("deleting VirtualBox guest property %q: error %d", name, ret)
	}
	return nil
}
