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
	"path/filepath"
	"syscall"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"

	"github.com/vincent-petithory/dataurl"
)

const (
	PresetPath               string      = "/etc/systemd/system-preset/20-ignition.preset"
	DefaultPresetPermissions os.FileMode = 0644
)

func (ut Util) FileFromSystemdUnit(unit types.Unit, runtime bool) (FetchOp, error) {
	if unit.Contents == nil {
		empty := ""
		unit.Contents = &empty
	}
	u, err := url.Parse(dataurl.EncodeBytes([]byte(*unit.Contents)))
	if err != nil {
		return FetchOp{}, err
	}

	var path string
	if runtime {
		path = SystemdRuntimeUnitsPath()
	} else {
		path = SystemdUnitsPath()
	}

	if path, err = ut.JoinPath(path, unit.Name); err != nil {
		return FetchOp{}, err
	}

	return FetchOp{
		Node: types.Node{
			Path: path,
		},
		Url: *u,
	}, nil
}

func (ut Util) FileFromSystemdUnitDropin(unit types.Unit, dropin types.Dropin, runtime bool) (FetchOp, error) {
	if dropin.Contents == nil {
		empty := ""
		dropin.Contents = &empty
	}
	u, err := url.Parse(dataurl.EncodeBytes([]byte(*dropin.Contents)))
	if err != nil {
		return FetchOp{}, err
	}

	var path string
	if runtime {
		path = SystemdRuntimeDropinsPath(string(unit.Name))
	} else {
		path = SystemdDropinsPath(string(unit.Name))
	}

	if path, err = ut.JoinPath(path, dropin.Name); err != nil {
		return FetchOp{}, err
	}

	return FetchOp{
		Node: types.Node{
			Path: path,
		},
		Url: *u,
	}, nil
}

// MaskUnit writes a symlink to /dev/null to mask the specified unit and returns the path of that unit
// without the sysroot prefix
func (ut Util) MaskUnit(unit types.Unit) (string, error) {
	path, err := ut.JoinPath(SystemdUnitsPath(), unit.Name)
	if err != nil {
		return "", err
	}

	if err := MkdirForFile(path); err != nil {
		return "", err
	}
	if err := os.RemoveAll(path); err != nil {
		return "", err
	}
	if err := os.Symlink("/dev/null", path); err != nil {
		return "", err
	}
	// not the same as the path above, since this lacks the sysroot prefix
	return filepath.Join("/", SystemdUnitsPath(), unit.Name), nil
}

// UnmaskUnit deletes the symlink to /dev/null for a masked unit
func (ut Util) UnmaskUnit(unit types.Unit) error {
	path, err := ut.JoinPath(SystemdUnitsPath(), unit.Name)
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
		if err = os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

// IsUnitMasked returns true/false if a systemd unit is masked
func (ut Util) IsUnitMasked(unit types.Unit) (bool, error) {
	path, err := ut.JoinPath(SystemdUnitsPath(), unit.Name)
	if err != nil {
		return false, err
	}

	target, err := os.Readlink(path)
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

	return true, nil
}

func (ut Util) EnableUnit(enabledUnit string) error {
	return ut.appendLineToPreset(fmt.Sprintf("enable %s", enabledUnit))
}

// presets link in /etc, which doesn't make sense for runtime units
// Related: https://github.com/coreos/ignition/v2/issues/588
func (ut Util) EnableRuntimeUnit(unit types.Unit, target string) error {
	// unless we're running tests locally, we want to affect /run, which will
	// be carried into the pivot, not a directory named /$DestDir/run
	if !distro.BlackboxTesting() {
		ut.DestDir = "/"
	}

	nodePath, err := ut.JoinPath(SystemdRuntimeUnitWantsPath(target), unit.Name)
	if err != nil {
		return err
	}
	targetPath, err := ut.JoinPath("/", SystemdRuntimeUnitsPath(), unit.Name)
	if err != nil {
		return err
	}

	link := types.Link{
		Node: types.Node{
			// XXX(jl): make Wants/Required a parameter
			Path: nodePath,
		},
		LinkEmbedded1: types.LinkEmbedded1{
			Target: targetPath,
		},
	}

	return ut.WriteLink(link)
}

func (ut Util) DisableUnit(disabledUnit string) error {
	return ut.appendLineToPreset(fmt.Sprintf("disable %s", disabledUnit))
}

func (ut Util) appendLineToPreset(data string) error {
	path, err := ut.JoinPath(PresetPath)
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
