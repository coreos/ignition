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
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	DefaultDirectoryPermissions os.FileMode = 0755
	DefaultFilePermissions      os.FileMode = 0755
)

var (
	ErrNotDirectory = errors.New("file is not a directory")
)

func WriteFile(filename, contents string) error {
	var err error

	dir := filepath.Dir(filename)
	if err := EnsureDirectoryExists(dir); err != nil {
		return err
	}

	// Create a temporary file in the same directory to ensure it's on the same filesystem
	var tmp *os.File
	if tmp, err = ioutil.TempFile(dir, "tmp"); err != nil {
		return err
	}
	tmp.Close()

	if err := ioutil.WriteFile(tmp.Name(), []byte(contents), DefaultFilePermissions); err != nil {
		return err
	}

	// Ensure the permissions are as requested (since WriteFile can be affected by sticky bit)
	if err := os.Chmod(tmp.Name(), DefaultFilePermissions); err != nil {
		return err
	}

	if err := os.Rename(tmp.Name(), filename); err != nil {
		return err
	}

	return nil
}

func EnsureDirectoryExists(dir string) error {
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return ErrNotDirectory
		}
		return nil
	}
	return os.MkdirAll(dir, DefaultDirectoryPermissions)
}
