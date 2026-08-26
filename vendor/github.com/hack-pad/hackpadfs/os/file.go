package os

import (
	"io"
	"os"
	"time"

	"github.com/hack-pad/hackpadfs"
)

type file struct {
	fs     *FS
	osFile *os.File
}

func (fs *FS) wrapFile(f *os.File) hackpadfs.File {
	return &file{fs: fs, osFile: f}
}

// Chmod implements hackpadfs.ChmoderFile
func (f *file) Chmod(mode hackpadfs.FileMode) error {
	return f.fs.wrapErr(f.osFile.Chmod(mode))
}

// Chown implements hackpadfs.ChownerFile
func (f *file) Chown(uid, gid int) error {
	return f.fs.wrapErr(f.osFile.Chown(uid, gid))
}

func (f *file) Close() error {
	return f.fs.wrapErr(f.osFile.Close())
}

// Name returns this file's name.
func (f *file) Name() string {
	return f.osFile.Name()
}

func (f *file) Read(b []byte) (n int, err error) {
	n, err = f.osFile.Read(b)
	return n, f.fs.wrapErr(err)
}

func (f *file) ReadAt(b []byte, off int64) (n int, err error) {
	n, err = f.osFile.ReadAt(b, off)
	return n, f.fs.wrapErr(err)
}

func (f *file) ReadDir(n int) ([]hackpadfs.DirEntry, error) {
	entries, err := f.osFile.ReadDir(n)
	return entries, f.fs.wrapErr(err)
}

func (f *file) ReadFrom(r io.Reader) (n int64, err error) {
	n, err = f.osFile.ReadFrom(r)
	return n, f.fs.wrapErr(err)
}

// Seek implements hackpadfs.SeekerFile
func (f *file) Seek(offset int64, whence int) (ret int64, err error) {
	ret, err = f.osFile.Seek(offset, whence)
	return ret, f.fs.wrapErr(err)
}

func (f *file) SetDeadline(t time.Time) error {
	return f.fs.wrapErr(f.osFile.SetDeadline(t))
}

func (f *file) SetReadDeadline(t time.Time) error {
	return f.fs.wrapErr(f.osFile.SetReadDeadline(t))
}

func (f *file) SetWriteDeadline(t time.Time) error {
	return f.fs.wrapErr(f.osFile.SetWriteDeadline(t))
}

// Stat implements hackpadfs.StaterFile
func (f *file) Stat() (hackpadfs.FileInfo, error) {
	info, err := f.osFile.Stat()
	return info, f.fs.wrapErr(err)
}

// Sync implements hackpadfs.SycnerFile
func (f *file) Sync() error {
	return f.fs.wrapErr(f.osFile.Sync())
}

// Truncate implements hackpadfs.TruncaterFile
func (f *file) Truncate(size int64) error {
	return f.fs.wrapErr(f.osFile.Truncate(size))
}

func (f *file) Write(b []byte) (n int, err error) {
	n, err = f.osFile.Write(b)
	return n, f.fs.wrapErr(err)
}

func (f *file) WriteAt(b []byte, off int64) (n int, err error) {
	n, err = f.osFile.WriteAt(b, off)
	return n, f.fs.wrapErr(err)
}

func (f *file) WriteString(s string) (n int, err error) {
	n, err = f.osFile.WriteString(s)
	return n, f.fs.wrapErr(err)
}
