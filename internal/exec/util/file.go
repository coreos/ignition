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
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	cutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/util"

	"golang.org/x/sys/unix"
)

const (
	DefaultDirectoryPermissions os.FileMode = 0755
	DefaultFilePermissions      os.FileMode = 0644
)

type FetchMode int

const (
	FetchReplace FetchMode = iota
	FetchAppend
	FetchExtract
)

type ArchiveType string

const (
	ArchiveTAR ArchiveType = "tar"
)

type FetchOp struct {
	Hash         hash.Hash
	Url          url.URL
	FetchOptions resource.FetchOptions
	Mode         FetchMode
	ArchiveType  ArchiveType
	Node         types.Node
}

func MakeFetchOp(l *log.Logger, node types.Node, contents types.Resource) (FetchOp, error) {
	var expectedSum []byte

	uri, err := url.Parse(*contents.Source)
	if err != nil {
		return FetchOp{}, err
	}

	hasher, err := util.GetHasher(contents.Verification)
	if err != nil {
		l.Crit("Error verifying file %q: %v", node.Path, err)
		return FetchOp{}, err
	}

	if hasher != nil {
		// explicitly ignoring the error here because the config should already
		// be validated by this point
		_, expectedSumString, _ := util.HashParts(contents.Verification)
		expectedSum, err = hex.DecodeString(expectedSumString)
		if err != nil {
			l.Crit("Error parsing verification string %q: %v", expectedSumString, err)
			return FetchOp{}, err
		}
	}
	compression := ""
	if contents.Compression != nil {
		compression = *contents.Compression
	}

	var headers http.Header
	if contents.HTTPHeaders != nil && len(contents.HTTPHeaders) > 0 {
		headers, err = contents.HTTPHeaders.Parse()
		if err != nil {
			return FetchOp{}, err
		}
	}

	return FetchOp{
		Hash: hasher,
		Node: node,
		Url:  *uri,
		FetchOptions: resource.FetchOptions{
			Hash:        hasher,
			Compression: compression,
			ExpectedSum: expectedSum,
			Headers:     headers,
		},
	}, nil
}

// PrepareFetches converts a given logger, http client, and types.File into a
// FetchOp. This includes operations such as parsing the source URL, generating
// a hasher, and performing user/group name lookups. If an error is encountered,
// the issue will be logged and nil will be returned.
func (u Util) PrepareFetches(l *log.Logger, f types.File) ([]FetchOp, error) {
	ops := []FetchOp{}

	if f.Contents.Source != nil {
		if base, err := MakeFetchOp(l, f.Node, f.Contents); err != nil {
			return nil, err
		} else {
			ops = append(ops, base)
		}
	}

	for _, appendee := range f.Append {
		if op, err := MakeFetchOp(l, f.Node, appendee); err != nil {
			return nil, err
		} else {
			op.Mode = FetchAppend
			ops = append(ops, op)
		}
	}

	return ops, nil
}

func (u Util) WriteLink(s types.Link) error {
	path := s.Path

	if err := MkdirForFile(path); err != nil {
		return fmt.Errorf("could not create leading directories: %v", err)
	}

	if cutil.IsTrue(s.Hard) {
		targetPath, err := u.JoinPath(*s.Target)
		if err != nil {
			return err
		}
		return os.Link(targetPath, path)
	}

	if err := os.Symlink(*s.Target, path); err != nil {
		return fmt.Errorf("could not create symlink: %v", err)
	}

	if err := u.SetPermissions(nil, s.Node); err != nil {
		return fmt.Errorf("error setting permissions of %s: %v", s.Path, err)
	}
	return nil
}

func (u Util) SetPermissions(mode *int, node types.Node) error {
	if mode != nil {
		if err := os.Chmod(node.Path, toFileMode(*mode)); err != nil {
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

// toFileMode converts Go permission bits to POSIX permission bits.
func toFileMode(m int) os.FileMode {
	mode := uint32(m)
	res := os.FileMode(mode & 0777)

	if mode&syscall.S_ISGID != 0 {
		res |= os.ModeSetgid
	}
	if mode&syscall.S_ISUID != 0 {
		res |= os.ModeSetuid
	}
	if mode&syscall.S_ISVTX != 0 {
		res |= os.ModeSticky
	}
	return res
}

// PerformFetch performs a fetch operation generated by PrepareFetch, retrieving
// the file and writing it to disk. Any encountered errors are returned.
func (u Util) PerformFetch(f FetchOp) error {
	path := f.Node.Path

	if err := MkdirForFile(path); err != nil {
		return err
	}

	// Create a temporary file in the same directory to ensure it's on the same filesystem
	tmp, err := os.CreateTemp(filepath.Dir(path), "tmp")
	if err != nil {
		return err
	}
	defer tmp.Close()

	// os.CreateTemp defaults to 0600
	if err := tmp.Chmod(DefaultFilePermissions); err != nil {
		return err
	}

	// sometimes the following line will fail (the file might be renamed),
	// but that's ok (we wanted to keep the file in that case).
	defer os.Remove(tmp.Name())

	err = u.Fetcher.Fetch(f.Url, tmp, f.FetchOptions)
	if err != nil {
		u.Crit("Error fetching file %q: %v", path, err)
		return err
	}

	switch f.Mode {
	case FetchAppend:
		// Make sure that we're appending to a file
		finfo, err := os.Lstat(path)
		switch {
		case os.IsNotExist(err):
			// No problem, we'll create it.
			break
		case err != nil:
			return err
		default:
			if !finfo.Mode().IsRegular() {
				return fmt.Errorf("can only append to files: %q", path)
			}
		}

		// Open with the default permissions, we'll chown/chmod it later
		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, DefaultFilePermissions)
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err = tmp.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err = io.Copy(targetFile, tmp); err != nil {
			return err
		}
	case FetchReplace:
		if err = os.Rename(tmp.Name(), path); err != nil {
			return err
		}
	case FetchExtract:
		var walker archiveWalker
		switch f.ArchiveType {
		case ArchiveTAR:
			walker = tarWalker{}
		default:
			return fmt.Errorf("unsupported archive type %s", string(f.ArchiveType))
		}
		if err := u.extract(walker, tmp.Name(), path); err != nil {
			return err
		}
	}

	return nil
}

// MkdirForFile helper creates the directory components of path.
func MkdirForFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), DefaultDirectoryPermissions)
}

// FindFirstMissingPathComponent returns the path up to the first component
// which was found to be missing, or the whole path if it already exists.
func FindFirstMissingPathComponent(path string) (string, error) {
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
		return 0, fmt.Errorf("no such user %q: %v", name, err)
	}
	uid, err := strconv.ParseInt(usr.Uid, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("couldn't parse uid %q: %v", usr.Uid, err)
	}
	return int(uid), nil
}

func (u Util) getGroupID(name string) (int, error) {
	g, err := u.groupLookup(name)
	if err != nil {
		return 0, fmt.Errorf("no such group %q: %v", name, err)
	}
	gid, err := strconv.ParseInt(g.Gid, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("couldn't parse gid %q: %v", g.Gid, err)
	}
	return int(gid), nil
}
