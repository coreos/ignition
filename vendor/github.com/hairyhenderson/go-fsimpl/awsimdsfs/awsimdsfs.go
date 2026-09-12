package awsimdsfs

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

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/internal"
)

// withIMDSClienter is an fs.FS that can be configured to use the given Secrets
// Manager client.
type withIMDSClienter interface {
	WithIMDSClient(imdsclient IMDSClient) fs.FS
}

// WithIMDSClientFS overrides the AWS IMDS client used by fs, if the filesystem
// supports it (i.e. has a WithIMDSClient method). This can be used for
// configuring specialized client options.
//
// Note that this should not be used together with [fsimpl.WithHTTPClientFS].
// If you wish only to override the HTTP client, use WithHTTPClient alone.
func WithIMDSClientFS(imdsclient IMDSClient, fsys fs.FS) fs.FS {
	if fsys, ok := fsys.(withIMDSClienter); ok {
		return fsys.WithIMDSClient(imdsclient)
	}

	return fsys
}

type awsimdsFS struct {
	ctx        context.Context
	base       *url.URL
	httpclient *http.Client
	imdsclient IMDSClient
	root       string
}

// New provides a filesystem (an fs.FS) backed by the AWS IMDS,
// rooted at the given URL.
//
// A context can be given by using WithContextFS.
func New(u *url.URL) (fs.FS, error) {
	if u.Scheme != "aws+imds" {
		return nil, fmt.Errorf("invalid URL scheme %q", u.Scheme)
	}

	root := u.Path
	if len(root) > 0 && root[0] == '/' {
		root = root[1:]
	}

	return &awsimdsFS{
		ctx:  context.Background(),
		base: u,
		root: root,
	}, nil
}

// FS is used to register this filesystem with an fsimpl.FSMux
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, "aws+imds")

var (
	_ fs.FS                     = (*awsimdsFS)(nil)
	_ fs.ReadDirFS              = (*awsimdsFS)(nil)
	_ fs.SubFS                  = (*awsimdsFS)(nil)
	_ internal.WithContexter    = (*awsimdsFS)(nil)
	_ internal.WithHTTPClienter = (*awsimdsFS)(nil)
	_ withIMDSClienter          = (*awsimdsFS)(nil)
)

func (f awsimdsFS) URL() string {
	return f.base.String()
}

func (f *awsimdsFS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *awsimdsFS) WithHTTPClient(client *http.Client) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.httpclient = client

	return &fsys
}

func (f *awsimdsFS) WithIMDSClient(imdsclient IMDSClient) fs.FS {
	if imdsclient == nil {
		return f
	}

	fsys := *f
	fsys.imdsclient = imdsclient

	return &fsys
}

func (f *awsimdsFS) getClient(ctx context.Context) (IMDSClient, error) {
	if f.imdsclient != nil {
		return f.imdsclient, nil
	}

	opts := [](func(*config.LoadOptions) error){}
	if f.httpclient != nil {
		opts = append(opts, config.WithHTTPClient(f.httpclient))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	optFns := [](func(*imds.Options)){}

	// setting a host in the URL is only intended for test purposes
	if f.base.Host != "" {
		optFns = append(optFns, func(o *imds.Options) {
			o.Endpoint = "http://" + f.base.Host
		})
	}

	f.imdsclient = imds.NewFromConfig(cfg, optFns...)

	return f.imdsclient, nil
}

func (f *awsimdsFS) Sub(name string) (fs.FS, error) {
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

func (f *awsimdsFS) Open(name string) (fs.File, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	client, err := f.getClient(f.ctx)
	if err != nil {
		return nil, err
	}

	file := &awsimdsFile{
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

func (f *awsimdsFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	client, err := f.getClient(f.ctx)
	if err != nil {
		return nil, err
	}

	dir := &awsimdsFile{
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

type awsimdsFile struct {
	ctx    context.Context
	fi     fs.FileInfo
	client IMDSClient
	body   io.Reader
	name   string
	root   string

	children []*awsimdsFile
	diroff   int
}

var _ fs.ReadDirFile = (*awsimdsFile)(nil)

func (f *awsimdsFile) Close() error {
	if closer, ok := f.body.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}

func (f *awsimdsFile) Read(p []byte) (int, error) {
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

func (f *awsimdsFile) Stat() (fs.FileInfo, error) {
	if f.fi != nil {
		return f.fi, nil
	}

	err := f.getValue()
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
	}

	return f.fi, nil
}

// convertAWSError converts an AWS error to an error suitable for returning
// from the package. We don't want to leak SDK error types.
func convertAWSError(err error) error {
	if err == nil {
		return nil
	}

	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) {
		switch respErr.Response.StatusCode {
		case http.StatusNotFound:
			return fmt.Errorf("%w: %s", fs.ErrNotExist, respErr.Response.Status)
		default:
			return fmt.Errorf("%w: HTTP error %s", fs.ErrInvalid, respErr.Response.Status)
		}
	}

	var opErr *smithy.OperationError
	if errors.As(err, &opErr) {
		return fmt.Errorf("%w: %s", err, opErr.OperationName)
	}

	err = fmt.Errorf("%T: %w", err, err)

	return err
}

// getValue gets the value from IMDS and populates body and fi. SDK errors will
// not be leaked, instead they will be converted to more general errors.
func (f *awsimdsFile) getValue() error {
	var (
		err error
		rc  io.ReadCloser
	)

	fullPath := path.Join(f.root, f.name)

	if isIMDSDirectory(fullPath) {
		f.fi = internal.DirInfo(f.name, time.Time{})

		return nil
	}

	if s, ok := strings.CutPrefix(fullPath, "meta-data"); ok {
		rc, err = f.getMetaData(s)
	} else if s, ok := strings.CutPrefix(fullPath, "user-data"); ok && s == "" {
		rc, err = f.getUserData()
	} else if s, ok := strings.CutPrefix(fullPath, "dynamic"); ok {
		rc, err = f.getDynamicData(s)
	} else {
		return &fs.PathError{
			Op:   "read",
			Path: f.name,
			Err:  fmt.Errorf("%w: invalid prefix for %q", fs.ErrNotExist, fullPath),
		}
	}

	if rc != nil {
		f.fi, f.body, err = f.readData(f.name, rc)
	}

	return convertAWSError(err)
}

func (f *awsimdsFile) readData(name string, content io.ReadCloser) (fi fs.FileInfo, r io.Reader, err error) {
	b := &bytes.Buffer{}

	n, err := io.Copy(b, content)
	if err != nil {
		return nil, nil, fmt.Errorf("copy: %w", err)
	}

	return internal.FileInfo(name, n, 0o444, time.Time{}, ""), b, nil
}

func (f *awsimdsFile) getMetaData(s string) (io.ReadCloser, error) {
	s = strings.TrimPrefix(s, "/")

	out, err := f.client.GetMetadata(f.ctx, &imds.GetMetadataInput{Path: s})
	if err != nil {
		return nil, fmt.Errorf("getMetaData: %w", err)
	}

	if out != nil && out.Content != nil {
		return out.Content, nil
	}

	return nil, nil
}

func (f *awsimdsFile) getDynamicData(s string) (io.ReadCloser, error) {
	s = strings.TrimPrefix(s, "/")

	out, err := f.client.GetDynamicData(f.ctx, &imds.GetDynamicDataInput{Path: s})
	if err != nil {
		return nil, fmt.Errorf("getDynamicData: %w", err)
	}

	if out != nil && out.Content != nil {
		return out.Content, nil
	}

	return nil, nil
}

func (f *awsimdsFile) getUserData() (io.ReadCloser, error) {
	out, err := f.client.GetUserData(f.ctx, &imds.GetUserDataInput{})
	if err != nil {
		return nil, fmt.Errorf("getUserData: %w", err)
	}

	if out != nil && out.Content != nil {
		return out.Content, nil
	}

	return nil, nil
}

// listPrefix returns the prefix for this directory
func (f *awsimdsFile) listPrefix() string {
	// when listing "." at the root (or opaque root), avoid returning "//"
	if f.name == "." && (f.root == "" || f.root == "/") {
		return f.root
	}

	return path.Join(f.root, f.name) + "/"
}

func (f *awsimdsFile) parseListOutput(r io.ReadCloser) ([]string, error) {
	prefix := f.listPrefix()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("readAll: %w", err)
	}

	children := []string{}

	for line := range strings.SplitSeq(string(b), "\n") {
		if line == "" {
			continue
		}

		children = append(children, prefix+line)
	}

	return children, nil
}

func (f *awsimdsFile) listMetadata() ([]string, error) {
	fullPath := path.Join(f.root, f.name)
	fullPath = strings.TrimPrefix(fullPath, "meta-data")
	fullPath = strings.Trim(fullPath, "/")

	out, err := f.client.GetMetadata(f.ctx, &imds.GetMetadataInput{Path: fullPath})
	if err != nil {
		return nil, fmt.Errorf("getMetadata: %w", err)
	}

	return f.parseListOutput(out.Content)
}

func (f *awsimdsFile) listDynamic() ([]string, error) {
	fullPath := path.Join(f.root, f.name)
	fullPath = strings.TrimPrefix(fullPath, "dynamic")
	fullPath = strings.Trim(fullPath, "/")

	out, err := f.client.GetDynamicData(f.ctx, &imds.GetDynamicDataInput{Path: fullPath})
	if err != nil {
		return nil, fmt.Errorf("getDynamicData: %w", err)
	}

	return f.parseListOutput(out.Content)
}

// special-case - the root directory can't technically be listed through the
// SDK, so we have to fake it. but the children are all known, so we can just
// hard-code them.
func (f *awsimdsFile) listRoot() error {
	fullPath := path.Join(f.root, f.name)

	// we have to get user-data so we can set its size correctly
	rc, err := f.getUserData()
	if err != nil {
		return fmt.Errorf("getUserData: %w", err)
	}

	udFi, _, err := f.readData("user-data", rc)
	if err != nil {
		return fmt.Errorf("readData: %w", err)
	}

	f.children = []*awsimdsFile{
		{
			ctx:    f.ctx,
			name:   "dynamic",
			root:   fullPath,
			client: f.client,
			fi:     internal.DirInfo("dynamic", time.Time{}),
		},
		{
			ctx:    f.ctx,
			name:   "meta-data",
			root:   fullPath,
			client: f.client,
			fi:     internal.DirInfo("meta-data", time.Time{}),
		},
		{
			ctx:    f.ctx,
			name:   "user-data",
			root:   fullPath,
			client: f.client,
			fi:     udFi,
		},
	}

	return nil
}

// list assignes a sorted list of the children of this directory to f.children
func (f *awsimdsFile) list() error {
	fullPath := path.Join(f.root, f.name)

	if fullPath == "" || fullPath == "/" || fullPath == "." {
		return f.listRoot()
	}

	var (
		children []string
		err      error
	)

	if _, ok := strings.CutPrefix(fullPath, "meta-data"); ok {
		children, err = f.listMetadata()
		if err != nil {
			return err
		}
	} else if _, ok := strings.CutPrefix(fullPath, "dynamic"); ok {
		children, err = f.listDynamic()
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("%w: invalid prefix for %q", fs.ErrInvalid, fullPath)
	}

	return f.populateChildren(children)
}

// populateChildren creates a list of children from the given list of names,
// sorts and assigns it to f.children
func (f *awsimdsFile) populateChildren(children []string) error {
	// track files that we've already seen - we don't want to add duplicates
	seen := map[string]struct{}{}

	for _, entry := range children {
		entry = strings.Trim(entry, "/")
		parts := strings.Split(entry, "/")
		name := parts[len(parts)-1]

		if _, ok := seen[name]; ok {
			continue
		}

		seen[name] = struct{}{}

		child := awsimdsFile{
			ctx:    f.ctx,
			name:   name,
			root:   path.Join(f.root, f.name),
			client: f.client,
		}

		fi, err := child.Stat()
		if err != nil {
			return err
		}

		child.fi = fi

		f.children = append(f.children, &child)
	}

	// the AWS SDK doesn't sort the list of children, so we do it here
	sort.Slice(f.children, func(i, j int) bool {
		return f.children[i].name < f.children[j].name
	})

	return nil
}

// If n > 0, ReadDir returns at most n DirEntry structures.
// In this case, if ReadDir returns an empty slice, it will return
// a non-nil error explaining why.
// At the end of a directory, the error is io.EOF.
//
// If n <= 0, ReadDir returns all the DirEntry values from the directory
// in a single slice. In this case, if ReadDir succeeds (reads all the way
// to the end of the directory), it returns the slice and a nil error.
// If it encounters an error before the end of the directory,
// ReadDir returns the DirEntry list read until that point and a non-nil error.
func (f *awsimdsFile) ReadDir(n int) ([]fs.DirEntry, error) {
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
