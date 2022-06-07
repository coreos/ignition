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
	"path/filepath"

	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
)

const SystemPresetPath = "etc/systemd/system-preset/20-ignition.preset"
const GlobalPresetPath = "etc/systemd/user-preset/20-ignition.preset"
const UserPresetPath = "etc/systemd/user-preset/21-ignition.preset"

const SystemUnitPath = "etc/systemd/system"
const GlobalUnitPath = "etc/systemd/user"
const UserUnitPath = ".config/systemd/user"

func (u Util) SystemdUnitPaths(unit types.Unit) ([]string, error) {
	var paths []string
	switch GetUnitScope(unit) {
	case UserUnit:
		for _, user := range unit.Users {
			home, err := u.GetUserHomeDirByName(string(user))
			if err != nil {
				return nil, err
			}
			paths = append(paths, filepath.Join(home, UserUnitPath))
		}
	case SystemUnit:
		paths = append(paths, SystemUnitPath)
	case GlobalUnit:
		paths = append(paths, GlobalUnitPath)
	default:
		paths = append(paths, SystemUnitPath)
	}
	return paths, nil
}

func (u Util) SystemdPresetPath(scope UnitScope) string {
	switch scope {
	case UserUnit:
		return UserPresetPath
	case SystemUnit:
		return SystemPresetPath
	case GlobalUnit:
		return GlobalPresetPath
	default:
		return SystemPresetPath
	}
}

func (u Util) SystemdDropinsPaths(unit types.Unit) ([]string, error) {
	var paths []string
	unitpaths, err := u.SystemdUnitPaths(unit)
	if err != nil {
		return nil, err
	}
	for _, path := range unitpaths {
		paths = append(paths, filepath.Join(path, unit.Name+".d"))
	}
	return paths, err
}
