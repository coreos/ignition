package billyadapter

import (
	"io/fs"
	"time"

	"github.com/go-git/go-billy/v5"
)

// FrozenModTimeFilesystem is a workaround for annoying behaviour in billy's
// memfs FileInfo implementation, which always returns time.Now() on calls to
// ModTime().
func FrozenModTimeFilesystem(fs billy.Filesystem, modTime time.Time) billy.Filesystem {
	return &frozenModTimeFilesystem{fs, modTime}
}

type frozenModTimeFilesystem struct {
	billy.Filesystem
	modTime time.Time
}

var _ billy.Filesystem = (*frozenModTimeFilesystem)(nil)

func (f *frozenModTimeFilesystem) Stat(name string) (fs.FileInfo, error) {
	fi, err := f.Filesystem.Stat(name)
	if err != nil {
		return nil, err
	}

	fi = &frozenModTimeFileInfo{fi, f.modTime}

	return fi, err
}

func (f *frozenModTimeFilesystem) ReadDir(name string) ([]fs.FileInfo, error) {
	fis, err := f.Filesystem.ReadDir(name)
	if err != nil {
		return nil, err
	}

	for i, fi := range fis {
		fi = &frozenModTimeFileInfo{fi, f.modTime}
		fis[i] = fi
	}

	return fis, err
}

type frozenModTimeFileInfo struct {
	fs.FileInfo
	modTime time.Time
}

var _ fs.FileInfo = (*frozenModTimeFileInfo)(nil)

func (fi *frozenModTimeFileInfo) ModTime() time.Time {
	return fi.modTime
}
