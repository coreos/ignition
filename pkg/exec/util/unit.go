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

	"github.com/coreos/ignition/pkg/config"
)

func FileFromUnit(root string, unit config.Unit) *File {
	return &File{
		config.File{
			Path:     filepath.Join(root, SystemdUnitsPath(), string(unit.Name)),
			Contents: unit.Contents,
			Mode:     DefaultFilePermissions,
			Uid:      0,
			Gid:      0,
		},
	}
}

func FileFromUnitDropin(root string, unit config.Unit, dropin config.UnitDropIn) *File {
	return &File{
		config.File{
			Path:     filepath.Join(root, SystemdDropinsPath(string(unit.Name)), string(dropin.Name)),
			Contents: dropin.Contents,
			Mode:     DefaultFilePermissions,
			Uid:      0,
			Gid:      0,
		},
	}
}
