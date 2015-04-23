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
	"path/filepath"
)

type Filesystem struct {
	Device  DevicePath
	Format  FilesystemFormat
	Options MkfsOptions
	Files   []File
}

type DevicePath string

func (d *DevicePath) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return d.unmarshal(unmarshal)
}

func (d *DevicePath) UnmarshalJSON(data []byte) error {
	return d.unmarshal(func(td interface{}) error {
		return json.Unmarshal(data, td)
	})
}

type devicePath DevicePath

func (d *DevicePath) unmarshal(unmarshal func(interface{}) error) error {
	td := devicePath(*d)
	if err := unmarshal(&td); err != nil {
		return err
	}
	nd := DevicePath(td)
	if err := nd.assertValid(); err != nil {
		return err
	}
	*d = nd
	return nil
}

func (d DevicePath) assertValid() error {
	if !filepath.IsAbs(string(d)) {
		return errors.New("device path not absolute")
	}
	return nil
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
	nf := FilesystemFormat(tf)
	if err := nf.assertValid(); err != nil {
		return err
	}
	*f = nf
	return nil
}

func (f FilesystemFormat) assertValid() error {
	switch f {
	case "ext4", "btrfs":
		return nil
	default:
		return errors.New("invalid filesystem")
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
	no := MkfsOptions(to)
	if err := no.assertValid(); err != nil {
		return err
	}
	*o = no
	return nil
}

func (o MkfsOptions) assertValid() error {
	return nil
}
