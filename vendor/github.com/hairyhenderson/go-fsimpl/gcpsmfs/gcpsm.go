package gcpsmfs

import (
	"bytes"
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

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/internal"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// withSMClienter is an fs.FS that can be configured to use the given Secrets
// Manager client.
type withSMClienter interface {
	WithSMClient(smclient SecretManagerClient) fs.FS
}

// WithSMClientFS overrides the GCP Secrets Manager client used by fs, if the
// filesystem supports it (i.e. has a WithSMClient method). This can be used for
// configuring specialized client options.
//
// Note that this should not be used together with WithHTTPClient. If you wish
// only to override the HTTP client, use WithHTTPClient alone.
func WithSMClientFS(smclient SecretManagerClient, fsys fs.FS) fs.FS {
	if fsys, ok := fsys.(withSMClienter); ok {
		return fsys.WithSMClient(smclient)
	}

	return fsys
}

type gcpsmFS struct {
	ctx        context.Context
	smclient   SecretManagerClient
	base       *url.URL
	httpclient *http.Client
	project    string
}

// New provides a filesystem (an fs.FS) backed by the GCP Secret Manager,
// rooted at the given URL.
//
// The URL should use the "gcp+sm" scheme and one of the following formats:
//   - "gcp+sm:///projects/<project-id>" to set an explicit project context
//   - "gcp+sm:///" for no project context
//
// A context can be given by using WithContextFS.
func New(u *url.URL) (fs.FS, error) {
	if u.Scheme != "gcp+sm" {
		return nil, fmt.Errorf("invalid URL scheme %q", u.Scheme)
	}

	f := &gcpsmFS{
		ctx:  context.Background(),
		base: u,
	}

	// Normalize the path and validate it matches one of the supported forms:
	//   - "/" for no project context
	//   - "/projects/<project-id>" (optionally with a trailing slash)
	cleanPath := path.Clean(u.Path)
	// path.Clean("") returns ".", so treat that as no project context as well.
	if cleanPath == "." || cleanPath == "/" {
		// No project context.
		return f, nil
	}

	if project, found := strings.CutPrefix(cleanPath, "/projects/"); found {
		// Reject paths with extra segments like "/projects/p/secrets/foo".
		if project == "" || strings.Contains(project, "/") {
			return nil, fmt.Errorf("invalid gcp+sm URL path %q: expected /projects/<project-id> or /", u.Path)
		}

		f.project = project

		return f, nil
	}

	return nil, fmt.Errorf("invalid gcp+sm URL path %q: expected /projects/<project-id> or /", u.Path)
}

// FS is used to register this filesystem with an fsimpl.FSMux
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, "gcp+sm")

var (
	_ fs.FS                     = (*gcpsmFS)(nil)
	_ fs.ReadFileFS             = (*gcpsmFS)(nil)
	_ fs.ReadDirFS              = (*gcpsmFS)(nil)
	_ internal.WithContexter    = (*gcpsmFS)(nil)
	_ internal.WithHTTPClienter = (*gcpsmFS)(nil)
	_ withSMClienter            = (*gcpsmFS)(nil)
)

func (f gcpsmFS) URL() string {
	return f.base.String()
}

func (f *gcpsmFS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *gcpsmFS) WithHTTPClient(client *http.Client) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.httpclient = client

	return &fsys
}

func (f *gcpsmFS) WithSMClient(smclient SecretManagerClient) fs.FS {
	if smclient == nil {
		return f
	}

	fsys := *f
	fsys.smclient = smclient

	return &fsys
}

func (f *gcpsmFS) getClient() (SecretManagerClient, error) {
	if f.smclient != nil {
		return f.smclient, nil
	}

	opts := []option.ClientOption{}
	if f.httpclient != nil {
		opts = append(opts, option.WithHTTPClient(f.httpclient))
	}

	c, err := secretmanager.NewClient(f.ctx, opts...)
	if err != nil {
		return nil, err
	}

	f.smclient = &clientAdapter{c}

	return f.smclient, nil
}

func (f *gcpsmFS) getProjectAndFileName(name string) (string, string, error) {
	// First, assume that the project is in the FS definition, not the path name
	project := f.project
	fileName := name

	// If no project is given by the FS, it must be in the file name, and must be extracted
	if project == "" {
		parts := strings.Split(name, "/")
		if len(parts) != 4 || parts[0] != "projects" || parts[2] != "secrets" {
			return "", "", &fs.PathError{Op: "getProjectAndFileName", Path: name, Err: fs.ErrInvalid}
		}

		project = parts[1]
		if project == "" {
			return "", "", &fs.PathError{Op: "getProjectAndFileName", Path: name, Err: fs.ErrInvalid}
		}

		fileName = strings.TrimPrefix(path.Base(parts[3]), ".")
	}

	if strings.Contains(fileName, "/") {
		return "", "", &fs.PathError{Op: "getProjectAndFileName", Path: name, Err: fs.ErrInvalid}
	}

	return project, fileName, nil
}

func (f *gcpsmFS) Open(name string) (fs.File, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	client, err := f.getClient()
	if err != nil {
		return nil, err
	}

	project, fileName, err := f.getProjectAndFileName(name)
	if err != nil {
		return nil, err
	}

	file := &gcpsmFile{
		ctx:     f.ctx,
		name:    fileName,
		project: project,
		client:  client,
	}

	if fileName == "." {
		return file, nil
	}

	return file, nil
}

func (f *gcpsmFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	if name != "." {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}

	client, err := f.getClient()
	if err != nil {
		return nil, err
	}

	project := f.project

	if project == "" {
		return nil, errors.New("listing secrets requires a project in the URL (e.g. gcp+sm:///projects/<project-id>)")
	}

	dir := &gcpsmFile{
		ctx:     f.ctx,
		name:    name,
		project: project,
		client:  client,
	}

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}

	return entries, nil
}

func (f *gcpsmFS) ReadFile(name string) ([]byte, error) {
	file, err := f.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

type gcpsmFile struct {
	ctx     context.Context
	name    string
	project string
	client  SecretManagerClient

	// modTime is set by ensureModTime after GetSecretVersion; nil means not loaded yet.
	modTime *time.Time
	body    io.Reader
	content []byte

	children []gcpsmFile
	diroff   int
}

var _ fs.ReadDirFile = (*gcpsmFile)(nil)

func (f *gcpsmFile) Close() error {
	return nil
}

func (f *gcpsmFile) Read(p []byte) (int, error) {
	if err := f.loadContent(); err != nil {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: err}
	}

	return f.body.Read(p)
}

func (f *gcpsmFile) getResourceName() string {
	return fmt.Sprintf("projects/%s/secrets/%s/versions/latest", f.project, f.name)
}

// fileInfo returns metadata for this handle. Directories (name ".") need no fetch;
// secret files must have loaded content and modTime (see loadContent / ensureModTime).
func (f *gcpsmFile) fileInfo() fs.FileInfo {
	if f.name == "." {
		return internal.DirInfo(f.name, time.Time{})
	}

	mt := time.Time{}
	if f.modTime != nil {
		mt = *f.modTime
	}

	return internal.FileInfo(f.name, int64(len(f.content)), 0o444, mt, "")
}

func (f *gcpsmFile) Stat() (fs.FileInfo, error) {
	if f.name == "." {
		return f.fileInfo(), nil
	}

	if err := f.loadContent(); err != nil {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
	}

	if err := f.ensureModTime(); err != nil {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
	}

	return f.fileInfo(), nil
}

// loadContent fetches the secret payload via AccessSecretVersion (one RPC when not cached).
func (f *gcpsmFile) loadContent() error {
	if f.content != nil {
		return nil
	}

	resourceName := f.getResourceName()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: resourceName,
	}

	resp, err := f.client.AccessSecretVersion(f.ctx, req)
	if err != nil {
		return convertGCPError(err)
	}

	var payload []byte
	if resp.Payload != nil {
		payload = resp.Payload.Data
	}

	if payload == nil {
		payload = []byte{}
	}

	f.content = payload
	f.body = bytes.NewReader(f.content)

	return nil
}

// ensureModTime loads version metadata via GetSecretVersion (one RPC when not cached).
func (f *gcpsmFile) ensureModTime() error {
	if f.modTime != nil {
		return nil
	}

	resourceName := f.getResourceName()

	getReq := &secretmanagerpb.GetSecretVersionRequest{
		Name: resourceName,
	}

	getResp, err := f.client.GetSecretVersion(f.ctx, getReq)
	if err != nil {
		return convertGCPError(err)
	}

	t := time.Time{}
	if getResp != nil && getResp.GetCreateTime() != nil {
		t = getResp.GetCreateTime().AsTime()
	}

	f.modTime = &t

	return nil
}

func (f *gcpsmFile) ReadDir(n int) ([]fs.DirEntry, error) {
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
		entries[i-low] = internal.FileInfoDirEntry(f.children[i].fileInfo())
	}

	f.diroff = high

	return entries, nil
}

func (f *gcpsmFile) list() error {
	parent := "projects/" + f.project
	req := &secretmanagerpb.ListSecretsRequest{
		Parent: parent,
	}

	it := f.client.ListSecrets(f.ctx, req)

	var entries []gcpsmFile

	for {
		secret, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return convertGCPError(err)
		}

		// Name is full resource name: projects/{project}/secrets/{name}
		parts := strings.Split(secret.Name, "/")
		name := parts[len(parts)-1]

		child := gcpsmFile{
			ctx:     f.ctx,
			name:    name,
			project: f.project,
			client:  f.client,
		}

		err = child.loadContent()
		if err != nil {
			return fmt.Errorf("while fetching secret %s: %w", name, err)
		}

		err = child.ensureModTime()
		if err != nil {
			return fmt.Errorf("while fetching secret %s: %w", name, err)
		}

		entries = append(entries, child)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	f.children = entries

	return nil
}

func convertGCPError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	switch st.Code() { //nolint:exhaustive
	case codes.NotFound:
		return fmt.Errorf("%w: %s", fs.ErrNotExist, st.Message())
	case codes.PermissionDenied:
		return fmt.Errorf("%w: %s", fs.ErrPermission, st.Message())
	case codes.Unauthenticated:
		return fmt.Errorf("%w: %s", fs.ErrPermission, st.Message())
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %s", fs.ErrInvalid, st.Message())
	default:
		return err
	}
}
