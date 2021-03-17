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

package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	cutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"

	"golang.org/x/sys/unix"
)

const (
	DefaultDirectoryPermissions os.FileMode = 0755
	DefaultFilePermissions      os.FileMode = 0644
)

func (u Util) WriteLink(s types.Link) error {
	path := s.Path

	if err := MkdirForFile(path); err != nil {
		return fmt.Errorf("Could not create leading directories: %v", err)
	}

	if cutil.IsTrue(s.Hard) {
		targetPath, err := u.JoinPath(*s.Target)
		if err != nil {
			return err
		}
		return os.Link(targetPath, path)
	}

	if err := os.Symlink(*s.Target, path); err != nil {
		return fmt.Errorf("Could not create symlink: %v", err)
	}

	if err := u.SetPermissions(nil, s.Node); err != nil {
		return fmt.Errorf("error setting permissions of %s: %v", s.Path, err)
	}
	return nil
}

func (u Util) SetPermissions(mode *int, node types.Node) error {
	if mode != nil {
		mode := os.FileMode(*mode)
		if err := os.Chmod(node.Path, mode); err != nil {
			return fmt.Errorf("failed to change mode of %s: %v", node.Path, err)
		}
	}

	defaultUid, defaultGid, _ := getFileOwnerAndMode(node.Path)
	uid, gid, err := u.ResolveNodeUidAndGid(node, defaultUid, defaultGid)
	if err != nil {
		return fmt.Errorf("failed to determine correct uid and gid for %s: %v", node.Path, err)
	}
	if err := os.Lchown(node.Path, uid, gid); err != nil {
		return fmt.Errorf("failed to change ownership of %s: %v", node.Path, err)
	}
	return nil
}

// MkdirForFile helper creates the directory components of path.
func MkdirForFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), DefaultDirectoryPermissions)
}

// FindFirstMissingDirForFile returns the first component which was found to be
// missing for the path.
func FindFirstMissingDirForFile(path string) (string, error) {
	entry := path
	dir := filepath.Dir(path)
	for {
		exists := true
		if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
			exists = false
		} else if err != nil {
			return "", err
		}

		// also sanity check we didn't somehow get all the way up to /sysroot
		if dir == "/" {
			return "", fmt.Errorf("/ doesn't seem to exist")
		}
		if exists {
			return entry, nil
		}
		entry = dir
		dir = filepath.Dir(dir)
	}
}

// FilesystemIsEmpty checks the mountpoint of a filesystem to see whether
// the filesystem is empty.
// Adapted from https://stackoverflow.com/a/30708914
func FilesystemIsEmpty(dirpath string) (bool, error) {
	dfd, err := os.Open(dirpath)
	if err != nil {
		return false, err
	}
	defer dfd.Close()

	for {
		names, err := dfd.Readdirnames(1)
		if err == io.EOF {
			return true, nil
		} else if err != nil {
			return false, err
		}

		switch names[0] {
		case "lost+found":
			// valid in a fresh filesystem; keep reading
		default:
			return false, nil
		}
	}
}

// getFileOwner will return the uid and gid for the file at a given path. If the
// file doesn't exist, or some other error is encountered when running stat on
// the path, 0, 0, and 0 will be returned.
func getFileOwnerAndMode(path string) (int, int, os.FileMode) {
	info := unix.Stat_t{}
	if err := unix.Stat(path, &info); err != nil {
		return 0, 0, 0
	}
	return int(info.Uid), int(info.Gid), os.FileMode(info.Mode)
}

// ResolveNodeUidAndGid attempts to convert a types.Node into a concrete uid and
// gid. If the node has the User.ID field set, that's used for the uid. If the
// node has the User.Name field set, a username -> uid lookup is performed. If
// neither are set, it returns the passed in defaultUid. The logic is identical
// for gids with equivalent fields.
func (u Util) ResolveNodeUidAndGid(n types.Node, defaultUid, defaultGid int) (int, int, error) {
	var err error
	uid, gid := defaultUid, defaultGid

	if n.User.ID != nil {
		uid = *n.User.ID
	} else if cutil.NotEmpty(n.User.Name) {
		uid, err = u.getUserID(*n.User.Name)
		if err != nil {
			return 0, 0, err
		}
	}

	if n.Group.ID != nil {
		gid = *n.Group.ID
	} else if cutil.NotEmpty(n.Group.Name) {
		gid, err = u.getGroupID(*n.Group.Name)
		if err != nil {
			return 0, 0, err
		}
	}
	return uid, gid, nil
}

func (u Util) getUserID(name string) (int, error) {
	usr, err := u.userLookup(name)
	if err != nil {
		return 0, fmt.Errorf("No such user %q: %v", name, err)
	}
	uid, err := strconv.ParseInt(usr.Uid, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("Couldn't parse uid %q: %v", usr.Uid, err)
	}
	return int(uid), nil
}

func (u Util) getGroupID(name string) (int, error) {
	g, err := u.groupLookup(name)
	if err != nil {
		return 0, fmt.Errorf("No such group %q: %v", name, err)
	}
	gid, err := strconv.ParseInt(g.Gid, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("Couldn't parse gid %q: %v", g.Gid, err)
	}
	return int(gid), nil
}
