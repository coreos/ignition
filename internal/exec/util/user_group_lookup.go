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

//go:build linux
// +build linux

package util

// See blkid.go for compiler warning comment.

// #cgo CFLAGS: -Werror=implicit-function-declaration
// #include "user_group_lookup.h"
import "C"

import (
	"fmt"
	"github.com/coreos/ignition/v2/internal/distro"
	"os/user"
)

// userLookup looks up the user in u.DestDir.
func (u Util) userLookup(name string) (*user.User, error) {
	if distro.UserGroupLookupUsingGo() {
		return lookupUser(name, u.DestDir)
	}
	res := &C.lookup_res_t{}

	if ret, err := C.user_lookup(C.CString(u.DestDir),
		C.CString(name), res); ret < 0 {
		return nil, fmt.Errorf("lookup failed: %v", err)
	}

	if res.name == nil {
		return nil, user.UnknownUserError(fmt.Sprintf("user %q not found", name))
	}

	usr := &user.User{
		Name:    C.GoString(res.name),
		Uid:     fmt.Sprintf("%d", int(res.uid)),
		Gid:     fmt.Sprintf("%d", int(res.gid)),
		HomeDir: C.GoString(res.home),
	}

	C.user_lookup_res_free(res)

	return usr, nil
}

// groupLookup looks up the group in u.DestDir.
func (u Util) groupLookup(name string) (*user.Group, error) {
	if distro.UserGroupLookupUsingGo() {
		return lookupGroup(name, u.DestDir)
	}
	res := &C.lookup_res_t{}

	if ret, err := C.group_lookup(C.CString(u.DestDir),
		C.CString(name), res); ret < 0 {
		return nil, fmt.Errorf("lookup failed: %v", err)
	}

	if res.name == nil {
		return nil, user.UnknownGroupError(fmt.Sprintf("group %q not found", name))
	}

	grp := &user.Group{
		Name: C.GoString(res.name),
		Gid:  fmt.Sprintf("%d", int(res.gid)),
	}

	C.group_lookup_res_free(res)

	return grp, nil
}
