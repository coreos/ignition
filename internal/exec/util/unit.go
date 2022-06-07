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
	"net/url"
	"os"
	"os/exec"
	"syscall"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"

	"github.com/vincent-petithory/dataurl"
)

const (
	DefaultPresetPermissions os.FileMode = 0644
)

type UnitScope string

const (
	SystemUnit UnitScope = "system"
	UserUnit   UnitScope = "user"
	GlobalUnit UnitScope = "global"
)

func GetUnitScope(unit types.Unit) UnitScope {
	if util.NilOrEmpty(unit.Scope) {
		return SystemUnit
	}

	switch *unit.Scope {
	case "user", "system", "global":
		return UnitScope(*unit.Scope)
	default:
		panic("Error: Invalid scope defined")
	}
}

func (ut Util) FilesFromSystemdUnit(unit types.Unit) ([]FetchOp, error) {
	var fetchops []FetchOp

	if unit.Contents == nil {
		empty := ""
		unit.Contents = &empty
	}
	u, err := url.Parse(dataurl.EncodeBytes([]byte(*unit.Contents)))
	if err != nil {
		return []FetchOp{}, err
	}

	UnitPaths, err := ut.SystemdUnitPaths(unit)
	if err != nil {
		return []FetchOp{}, err
	}
	for _, path := range UnitPaths {
		fpath, err := ut.JoinPath(path, unit.Name)
		if err != nil {
			return []FetchOp{}, err
		}

		fetchops = append(fetchops, FetchOp{Node: types.Node{Path: fpath}, Url: *u})
	}

	return fetchops, nil
}

func (ut Util) FilesFromSystemdUnitDropin(unit types.Unit, dropin types.Dropin) ([]FetchOp, error) {
	var fetchops []FetchOp

	if dropin.Contents == nil {
		empty := ""
		dropin.Contents = &empty
	}

	u, err := url.Parse(dataurl.EncodeBytes([]byte(*dropin.Contents)))
	if err != nil {
		return []FetchOp{}, err
	}

	DropinsPaths, err := ut.SystemdDropinsPaths(unit)
	if err != nil {
		return []FetchOp{}, err
	}
	for _, path := range DropinsPaths {
		fpath, err := ut.JoinPath(path, dropin.Name)
		if err != nil {
			return []FetchOp{}, err
		}
		fetchops = append(fetchops, FetchOp{Node: types.Node{Path: fpath}, Url: *u})
	}

	return fetchops, nil
}

// MaskUnit writes a symlink to /dev/null to mask the specified unit
func (ut Util) MaskUnit(unit types.Unit) error {
	UnitPaths, err := ut.SystemdUnitPaths(unit)
	if err != nil {
		return err
	}
	for _, path := range UnitPaths {
		unitpath, err := ut.JoinPath(path, unit.Name)
		if err != nil {
			return err
		}

		if err := MkdirForFile(unitpath); err != nil {
			return err
		}
		if err := os.RemoveAll(unitpath); err != nil {
			return err
		}
		if err := os.Symlink("/dev/null", unitpath); err != nil {
			return err
		}
	}
	return nil
}

// UnmaskUnit deletes the symlink to /dev/null for a masked unit
func (ut Util) UnmaskUnit(unit types.Unit) error {
	UnitPaths, err := ut.SystemdUnitPaths(unit)
	if err != nil {
		return err
	}

	for _, path := range UnitPaths {
		unitpath, err := ut.JoinPath(path, unit.Name)
		if err != nil {
			return err
		}
		// Make a final check to make sure the unit is masked
		masked, err := ut.IsUnitMasked(unit)
		if err != nil {
			return err
		}
		// If masked, remove the symlink
		if masked {
			if err = os.Remove(unitpath); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsUnitMasked returns true/false if a systemd unit is masked
func (ut Util) IsUnitMasked(unit types.Unit) (bool, error) {
	UnitPaths, err := ut.SystemdUnitPaths(unit)
	if err != nil {
		return false, err
	}
	for _, path := range UnitPaths {
		unitpath, err := ut.JoinPath(path, unit.Name)
		if err != nil {
			return false, err
		}

		target, err := os.Readlink(unitpath)
		if err != nil {
			if os.IsNotExist(err) {
				// The path doesn't exist, hence the unit isn't masked
				return false, nil
			} else if e, ok := err.(*os.PathError); ok && e.Err == syscall.EINVAL {
				// The path isn't a symlink, hence the unit isn't masked
				return false, nil
			} else {
				return false, err
			}
		}

		if target != "/dev/null" {
			// The symlink doesn't point to /dev/null, hence the unit isn't masked
			return false, nil
		}
	}
	return true, nil
}

func (ut Util) EnableUnit(enabledUnit string, scope UnitScope) error {
	return ut.appendLineToPreset(fmt.Sprintf("enable %s", enabledUnit), ut.SystemdPresetPath(scope))
}

func (ut Util) DisableUnit(disabledUnit string, scope UnitScope) error {
	// We need to delete any enablement symlinks for a unit before sending it to a
	// preset directive. This will help to disable that unit completely.
	// For more information: https://github.com/coreos/fedora-coreos-tracker/issues/392
	// This is a short-term solution until the upstream systemd PR
	// (https://github.com/systemd/systemd/pull/15205) gets accepted.
	if err := ut.Logger.LogOp(
		func() error {
			args := []string{"--root", ut.DestDir, "disable", disabledUnit}
			if output, err := exec.Command(distro.SystemctlCmd(), args...).CombinedOutput(); err != nil {
				return fmt.Errorf("cannot remove symlink(s) for %s: %v: %q", disabledUnit, err, string(output))
			}
			return nil
		},
		"removing enablement symlink(s) for %q", disabledUnit,
	); err != nil {
		return err
	}
	return ut.appendLineToPreset(fmt.Sprintf("disable %s", disabledUnit), ut.SystemdPresetPath(scope))
}

func (ut Util) appendLineToPreset(data string, presetpath string) error {
	path, err := ut.JoinPath(presetpath)
	if err != nil {
		return err
	}

	if err := MkdirForFile(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, DefaultPresetPermissions)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(data + "\n")
	return err
}
