package gcpmetafs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/internal"
)

// withMetadataClienter is an fs.FS that can be configured to use the given Metadata
// client.
type withMetadataClienter interface {
	WithMetadataClient(client MetadataClient) fs.FS
}

// WithMetadataClientFS overrides the GCP Metadata client used by fs, if the filesystem
// supports it (i.e. has a WithMetadataClient method). This can be used for
// configuring specialized client options.
//
// Note that this should not be used together with [fsimpl.WithHTTPClientFS].
// If you wish only to override the HTTP client, use WithHTTPClient alone.
func WithMetadataClientFS(client MetadataClient, fsys fs.FS) fs.FS {
	if fsys, ok := fsys.(withMetadataClienter); ok {
		return fsys.WithMetadataClient(client)
	}

	return fsys
}

type gcpmetaFS struct {
	ctx        context.Context
	base       *url.URL
	httpclient *http.Client
	metaclient MetadataClient
	root       string
}

var (
	_ fs.FS                     = (*gcpmetaFS)(nil)
	_ fs.ReadDirFS              = (*gcpmetaFS)(nil)
	_ fs.SubFS                  = (*gcpmetaFS)(nil)
	_ internal.WithContexter    = (*gcpmetaFS)(nil)
	_ internal.WithHTTPClienter = (*gcpmetaFS)(nil)
	_ withMetadataClienter      = (*gcpmetaFS)(nil)
)

// New provides a filesystem (an fs.FS) backed by the GCP VM Metadata Service,
// rooted at the given URL.
//
// A context can be given by using WithContextFS.
func New(u *url.URL) (fs.FS, error) {
	if u.Scheme != "gcp+meta" {
		return nil, fmt.Errorf("invalid URL scheme %q", u.Scheme)
	}

	root := strings.TrimPrefix(u.Path, "/")

	return &gcpmetaFS{
		ctx:  context.Background(),
		base: u,
		root: root,
	}, nil
}

// FS is used to register this filesystem with an fsimpl.FSMux
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, "gcp+meta")

func (f gcpmetaFS) URL() string {
	return f.base.String()
}

func (f *gcpmetaFS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *gcpmetaFS) WithHTTPClient(client *http.Client) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.httpclient = client

	return &fsys
}

func (f *gcpmetaFS) WithMetadataClient(client MetadataClient) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.metaclient = client

	return &fsys
}

func (f *gcpmetaFS) getClient() MetadataClient {
	if f.metaclient != nil {
		return f.metaclient
	}

	f.metaclient = metadata.NewClient(f.httpclient)

	return f.metaclient
}

func (f *gcpmetaFS) Sub(name string) (fs.FS, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "sub", Path: name, Err: fs.ErrInvalid}
	}

	if name == "." || name == "" {
		return f, nil
	}

	fsys := *f
	fsys.root = path.Join(fsys.root, name)

	return &fsys, nil
}

func (f *gcpmetaFS) Open(name string) (fs.File, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	client := f.getClient()

	file := &gcpmetaFile{
		ctx:    f.ctx,
		name:   strings.TrimPrefix(path.Base(name), "."),
		root:   strings.TrimPrefix(path.Join(f.root, path.Dir(name)), "."),
		client: client,
	}

	if name == "." {
		file.fi = internal.DirInfo(file.name, time.Time{})

		return file, nil
	}

	return file, nil
}

func (f *gcpmetaFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	client := f.getClient()

	dir := &gcpmetaFile{
		ctx:    f.ctx,
		name:   name,
		root:   f.root,
		client: client,
		fi:     internal.DirInfo(name, time.Time{}),
	}

	des, err := dir.ReadDir(-1)
	if err != nil {
		return nil, &fs.PathError{Op: "readDir", Path: name, Err: err}
	}

	return des, nil
}

// errIsDirectory is a copy of EISDIR for our purposes
var errIsDirectory = errors.New("is a directory")

type gcpmetaFile struct {
	ctx    context.Context
	fi     fs.FileInfo
	client MetadataClient
	body   io.Reader
	name   string
	root   string

	children []*gcpmetaFile
	diroff   int
}

var _ fs.ReadDirFile = (*gcpmetaFile)(nil)

func (f *gcpmetaFile) Close() error {
	if closer, ok := f.body.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}

func (f *gcpmetaFile) Read(p []byte) (int, error) {
	if f.body == nil {
		err := f.getValue()
		if err != nil {
			return 0, &fs.PathError{Op: "read", Path: f.name, Err: err}
		}
	}

	if f.fi.IsDir() {
		return 0, fmt.Errorf("%w: %s", errIsDirectory, f.name)
	}

	return f.body.Read(p)
}

func (f *gcpmetaFile) Stat() (fs.FileInfo, error) {
	if f.fi != nil {
		return f.fi, nil
	}

	err := f.getValue()
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
	}

	return f.fi, nil
}

// isGuestAttributesDisabled checks if the error is due to guest attributes being disabled
func isGuestAttributesDisabled(err error) bool {
	var metaErr *metadata.Error

	return errors.As(err, &metaErr) &&
		metaErr.Code == http.StatusForbidden &&
		strings.Contains(metaErr.Message, "Guest attributes endpoint access is disabled")
}

// isTokenError checks if the error is due to token-related issues
func isTokenError(err *metadata.Error) bool {
	return err.Code == http.StatusBadRequest &&
		strings.Contains(err.Message, "non-empty audience parameter required")
}

// convertMetadataError converts a GCP Metadata error to an error suitable for returning
// from the package. We don't want to leak SDK error types.
func convertMetadataError(err error) error {
	if err == nil {
		return nil
	}

	// NotDefinedError is a 404
	var ndErr metadata.NotDefinedError
	if errors.As(err, &ndErr) {
		return fs.ErrNotExist
	}

	// 403s can also generally be treated as not existing
	if isGuestAttributesDisabled(err) {
		return fs.ErrNotExist
	}

	// For token-related errors, return invalid
	var metaErr *metadata.Error
	if errors.As(err, &metaErr) && isTokenError(metaErr) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, metaErr.Message)
	}

	return err
}

// getValue gets the value from the metadata service and populates body and fi.
func (f *gcpmetaFile) getValue() error {
	fullPath := path.Join(f.root, f.name)

	if isMetadataDirectory(fullPath) {
		f.fi = internal.DirInfo(f.name, time.Time{})

		return nil
	}

	data, err := f.client.GetWithContext(f.ctx, fullPath)
	if err != nil {
		return convertMetadataError(err)
	}

	f.fi, f.body = f.readData(f.name, data)

	return nil
}

func (f *gcpmetaFile) readData(name string, content string) (fs.FileInfo, io.Reader) {
	reader := strings.NewReader(content)
	size := int64(len(content))

	return internal.FileInfo(name, size, 0o444, time.Time{}, ""), reader
}

// listPrefix returns the prefix for this directory
func (f *gcpmetaFile) listPrefix() string {
	// when listing "." at the root (or opaque root), avoid returning "//"
	if f.name == "." && (f.root == "" || f.root == "/") {
		return f.root
	}

	// Use path.Join and ensure it ends with a slash
	prefix := path.Join(f.root, f.name)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return prefix
}

// list assigns a sorted list of the children of this directory to f.children
func (f *gcpmetaFile) list() error {
	fullPath := path.Join(f.root, f.name)

	// Handle special directories
	if fullPath == "" || fullPath == "/" || fullPath == "." {
		return f.listRoot()
	}

	if fullPath == "instance" {
		return f.listInstance()
	}

	if strings.HasPrefix(fullPath, "instance/service-accounts/") &&
		strings.Count(fullPath, "/") == 2 {
		return f.listServiceAccount()
	}

	// Fetch directory listing for other paths
	children, err := f.fetchDirectoryListing(fullPath)
	if err != nil {
		// For 403 errors related to guest attributes, return empty children
		if isGuestAttributesDisabled(err) {
			f.children = []*gcpmetaFile{}

			return nil
		}

		return err
	}

	return f.populateChildren(children)
}

// listRoot handles the root directory listing
func (f *gcpmetaFile) listRoot() error {
	f.children = []*gcpmetaFile{
		{
			ctx:    f.ctx,
			name:   "instance",
			root:   f.listPrefix(),
			client: f.client,
			fi:     internal.DirInfo("instance", time.Time{}),
		},
		{
			ctx:    f.ctx,
			name:   "project",
			root:   f.listPrefix(),
			client: f.client,
			fi:     internal.DirInfo("project", time.Time{}),
		},
	}

	return nil
}

// listInstance handles the instance directory listing
func (f *gcpmetaFile) listInstance() error {
	// Try to get the directory listing from the API first
	children, err := f.fetchDirectoryListing("instance")
	if err == nil && len(children) > 0 {
		// If we got a successful response, use it
		return f.populateChildren(children)
	}

	// Fallback to hardcoded paths if API fails or returns empty
	knownPaths := []string{
		"id", "name", "hostname", "zone", "machine-type", "description",
		"tags", "scheduling", "network-interfaces", "disks", "licenses",
		"cpu-platform", "maintenance-event", "service-accounts",
		"attributes", "image", "preempted",
	}

	return f.createChildrenFromPaths(knownPaths)
}

// listServiceAccount handles service account directory listings
func (f *gcpmetaFile) listServiceAccount() error {
	knownPaths := []string{
		"aliases", "email", "scopes",
	}

	return f.createChildrenFromPaths(knownPaths)
}

// createChildrenFromPaths creates children from a list of known paths
func (f *gcpmetaFile) createChildrenFromPaths(paths []string) error {
	children := make([]string, 0, len(paths))
	prefix := f.listPrefix()

	for _, entry := range paths {
		children = append(children, path.Join(prefix, entry))
	}

	return f.populateChildren(children)
}

// fetchDirectoryListing gets the directory listing from the metadata service
func (f *gcpmetaFile) fetchDirectoryListing(fullPath string) ([]string, error) {
	// Normalize path and ensure it ends with a slash for directory listing
	requestPath := strings.TrimPrefix(fullPath, "/")
	if !strings.HasSuffix(requestPath, "/") {
		requestPath += "/"
	}

	// Get directory listing
	data, err := f.client.GetWithContext(f.ctx, requestPath)
	if err != nil {
		var metaErr *metadata.Error
		if errors.As(err, &metaErr) && isTokenError(metaErr) {
			// For token-related errors, return nil to indicate no children
			return nil, nil
		}

		// Pass through other errors to be handled by the caller
		return nil, fmt.Errorf("getDirectoryListing %q: %w", requestPath, err)
	}

	// Process entries
	entries := []string{}

	lines := strings.SplitSeq(data, "\n")
	for line := range lines {
		if line != "" {
			entries = append(entries, line)
		}
	}

	// Generate full paths
	prefix := f.listPrefix()

	children := make([]string, 0, len(entries))
	for _, entry := range entries {
		// Use path.Join for consistent path handling
		children = append(children, path.Join(prefix, entry))
	}

	return children, nil
}

// populateChildren creates a list of children from the given list of names,
// sorts and assigns it to f.children
func (f *gcpmetaFile) populateChildren(children []string) error {
	// track files that we've already seen - we don't want to add duplicates
	seen := map[string]struct{}{}
	f.children = []*gcpmetaFile{} // Initialize to empty slice

	for _, entry := range children {
		entry = strings.Trim(entry, "/")
		parts := strings.Split(entry, "/")
		name := parts[len(parts)-1]

		if _, ok := seen[name]; ok {
			continue
		}

		seen[name] = struct{}{}

		child := gcpmetaFile{
			ctx:    f.ctx,
			name:   name,
			root:   path.Join(f.root, f.name),
			client: f.client,
		}

		// For known paths in special directories, we can determine if it's a directory
		// without making a request
		if isMetadataDirectory(path.Join(child.root, child.name)) {
			child.fi = internal.DirInfo(name, time.Time{})
		} else {
			// Try to stat the file, but don't fail if it doesn't exist
			fi, err := child.Stat()
			if err != nil {
				// Skip this child if it doesn't exist
				if errors.Is(err, fs.ErrNotExist) {
					continue
				}

				return fmt.Errorf("stat %q: %w", name, err)
			}

			child.fi = fi
		}

		f.children = append(f.children, &child)
	}

	// Sort the list of children
	sort.Slice(f.children, func(i, j int) bool {
		return f.children[i].name < f.children[j].name
	})

	return nil
}

// ReadDir implements fs.ReadDirFile
func (f *gcpmetaFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.children == nil {
		if err := f.list(); err != nil {
			return nil, fmt.Errorf("list: %w", err)
		}
	}

	if n > 0 && f.diroff >= len(f.children) {
		return nil, io.EOF
	}

	low := f.diroff
	high := f.diroff + n

	// clamp high at the max, and ensure it's higher than low
	if high >= len(f.children) || high <= low {
		high = len(f.children)
	}

	entries := make([]fs.DirEntry, high-low)
	for i := low; i < high; i++ {
		entries[i-low] = internal.FileInfoDirEntry(f.children[i].fi)
	}

	f.diroff = high

	return entries, nil
}
