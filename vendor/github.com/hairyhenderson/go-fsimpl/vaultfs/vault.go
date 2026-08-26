package vaultfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/internal"
	"github.com/hairyhenderson/go-fsimpl/vaultfs/vaultauth"
	"github.com/hashicorp/vault/api"
)

type vaultFS struct {
	ctx  context.Context
	base *url.URL
	auth api.AuthMethod

	client *refCountedClient
}

// New creates a filesystem for the Vault endpoint rooted at u.
//
// It is especially important to make sure that opened files are closed,
// otherwise a Vault token may be leaked!
//
// The filesystem may be configured with:
//
//   - [vaultauth.WithAuthMethod] (set the auth method)
//   - [fsimpl.WithContextFS] (inject a context)
//   - [fsimpl.WithHeaderFS] (inject custom HTTP headers)
func New(u *url.URL) (fs.FS, error) {
	if u == nil {
		return nil, errors.New("url must not be nil")
	}

	if u.Path == "" {
		u.Path = "/"
	}

	if !strings.HasSuffix(u.Path, "/") {
		return nil, fmt.Errorf("invalid url path %q must be a prefix ending with \"/\"", u)
	}

	config, err := vaultConfig(u)
	if err != nil {
		return nil, fmt.Errorf("vault configuration error: %w", err)
	}

	c, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("vault client creation failed: %w", err)
	}

	fsys := newWithVaultClient(u, nil)
	fsys = WithClient(c, fsys).(*vaultFS)
	fsys.auth = vaultauth.NewTokenAuth("")

	return fsys, nil
}

func newWithVaultClient(u *url.URL, client *refCountedClient) *vaultFS {
	base := *u
	if base.Path == "" || base.Path == "/" {
		base.Path = "/v1/"
	} else if !strings.HasPrefix(base.Path, "/v1") {
		base.Path = path.Join("/v1", path.Dir(base.Path)) + "/"
	}

	return &vaultFS{
		ctx:    context.Background(),
		client: client,
		base:   &base,
	}
}

func vaultConfig(u *url.URL) (*api.Config, error) {
	config := api.DefaultConfig()
	if config.Error != nil {
		return nil, config.Error
	}

	// handle compound URL scheme not supported by the client, but only if the
	// URL has a host part set - otherwise use the scheme from $VAULT_ADDR, as
	// set by api.DefaultConfig() above
	if u.Host != "" {
		scheme := strings.TrimPrefix(u.Scheme, "vault+")
		if scheme == "vault" {
			scheme = "https"
		}

		config.Address = scheme + "://" + u.Host
	}

	return config, nil
}

// FS is used to register this filesystem with an fsimpl.FSMux
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, "vault", "vault+http", "vault+https")

var (
	_ fs.FS                  = (*vaultFS)(nil)
	_ fs.ReadFileFS          = (*vaultFS)(nil)
	_ internal.WithContexter = (*vaultFS)(nil)
	_ internal.WithHeaderer  = (*vaultFS)(nil)
	_ withClienter           = (*vaultFS)(nil)
	_ withConfiger           = (*vaultFS)(nil)
)

func (f vaultFS) URL() string {
	return f.base.String()
}

func (f *vaultFS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *vaultFS) WithHeader(headers http.Header) fs.FS {
	if headers == nil {
		return f
	}

	fsys := *f

	for k, vs := range headers {
		for _, v := range vs {
			fsys.client.AddHeader(k, v)
		}
	}

	return &fsys
}

func (f *vaultFS) WithClient(client *api.Client) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.client = newRefCountedClient(client)

	return &fsys
}

func (f *vaultFS) WithConfig(config *api.Config) fs.FS {
	if config == nil {
		return f
	}

	// handle compound URL scheme not supported by the client, but only if the
	// URL has a host part set - otherwise use the scheme from $VAULT_ADDR, as
	// set by api.DefaultConfig() above
	if f.base.Host != "" {
		scheme := strings.TrimPrefix(f.base.Scheme, "vault+")
		if scheme == "vault" {
			scheme = "https"
		}

		config.Address = scheme + "://" + f.base.Host
	}

	client, err := api.NewClient(config)
	if err != nil {
		slog.ErrorContext(f.ctx, "vaultfs: failed to create vault client with user-supplied configuration",
			slog.Any("error", err))

		return nil
	}

	return f.WithClient(client)
}

func (f vaultFS) WithAuthMethod(auth api.AuthMethod) fs.FS {
	fsys := f
	fsys.auth = auth

	return &fsys
}

func (f vaultFS) Open(name string) (fs.File, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	u, err := internal.SubURL(f.base, name)
	if err != nil {
		return nil, err
	}

	if f.auth == nil {
		return nil, fmt.Errorf("missing vault auth method: %q", f.client.Token())
	}

	return newVaultFile(f.ctx, name, u, f.client, f.auth), nil
}

// ReadFile implements fs.ReadFileFS
//
// This implementation minimises Vault token uses by avoiding an extra Stat.
func (f vaultFS) ReadFile(name string) ([]byte, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: fs.ErrInvalid}
	}

	opened, err := f.Open(name)
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	b, err := io.ReadAll(opened)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// newVaultFile opens a vault file/dir for reading - if this file is not closed
// a vault token may be leaked!
func newVaultFile(ctx context.Context, name string, u *url.URL, client *refCountedClient, auth api.AuthMethod) *vaultFile {
	// add reference to shared client - will be removed on Close
	client.AddRef()

	return &vaultFile{
		ctx:    ctx,
		name:   name,
		u:      u,
		client: client,
		auth:   auth,
	}
}

type vaultFile struct {
	ctx  context.Context
	name string

	mountInfo *mountInfo

	u      *url.URL
	client *refCountedClient
	auth   api.AuthMethod

	body     io.ReadCloser
	children []string
	diridx   int

	closed atomic.Int32
}

type mountInfo struct {
	*api.MountOutput
	name       string
	secretPath string
}

var _ fs.ReadDirFile = (*vaultFile)(nil)

func (f *vaultFile) request() (*api.KVSecret, *api.Secret, error) {
	mountInfo, err := f.getMountInfo(f.ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get mount info: %w", err)
	}

	// it's a KVv2 Get operation with the right type, version, and especially if
	// the secret path is set - otherwise it might need to be a list operation
	if mountInfo.secretPath != "" && mountInfo.Type == "kv" && mountInfo.Options["version"] == "2" {
		var kv *api.KVSecret

		kv, err = f.kv2request(f.ctx, mountInfo.name, mountInfo.secretPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get KV v2 secret: %w", err)
		}

		return kv, nil, nil
	}

	secret, err := f.rawRequest(http.MethodGet)
	if err != nil {
		return nil, nil, err
	}

	return nil, secret, nil
}

func (f *vaultFile) kv2request(ctx context.Context, mount, secret string) (kv *api.KVSecret, err error) {
	kv2client := f.client.KVv2(mount)

	version := 0
	if ver := f.u.Query().Get("version"); ver != "" {
		version, err = strconv.Atoi(ver)
		if err != nil {
			return nil, fmt.Errorf("invalid version %q requested: %w", ver, err)
		}
	}

	return kv2client.GetVersion(ctx, secret, version)
}

// rawRequest makes a raw request to Vault by constructing a new request from
// the method and URL, and returns the parsed secret.
//
// This should probably be replaced with a call to the logical client (either
// Read, Write, or List) in the future, especially as the RawRequestWithContext
// method is deprecated.
func (f *vaultFile) rawRequest(method string) (*api.Secret, error) {
	req, err := f.newRequest(method)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault request: %w", err)
	}

	//nolint:staticcheck
	resp, err := f.client.RawRequestWithContext(f.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("http %s %s failed with: %w", method, f.u.Path,
			vaultFSError(err))
	}

	if resp.StatusCode == 0 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %s %s failed with status %d", method, f.u, resp.StatusCode)
	}

	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vault secret: %w", err)
	}

	return secret, nil
}

func (f *vaultFile) newRequest(method string) (*api.Request, error) {
	values := f.u.Query()
	if len(values) > 0 && method == http.MethodGet {
		method = http.MethodPost
	}

	req := f.client.NewRequest(method, f.u.Path)
	if method == http.MethodGet {
		req.Params = values
	} else if len(values) > 0 {
		data := map[string]any{}

		for k, vs := range values {
			for _, v := range vs {
				data[k] = v
			}
		}

		err := req.SetJSONBody(data)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

// Close the file. Will error on second call. Decrements the ref count on first
// call and logs out of vault when the ref count reaches zero.
func (f *vaultFile) Close() error {
	// important to know the state of the file so that we don't
	if f.closed.Load() == 1 {
		return &fs.PathError{Op: "close", Path: f.name, Err: fs.ErrClosed}
	}

	// mark closed
	f.closed.Store(1)

	f.client.RemoveRef()

	if f.client.Refs() == 0 {
		// the token auth method manages its own logout, to avoid revoking the
		// token, which shouldn't be managed here
		if lauth, ok := f.auth.(authLogouter); ok {
			lauth.Logout(f.ctx, f.client.Client)
		} else {
			revokeToken(f.ctx, f.client.Client)
		}
	}

	if f.body == nil {
		return nil
	}

	return f.body.Close()
}

func (f *vaultFile) Read(p []byte) (int, error) {
	if f.body != nil {
		return f.body.Read(p)
	}

	kvsec, s, err := f.request()
	if err != nil {
		return 0, err
	}

	var b []byte

	if (s != nil && s.Data != nil) || (kvsec != nil && kvsec.Data != nil) {
		if kvsec != nil {
			b, err = json.Marshal(kvsec.Data)
		} else {
			b, err = json.Marshal(s.Data)
		}

		if err != nil {
			return 0, &fs.PathError{
				Op: "read", Path: f.name,
				Err: fmt.Errorf("unexpected failure to marshal vault secret: %w", err),
			}
		}
	}

	f.body = io.NopCloser(bytes.NewReader(b))

	return f.body.Read(p)
}

//nolint:gocyclo
func (f *vaultFile) Stat() (fs.FileInfo, error) {
	kvsec, secret, err := f.request()

	rerr := &api.ResponseError{}
	if errors.As(err, &rerr) && rerr.StatusCode != http.StatusNotFound {
		return nil, &fs.PathError{
			Op: "stat", Path: f.name,
			Err: vaultFSError(err),
		}
	} else if err != nil {
		// if it's a 404 it might be a directory - let's try to LIST it instead
		_, lerr := f.list()
		if lerr != nil {
			// return the original error, not the LIST error
			return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
		}

		fi := internal.DirInfo(strings.TrimSuffix(path.Base(f.name), "/"), time.Time{})

		return fi, nil
	}

	if (secret == nil || secret.Data == nil) && (kvsec == nil || kvsec.Data == nil) {
		return nil, &fs.PathError{
			Op: "stat", Path: f.name, Err: errors.New("malformed secret"),
		}
	}

	var (
		b       []byte
		modTime time.Time
	)

	if kvsec != nil {
		b, err = json.Marshal(kvsec.Data)
		modTime = createdTimeFromData(kvsec)
	} else {
		b, err = json.Marshal(secret.Data)
	}

	if err != nil {
		return nil, &fs.PathError{
			Op: "stat", Path: f.name,
			Err: fmt.Errorf("malformed secret: %w", err),
		}
	}

	return internal.FileInfo(
		strings.TrimSuffix(path.Base(f.name), "/"),
		int64(len(b)),
		0o444,
		modTime,
		"application/json",
	), nil
}

func (f *vaultFile) list() ([]string, error) {
	mi, err := f.getMountInfo(f.ctx)
	if err != nil {
		return nil, fmt.Errorf("get mount info: %w", err)
	}

	p := path.Join(mi.name, mi.secretPath)
	// if it's a KVv2 mount, we must inject "metadata/" into the path, because
	// the logical client expects raw paths
	if mi.Type == "kv" && mi.Options["version"] == "2" {
		p = path.Join(mi.name, "metadata", mi.secretPath)
	}

	s, err := f.client.Logical().ListWithContext(f.ctx, p)
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", vaultFSError(err))
	}

	if s == nil {
		return nil, fmt.Errorf("nil response from vault LIST %q: %w", f.u.Path, fs.ErrNotExist)
	}

	keys, ok := s.Data["keys"]
	if !ok {
		return nil, fmt.Errorf("keys missing from vault LIST response %+v", s)
	}

	k, ok := keys.([]any)
	if !ok {
		return nil, fmt.Errorf("keys returned in unexpected format from vault LIST response: %#v", keys)
	}

	dirkeys := make([]string, len(k))

	for i := range k {
		if s, ok := k[i].(string); ok {
			dirkeys[i] = s
		}
	}

	return dirkeys, nil
}

func (f *vaultFile) childFile(childName string) *vaultFile {
	parent := *f.u
	parent.Path += "/"

	u, _ := url.Parse(childName)
	childURL := (&parent).ResolveReference(u)

	return newVaultFile(f.ctx, childName, childURL, f.client, f.auth)
}

func (f *vaultFile) ReadDir(n int) ([]fs.DirEntry, error) {
	// first call lists everything and caches the entries
	if f.children == nil {
		entries, err := f.list()
		if err != nil {
			return nil, &fs.PathError{Op: "readDir", Path: f.name, Err: err}
		}

		f.children = entries
	}

	dirents := []fs.DirEntry{}

	for i := f.diridx; n <= 0 || (n > 0 && i < n+f.diridx); i++ {
		// if we don't have enough children left
		if i >= len(f.children) {
			if n <= 0 {
				break
			}

			return dirents, io.EOF
		}

		childName := f.children[i]

		// vault lists directories with trailing slashes
		if strings.HasSuffix(childName, "/") {
			fi := internal.DirInfo(childName[:len(childName)-1], time.Time{})
			dirents = append(dirents, internal.FileInfoDirEntry(fi))

			continue
		}

		child := f.childFile(childName)
		defer child.Close()

		fi, err := child.Stat()
		if err != nil {
			return nil, &fs.PathError{Op: "readDir", Path: f.name, Err: err}
		}

		dirents = append(dirents, internal.FileInfoDirEntry(fi))
	}

	f.diridx += len(dirents)

	return dirents, nil
}

// vaultFSError converts from a vault API error to an appropriate filesystem
// error, preventing Vault API types from leaking
func vaultFSError(err error) error {
	rerr := &api.ResponseError{}
	if errors.As(err, &rerr) {
		errDetails := strings.Join(rerr.Errors, ", ")
		if errDetails != "" {
			errDetails = ", details: " + errDetails
		}

		return fmt.Errorf("%s %s - %d%s: %w",
			rerr.HTTPMethod,
			rerr.URL,
			rerr.StatusCode,
			errDetails,
			fs.ErrNotExist,
		)
	}

	return err
}

// getMountInfo calls the undocumented sys/internal/ui/mounts endpoint to set
// the file's mount metadata. This is used in preference to the sys/mounts
// API because this one works read-only roles (!). The result is cached.
func (f *vaultFile) getMountInfo(ctx context.Context) (*mountInfo, error) {
	if f.mountInfo != nil {
		return f.mountInfo, nil
	}

	if f.client.Token() == "" {
		secret, err := f.auth.Login(ctx, f.client.Client)
		if err != nil {
			return nil, fmt.Errorf("vault login failure: %w", err)
		}

		f.client.SetToken(secret.Auth.ClientToken)
	}

	resp, err := f.client.Logical().ReadRawWithContext(ctx, "sys/internal/ui/mounts")
	if err != nil {
		return nil, fmt.Errorf("read mount info: %w", err)
	}

	s, err := f.client.Logical().ParseRawResponseAndCloseBody(resp, err)
	if err != nil {
		return nil, fmt.Errorf("parse mount info: %w", err)
	}

	rawMounts, ok := s.Data["secret"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected mount info format: %#v", s.Data)
	}

	mi, err := findMountInfo(f.u.Path, rawMounts)
	if err != nil {
		return nil, err
	}

	if mi == nil {
		return nil, fmt.Errorf("mount not found for %q", f.u.Path)
	}

	f.mountInfo = mi

	return f.mountInfo, nil
}

func findMountInfo(rawFilePath string, rawMounts map[string]any) (*mountInfo, error) {
	for mountName, mountOpts := range rawMounts {
		mountPrefix := path.Join("/v1", mountName)

		if strings.HasPrefix(rawFilePath, mountPrefix) {
			v, ok := mountOpts.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("unexpected mount info format for %q: %#v", mountName, v)
			}

			mount := &api.MountOutput{Type: v["type"].(string)}

			opts, ok := v["options"].(map[string]any)
			if ok {
				mount.Options = make(map[string]string, len(opts))
				for k, v := range opts {
					mount.Options[k] = v.(string)
				}
			}

			// build secretPath - it's the part after the mount name, including the
			// / prefix
			spath := strings.TrimPrefix(rawFilePath, mountPrefix)

			return &mountInfo{name: mountName, secretPath: spath, MountOutput: mount}, nil
		}
	}

	return nil, nil
}

func createdTimeFromData(kvsec *api.KVSecret) time.Time {
	metadata := kvsec.VersionMetadata
	if metadata == nil {
		return time.Time{}
	}

	return metadata.CreatedTime
}
