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
	"os"
	"path/filepath"

	"github.com/coreos/ignition/config"
)

const (
	presetPath = "/etc/systemd/system-preset/20-ignition.preset"
)

func FileFromUnit(unit config.Unit) *File {
	return &File{
		config.File{
			Path:     filepath.Join(SystemdUnitsPath(), string(unit.Name)),
			Contents: unit.Contents,
			Mode:     DefaultFilePermissions,
			Uid:      0,
			Gid:      0,
		},
	}
}

func FileFromUnitDropin(unit config.Unit, dropin config.UnitDropIn) *File {
	return &File{
		config.File{
			Path:     filepath.Join(SystemdDropinsPath(string(unit.Name)), string(dropin.Name)),
			Contents: dropin.Contents,
			Mode:     DefaultFilePermissions,
			Uid:      0,
			Gid:      0,
		},
	}
}

func (d *DestDir) MaskUnit(unit config.Unit) error {
	path := d.JoinPath(SystemdUnitsPath(), string(unit.Name))
	if err := mkdirForFile(path); err != nil {
		return err
	}
	return os.Symlink("/dev/null", path)
}

func (d *DestDir) EnableUnit(unit config.Unit) error {
	path := d.JoinPath(presetPath)
	if err := mkdirForFile(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0444)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("enable %s\n", unit.Name))
	return err
}
