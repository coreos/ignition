package consulfs

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
	"sync/atomic"
	"time"

	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/internal"
	"github.com/hashicorp/consul/api/v2"
)

type consulFS struct {
	ctx  context.Context
	base *url.URL

	client    *api.Client
	config    *api.Config
	queryOpts *api.QueryOptions
	header    http.Header
	token     string
}

// New creates a filesystem for the Consul KV endpoint rooted at u.
func New(u *url.URL) (fs.FS, error) {
	if u == nil {
		return nil, errors.New("url must not be nil")
	}

	if u.Path == "" {
		u.Path = "/"
	}

	if !strings.HasSuffix(u.Path, "/") {
		return nil, fmt.Errorf("invalid url: path %q must be a prefix ending with \"/\"", u)
	}

	return &consulFS{
		ctx:  context.Background(),
		base: u,
	}, nil
}

// FS is used to register this filesystem with an [fsimpl.FSMux]
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, "consul", "consul+http", "consul+https")

var (
	_ fs.FS                  = (*consulFS)(nil)
	_ fs.ReadFileFS          = (*consulFS)(nil)
	_ internal.WithContexter = (*consulFS)(nil)
	_ internal.WithHeaderer  = (*consulFS)(nil)
	_ withConfiger           = (*consulFS)(nil)
	_ withQueryOptionser     = (*consulFS)(nil)
	_ withTokener            = (*consulFS)(nil)
)

func (f consulFS) URL() string {
	return f.base.String()
}

func (f *consulFS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *consulFS) WithHeader(header http.Header) fs.FS {
	if header == nil {
		return f
	}

	fsys := *f
	fsys.header = header

	if fsys.client != nil {
		for k, vs := range header {
			for _, v := range vs {
				fsys.client.AddHeader(k, v)
			}
		}
	}

	return &fsys
}

func (f *consulFS) WithConfig(config *api.Config) fs.FS {
	if config == nil {
		return f
	}

	fsys := *f
	fsys.client = nil
	fsys.config = config

	return &fsys
}

func (f consulFS) WithToken(token string) fs.FS {
	fsys := f
	fsys.client = nil
	fsys.token = token

	return &fsys
}

func (f consulFS) WithQueryOptions(opts *api.QueryOptions) fs.FS {
	fsys := f
	fsys.queryOpts = opts

	return &fsys
}

func getAddress(u *url.URL) string {
	// handle compound URL scheme not supported by the client, but only if the
	// URL has a host part set - otherwise just use the defaults
	if u.Host != "" {
		scheme := strings.TrimPrefix(u.Scheme, "consul+")
		if scheme == "consul" {
			scheme = "http"
		}

		return scheme + "://" + u.Host
	}

	return ""
}

// initClient must be called before referencing f.client
func (f *consulFS) initClient() error {
	if f.client != nil {
		return nil
	}

	config := f.config
	if config == nil {
		config = &api.Config{}
	}

	addr := getAddress(f.base)
	if addr != "" {
		config.Address = addr
	}

	// set the token if one was provided
	if f.token != "" {
		config.Token = f.token
	}

	c, err := api.NewClient(config)
	if err != nil {
		return fmt.Errorf("consul client creation failed: %w", err)
	}

	if f.header != nil {
		c.SetHeaders(f.header)
	}

	f.client = c

	return nil
}

func (f *consulFS) Open(name string) (fs.File, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	u, err := internal.SubURL(f.base, name)
	if err != nil {
		return nil, err
	}

	if err = f.initClient(); err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	return &consulFile{
		ctx:       f.ctx,
		name:      name,
		u:         u,
		client:    f.client,
		queryOpts: f.queryOpts,
	}, nil
}

// ReadFile implements fs.ReadFileFS
func (f *consulFS) ReadFile(name string) ([]byte, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{
			Op: "readFile", Path: name,
			Err: fs.ErrInvalid,
		}
	}

	u, err := internal.SubURL(f.base, name)
	if err != nil {
		return nil, err
	}

	if err = f.initClient(); err != nil {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: err}
	}

	kvPair, _, err := f.client.KV().Get(u.Path, f.queryOpts.WithContext(f.ctx))
	if err != nil {
		return nil, &fs.PathError{
			Op: "readFile", Path: u.Path,
			Err: fmt.Errorf("kv.Get: %w", err),
		}
	}

	return kvPair.Value, nil
}

type consulFile struct {
	ctx       context.Context
	name      string
	u         *url.URL
	client    *api.Client
	kv        *api.KV
	queryOpts *api.QueryOptions

	body     io.ReadCloser
	fi       fs.FileInfo
	children []string
	diridx   int

	closed atomic.Int32
}

var _ fs.ReadDirFile = (*consulFile)(nil)

// Close the file. Will error on second call.
func (f *consulFile) Close() error {
	if f.closed.Load() == 1 {
		return &fs.PathError{Op: "close", Path: f.name, Err: fs.ErrClosed}
	}

	// mark closed
	f.closed.Store(1)

	if f.body == nil {
		return nil
	}

	return f.body.Close()
}

func (f *consulFile) get() error {
	if f.kv == nil {
		f.kv = f.client.KV()
	}

	key := strings.TrimPrefix(f.u.Path, "/")

	kvPair, _, err := f.kv.Get(key, f.queryOpts.WithContext(f.ctx))
	if err != nil {
		return fmt.Errorf("kv.Get: %w", err)
	}

	if kvPair == nil {
		return fs.ErrNotExist
	}

	name := f.name
	if !strings.HasSuffix(kvPair.Key, "/") {
		name = path.Base(name)
	}

	f.fi = internal.FileInfo(name, int64(len(kvPair.Value)),
		0o444, time.Time{}, "",
	)

	f.body = io.NopCloser(bytes.NewReader(kvPair.Value))

	return nil
}

func (f *consulFile) list() ([]string, error) {
	if f.kv == nil {
		f.kv = f.client.KV()
	}

	key := strings.TrimPrefix(f.u.Path, "/")
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	keys, _, err := f.kv.Keys(key, "/", f.queryOpts.WithContext(f.ctx))
	if err != nil {
		return nil, fmt.Errorf("kv.Keys: %w", err)
	}

	// can't have empty directories as they're just prefixes with consul
	if len(keys) == 0 {
		return nil, fs.ErrNotExist
	}

	// the listing is recursive, but we only want direct children
	keys = onlyChildren(key, keys)

	return keys, nil
}

// onlyChildren returns the sorted slice of keys that are direct children of the
// given key.
func onlyChildren(parent string, keys []string) []string {
	keyMap := map[string]struct{}{}

	for _, key := range keys {
		child := strings.TrimPrefix(key, parent)
		if strings.Contains(child, "/") {
			child, _, _ = strings.Cut(child, "/")

			// is a dir, so still needs a trailing slash
			child += "/"
		}

		keyMap[parent+child] = struct{}{}
	}

	keys = make([]string, 0, len(keyMap))
	for k := range keyMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func (f *consulFile) Read(p []byte) (int, error) {
	if f.body != nil {
		return f.body.Read(p)
	}

	err := f.get()
	if err != nil {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: err}
	}

	return f.body.Read(p)
}

func (f *consulFile) Stat() (fs.FileInfo, error) {
	if f.fi != nil {
		return f.fi, nil
	}

	key := strings.TrimPrefix(f.u.Path, "/")

	// If key is empty, we're at root - it's always a directory, skip get()
	// because Consul KV API doesn't accept empty key names for GET
	if key == "" {
		_, err := f.list()
		if err != nil {
			return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
		}

		f.fi = internal.DirInfo(path.Base(f.name), time.Time{})

		return f.fi, nil
	}

	err := f.get()
	if err == nil {
		return f.fi, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
	}

	_, err = f.list()
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
	}

	f.fi = internal.DirInfo(path.Base(f.name), time.Time{})

	return f.fi, nil
}

func (f *consulFile) childFile(childName string) *consulFile {
	parent := *f.u
	if !strings.HasSuffix(parent.Path, "/") {
		parent.Path += "/"
	}

	childURL, _ := internal.SubURL(&parent, childName)

	cf := &consulFile{
		ctx:       f.ctx,
		name:      childName,
		u:         childURL,
		client:    f.client,
		queryOpts: f.queryOpts,
	}

	return cf
}

func (f *consulFile) ReadDir(n int) ([]fs.DirEntry, error) {
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

		// consul lists directories with trailing slashes
		if strings.HasSuffix(childName, "/") {
			name := childName[:len(childName)-1]
			name = path.Base(name)

			fi := internal.DirInfo(name, time.Time{})
			dirents = append(dirents, internal.FileInfoDirEntry(fi))

			continue
		}

		child := f.childFile(path.Base(childName))
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
