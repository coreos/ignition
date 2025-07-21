// Package filefs wraps os.DirFS to provide a local filesystem for file:// URLs.
package filefs

import (
	"io/fs"
	"net/url"
	"os"

	"github.com/hairyhenderson/go-fsimpl"
)

type fileFS struct {
	root fs.FS
}

// New returns a filesystem (an fs.FS) for the tree of files rooted at the
// directory root. This filesystem is suitable for use with the 'file:' URL
// scheme, and interacts with the local filesystem.
//
// This is effectively a wrapper for os.DirFS.
func New(u *url.URL) (fs.FS, error) {
	rootPath := pathForDirFS(u)

	return &fileFS{root: os.DirFS(rootPath)}, nil
}

// return the correct filesystem path for the given URL. Supports Windows paths
// and UNCs as well
func pathForDirFS(u *url.URL) string {
	if u.Path == "" {
		return ""
	}

	rootPath := u.Path
	if len(rootPath) >= 3 {
		if rootPath[0] == '/' && rootPath[2] == ':' {
			rootPath = rootPath[1:]
		}
	}

	// a file:// URL with a host part should be interpreted as a UNC
	switch u.Host {
	case ".":
		rootPath = "//./" + rootPath
	case "":
		// nothin'
	default:
		rootPath = "//" + u.Host + rootPath
	}

	return rootPath
}

// FS is used to register this filesystem with an fsimpl.FSMux
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, "file")

var (
	_ fs.FS         = (*fileFS)(nil)
	_ fs.ReadDirFS  = (*fileFS)(nil)
	_ fs.ReadFileFS = (*fileFS)(nil)
	_ fs.StatFS     = (*fileFS)(nil)
	_ fs.GlobFS     = (*fileFS)(nil)
	_ fs.SubFS      = (*fileFS)(nil)
)

func (f *fileFS) Open(name string) (fs.File, error) {
	return f.root.Open(name)
}

func (f *fileFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(f.root, name)
}

func (f *fileFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(f.root, name)
}

func (f *fileFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(f.root, name)
}

func (f *fileFS) Glob(name string) ([]string, error) {
	return fs.Glob(f.root, name)
}

func (f *fileFS) Sub(name string) (fs.FS, error) {
	return fs.Sub(f.root, name)
}
