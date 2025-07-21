package os

import (
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hack-pad/hackpadfs"
)

const osPathOp = "ospath"

// ToOSPath converts a valid 'io/fs' package path to the equivalent 'os' package path for this FS
func (fs *FS) ToOSPath(fsPath string) (string, error) {
	osPath, err := fs.rootedPath(osPathOp, fsPath)
	if err != nil { // handle typed err
		return "", err
	}
	return osPath, nil
}

func (fs *FS) rootedPath(op, name string) (string, *hackpadfs.PathError) {
	return fs.toOSPath(runtime.GOOS, filepath.Separator, op, name)
}

func (fs *FS) toOSPath(goos string, separator rune, op, fsPath string) (string, *hackpadfs.PathError) {
	if !hackpadfs.ValidPath(fsPath) {
		return "", &hackpadfs.PathError{Op: op, Path: fsPath, Err: hackpadfs.ErrInvalid}
	}
	fsPath = path.Join("/", fs.root, fsPath)
	filePath := joinSepPath(string(separator), fs.getVolumeName(goos), fromSeparator(separator, fsPath))
	return filePath, nil
}

func joinSepPath(separator, elem1, elem2 string) string {
	elem1 = strings.TrimRight(elem1, separator)
	elem2 = strings.TrimLeft(elem2, separator)
	return elem1 + separator + elem2
}

func fromSeparator(separator rune, path string) string {
	if separator == '/' {
		return path
	}
	return strings.ReplaceAll(path, "/", string(separator))
}

func toSeparator(separator rune, path string) string {
	if separator == '/' {
		return path
	}
	return strings.ReplaceAll(path, string(separator), "/")
}

func (fs *FS) getVolumeName(goos string) string {
	if goos == goosWindows && fs.volumeName == "" {
		return `C:`
	}
	return fs.volumeName
}

// FromOSPath converts an absolute 'os' package path to the valid equivalent 'io/fs' package path for this FS.
//
// Returns an error for any of the following conditions:
//   - The path is not absolute.
//   - The path does not match fs's volume name set by SubVolume().
//   - The path does not share fs's root path set by Sub().
func (fs *FS) FromOSPath(osPath string) (string, error) {
	if !filepath.IsAbs(osPath) {
		return "", &hackpadfs.PathError{Op: osPathOp, Path: osPath, Err: hackpadfs.ErrInvalid}
	}
	return fs.fromOSPath(runtime.GOOS, filepath.Separator, filepath.VolumeName, osPathOp, osPath)
}

func (fs *FS) fromOSPath(
	goos string, separator rune, getVolumeName func(string) string,
	op, osPath string,
) (string, error) {
	errInvalid := &hackpadfs.PathError{Op: op, Path: osPath, Err: hackpadfs.ErrInvalid}
	fsVolumeName := fs.getVolumeName(goos)
	if getVolumeName(osPath) != fsVolumeName {
		return "", errInvalid
	}

	// remove volume name prefix
	osPath = strings.TrimPrefix(osPath, fsVolumeName)
	osPath = strings.TrimPrefix(osPath, string(separator))

	// remove root fs path prefix
	fsPath := toSeparator(separator, osPath)
	if fs.root != "" && fsPath != fs.root && !strings.HasPrefix(fsPath, fs.root+"/") {
		return "", errInvalid
	}
	fsPath = strings.TrimPrefix(fsPath, fs.root)
	fsPath = strings.TrimPrefix(fsPath, "/")

	if fsPath == "" {
		fsPath = "."
	}
	return fsPath, nil
}
