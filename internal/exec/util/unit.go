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
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/coreos/ignition/v2/config/v3_2_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/util"
)

const (
	PresetPath               string      = "/etc/systemd/system-preset/20-ignition.preset"
	DefaultPresetPermissions os.FileMode = 0644
)

func (ut Util) getUnitFetch(path string, contents types.Resource) (FetchOp, error) {
	u, err := url.Parse(*contents.Source)
	if err != nil {
		ut.Logger.Crit("Unable to parse systemd contents URL: %s", err)
		return FetchOp{}, err
	}
	hasher, err := util.GetHasher(contents.Verification)
	if err != nil {
		ut.Logger.Crit("Unable to get hasher: %s", err)
		return FetchOp{}, err
	}

	var expectedSum []byte
	if hasher != nil {
		// explicitly ignoring the error here because the config should already
		// be validated by this point
		_, expectedSumString, _ := util.HashParts(contents.Verification)
		expectedSum, err = hex.DecodeString(expectedSumString)
		if err != nil {
			ut.Logger.Crit("Error parsing verification string %q: %v", expectedSumString, err)
			return FetchOp{}, err
		}
	}

	var compression string
	if contents.Compression != nil {
		compression = *contents.Compression
	}

	return FetchOp{
		Hash: hasher,
		Node: types.Node{
			Path: path,
		},
		Url: *u,
		FetchOptions: resource.FetchOptions{
			Hash:        hasher,
			ExpectedSum: expectedSum,
			Compression: compression,
		},
	}, nil
}

func (ut Util) FileFromSystemdUnit(unit types.Unit, runtime bool) (FetchOp, error) {
	var path string
	var err error
	if runtime {
		path = SystemdRuntimeUnitsPath()
	} else {
		path = SystemdUnitsPath()
	}

	if path, err = ut.JoinPath(path, unit.Name); err != nil {
		return FetchOp{}, err
	}

	return ut.getUnitFetch(path, unit.Contents)
}

func (ut Util) FileFromSystemdUnitDropin(unit types.Unit, dropin types.Dropin, runtime bool) (FetchOp, error) {
	var path string
	var err error
	if runtime {
		path = SystemdRuntimeDropinsPath(string(unit.Name))
	} else {
		path = SystemdDropinsPath(string(unit.Name))
	}

	if path, err = ut.JoinPath(path, dropin.Name); err != nil {
		return FetchOp{}, err
	}

	return ut.getUnitFetch(path, unit.Contents)
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
