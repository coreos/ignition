package hackpadfs

import "strings"

func stripErrPathPrefix(err error, name, mountSubPath string) error {
	if err == nil {
		return err
	}
	prefix := strings.TrimSuffix(mountSubPath, name)
	switch err := err.(type) {
	case *PathError:
		return &PathError{
			Op:   err.Op,
			Path: strings.TrimPrefix(err.Path, prefix),
			Err:  err.Err,
		}
	case *LinkError:
		return &LinkError{
			Op:  err.Op,
			Old: strings.TrimPrefix(err.Old, prefix),
			New: strings.TrimPrefix(err.New, prefix),
			Err: err.Err,
		}
	default:
		return err
	}
}
