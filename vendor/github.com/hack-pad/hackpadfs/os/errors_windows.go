//go:build windows
// +build windows

package os

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/hack-pad/hackpadfs"
)

// wrapNonStandardErrors maps an operating system-specific error to a common type.
// Only implemented for built-in standard library os errors, no other custom FS errors.
func (fs *FS) wrapNonStandardErrors(err error) error {
	switch e := err.(type) {
	case *hackpadfs.PathError:
		errCopy := *e
		errCopy.Err = fs.mapNonStandardError(errCopy.Err)
		err = &errCopy
	case *hackpadfs.LinkError:
		errCopy := *e
		errCopy.Err = fs.mapNonStandardError(errCopy.Err)
		err = &errCopy
	default:
		err = fs.mapNonStandardError(err)
	}
	return err
}

func (fs *FS) mapNonStandardError(err error) error {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return err
	}
	// Values from https://docs.microsoft.com/en-us/windows/win32/debug/system-error-codes--0-499-
	const (
		ERROR_NEGATIVE_SEEK = syscall.Errno(0x83)
		ERROR_DIR_NOT_EMPTY = syscall.Errno(0x91)
	)
	switch errno {
	case ERROR_NEGATIVE_SEEK:
		return &mappedErr{hackpadfs.ErrInvalid, errno}
	case ERROR_DIR_NOT_EMPTY:
		return &mappedErr{hackpadfs.ErrNotEmpty, errno}
	default:
		return err
	}
}

type mappedErr struct {
	normalized error
	original   error
}

func (m *mappedErr) Error() string {
	return fmt.Sprintf("%s: %s", m.normalized.Error(), m.original.Error())
}

func (m *mappedErr) Is(err error) bool {
	return errors.Is(err, m.normalized)
}

func (m *mappedErr) Unwrap() error {
	return m.original
}
