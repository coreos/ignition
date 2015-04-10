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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/coreos/ignition/pkg/config"
)

const (
	DefaultDirectoryPermissions config.FileMode = 0755
	DefaultFilePermissions      config.FileMode = 0644
)

// in-memory representation of a file
type File struct {
	config.File
}

func WriteFile(f *File) error {
	var err error

	dir := filepath.Dir(f.Path)
	if err := os.MkdirAll(dir, os.FileMode(DefaultDirectoryPermissions)); err != nil {
		return err
	}

	// Create a temporary file in the same directory to ensure it's on the same filesystem
	var tmp *os.File
	if tmp, err = ioutil.TempFile(dir, "tmp"); err != nil {
		return err
	}
	tmp.Close()
	defer func() {
		if err != nil {
			os.Remove(tmp.Name())
		}
	}()

	if err := ioutil.WriteFile(tmp.Name(), []byte(f.Contents), os.FileMode(f.Mode)); err != nil {
		return err
	}

	// XXX(vc): Note that we assume to be operating on the file we just wrote, this is only guaranteed
	// by using syscall.Fchown() and syscall.Fchmod()

	// Ensure the ownership and mode are as requested (since WriteFile can be affected by sticky bit)
	if err := os.Chown(tmp.Name(), f.Uid, f.Gid); err != nil {
		return err
	}

	if err := os.Chmod(tmp.Name(), os.FileMode(f.Mode)); err != nil {
		return err
	}

	if err := os.Rename(tmp.Name(), f.Path); err != nil {
		return err
	}

	return nil
}
