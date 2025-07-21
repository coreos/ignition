package internal

import (
	"context"
	"io/fs"
	"net/http"
	"time"
)

// A few convenience functions and types intended for use only inside this
// module for now.

// FileInfo creates a static fs.FileInfo with the given properties.
// The result is also a fs.DirEntry and can be safely cast.
func FileInfo(name string, size int64, mode fs.FileMode, modTime time.Time, contentType string) fs.FileInfo {
	return &staticFileInfo{
		name:        name,
		size:        size,
		mode:        mode,
		modTime:     modTime,
		contentType: contentType,
	}
}

// DirInfo creates a fs.FileInfo for a directory with the given name. Use
// FileInfo to set other values.
func DirInfo(name string, modTime time.Time) fs.FileInfo {
	return FileInfo(name, 0, fs.ModeDir|0o555, modTime, "")
}

type staticFileInfo struct {
	modTime     time.Time
	name        string
	contentType string
	size        int64
	mode        fs.FileMode
}

var (
	_ fs.FileInfo         = (*staticFileInfo)(nil)
	_ fs.DirEntry         = (*staticFileInfo)(nil)
	_ ContentTypeFileInfo = (*staticFileInfo)(nil)
)

func (fi staticFileInfo) ContentType() string         { return fi.contentType }
func (fi staticFileInfo) IsDir() bool                 { return fi.Mode().IsDir() }
func (fi staticFileInfo) Mode() fs.FileMode           { return fi.mode }
func (fi *staticFileInfo) ModTime() time.Time         { return fi.modTime }
func (fi staticFileInfo) Name() string                { return fi.name }
func (fi staticFileInfo) Size() int64                 { return fi.size }
func (fi staticFileInfo) Sys() any                    { return nil }
func (fi *staticFileInfo) Info() (fs.FileInfo, error) { return fi, nil }
func (fi staticFileInfo) Type() fs.FileMode           { return fi.Mode().Type() }

// FileInfoDirEntry adapts a fs.FileInfo into a fs.DirEntry. If it doesn't
// already implement fs.DirEntry, it will be wrapped to always return the
// same fs.FileInfo.
func FileInfoDirEntry(fi fs.FileInfo) fs.DirEntry {
	de, ok := fi.(fs.DirEntry)
	if ok {
		return de
	}

	return &fileinfoDirEntry{fi}
}

// a wrapper to make a fs.FileInfo into an fs.DirEntry
type fileinfoDirEntry struct {
	fs.FileInfo
}

var _ fs.DirEntry = (*fileinfoDirEntry)(nil)

func (fi *fileinfoDirEntry) Info() (fs.FileInfo, error) { return fi, nil }
func (fi *fileinfoDirEntry) Type() fs.FileMode          { return fi.Mode().Type() }

// ContentTypeFileInfo is an fs.FileInfo that can also report its content type
type ContentTypeFileInfo interface {
	fs.FileInfo

	ContentType() string
}

// WithContexter is an fs.FS that can be configured with a custom context
type WithContexter interface {
	WithContext(ctx context.Context) fs.FS
}

// WithHTTPClienter is an fs.FS that can be configured with a custom http.Client
type WithHTTPClienter interface {
	WithHTTPClient(client *http.Client) fs.FS
}

// WithHeaderer is an fs.FS that can be configured to send a custom http.Header
type WithHeaderer interface {
	WithHeader(headers http.Header) fs.FS
}

// WithIMDSFSer overrides the IMDS filesystem used by fs, if the filesystem
// supports it (i.e. has a WithIMDSFS method). This can be used for overriding
// the IMDS filesystem used by the fs to discover the region.
type WithIMDSFSer interface {
	WithIMDSFS(imdsfs fs.FS) fs.FS
}
