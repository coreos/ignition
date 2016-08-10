// Copyright 2016 CoreOS, Inc.
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

package types

import (
	"errors"
)

var (
	ErrFilesystemInvalidFormat = errors.New("invalid filesystem format")
	ErrFilesystemNoMountPath   = errors.New("filesystem is missing mount or path")
	ErrFilesystemMountAndPath  = errors.New("filesystem has both mount and path defined")
)

type Filesystem struct {
	Name  string           `json:"name,omitempty"`
	Mount *FilesystemMount `json:"mount,omitempty"`
	Path  *Path            `json:"path,omitempty"`
}

type FilesystemMount struct {
	Device Path              `json:"device,omitempty"`
	Format FilesystemFormat  `json:"format,omitempty"`
	Create *FilesystemCreate `json:"create,omitempty"`
}

type FilesystemCreate struct {
	Force   bool        `json:"force,omitempty"`
	Options MkfsOptions `json:"options,omitempty"`
}

func (f Filesystem) AssertValid() error {
	if f.Mount == nil && f.Path == nil {
		return ErrFilesystemNoMountPath
	}
	if f.Mount != nil && f.Path != nil {
		return ErrFilesystemMountAndPath
	}
	return nil
}

type FilesystemFormat string

func (f FilesystemFormat) AssertValid() error {
	switch f {
	case "ext4", "btrfs", "xfs":
		return nil
	default:
		return ErrFilesystemInvalidFormat
	}
}

type MkfsOptions []string
