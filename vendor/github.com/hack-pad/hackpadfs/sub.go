package hackpadfs

import (
	"path"
)

var _ interface {
	FS
	MountFS
} = &subFS{}

type subFS struct {
	rootFS   FS
	basePath string
}

func newSubFS(fs FS, dir string) (FS, error) {
	if !ValidPath(dir) {
		return nil, &PathError{Op: "sub", Path: dir, Err: ErrInvalid}
	}
	return &subFS{
		basePath: dir,
		rootFS:   fs,
	}, nil
}

func (fs *subFS) Open(name string) (File, error) {
	if !ValidPath(name) {
		return nil, &PathError{Op: "open", Path: name, Err: ErrInvalid}
	}
	mount, subPath := fs.Mount(name)
	file, err := mount.Open(subPath)
	return file, stripErrPathPrefix(err, name, subPath)
}

func (fs *subFS) Mount(p string) (mount FS, subPath string) {
	if !ValidPath(p) {
		return fs.rootFS, p
	}
	return fs.rootFS, path.Join(fs.basePath, p)
}
