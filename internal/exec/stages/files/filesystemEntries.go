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
	"strings"

	"github.com/coreos/ignition/v2/config/v3_1_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
)

// createFilesystemsEntries creates the files described in config.Storage.{Files,Directories}.
func (s *stage) createFilesystemsEntries(config types.Config) error {
	s.Logger.PushPrefix("createFilesystemsFiles")
	defer s.Logger.PopPrefix()

	entries, err := s.getOrderedCreationList(config)
	if err != nil {
		return err
	}

	if err := s.createEntries(entries); err != nil {
		return fmt.Errorf("failed to create files: %v", err)
	}

	return nil
}

// filesystemEntry represent a thing that knows how to create itself.
type filesystemEntry interface {
	// create creates the entry if specified. It assumes that if overwrite=true then any existing
	// files at the path will have been deleted.
	create(l *log.Logger, u util.Util) error
	node() types.Node
}

type fileEntry types.File

func (tmp fileEntry) node() types.Node {
	return types.File(tmp).Node
}

func (tmp fileEntry) create(l *log.Logger, u util.Util) error {
	f := types.File(tmp)

	empty := "" // golang--

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
	// Cases where there is file there
	case !regular:
		return fmt.Errorf("error creating file %q: A non regular file exists there already and overwrite is false", f.Path)
	case f.Contents.Source != nil:
		return fmt.Errorf("error creating file %q: A file exists there already and overwrite is false", f.Path)
	case regular && f.Contents.Source == nil:
		break
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
	if err := u.SetPermissions(f.Mode, f.Node); err != nil {
		return fmt.Errorf("error setting file permissions for %s: %v", f.Path, err)
	}
	return nil
}

type dirEntry types.Directory

func (tmp dirEntry) node() types.Node {
	return types.Directory(tmp).Node
}

func (tmp dirEntry) create(l *log.Logger, u util.Util) error {
	d := types.Directory(tmp)
	st, err := os.Lstat(d.Path)
	switch {
	case os.IsNotExist(err):
		// use default perms, we'll fix it later
		if err := os.MkdirAll(d.Path, util.DefaultDirectoryPermissions); err != nil {
			return fmt.Errorf("Failed to create directory %s: %v", d.Path, err)
		}
	case err != nil:
		return fmt.Errorf("stat() failed on %s: %v", d.Path, err)
	case !st.Mode().IsDir():
		return fmt.Errorf("error creating directory %s: A non-directory already exists and overwrite is false", d.Path)
	}

	if err := u.SetPermissions(d.Mode, d.Node); err != nil {
		return fmt.Errorf("error setting directory permissions for %s: %v", d.Path, err)
	}
	return nil
}

type linkEntry types.Link

func (tmp linkEntry) node() types.Node {
	return types.Link(tmp).Node
}

func (tmp linkEntry) create(l *log.Logger, u util.Util) error {
	s := types.Link(tmp)
	hard := s.Hard != nil && *s.Hard
	st, err := os.Lstat(s.Path)
	switch {
	case os.IsNotExist(err):
		break
	case err != nil:
		return fmt.Errorf("stat() failed on %s: %v", s.Path, err)
	case hard:
		// check that the file at that path points to the same inode as target
		targetPath, err := u.JoinPath(s.Target)
		if err != nil {
			return fmt.Errorf("error resolving target path of hard link %s: %v", s.Path, err)
		}
		targetst, err := os.Lstat(targetPath)
		if err != nil {
			return fmt.Errorf("error creating hard link %s: target does not exist or stat() returned an err: %v", s.Path, err)
		}
		if !os.SameFile(st, targetst) {
			return fmt.Errorf("error creating hard link %s: a file already exists at that path but is not the target and overwrite is false", s.Path)
		}
		l.Info("Hardlink %s to %s already exists, doing nothing", s.Path, s.Target)
		return nil
	case !hard:
		// if the existing file is a symlink, check that its target is correct
		if st.Mode()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(s.Path); err != nil {
				return fmt.Errorf("error reading link at %s: %v", s.Path, err)
			} else if filepath.Clean(target) != filepath.Clean(s.Target) {
				return fmt.Errorf("error creating symlink %s: a symlink exists at that path but points to %s, not %s and overwrite is false", s.Path, target, s.Target)
			} else {
				l.Info("Symlink %s to %s already exists, doing nothing", s.Path, s.Target)
				return nil
			}
		}
		return fmt.Errorf("error creating symlink %s: a non-symlink already exists at that path and overwrite is false", s.Path)
	}

	if err := l.LogOp(
		func() error {
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
	// Map from paths in the config to where they resolve for duplicate checking
	paths := map[string]string{}
	for _, d := range config.Storage.Directories {
		path, err := s.JoinPath(d.Path)
		if err != nil {
			return nil, err
		}
		if existing, ok := paths[path]; ok {
			return nil, fmt.Errorf("Directory at %s resolved to %s after symlink chasing, but another entry with path %s also resolves there",
				d.Path, path, existing)
		}
		paths[path] = d.Path
		d.Path = path
		entries = append(entries, dirEntry(d))
	}

	for _, f := range config.Storage.Files {
		path, err := s.JoinPath(f.Path)
		if err != nil {
			return nil, err
		}
		if existing, ok := paths[path]; ok {
			return nil, fmt.Errorf("File at %s resolved to %s after symlink chasing, but another entry with path %s also resolves there",
				f.Path, path, existing)
		}
		paths[path] = f.Path
		f.Path = path
		entries = append(entries, fileEntry(f))
	}

	for _, l := range config.Storage.Links {
		path, err := s.JoinPath(l.Path)
		if err != nil {
			return nil, err
		}
		if existing, ok := paths[path]; ok {
			return nil, fmt.Errorf("Link at %s resolved to %s after symlink chasing, but another entry with path %s also resolves there",
				l.Path, path, existing)
		}
		paths[path] = l.Path
		l.Path = path
		entries = append(entries, linkEntry(l))
	}
	sort.Slice(entries, func(i, j int) bool { return util.Depth(entries[i].node().Path) < util.Depth(entries[j].node().Path) })

	return entries, nil
}

func (s *stage) removePathOnOverwrite(e filesystemEntry) error {
	if e.node().Overwrite != nil && *e.node().Overwrite {
		return os.RemoveAll(e.node().Path)
	}
	return nil
}

func (s *stage) relabelDirsForFile(path string) error {
	if !s.relabeling() {
		return nil
	}
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
		// somehow get all the way up to /sysroot
		if exists || dir == s.DestDir {
			break
		}
		relabelFrom = dir
		dir = filepath.Dir(dir)
	}
	// trim off prefix since this needs to be relative to the sysroot
	s.relabel(relabelFrom[len(s.DestDir):])

	return nil
}

// createEntries creates any files or directories listed for the filesystem in Storage.{Files,Directories}.
func (s *stage) createEntries(entries []filesystemEntry) error {
	s.Logger.PushPrefix("createFiles")
	defer s.Logger.PopPrefix()

	for _, e := range entries {
		path := e.node().Path
		if !strings.HasPrefix(path, s.DestDir) {
			panic(fmt.Sprintf("Entry path %s isn't under prefix %s", path, s.DestDir))
		}

		if err := s.relabelDirsForFile(path); err != nil {
			return fmt.Errorf("error relabeling paths for %s: %v", path, err)
		}
		if err := s.removePathOnOverwrite(e); err != nil {
			return fmt.Errorf("error removing existing file %s: %v", path, err)
		}
		if err := e.create(s.Logger, s.Util); err != nil {
			return fmt.Errorf("error creating %s: %v", path, err)
		}
	}
	return nil
}
