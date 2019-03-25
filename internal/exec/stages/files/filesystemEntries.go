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
	"github.com/coreos/ignition/config/v3_0/types"
	"github.com/coreos/ignition/internal/exec/util"
	"github.com/coreos/ignition/internal/log"
)

// createFilesystemsEntries creates the files described in config.Storage.{Files,Directories}.
func (s *stage) createFilesystemsEntries(config types.Config) error {
	s.Logger.PushPrefix("createFilesystemsFiles")
	defer s.Logger.PopPrefix()

	entryMap, err := s.getOrderedCreationList(config)
	if err != nil {
		return err
	}

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
	empty := "" // golang--

	canOverwrite := f.Overwrite != nil && *f.Overwrite
	st, err := os.Lstat(f.Path)
	regular := (st == nil) || st.Mode().IsRegular()
	switch {
	case os.IsNotExist(err) && f.Contents.Source == nil:
		// set f.Contents so we create an empty file
		f.Contents.Source = &empty
	case os.IsNotExist(err):
		break
	case err != nil:
		return err
	// 3/8 cases where we need to overwrite but can't
	case !canOverwrite && !regular:
		return fmt.Errorf("error creating file %q: A non regular file exists there already and overwrite is false", f.Path)
	case !canOverwrite && regular && f.Contents.Source != nil:
		return fmt.Errorf("error creating file %q: A file exists there already and overwrite is false", f.Path)
	// 2/8 cases where we don't need to do anything
	case regular && f.Contents.Source == nil:
		break
	//  3/8 cases where we need to delete the node first
	case canOverwrite && !regular && f.Contents.Source == nil:
		// If we're deleting the file we need set f.Contents so it creates an empty file
		f.Contents.Source = &empty
		fallthrough
	case canOverwrite && !regular && f.Contents.Source != nil:
		fallthrough
	case canOverwrite && f.Contents.Source != nil:
		if err := os.RemoveAll(f.Path); err != nil {
			return fmt.Errorf("error creating file %q: could not remove existing node at that path: %v", f.Path, err)
		}
	// somehow we missed a case, internal bug
	default:
		return fmt.Errorf("Ignition encountered an internal error processing %q and must die now. Please file a bug", f.Path)
	}

	fetchOps, err := u.PrepareFetches(l, f)
	if err != nil {
		return fmt.Errorf("failed to resolve file %q: %v", f.Path, err)
	}

	for _, op := range fetchOps {
		msg := "writing file %q"
		if op.Append {
			msg = "appending to file %q"
		}
		if err := l.LogOp(
			func() error {
				return u.PerformFetch(op)
			}, msg, f.Path,
		); err != nil {
			return fmt.Errorf("failed to create file %q: %v", op.Node.Path, err)
		}
	}
	u.SetPermissions(f)

	return nil
}

type dirEntry types.Directory

func (tmp dirEntry) getPath() string {
	return types.Directory(tmp).Path
}

func (tmp dirEntry) create(l *log.Logger, u util.Util) error {
	d := types.Directory(tmp)

	err := l.LogOp(func() error {
		path := d.Path

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
			d.Mode = configUtil.IntToPtr(int(util.DefaultDirectoryPermissions))
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

// getOrderedCreationList resolves all symlinks in the node paths and sets the path to be
// prepended by the sysroot. It orders the list from shallowest (e.g. /a) to deepeset
// (e.g. /a/b/c/d/e).
func (s stage) getOrderedCreationList(config types.Config) ([]filesystemEntry, error) {
	entries := []filesystemEntry{}
	// Add directories first to ensure they are created before files.
	for _, d := range config.Storage.Directories {
		path, err := s.JoinPath(d.Path)
		if err != nil {
			return nil, err
		}
		d.Path = path
		entries = append(entries, dirEntry(d))
	}

	for _, f := range config.Storage.Files {
		path, err := s.JoinPath(f.Path)
		if err != nil {
			return nil, err
		}
		f.Path = path
		entries = append(entries, fileEntry(f))
	}

	for _, l := range config.Storage.Links {
		path, err := s.JoinPath(l.Path)
		if err != nil {
			return nil, err
		}
		l.Path = path
		entries = append(entries, linkEntry(l))
	}
	sort.Slice(entries, func(i, j int) bool { return util.Depth(entries[i].getPath()) < util.Depth(entries[j].getPath()) })

	return entries, nil
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
				exists := true
				if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
					exists = false
				} else if err != nil {
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
