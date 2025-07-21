//go:build !windows
// +build !windows

package os

// wrapNonStandardErrors maps an operating system-specific error to a common type.
// Only implemented for built-in standard library os errors, no other custom FS errors.
func (fs *FS) wrapNonStandardErrors(err error) error {
	return err
}
