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
	"os"
	"path/filepath"
	"sort"

	configUtil "github.com/coreos/ignition/config/util"
	"github.com/coreos/ignition/internal/config/types"
	"github.com/coreos/ignition/internal/exec/util"
	"github.com/coreos/ignition/internal/log"
)

// createFilesystemsEntries creates the files described in config.Storage.{Files,Directories}.
func (s *stage) createFilesystemsEntries(config types.Config) error {
	s.Logger.PushPrefix("createFilesystemsFiles")
	defer s.Logger.PopPrefix()

	entryMap := s.getOrderedCreationList(config)

	if err := s.createEntries(entryMap); err != nil {
		return fmt.Errorf("failed to create files: %v", err)
	}

	return nil
}

// filesystemEntry represent a thing that knows how to create itself.
type filesystemEntry interface {
	create(l *log.Logger, u util.Util) error
	getPath() string
}

type fileEntry types.File

func (tmp fileEntry) getPath() string {
	return types.File(tmp).Path
}

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

func (tmp dirEntry) getPath() string {
	return types.Directory(tmp).Path
}

func (tmp dirEntry) create(l *log.Logger, u util.Util) error {
	d := types.Directory(tmp)

	err := l.LogOp(func() error {
		path, err := u.JoinPath(string(d.Path))
		if err != nil {
			return err
		}

		if err := u.DeletePathOnOverwrite(d.Node); err != nil {
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

func (tmp linkEntry) getPath() string {
	return types.Link(tmp).Path
}

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

// ByPathSegments is used to sort directories so /foo gets created before /foo/bar if they are both specified.
type ByPathSegments []filesystemEntry

func (lst ByPathSegments) Len() int { return len(lst) }

func (lst ByPathSegments) Swap(i, j int) {
	lst[i], lst[j] = lst[j], lst[i]
}

func (lst ByPathSegments) Less(i, j int) bool {
	return depth(lst[i].getPath()) < depth(lst[j].getPath())
}

func depth(path string) uint {
	var count uint = 0
	for p := filepath.Clean(path); p != "/"; count++ {
		p = filepath.Dir(p)
	}
	return count
}

// mapEntriesToFilesystems builds a map of filesystems to files. If multiple
// definitions of the same filesystem are present, only the final definition is
// used. The directories are sorted to ensure /foo gets created before /foo/bar.
func (s stage) getOrderedCreationList(config types.Config) []filesystemEntry {
	entries := []filesystemEntry{}

	// Add directories first to ensure they are created before files.
	for _, d := range config.Storage.Directories {
		entries = append(entries, dirEntry(d))
	}

	for _, f := range config.Storage.Files {
		entries = append(entries, fileEntry(f))
	}

	for _, sy := range config.Storage.Links {
		entries = append(entries, linkEntry(sy))
	}
	sort.Stable(ByPathSegments(entries))

	return entries
}

// createEntries creates any files or directories listed for the filesystem in Storage.{Files,Directories}.
func (s *stage) createEntries(files []filesystemEntry) error {
	s.Logger.PushPrefix("createFiles")
	defer s.Logger.PopPrefix()

	for _, e := range files {
		path := e.getPath()
		if s.relabeling() {
			// relabel from the first parent dir that we'll have to create --
			// alternatively, we could make `MkdirForFile` fancier instead of
			// using `os.MkdirAll`, though that's quite a lot of levels to plumb
			// through
			relabelFrom := path
			dir := filepath.Dir(path)
			for {
				exists, err := s.Util.PathExists(dir)
				if err != nil {
					return err
				}
				// we're done on the first hit -- also sanity check we didn't
				// somehow get all the way up to /
				if exists || dir == "/" {
					break
				}
				relabelFrom = dir
				dir = filepath.Dir(dir)
			}
			s.relabel(relabelFrom)
		}
		if err := e.create(s.Logger, s.Util); err != nil {
			return err
		}
	}
	return nil
}
