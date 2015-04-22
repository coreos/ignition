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
	"errors"
	"strings"
)

type Filesystem struct {
	Device  DevicePath
	Format  FilesystemFormat
	Options MkfsOptions
	Files   []File
}

type DevicePath string

func (d *DevicePath) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var device string
	if err := unmarshal(&device); err != nil {
		return err
	}
	*d = DevicePath(device)

	if !strings.HasPrefix(string(device), "/dev") {
		return errors.New("invalid device path")
	}
	return nil
}

type FilesystemFormat string

func (f *FilesystemFormat) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var format string
	if err := unmarshal(&format); err != nil {
		return err
	}
	*f = FilesystemFormat(format)

	switch format {
	case "ext4", "btrfs":
	default:
		return errors.New("invalid filesystem")
	}
	return nil
}

type MkfsOptions string

func (o *MkfsOptions) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var options string
	if err := unmarshal(&options); err != nil {
		return err
	}
	*o = MkfsOptions(options)

	return nil
}
