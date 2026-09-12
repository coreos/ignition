package billyadapter

import (
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/hairyhenderson/go-fsimpl/internal"
)

func BillyToFS(bfs billy.Filesystem) fs.ReadDirFS {
	return &billyFS{bfs}
}

type billyFS struct {
	bfs billy.Filesystem
}

var _ fs.ReadDirFS = (*billyFS)(nil)

func billyPath(p string) string {
	if p == "." {
		return string(filepath.Separator)
	}

	return string(filepath.Separator) + p
}

func isBillyDir(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "cannot open directory")
}

func (f *billyFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	origName := name
	name = billyPath(name)

	bf, err := f.bfs.Open(name)
	if isBillyDir(err) {
		return makeBillyDir(f.bfs, name)
	}

	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: origName, Err: err}
	}

	fi, err := f.bfs.Stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: origName, Err: err}
	}

	file := &billyFile{bf, fi}

	return file, nil
}

func (f *billyFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	name = billyPath(name)

	dir, err := makeBillyDir(f.bfs, name)
	if err != nil {
		return nil, err
	}

	return dir.ReadDir(-1)
}

type billyFile struct {
	file billy.File
	fi   fs.FileInfo
}

var _ fs.File = (*billyFile)(nil)

func (f *billyFile) Stat() (fs.FileInfo, error) {
	return f.fi, nil
}

func (f *billyFile) Read(p []byte) (int, error) {
	return f.file.Read(p)
}

func (f *billyFile) Close() error {
	return f.file.Close()
}

func makeBillyDir(bfs billy.Filesystem, name string) (*billyDir, error) {
	fis, err := bfs.ReadDir(name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}

	fi, err := bfs.Stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}

	dir := billyDir{
		name:     name,
		fi:       fi,
		children: fis,
	}

	return &dir, nil
}

type billyDir struct {
	name     string
	fi       fs.FileInfo
	children []fs.FileInfo
	diroff   int
}

var _ fs.ReadDirFile = (*billyDir)(nil)

func (f billyDir) Stat() (fs.FileInfo, error) { return f.fi, nil }
func (f billyDir) Read(_ []byte) (int, error) { return 0, nil }
func (f billyDir) Close() error               { return nil }

// If n > 0, ReadDir returns at most n DirEntry structures.
// In this case, if ReadDir returns an empty slice, it will return
// a non-nil error explaining why.
// At the end of a directory, the error is io.EOF.
//
// If n <= 0, ReadDir returns all the DirEntry values from the directory
// in a single slice. In this case, if ReadDir succeeds (reads all the way
// to the end of the directory), it returns the slice and a nil error.
// If it encounters an error before the end of the directory,
// ReadDir returns the DirEntry list read until that point and a non-nil error.
func (f *billyDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if n > 0 && f.diroff >= len(f.children) {
		return nil, io.EOF
	}

	low := f.diroff
	high := f.diroff + n

	// clamp high at the max, and ensure it's higher than low
	if high >= len(f.children) || high <= low {
		high = len(f.children)
	}

	entries := dirents(f.children[low:high])

	f.diroff = high

	return entries, nil
}

func dirents(children []fs.FileInfo) []fs.DirEntry {
	entries := make([]fs.DirEntry, len(children))

	for i, fi := range children {
		entries[i] = internal.FileInfoDirEntry(fi)
	}

	return entries
}
