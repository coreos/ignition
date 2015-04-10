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

package config

import (
	"fmt"
	"os"
)

type FileMode os.FileMode

type File struct {
	Path     string
	Contents string
	Mode     FileMode
	// FIXME(vc) make these strings and add resolution to WriteFile
	Uid int
	Gid int
}

func (m *FileMode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var mode int
	if err := unmarshal(&mode); err != nil {
		return err
	}
	if (mode &^ 07777) != 0 {
		return fmt.Errorf("illegal file mode %#o", mode)
	}

	*m = FileMode(mode)
	return nil
}
