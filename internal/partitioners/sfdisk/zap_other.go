// Copyright 2024 Red Hat, Inc.
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

//go:build !linux

package sfdisk

import "os"

// diskGeometry falls back to the file size with a 512-byte sector on
// non-Linux platforms (image files under test); the production target
// is the Linux ioctl implementation.
func diskGeometry(f *os.File) (int64, int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, 0, err
	}
	return fi.Size(), 512, nil
}

// rereadPartitionTable is a no-op off Linux (no block layer).
func rereadPartitionTable(f *os.File) {}
