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

package prepivot

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/exec/stages"
	"github.com/coreos/ignition/src/exec/util"
	"github.com/coreos/ignition/src/log"
)

const (
	name = "prepivot"
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger *log.Logger, root string) stages.Stage {
	return &stage{util.Util{
		DestDir: root,
		Logger:  logger,
	}}
}

func (creator) Name() string {
	return name
}

type stage struct {
	util.Util
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config config.Config) bool {
	if err := s.createPasswd(config); err != nil {
		s.Logger.Crit("failed to create users/groups: %v", err)
		return false
	}

	if err := s.createFilesystemsFiles(config); err != nil {
		s.Logger.Crit("failed to create files: %v", err)
		return false
	}

	if err := s.createUnits(config); err != nil {
		s.Logger.Crit("failed to create units: %v", err)
		return false
	}
	return true
}

// createUnits creates the units listed under systemd.units and networkd.units.
func (s stage) createUnits(config config.Config) error {
	for _, unit := range config.Systemd.Units {
		if err := s.writeSystemdUnit(unit); err != nil {
			return err
		}
		if unit.Enable {
			if err := s.Logger.LogOp(
				func() error { return s.EnableUnit(unit) },
				"enabling unit %q", unit.Name,
			); err != nil {
				return err
			}
		}
		if unit.Mask {
			if err := s.Logger.LogOp(
				func() error { return s.MaskUnit(unit) },
				"masking unit %q", unit.Name,
			); err != nil {
				return err
			}
		}
	}
	for _, unit := range config.Networkd.Units {
		if err := s.writeNetworkdUnit(unit); err != nil {
			return err
		}
	}
	return nil
}

// writeSystemdUnit creates the specified unit and any dropins for that unit.
// If the contents of the unit or are empty, the unit is not created. The same
// applies to the unit's dropins.
func (s stage) writeSystemdUnit(unit config.SystemdUnit) error {
	return s.Logger.LogOp(func() error {
		for _, dropin := range unit.DropIns {
			if dropin.Contents == "" {
				continue
			}

			f := util.FileFromUnitDropin(unit, dropin)
			if err := s.Logger.LogOp(
				func() error { return s.WriteFile(f) },
				"writing dropin %q at %q", dropin.Name, f.Path,
			); err != nil {
				return err
			}
		}

		if unit.Contents == "" {
			return nil
		}

		f := util.FileFromSystemdUnit(unit)
		if err := s.Logger.LogOp(
			func() error { return s.WriteFile(f) },
			"writing unit %q at %q", unit.Name, f.Path,
		); err != nil {
			return err
		}

		return nil
	}, "writing unit %q", unit.Name)
}

// writeNetworkdUnit creates the specified unit. If the contents of the unit or
// are empty, the unit is not created.
func (s stage) writeNetworkdUnit(unit config.NetworkdUnit) error {
	return s.Logger.LogOp(func() error {
		if unit.Contents == "" {
			return nil
		}

		f := util.FileFromNetworkdUnit(unit)
		if err := s.Logger.LogOp(
			func() error { return s.WriteFile(f) },
			"writing unit %q at %q", unit.Name, f.Path,
		); err != nil {
			return err
		}

		return nil
	}, "writing unit %q", unit.Name)
}

// createPasswd creates the users and groups as described in config.Passwd.
func (s stage) createPasswd(config config.Config) error {
	if err := s.createGroups(config); err != nil {
		return fmt.Errorf("failed to create groups: %v", err)
	}

	if err := s.createUsers(config); err != nil {
		return fmt.Errorf("failed to create users: %v", err)
	}

	return nil
}

// createUsers creates the users as described in config.Passwd.Users.
func (s stage) createUsers(config config.Config) error {
	if len(config.Passwd.Users) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createUsers")
	defer s.Logger.PopPrefix()

	for _, u := range config.Passwd.Users {
		if err := s.CreateUser(u); err != nil {
			return fmt.Errorf("failed to create user %q: %v",
				u.Name, err)
		}

		if err := s.SetPasswordHash(u); err != nil {
			return fmt.Errorf("failed to set password for %q: %v",
				u.Name, err)
		}

		if err := s.AuthorizeSSHKeys(u); err != nil {
			return fmt.Errorf("failed to add keys to user %q: %v",
				u.Name, err)
		}
	}

	return nil
}

// createGroups creates the users as described in config.Passwd.Groups.
func (s stage) createGroups(config config.Config) error {
	if len(config.Passwd.Groups) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createGroups")
	defer s.Logger.PopPrefix()

	for _, g := range config.Passwd.Groups {
		if err := s.CreateGroup(g); err != nil {
			return fmt.Errorf("failed to create group %q: %v",
				g.Name, err)
		}
	}

	return nil
}

// createFilesystemsFiles creates the files described in config.Storage.Filesystems.
func (s stage) createFilesystemsFiles(config config.Config) error {
	if len(config.Storage.Filesystems) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createFilesystemsFiles")
	defer s.Logger.PopPrefix()

	for _, fs := range config.Storage.Filesystems {
		if err := s.createFiles(fs); err != nil {
			return fmt.Errorf("failed to create files %q: %v", fs.Device, err)
		}
	}

	return nil
}

// createFiles creates any files listed for the filesystem in fs.Files.
func (s stage) createFiles(fs config.Filesystem) error {
	if len(fs.Files) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createFiles")
	defer s.Logger.PopPrefix()

	mnt, err := ioutil.TempDir("", "ignition-files")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.Remove(mnt)

	dev := string(fs.Device)
	format := string(fs.Format)

	if err := s.Logger.LogOp(
		func() error { return syscall.Mount(dev, mnt, format, 0, "") },
		"mounting %q at %q", dev, mnt,
	); err != nil {
		return fmt.Errorf("failed to mount device %q at %q: %v", dev, mnt, err)
	}
	defer s.Logger.LogOp(
		func() error { return syscall.Unmount(mnt, 0) },
		"unmounting %q at %q", dev, mnt,
	)

	u := util.Util{
		Logger:  s.Logger,
		DestDir: mnt,
	}
	for _, f := range fs.Files {
		if err := s.Logger.LogOp(
			func() error { return u.WriteFile(&f) },
			"writing file %q", string(f.Path),
		); err != nil {
			return fmt.Errorf("failed to create file %q: %v", f.Path, err)
		}
	}

	return nil
}
