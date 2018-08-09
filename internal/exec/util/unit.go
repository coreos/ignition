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

	configUtil "github.com/coreos/ignition/config/util"
	"github.com/coreos/ignition/internal/config/types"
	"github.com/coreos/ignition/internal/distro"

	"github.com/vincent-petithory/dataurl"
)

const (
	PresetPath               string      = "/etc/systemd/system-preset/20-ignition.preset"
	DefaultPresetPermissions os.FileMode = 0644
)

func FileFromSystemdUnit(unit types.Unit, runtime bool) (*FetchOp, error) {
	u, err := url.Parse(dataurl.EncodeBytes([]byte(unit.Contents)))
	if err != nil {
		return nil, err
	}

	var path string
	if runtime {
		path = SystemdRuntimeUnitsPath()
	} else {
		path = SystemdUnitsPath()
	}

	return &FetchOp{
		Path: filepath.Join(path, string(unit.Name)),
		Url:  *u,
		Mode: configUtil.IntToPtr(int(DefaultFilePermissions)),
	}, nil
}

func FileFromNetworkdUnit(unit types.Networkdunit) (*FetchOp, error) {
	u, err := url.Parse(dataurl.EncodeBytes([]byte(unit.Contents)))
	if err != nil {
		return nil, err
	}
	return &FetchOp{
		Path: filepath.Join(NetworkdUnitsPath(), string(unit.Name)),
		Url:  *u,
		Mode: configUtil.IntToPtr(int(DefaultFilePermissions)),
	}, nil
}

func FileFromSystemdUnitDropin(unit types.Unit, dropin types.SystemdDropin, runtime bool) (*FetchOp, error) {
	u, err := url.Parse(dataurl.EncodeBytes([]byte(dropin.Contents)))
	if err != nil {
		return nil, err
	}

	var path string
	if runtime {
		path = SystemdRuntimeDropinsPath(string(unit.Name))
	} else {
		path = SystemdDropinsPath(string(unit.Name))
	}

	return &FetchOp{
		Path: filepath.Join(path, string(dropin.Name)),
		Url:  *u,
		Mode: configUtil.IntToPtr(int(DefaultFilePermissions)),
	}, nil
}

func FileFromNetworkdUnitDropin(unit types.Networkdunit, dropin types.NetworkdDropin) (*FetchOp, error) {
	u, err := url.Parse(dataurl.EncodeBytes([]byte(dropin.Contents)))
	if err != nil {
		return nil, err
	}
	return &FetchOp{
		Path: filepath.Join(NetworkdDropinsPath(string(unit.Name)), string(dropin.Name)),
		Url:  *u,
		Mode: configUtil.IntToPtr(int(DefaultFilePermissions)),
	}, nil
}

func (u Util) MaskUnit(unit types.Unit) error {
	path := u.JoinPath(SystemdUnitsPath(), string(unit.Name))
	if err := MkdirForFile(path); err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return os.Symlink("/dev/null", path)
}

func (u Util) EnableUnit(unit types.Unit) error {
	return u.appendLineToPreset(fmt.Sprintf("enable %s", unit.Name))
}

// presets link in /etc, which doesn't make sense for runtime units
// Related: https://github.com/coreos/ignition/issues/588
func (u Util) EnableRuntimeUnit(unit types.Unit, target string) error {
	// unless we're running tests locally, we want to affect /run, which will
	// be carried into the pivot, not a directory named /$DestDir/run
	if !distro.BlackboxTesting() {
		u.DestDir = "/"
	}

	link := types.Link{
		Node: types.Node{
			Filesystem: "root",
			// XXX(jl): make Wants/Required a parameter
			Path: filepath.Join(SystemdRuntimeUnitWantsPath(target), string(unit.Name)),
		},
		LinkEmbedded1: types.LinkEmbedded1{
			Target: filepath.Join("/", SystemdRuntimeUnitsPath(), string(unit.Name)),
		},
	}

	return u.WriteLink(link)
}

func (u Util) DisableUnit(unit types.Unit) error {
	return u.appendLineToPreset(fmt.Sprintf("disable %s", unit.Name))
}

func (u Util) appendLineToPreset(data string) error {
	path := u.JoinPath(PresetPath)
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
