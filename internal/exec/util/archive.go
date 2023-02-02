// Copyright 2022 Red Hat, Inc.
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
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/coreos/ignition/v2/internal/log"
)

const filemode = 07777

type archiveWalker interface {
	Walk(l *log.Logger, path string, fn WalkFunc) error
}

type FileInfo struct {
	Name     string
	Linkname string
	Mode     int64
	Size     int64
	UID      int
	GID      int
	Xattrs   map[string]string
}

type WalkFunc func(fi FileInfo, r io.Reader) error

type tarWalker struct{}

func (tarWalker) Walk(l *log.Logger, path string, fn WalkFunc) error {
	ar, err := os.Open(path)
	if err != nil {
		return err
	}
	defer ar.Close()

	rd := tar.NewReader(ar)
	for {
		hdr, err := rd.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}

		fi := FileInfo{
			Name:   hdr.Name,
			Mode:   hdr.Mode & filemode,
			Size:   hdr.Size,
			UID:    hdr.Uid,
			GID:    hdr.Gid,
			Xattrs: make(map[string]string),
		}

		switch hdr.Typeflag {
		case tar.TypeLink:
			fi.Linkname = hdr.Linkname
			fallthrough
		case tar.TypeReg:
			fi.Mode |= unix.S_IFREG
		case tar.TypeSymlink:
			fi.Linkname = hdr.Linkname
			fi.Mode |= unix.S_IFLNK
		case tar.TypeDir:
			fi.Mode |= unix.S_IFDIR
		default:
			l.Warning("Unsupported TAR file type %q, skipping it.", hdr.Typeflag)
		}

		// Prefer user/group names for portability. Most archives don't have
		// users/groups other than 0, but a file owned by "bin" may have
		// different IDs depending on the distribution.
		if hdr.Uname != "" {
			usr, err := user.Lookup(hdr.Uname)
			if err == nil {
				fi.UID, err = strconv.Atoi(usr.Uid)
			}
			if err != nil {
				l.Warning("could not look up user %v, defaulting to UID %d: %v", hdr.Uname, fi.UID, err)
			}
		}
		if hdr.Gname != "" {
			grp, err := user.LookupGroup(hdr.Gname)
			if err == nil {
				fi.GID, err = strconv.Atoi(grp.Gid)
			}
			if err != nil {
				l.Warning("could not look up group %v, defaulting to GID %d: %v", hdr.Gname, fi.GID, err)
			}
		}

		for k, v := range hdr.PAXRecords {
			if !strings.HasPrefix(k, "SCHILY.xattr.") {
				continue
			}
			fi.Xattrs[k[len("SCHILY.xattr."):]] = v
		}

		if err := fn(fi, rd); err != nil {
			return err
		}
	}
}

func (u Util) extract(walker archiveWalker, from, to string) error {
	destdir, err := unix.Open(to, unix.O_PATH|unix.O_DIRECTORY, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: to, Err: err}
	}
	defer unix.Close(destdir)

	return walker.Walk(u.Logger, from, func(fi FileInfo, r io.Reader) error {

		u.Debug("extracting %v", fi.Name)

		// Make sure we do not follow symlinks or magic links, and that we
		// resolve paths relative to the destination. This is less expensive
		// than chroot, and prevents any nonsense with regard to tarballs
		// trying to write outside of the destination directory.
		const resolve = unix.RESOLVE_IN_ROOT | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_SYMLINKS

		dirfd, err := unix.Openat2(destdir, filepath.Dir(fi.Name), &unix.OpenHow{
			Flags:   unix.O_PATH | unix.O_DIRECTORY | unix.O_NOFOLLOW,
			Resolve: resolve,
		})
		if err != nil {
			return &os.PathError{Op: "open", Path: filepath.Dir(fi.Name), Err: err}
		}
		defer unix.Close(dirfd)

		name := filepath.Base(fi.Name)

		setattrs := func(fd int, name string) error {
			if err := unix.Fchownat(fd, name, fi.UID, fi.GID, unix.AT_EMPTY_PATH); err != nil {
				return &os.PathError{Op: "chown", Path: fi.Name, Err: err}
			}

			if name == "" {
				if err := unix.Fchmod(fd, uint32(fi.Mode&filemode)); err != nil {
					return &os.PathError{Op: "chmod", Path: fi.Name, Err: err}
				}

				for attr, val := range fi.Xattrs {
					if err := unix.Fsetxattr(fd, attr, []byte(val), 0); err != nil {
						return &os.PathError{Op: "setxattr", Path: fi.Name, Err: err}
					}
				}
			} else {
				if err := unix.Fchmodat(fd, name, uint32(fi.Mode&filemode), 0); err != nil {
					return &os.PathError{Op: "chmod", Path: fi.Name, Err: err}
				}

				// There is no fsetxattrat, but lsetxattr is good enough.
				for attr, val := range fi.Xattrs {
					if err := unix.Lsetxattr(fi.Name, attr, []byte(val), 0); err != nil {
						return &os.PathError{Op: "setxattr", Path: fi.Name, Err: err}
					}
				}
			}
			return nil
		}

		switch fi.Mode & unix.S_IFMT {
		case unix.S_IFREG:
			if fi.Linkname != "" {
				target := fi.Linkname
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(fi.Name), target)
				}

				srcfd, err := unix.Openat2(destdir, target, &unix.OpenHow{
					Flags:   unix.O_PATH | unix.O_NOFOLLOW,
					Resolve: resolve,
				})
				if err != nil {
					return &os.PathError{Op: "open", Path: target, Err: err}
				}
				defer unix.Close(srcfd)

				if err := unix.Linkat(srcfd, "", dirfd, name, unix.AT_EMPTY_PATH); err != nil {
					return &os.PathError{Op: "link", Path: fi.Name, Err: err}
				}
			} else {
				fd, err := unix.Openat2(dirfd, name, &unix.OpenHow{
					Flags:   unix.O_WRONLY | unix.O_CREAT | unix.O_TRUNC,
					Resolve: resolve,
				})
				if err != nil {
					return &os.PathError{Op: "open", Path: fi.Name, Err: err}
				}
				f := os.NewFile(uintptr(fd), fi.Name)
				defer f.Close()

				if _, err := io.Copy(f, r); err != nil {
					return &os.PathError{Op: "extract", Path: fi.Name, Err: err}
				}

				if err := setattrs(fd, ""); err != nil {
					return err
				}

				if err := f.Close(); err != nil {
					return err
				}
			}
		case unix.S_IFDIR:
			if err := unix.Mkdirat(dirfd, name, 0); err != nil && !errors.Is(err, unix.EEXIST) {
				return &os.PathError{Op: "mkdir", Path: fi.Name, Err: err}
			}
			if err := setattrs(dirfd, name); err != nil {
				return err
			}
		case unix.S_IFLNK:
			if err := unix.Symlinkat(fi.Linkname, dirfd, name); err != nil {
				return &os.PathError{Op: "symlink", Path: fi.Name, Err: err}
			}
			if err := setattrs(dirfd, name); err != nil {
				return err
			}
		default:
			return &os.PathError{Op: "walk", Path: fi.Name, Err: fmt.Errorf("Unsupported file type %v", fi.Mode&unix.S_IFMT)}
		}

		return nil
	})
}
