package hackpadfs

import (
	"io"
	gofs "io/fs"
	"syscall"
	"time"
)

// Flags are bit-wise OR'd with each other in fs.OpenFile().
// Exactly one of Read/Write flags must be specified, and any other flags can be OR'd together.
const (
	FlagReadOnly  int = syscall.O_RDONLY
	FlagWriteOnly int = syscall.O_WRONLY
	FlagReadWrite int = syscall.O_RDWR

	FlagAppend    int = syscall.O_APPEND
	FlagCreate    int = syscall.O_CREAT
	FlagExclusive int = syscall.O_EXCL
	FlagSync      int = syscall.O_SYNC
	FlagTruncate  int = syscall.O_TRUNC
)

// FileMode represents a file's mode and permission bits. Mirrors io/fs.FileMode.
type FileMode = gofs.FileMode

// Mode values are bit-wise OR'd with a file's permissions to form the FileMode. Mirror io/fs.Mode... values.
const (
	ModeDir        = gofs.ModeDir
	ModeAppend     = gofs.ModeAppend
	ModeExclusive  = gofs.ModeExclusive
	ModeTemporary  = gofs.ModeTemporary
	ModeSymlink    = gofs.ModeSymlink
	ModeDevice     = gofs.ModeDevice
	ModeNamedPipe  = gofs.ModeNamedPipe
	ModeSocket     = gofs.ModeSocket
	ModeSetuid     = gofs.ModeSetuid
	ModeSetgid     = gofs.ModeSetgid
	ModeCharDevice = gofs.ModeCharDevice
	ModeSticky     = gofs.ModeSticky
	ModeIrregular  = gofs.ModeIrregular

	ModeType = gofs.ModeType
	ModePerm = gofs.ModePerm
)

// FileInfo describes a file and is returned by Stat(). Mirrors io/fs.FileInfo.
type FileInfo = gofs.FileInfo

// DirEntry is an entry read from a directory. Mirrors io/fs.DirEntry.
type DirEntry = gofs.DirEntry

// File provides access to a file. Mirrors io/fs.File.
type File = gofs.File

// ReadWriterFile is a File that supports Write() operations.
type ReadWriterFile interface {
	File
	io.Writer
}

// ReaderAtFile is a File that supports ReadAt() operations.
type ReaderAtFile interface {
	File
	io.ReaderAt
}

// WriterAtFile is a File that supports WriteAt() operations.
type WriterAtFile interface {
	File
	io.WriterAt
}

// DirReaderFile is a File that supports ReadDir() operations. Mirrors io/fs.ReadDirFile.
type DirReaderFile interface {
	File
	ReadDir(n int) ([]DirEntry, error)
}

// SeekerFile is a File that supports Seek() operations.
type SeekerFile interface {
	File
	io.Seeker
}

// SyncerFile is a File that supports Sync() operations.
type SyncerFile interface {
	File
	Sync() error
}

// TruncaterFile is a File that supports Truncate() operations.
type TruncaterFile interface {
	File
	Truncate(size int64) error
}

// ChmoderFile is a File that supports Chmod() operations.
type ChmoderFile interface {
	File
	Chmod(mode FileMode) error
}

// ChownerFile is a File that supports Chown() operations.
type ChownerFile interface {
	File
	Chown(uid, gid int) error
}

// ChtimeserFile is a File that supports Chtimes() operations.
type ChtimeserFile interface {
	File
	Chtimes(atime time.Time, mtime time.Time) error
}

// ChmodFile runs file.Chmod() is available, fails with a not implemented error otherwise.
func ChmodFile(file File, mode FileMode) error {
	if file, ok := file.(ChmoderFile); ok {
		return file.Chmod(mode)
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return &PathError{Op: "chmod", Path: info.Name(), Err: ErrNotImplemented}
}

// ChownFile runs file.Chown() is available, fails with a not implemented error otherwise.
func ChownFile(file File, uid, gid int) error {
	if file, ok := file.(ChownerFile); ok {
		return file.Chown(uid, gid)
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return &PathError{Op: "chmod", Path: info.Name(), Err: ErrNotImplemented}
}

// ChtimesFile runs file.Chtimes() is available, fails with a not implemented error otherwise.
func ChtimesFile(file File, atime, mtime time.Time) error {
	if file, ok := file.(ChtimeserFile); ok {
		return file.Chtimes(atime, mtime)
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return &PathError{Op: "chtimes", Path: info.Name(), Err: ErrNotImplemented}
}

// ReadAtFile runs file.ReadAt() is available, fails with a not implemented error otherwise.
func ReadAtFile(file File, p []byte, off int64) (n int, err error) {
	if file, ok := file.(ReaderAtFile); ok {
		return file.ReadAt(p, off)
	}
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return 0, &PathError{Op: "readat", Path: info.Name(), Err: ErrNotImplemented}
}

// WriteFile runs file.Write() is available, fails with a not implemented error otherwise.
func WriteFile(file File, p []byte) (n int, err error) {
	if file, ok := file.(ReadWriterFile); ok {
		return file.Write(p)
	}
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return 0, &PathError{Op: "write", Path: info.Name(), Err: ErrNotImplemented}
}

// WriteAtFile runs file.WriteAt() is available, fails with a not implemented error otherwise.
func WriteAtFile(file File, p []byte, off int64) (n int, err error) {
	if file, ok := file.(WriterAtFile); ok {
		return file.WriteAt(p, off)
	}
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return 0, &PathError{Op: "writeat", Path: info.Name(), Err: ErrNotImplemented}
}

// ReadDirFile runs file.ReadDir() is available, fails with a not implemented error otherwise.
func ReadDirFile(file File, n int) ([]DirEntry, error) {
	if file, ok := file.(DirReaderFile); ok {
		return file.ReadDir(n)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return nil, &PathError{Op: "readdir", Path: info.Name(), Err: ErrNotImplemented}
}

// SeekFile runs file.Seek() is available, fails with a not implemented error otherwise.
func SeekFile(file File, offset int64, whence int) (int64, error) {
	if file, ok := file.(SeekerFile); ok {
		return file.Seek(offset, whence)
	}
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return 0, &PathError{Op: "seek", Path: info.Name(), Err: ErrNotImplemented}
}

// SyncFile runs file.Sync() is available, fails with a not implemented error otherwise.
func SyncFile(file File) error {
	if file, ok := file.(SyncerFile); ok {
		return file.Sync()
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return &PathError{Op: "sync", Path: info.Name(), Err: ErrNotImplemented}
}

// TruncateFile runs file.Truncate() is available, fails with a not implemented error otherwise.
func TruncateFile(file File, size int64) error {
	if file, ok := file.(TruncaterFile); ok {
		return file.Truncate(size)
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return &PathError{Op: "truncate", Path: info.Name(), Err: ErrNotImplemented}
}
