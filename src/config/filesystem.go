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
	"encoding/json"
	"errors"
)

var (
	ErrFilesystemRelativePath  = errors.New("device path not absolute")
	ErrFilesystemInvalidFormat = errors.New("invalid filesystem format")
)

type Filesystem struct {
	Device  DevicePath       `json:"device,omitempty"  yaml:"device"`
	Format  FilesystemFormat `json:"format,omitempty"  yaml:"format"`
	Options MkfsOptions      `json:"options,omitempty" yaml:"options"`
	Files   []File           `json:"files,omitempty"   yaml:"files"`
}

type FilesystemFormat string

func (f *FilesystemFormat) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return f.unmarshal(unmarshal)
}

func (f *FilesystemFormat) UnmarshalJSON(data []byte) error {
	return f.unmarshal(func(tf interface{}) error {
		return json.Unmarshal(data, tf)
	})
}

type filesystemFormat FilesystemFormat

func (f *FilesystemFormat) unmarshal(unmarshal func(interface{}) error) error {
	tf := filesystemFormat(*f)
	if err := unmarshal(&tf); err != nil {
		return err
	}
	*f = FilesystemFormat(tf)
	return f.assertValid()
}

func (f FilesystemFormat) assertValid() error {
	switch f {
	case "ext4", "btrfs":
		return nil
	default:
		return ErrFilesystemInvalidFormat
	}
}

type MkfsOptions []string

func (o *MkfsOptions) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return o.unmarshal(unmarshal)
}

func (o *MkfsOptions) UnmarshalJSON(data []byte) error {
	return o.unmarshal(func(to interface{}) error {
		return json.Unmarshal(data, to)
	})
}

type mkfsOptions MkfsOptions

func (o *MkfsOptions) unmarshal(unmarshal func(interface{}) error) error {
	to := mkfsOptions(*o)
	if err := unmarshal(&to); err != nil {
		return err
	}
	*o = MkfsOptions(to)
	return o.assertValid()
}

func (o MkfsOptions) assertValid() error {
	return nil
}
