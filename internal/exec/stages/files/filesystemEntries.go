// Copyright 2018 CoreOS, Inc.
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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	configUtil "github.com/coreos/ignition/config/util"
	"github.com/coreos/ignition/internal/config/types"
	"github.com/coreos/ignition/internal/exec/util"
	"github.com/coreos/ignition/internal/log"
)

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
	create(l *log.Logger, u util.Util) error
}

type fileEntry types.File

func (tmp fileEntry) create(l *log.Logger, u util.Util) error {
	f := types.File(tmp)

	fetchOp := u.PrepareFetch(l, f)
	if fetchOp == nil {
		return fmt.Errorf("failed to resolve file %q", f.Path)
	}

	msg := "writing file %q"
	if f.Append {
		msg = "appending to file %q"
	}

	if err := l.LogOp(
		func() error {
			err := u.DeletePathOnOverwrite(f.Node)
			if err != nil {
				return err
			}

			return u.PerformFetch(fetchOp)
		}, msg, string(f.Path),
	); err != nil {
		return fmt.Errorf("failed to create file %q: %v", fetchOp.Path, err)
	}

	return nil
}

type dirEntry types.Directory

func (tmp dirEntry) create(l *log.Logger, u util.Util) error {
	d := types.Directory(tmp)

	err := l.LogOp(func() error {
		path := filepath.Clean(u.JoinPath(string(d.Path)))

		err := u.DeletePathOnOverwrite(d.Node)
		if err != nil {
			return err
		}

		uid, gid, err := u.ResolveNodeUidAndGid(d.Node, 0, 0)
		if err != nil {
			return err
		}

		// Build a list of paths to create. Since os.MkdirAll only sets the mode for new directories and not the
		// ownership, we need to determine which directories will be created so we don't chown something that already
		// exists.
		newPaths := []string{path}
		for p := filepath.Dir(path); p != "/"; p = filepath.Dir(p) {
			_, err := os.Stat(p)
			if err == nil {
				break
			}
			if !os.IsNotExist(err) {
				return err
			}
			newPaths = append(newPaths, p)
		}

		if d.Mode == nil {
			d.Mode = configUtil.IntToPtr(0)
		}

		if err := os.MkdirAll(path, os.FileMode(*d.Mode)); err != nil {
			return err
		}

		for _, newPath := range newPaths {
			if err := os.Chmod(newPath, os.FileMode(*d.Mode)); err != nil {
				return err
			}
			if err := os.Chown(newPath, uid, gid); err != nil {
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

type linkEntry types.Link

func (tmp linkEntry) create(l *log.Logger, u util.Util) error {
	s := types.Link(tmp)

	if err := l.LogOp(
		func() error {
			err := u.DeletePathOnOverwrite(s.Node)
			if err != nil {
				return err
			}

			return u.WriteLink(s)
		}, "writing link %q -> %q", s.Path, s.Target,
	); err != nil {
		return fmt.Errorf("failed to create link %q: %v", s.Path, err)
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
	return depth(lst[i].Node) < depth(lst[j].Node)
}

func depth(n types.Node) uint {
	var count uint = 0
	for p := filepath.Clean(string(n.Path)); p != "/"; count++ {
		p = filepath.Dir(p)
	}
	return count
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

	for _, sy := range config.Storage.Links {
		if fs, ok := filesystems[sy.Filesystem]; ok {
			entryMap[fs] = append(entryMap[fs], linkEntry(sy))
		} else {
			s.Logger.Crit("the filesystem (%q), was not defined", sy.Filesystem)
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
		mnt = *fs.Path
	}

	u := util.Util{
		DestDir: mnt,
		Fetcher: s.Util.Fetcher,
		Logger:  s.Logger,
	}

	for _, e := range files {
		if err := e.create(s.Logger, u); err != nil {
			return err
		}
	}

	return nil
}
