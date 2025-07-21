package hackpadfs

import (
	"io/fs"
	"syscall"
)

// Errors commonly returned by file systems. Mirror their equivalents in the syscall and io/fs packages.
var (
	ErrInvalid    = syscall.EINVAL // TODO update to fs.ErrInvalid, once errors.Is supports it
	ErrPermission = fs.ErrPermission
	ErrExist      = fs.ErrExist
	ErrNotExist   = fs.ErrNotExist
	ErrClosed     = fs.ErrClosed

	ErrIsDir          = syscall.EISDIR
	ErrNotDir         = syscall.ENOTDIR
	ErrNotEmpty       = syscall.ENOTEMPTY
	ErrNotImplemented = syscall.ENOSYS

	SkipDir = fs.SkipDir
)

// PathError records a file system or file operation error and the path that caused it. Mirrors io/fs.PathError
type PathError = fs.PathError

// LinkError records a file system rename error and the paths that caused it. Mirrors os.LinkError
//
// NOTE: Is not identical to os.LinkError to avoid importing "os". Still resolves errors.Is() calls correctly.
type LinkError struct {
	Err error
	Op  string
	Old string
	New string
}

func (e *LinkError) Error() string {
	return e.Op + " " + e.Old + " " + e.New + ": " + e.Err.Error()
}

// Unwrap supports errors.Unwrap().
func (e *LinkError) Unwrap() error {
	return e.Err
}
