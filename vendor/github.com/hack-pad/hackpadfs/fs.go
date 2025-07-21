// Package hackpadfs defines many essential file and file system interfaces as well as helpers for use with the standard library's 'io/fs' package.
package hackpadfs

import (
	"errors"
	gofs "io/fs"
	gopath "path"
	"time"
)

// FS provides access to a file system and its files.
// It is the minimum functionality required for a file system, and mirrors Go's io/fs.FS interface.
type FS = gofs.FS

// SubFS is an FS that can return a subset of the current FS.
// The same effect as `chroot` in a program.
type SubFS interface {
	FS
	Sub(dir string) (FS, error)
}

// OpenFileFS is an FS that can open files with the given flags and can create with the given permission.
// Should matche the behavior of os.OpenFile().
type OpenFileFS interface {
	FS
	OpenFile(name string, flag int, perm FileMode) (File, error)
}

// CreateFS is an FS that can create files. Should match the behavior of os.Create().
type CreateFS interface {
	FS
	Create(name string) (File, error)
}

// MkdirFS is an FS that can make directories. Should match the behavior of os.Mkdir().
type MkdirFS interface {
	FS
	Mkdir(name string, perm FileMode) error
}

// MkdirAllFS is an FS that can make all missing directories in a given path. Should match the behavior of os.MkdirAll().
type MkdirAllFS interface {
	FS
	MkdirAll(path string, perm FileMode) error
}

// RemoveFS is an FS that can remove files or empty directories. Should match the behavior of os.Remove().
type RemoveFS interface {
	FS
	Remove(name string) error
}

// RemoveAllFS is an FS that can remove files or directories recursively. Should match the behavior of os.RemoveAll().
type RemoveAllFS interface {
	FS
	RemoveAll(name string) error
}

// RenameFS is an FS that can move files or directories. Should match the behavior of os.Rename().
type RenameFS interface {
	FS
	Rename(oldname, newname string) error
}

// StatFS is an FS that can stat files or directories. Should match the behavior of os.Stat().
type StatFS interface {
	FS
	Stat(name string) (FileInfo, error)
}

// LstatFS is an FS that can lstat files. Same as Stat, but returns file info of symlinks instead of their target. Should match the behavior of os.Lstat().
type LstatFS interface {
	FS
	Lstat(name string) (FileInfo, error)
}

// ChmodFS is an FS that can change file or directory permissions. Should match the behavior of os.Chmod().
type ChmodFS interface {
	FS
	Chmod(name string, mode FileMode) error
}

// ChownFS is an FS that can change file or directory ownership. Should match the behavior of os.Chown().
type ChownFS interface {
	FS
	Chown(name string, uid, gid int) error
}

// ChtimesFS is an FS that can change a file's access and modified timestamps. Should match the behavior of os.Chtimes().
type ChtimesFS interface {
	FS
	Chtimes(name string, atime time.Time, mtime time.Time) error
}

// ReadDirFS is an FS that can read a directory and return its DirEntry's. Should match the behavior of os.ReadDir().
type ReadDirFS interface {
	FS
	ReadDir(name string) ([]DirEntry, error)
}

// ReadFileFS is an FS that can read an entire file in one pass. Should match the behavior of os.ReadFile().
type ReadFileFS interface {
	FS
	ReadFile(name string) ([]byte, error)
}

// WriteFileFS is an FS that can write an entire file in one pass. Should match the behavior of os.WriteFile().
type WriteFileFS interface {
	FS
	WriteFile(name string, data []byte, perm FileMode) error
}

// SymlinkFS is an FS that can create symlinks. Should match the behavior of os.Symlink().
type SymlinkFS interface {
	FS
	Symlink(oldname, newname string) error
}

// MountFS is an FS that meshes one or more FS's together.
// Returns the FS for a file located at 'name' and its 'subPath' inside that FS.
type MountFS interface {
	FS
	Mount(name string) (mountFS FS, subPath string)
}

// ValidPath returns true if 'path' is a valid FS path. See io/fs.ValidPath() for details on FS-safe paths.
func ValidPath(path string) bool {
	return gofs.ValidPath(path)
}

// WalkDirFunc is the type of function called in WalkDir().
type WalkDirFunc = gofs.WalkDirFunc

// WalkDir recursively scans 'fs' starting at path 'root', calling 'fn' every time a new file or directory is visited.
func WalkDir(fs FS, root string, fn WalkDirFunc) error {
	return gofs.WalkDir(fs, root, fn)
}

// Sub attempts to call an optimized fs.Sub() if available. Falls back to a small MountFS implementation.
func Sub(fs FS, dir string) (FS, error) {
	if fs, ok := fs.(SubFS); ok {
		return fs.Sub(dir)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(dir)
		fs, err := Sub(mountFS, subPath)
		return fs, stripErrPathPrefix(err, dir, subPath)
	}
	return newSubFS(fs, dir)
}

// OpenFile attempts to call fs.Open() or fs.OpenFile() if available. Fails with a not implemented error otherwise.
func OpenFile(fs FS, name string, flag int, perm FileMode) (File, error) {
	if flag == FlagReadOnly {
		return fs.Open(name)
	}
	if fs, ok := fs.(OpenFileFS); ok {
		return fs.OpenFile(name, flag, perm)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		file, err := OpenFile(mountFS, subPath, flag, perm)
		return file, stripErrPathPrefix(err, name, subPath)
	}
	return nil, &PathError{Op: "open", Path: name, Err: ErrNotImplemented}
}

// Create attempts to call an optimized fs.Create() if available, falls back to OpenFile() with create flags.
func Create(fs FS, name string) (File, error) {
	if fs, ok := fs.(CreateFS); ok {
		return fs.Create(name)
	}
	return OpenFile(fs, name, FlagReadWrite|FlagCreate|FlagTruncate, 0o666)
}

// Mkdir creates a directory. Fails with a not implemented error if it's not a MkdirFS.
func Mkdir(fs FS, name string, perm FileMode) error {
	if fs, ok := fs.(MkdirFS); ok {
		return fs.Mkdir(name, perm)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		err := Mkdir(mountFS, subPath, perm)
		return stripErrPathPrefix(err, name, subPath)
	}
	return &PathError{Op: "mkdir", Path: name, Err: ErrNotImplemented}
}

// MkdirAll attempts to call an optimized fs.MkdirAll(), falls back to multiple fs.Mkdir() calls.
func MkdirAll(fs FS, path string, perm FileMode) error {
	if fs, ok := fs.(MkdirAllFS); ok {
		return fs.MkdirAll(path, perm)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(path)
		err := MkdirAll(mountFS, subPath, perm)
		return stripErrPathPrefix(err, path, subPath)
	}
	if !ValidPath(path) {
		return &PathError{Op: "mkdirall", Path: path, Err: ErrInvalid}
	}
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			err := Mkdir(fs, path[:i], perm)
			if err != nil {
				pathErr, ok := err.(*PathError)
				if !ok || !errors.Is(err, ErrExist) {
					return err
				}
				info, statErr := Stat(fs, pathErr.Path)
				if statErr != nil {
					return err
				}
				if !info.IsDir() {
					return &PathError{Op: "mkdir", Path: pathErr.Path, Err: ErrNotDir}
				}
			}
		}
	}
	return Mkdir(fs, path, perm)
}

// Remove removes a file with fs.Remove(). Fails with a not implemented error if it's not a RemoveFS.
func Remove(fs FS, name string) error {
	if fs, ok := fs.(RemoveFS); ok {
		return fs.Remove(name)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		err := Remove(mountFS, subPath)
		return stripErrPathPrefix(err, name, subPath)
	}
	return &PathError{Op: "remove", Path: name, Err: ErrNotImplemented}
}

// RemoveAll attempts to call an optimized fs.RemoveAll(), falls back to removing files and directories recursively.
func RemoveAll(fs FS, path string) error {
	if fs, ok := fs.(RemoveAllFS); ok {
		return fs.RemoveAll(path)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(path)
		err := RemoveAll(mountFS, subPath)
		return stripErrPathPrefix(err, path, subPath)
	}

	if !ValidPath(path) {
		return &PathError{Op: "removeall", Path: path, Err: ErrInvalid}
	}
	return removeAll(fs, path)
}

func removeAll(fs FS, path string) error {
	info, err := Stat(fs, path)
	if err != nil {
		if errors.Is(err, ErrNotExist) {
			err = nil
		}
		return err
	}
	if !info.IsDir() {
		err := Remove(fs, path)
		if errors.Is(err, ErrNotExist) {
			err = nil
		}
		return err
	}

	dir, err := ReadDir(fs, path)
	if err != nil {
		return &PathError{Op: "removeall", Path: path, Err: err}
	}
	for _, dirEntry := range dir {
		err := removeAll(fs, gopath.Join(path, dirEntry.Name()))
		if err != nil {
			return &PathError{Op: "removeall", Path: path, Err: err}
		}
	}
	if err := Remove(fs, path); err == nil || errors.Is(err, ErrNotExist) {
		return nil
	}
	return nil
}

// Rename moves files with fs.Rename(). Fails with a not implemented error if it's not a RenameFS.
func Rename(fs FS, oldName, newName string) error {
	if fs, ok := fs.(RenameFS); ok {
		return fs.Rename(oldName, newName)
	}
	return &LinkError{Op: "rename", Old: oldName, New: newName, Err: ErrNotImplemented}
}

// Stat attempts to call an optimized fs.Stat(), falls back to fs.Open() and file.Stat().
func Stat(fs FS, name string) (FileInfo, error) {
	if fs, ok := fs.(StatFS); ok {
		return fs.Stat(name)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		info, err := Stat(mountFS, subPath)
		return info, stripErrPathPrefix(err, name, subPath)
	}
	file, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return file.Stat()
}

// Lstat stats files and does not follow symlinks. Fails with a not implemented error if it's not a LstatFS.
func Lstat(fs FS, name string) (FileInfo, error) {
	if fs, ok := fs.(LstatFS); ok {
		return fs.Lstat(name)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		info, err := Lstat(mountFS, subPath)
		return info, stripErrPathPrefix(err, name, subPath)
	}
	return nil, &PathError{Op: "lstat", Path: name, Err: ErrNotImplemented}
}

// LstatOrStat attempts to call an optimized fs.LstatOrStat(), fs.Lstat(), or fs.Stat() - whichever is supported first.
func LstatOrStat(fs FS, name string) (FileInfo, error) {
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		info, err := LstatOrStat(mountFS, subPath)
		return info, stripErrPathPrefix(err, name, subPath)
	}
	info, err := Lstat(fs, name)
	if errors.Is(err, ErrNotImplemented) {
		info, err = Stat(fs, name)
	}
	return info, err
}

// Chmod attempts to call an optimized fs.Chmod(), falls back to opening the file and running file.Chmod().
func Chmod(fs FS, name string, mode FileMode) error {
	if fs, ok := fs.(ChmodFS); ok {
		return fs.Chmod(name, mode)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		err := Chmod(mountFS, subPath, mode)
		return stripErrPathPrefix(err, name, subPath)
	}
	file, err := fs.Open(name)
	if err != nil {
		return &PathError{Op: "chmod", Path: name, Err: err}
	}
	defer func() { _ = file.Close() }()
	return ChmodFile(file, mode)
}

// Chown attempts to call an optimized fs.Chown(), falls back to opening the file and running file.Chown().
func Chown(fs FS, name string, uid, gid int) error {
	if fs, ok := fs.(ChownFS); ok {
		return fs.Chown(name, uid, gid)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		err := Chown(mountFS, subPath, uid, gid)
		return stripErrPathPrefix(err, name, subPath)
	}
	file, err := fs.Open(name)
	if err != nil {
		return &PathError{Op: "chown", Path: name, Err: err}
	}
	defer func() { _ = file.Close() }()
	return ChownFile(file, uid, gid)
}

// Chtimes attempts to call an optimized fs.Chtimes(), falls back to opening the file and running file.Chtimes().
func Chtimes(fs FS, name string, atime time.Time, mtime time.Time) error {
	if fs, ok := fs.(ChtimesFS); ok {
		return fs.Chtimes(name, atime, mtime)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		err := Chtimes(mountFS, subPath, atime, mtime)
		return stripErrPathPrefix(err, name, subPath)
	}
	file, err := fs.Open(name)
	if err != nil {
		return &PathError{Op: "chtimes", Path: name, Err: err}
	}
	defer func() { _ = file.Close() }()
	return ChtimesFile(file, atime, mtime)
}

// ReadDir attempts to call an optimized fs.ReadDir(), falls back to io/fs.ReadDir().
func ReadDir(fs FS, name string) ([]DirEntry, error) {
	if fs, ok := fs.(ReadDirFS); ok {
		return fs.ReadDir(name)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		dirEntries, err := ReadDir(mountFS, subPath)
		return dirEntries, stripErrPathPrefix(err, name, subPath)
	}
	return gofs.ReadDir(fs, name)
}

// ReadFile attempts to call an optimized fs.ReadFile(), falls back to io/fs.ReadFile().
func ReadFile(fs FS, name string) ([]byte, error) {
	if fs, ok := fs.(ReadFileFS); ok {
		return fs.ReadFile(name)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		b, err := ReadFile(mountFS, subPath)
		return b, stripErrPathPrefix(err, name, subPath)
	}
	return gofs.ReadFile(fs, name)
}

// WriteFullFile attempts to call an optimized fs.WriteFile(), falls back to fs.OpenFile() with file.Write().
func WriteFullFile(fs FS, name string, data []byte, perm FileMode) error {
	if fs, ok := fs.(WriteFileFS); ok {
		return fs.WriteFile(name, data, perm)
	}
	if fs, ok := fs.(MountFS); ok {
		mountFS, subPath := fs.Mount(name)
		err := WriteFullFile(mountFS, subPath, data, perm)
		return stripErrPathPrefix(err, name, subPath)
	}

	f, err := OpenFile(fs, name, FlagWriteOnly|FlagCreate|FlagTruncate, perm)
	if err == nil {
		_, err = WriteFile(f, data)
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
	}
	return err
}

// Symlink creates a symlink. Fails with a not implemented error if it's not a SymlinkFS.
func Symlink(fs FS, oldname, newname string) error {
	if fs, ok := fs.(SymlinkFS); ok {
		return fs.Symlink(oldname, newname)
	}
	return &LinkError{Op: "symlink", Old: oldname, New: newname, Err: ErrNotImplemented}
}
