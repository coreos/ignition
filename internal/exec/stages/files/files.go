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

package files

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/exec/stages"
	eutil "github.com/coreos/ignition/internal/exec/util"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/util"
)

const (
	name = "files"
)

var (
	ErrFilesystemUndefined = errors.New("the referenced filesystem was not defined")
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger *log.Logger, client *util.HttpClient, root string) stages.Stage {
	return &stage{
		Util: eutil.Util{
			DestDir: root,
			Logger:  logger,
		},
		client: client,
	}
}

func (creator) Name() string {
	return name
}

type stage struct {
	eutil.Util

	client *util.HttpClient
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config types.Config) bool {
	if err := s.createPasswd(config); err != nil {
		s.Logger.Crit("failed to create users/groups: %v", err)
		return false
	}

	if err := s.createFilesystemsEntries(config); err != nil {
		s.Logger.Crit("failed to create files: %v", err)
		return false
	}

	if err := s.createUnits(config); err != nil {
		s.Logger.Crit("failed to create units: %v", err)
		return false
	}

	return true
}

// createFilesystemsEntries creates the files described in config.Storage.{Files,Directories}.
func (s stage) createFilesystemsEntries(config types.Config) error {
	if len(config.Storage.Filesystems) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createFilesystemsFiles")
	defer s.Logger.PopPrefix()

	entryMap, err := s.mapEntriesToFilesystems(config)
	if err != nil {
		return err
	}

	for fs, f := range entryMap {
		if err := s.createEntries(fs, f); err != nil {
			return fmt.Errorf("failed to create files: %v", err)
		}
	}

	return nil
}

// filesystemEntry represent a thing that knows how to create itself.
type filesystemEntry interface {
	create(l *log.Logger, c *util.HttpClient, u eutil.Util) error
}

type fileEntry types.File

func (tmp fileEntry) create(l *log.Logger, c *util.HttpClient, u eutil.Util) error {
	f := types.File(tmp)
	file := eutil.RenderFile(l, c, f)
	if file == nil {
		return fmt.Errorf("failed to resolve file %q", f.Path)
	}

	if err := l.LogOp(
		func() error { return u.WriteFile(file) },
		"writing file %q", string(f.Path),
	); err != nil {
		return fmt.Errorf("failed to create file %q: %v", file.Path, err)
	}

	return nil
}

type dirEntry types.Directory

func (tmp dirEntry) create(l *log.Logger, _ *util.HttpClient, u eutil.Util) error {
	d := types.Directory(tmp)
	err := l.LogOp(func() error {
		path := filepath.Clean(u.JoinPath(string(d.Path)))

		// Build a list of paths to create. Since os.MkdirAll only sets the mode for new directories and not the
		// ownership, we need to determine which directories will be created so we don't chown something that already
		// exists.
		newPaths := []string{}
		for p := path; p != "/"; p = filepath.Dir(p) {
			_, err := os.Stat(p)
			if err == nil {
				break
			}
			if !os.IsNotExist(err) {
				return err
			}
			newPaths = append(newPaths, p)
		}

		if err := os.MkdirAll(path, os.FileMode(d.Mode)); err != nil {
			return err
		}

		for _, newPath := range newPaths {
			if err := os.Chmod(newPath, os.FileMode(d.Mode)); err != nil {
				return err
			}
			if err := os.Chown(newPath, d.User.Id, d.Group.Id); err != nil {
				return err
			}
		}
		return nil
	}, "creating directory %q", string(d.Path))
	if err != nil {
		return fmt.Errorf("failed to create directory %q: %v", d.Path, err)
	}

	return nil
}

// ByDirectorySegments is used to sort directories so /foo gets created before /foo/bar if they are both specified.
type ByDirectorySegments []types.Directory

func (lst ByDirectorySegments) Len() int { return len(lst) }

func (lst ByDirectorySegments) Swap(i, j int) {
	lst[i], lst[j] = lst[j], lst[i]
}

func (lst ByDirectorySegments) Less(i, j int) bool {
	return lst[i].Depth() < lst[j].Depth()
}

// mapEntriesToFilesystems builds a map of filesystems to files. If multiple
// definitions of the same filesystem are present, only the final definition is
// used. The directories are sorted to ensure /foo gets created before /foo/bar.
func (s stage) mapEntriesToFilesystems(config types.Config) (map[types.Filesystem][]filesystemEntry, error) {
	filesystems := map[string]types.Filesystem{}
	for _, fs := range config.Storage.Filesystems {
		filesystems[fs.Name] = fs
	}

	entryMap := map[types.Filesystem][]filesystemEntry{}

	// Sort directories to ensure /a gets created before /a/b.
	sortedDirs := config.Storage.Directories
	sort.Sort(ByDirectorySegments(sortedDirs))

	// Add directories first to ensure they are created before files.
	for _, d := range sortedDirs {
		if fs, ok := filesystems[d.Filesystem]; ok {
			entryMap[fs] = append(entryMap[fs], dirEntry(d))
		} else {
			s.Logger.Crit("the filesystem (%q), was not defined", d.Filesystem)
			return nil, ErrFilesystemUndefined
		}
	}

	for _, f := range config.Storage.Files {
		if fs, ok := filesystems[f.Filesystem]; ok {
			entryMap[fs] = append(entryMap[fs], fileEntry(f))
		} else {
			s.Logger.Crit("the filesystem (%q), was not defined", f.Filesystem)
			return nil, ErrFilesystemUndefined
		}
	}

	return entryMap, nil
}

// createEntries creates any files or directories listed for the filesystem in Storage.{Files,Directories}.
func (s stage) createEntries(fs types.Filesystem, files []filesystemEntry) error {
	s.Logger.PushPrefix("createFiles")
	defer s.Logger.PopPrefix()

	var mnt string
	if fs.Path == nil {
		var err error
		mnt, err = ioutil.TempDir("", "ignition-files")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %v", err)
		}
		defer os.Remove(mnt)

		dev := string(fs.Mount.Device)
		format := string(fs.Mount.Format)

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
	} else {
		mnt = string(*fs.Path)
	}

	u := eutil.Util{
		Logger:  s.Logger,
		DestDir: mnt,
	}

	for _, e := range files {
		if err := e.create(s.Logger, s.client, u); err != nil {
			return err
		}
	}

	return nil
}

// createUnits creates the units listed under systemd.units and networkd.units.
func (s stage) createUnits(config types.Config) error {
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
func (s stage) writeSystemdUnit(unit types.SystemdUnit) error {
	return s.Logger.LogOp(func() error {
		for _, dropin := range unit.DropIns {
			if dropin.Contents == "" {
				continue
			}

			f := eutil.FileFromUnitDropin(unit, dropin)
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

		f := eutil.FileFromSystemdUnit(unit)
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
func (s stage) writeNetworkdUnit(unit types.NetworkdUnit) error {
	return s.Logger.LogOp(func() error {
		if unit.Contents == "" {
			return nil
		}

		f := eutil.FileFromNetworkdUnit(unit)
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
func (s stage) createPasswd(config types.Config) error {
	if err := s.createGroups(config); err != nil {
		return fmt.Errorf("failed to create groups: %v", err)
	}

	if err := s.createUsers(config); err != nil {
		return fmt.Errorf("failed to create users: %v", err)
	}

	return nil
}

// createUsers creates the users as described in config.Passwd.Users.
func (s stage) createUsers(config types.Config) error {
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
func (s stage) createGroups(config types.Config) error {
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
