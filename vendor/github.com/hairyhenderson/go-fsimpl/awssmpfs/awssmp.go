package awssmpfs

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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/awsimdsfs"
	"github.com/hairyhenderson/go-fsimpl/internal"
)

// withClienter is an [io/fs.FS] that can be configured to use the given
// Systems Manager client.
type withClienter interface {
	WithClient(client SSMClient) fs.FS
}

// WithClientFS overrides the AWS Systems Manager client used by fsys, if the
// filesystem supports it (i.e. has a WithClient method). This can be used for
// configuring specialized client options.
//
// Note that this should not be used together with [fsimpl.WithHTTPClientFS].
// If you wish only to override the HTTP client, use [fsimpl.WithHTTPClientFS]
// alone.
//
// Usually, client would be a [*github.com/aws/aws-sdk-go-v2/service/ssm.Client]
// created using [github.com/aws/aws-sdk-go-v2/service/ssm.New] or
// [github.com/aws/aws-sdk-go-v2/service/ssm.NewFromConfig].
func WithClientFS(client SSMClient, fsys fs.FS) fs.FS {
	if fsys, ok := fsys.(withClienter); ok {
		return fsys.WithClient(client)
	}

	return fsys
}

type awssmpFS struct {
	ctx        context.Context
	base       *url.URL
	httpclient *http.Client
	ssmclient  SSMClient
	imdsfs     fs.FS
	root       string
}

// New provides a filesystem (an [io/fs.FS]) backed by the AWS Systems Manager
// Parameter Store, rooted at the given URL. The URL should be a regular
// hierarchical URL (like "aws+smp:///foo/bar" or "aws+smp:///").
func New(u *url.URL) (fs.FS, error) {
	if u.Scheme != "aws+smp" {
		return nil, fmt.Errorf("invalid URL scheme %q", u.Scheme)
	}

	if u.Opaque != "" {
		return nil, fmt.Errorf("aws+smp URL must not be opaque %q", u.String())
	}

	// allow "aws+smp:" to mean "aws+smp:///"
	if u.Path == "" {
		u.Path = "/"
	}

	iu, _ := url.Parse("aws+imds:")

	imdsfs, err := awsimdsfs.New(iu)
	if err != nil {
		return nil, fmt.Errorf("couldn't create IMDS filesystem: %w", err)
	}

	return &awssmpFS{
		ctx:    context.Background(),
		base:   u,
		root:   u.Path,
		imdsfs: imdsfs,
	}, nil
}

// FS is used to register this filesystem with an [fsimpl.FSMux]
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, "aws+smp")

var (
	_ fs.FS                     = (*awssmpFS)(nil)
	_ fs.ReadFileFS             = (*awssmpFS)(nil)
	_ fs.ReadDirFS              = (*awssmpFS)(nil)
	_ fs.SubFS                  = (*awssmpFS)(nil)
	_ internal.WithContexter    = (*awssmpFS)(nil)
	_ internal.WithHTTPClienter = (*awssmpFS)(nil)
	_ withClienter              = (*awssmpFS)(nil)
	_ internal.WithIMDSFSer     = (*awssmpFS)(nil)
)

func (f awssmpFS) URL() string {
	return f.base.String()
}

func (f *awssmpFS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *awssmpFS) WithHTTPClient(client *http.Client) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.httpclient = client

	return &fsys
}

func (f *awssmpFS) WithClient(ssmclient SSMClient) fs.FS {
	if ssmclient == nil {
		return f
	}

	fsys := *f
	fsys.ssmclient = ssmclient

	return &fsys
}

func (f *awssmpFS) WithIMDSFS(imdsfs fs.FS) fs.FS {
	if imdsfs == nil {
		return f
	}

	fsys := *f
	fsys.imdsfs = imdsfs

	return &fsys
}

func (f *awssmpFS) getClient(ctx context.Context) (SSMClient, error) {
	if f.ssmclient != nil {
		return f.ssmclient, nil
	}

	opts := [](func(*config.LoadOptions) error){}
	if f.httpclient != nil {
		opts = append(opts, config.WithHTTPClient(f.httpclient))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	if cfg.Region == "" && f.imdsfs != nil {
		// if we have an IMDS filesystem, use it to get the region
		region, err := fs.ReadFile(f.imdsfs, "meta-data/placement/region")
		if err != nil {
			return nil, fmt.Errorf("couldn't get region from IMDS: %w", err)
		}

		cfg.Region = string(region)
	}

	optFns := []func(*ssm.Options){}

	// setting a host in the URL is only intended for test purposes
	if f.base.Host != "" {
		optFns = append(optFns, func(o *ssm.Options) {
			o.BaseEndpoint = aws.String("http://" + f.base.Host)
		})
	}

	f.ssmclient = ssm.NewFromConfig(cfg, optFns...)

	return f.ssmclient, nil
}

func (f *awssmpFS) Sub(name string) (fs.FS, error) {
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

func (f *awssmpFS) Open(name string) (fs.File, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	smclient, err := f.getClient(f.ctx)
	if err != nil {
		return nil, err
	}

	file := &awssmpFile{
		ctx:    f.ctx,
		name:   strings.TrimPrefix(path.Base(name), "."),
		root:   strings.TrimPrefix(path.Join(f.root, path.Dir(name)), "."),
		client: smclient,
	}

	if name == "." {
		file.fi = internal.DirInfo(file.name, time.Time{})

		return file, nil
	}

	return file, nil
}

func (f *awssmpFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	ssmclient, err := f.getClient(f.ctx)
	if err != nil {
		return nil, err
	}

	dir := &awssmpFile{
		ctx:    f.ctx,
		name:   name,
		root:   f.root,
		client: ssmclient,
		fi:     internal.DirInfo(name, time.Time{}),
	}

	des, err := dir.ReadDir(-1)
	if err != nil {
		return nil, &fs.PathError{Op: "readDir", Path: name, Err: err}
	}

	return des, nil
}

// ReadFile implements fs.ReadFileFS.
//
// This implementation is slightly more performant than calling Open and then
// reading the resulting fs.File.
func (f *awssmpFS) ReadFile(name string) ([]byte, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: fs.ErrInvalid}
	}

	smclient, err := f.getClient(f.ctx)
	if err != nil {
		return nil, err
	}

	out, err := smclient.GetParameter(f.ctx, &ssm.GetParameterInput{
		Name: aws.String(path.Join(f.root, name)),
		// decrypt the parameter if it's a SecureString
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: convertAWSError(err)}
	}

	return []byte(*out.Parameter.Value), nil
}

type awssmpFile struct {
	ctx    context.Context
	fi     fs.FileInfo
	client SSMClient
	body   io.Reader
	name   string
	root   string

	children []*awssmpFile
	diroff   int
}

var _ fs.ReadDirFile = (*awssmpFile)(nil)

func (f *awssmpFile) Close() error {
	// no-op - no state is kept
	return nil
}

func (f *awssmpFile) Read(p []byte) (int, error) {
	if f.body == nil {
		err := f.getParameter()
		if err != nil {
			return 0, &fs.PathError{Op: "read", Path: f.name, Err: err}
		}
	}

	return f.body.Read(p)
}

func (f *awssmpFile) Stat() (fs.FileInfo, error) {
	if f.fi != nil {
		return f.fi, nil
	}

	err := f.getParameter()
	if err == nil {
		return f.fi, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
	}

	// may be a directory, so attempt to list one child
	// no need for special handling for opaque paths, as "." will never hit this
	// code path (Open sets f.fi to a DirInfo)
	params, err := f.client.GetParametersByPath(f.ctx, &ssm.GetParametersByPathInput{
		Path:      aws.String(path.Join(f.root, f.name) + "/"),
		Recursive: aws.Bool(true),
	})
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: convertAWSError(err)}
	}

	if len(params.Parameters) == 0 {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: fs.ErrNotExist}
	}

	f.fi = internal.DirInfo(f.name, time.Time{})

	return f.fi, nil
}

// convertAWSError converts an AWS error to an error suitable for returning
// from the package. We don't want to leak SDK error types.
func convertAWSError(err error) error {
	// We can't find the parameter that you asked for.
	var nfErr *types.ParameterNotFound
	if errors.As(err, &nfErr) {
		return fmt.Errorf("%w: %s", fs.ErrNotExist, nfErr.ErrorMessage())
	}

	// An error occurred on the server side.
	var internalErr *types.InternalServerError
	if errors.As(err, &internalErr) {
		return fmt.Errorf("internal error: %s: %s", internalErr.ErrorCode(), internalErr.ErrorMessage())
	}

	// You provided an invalid value for a parameter.
	var paramErr *types.InvalidParameters
	if errors.As(err, &paramErr) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, paramErr.ErrorMessage())
	}

	return err
}

// getParameter gets the parameter value from AWS SSM Parameter Store and
// populates body and fi. SDK errors will not be leaked, instead they will be
// converted to more general errors.
func (f *awssmpFile) getParameter() error {
	fullPath := path.Join(f.root, f.name)

	out, err := f.client.GetParameter(f.ctx, &ssm.GetParameterInput{
		Name:           aws.String(fullPath),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("getParameter: %w", convertAWSError(err))
	}

	body := *out.Parameter.Value

	n := int64(len(body))
	f.body = strings.NewReader(body)

	// parameter versions are immutable, so the created date for this version
	// is also the last modified date
	modTime := out.Parameter.LastModifiedDate
	if modTime == nil {
		modTime = &time.Time{}
	}

	contentType := ""
	if out.Parameter.DataType != nil && *out.Parameter.DataType == "text" {
		contentType = "text/plain"
	}

	// populate fi
	f.fi = internal.FileInfo(f.name, n, 0o444, *modTime, contentType)

	return nil
}

// listPrefix returns the prefix for this directory
func (f *awssmpFile) listPrefix() string {
	// when listing "." at the root, avoid returning "//"
	if (f.name == "." || f.name == "") && f.root == "/" {
		return f.root
	}

	// we want the prefix to end with a /, but path.Join will remove it, so we
	// tack it on at the end
	return path.Join(f.root, f.name) + "/"
}

func (f *awssmpFile) listParameters() ([]types.Parameter, error) {
	prefix := f.listPrefix()
	paramList := []types.Parameter{}

	for token := (*string)(nil); ; {
		out, err := f.client.GetParametersByPath(f.ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(prefix),
			WithDecryption: aws.Bool(true),
			NextToken:      token,
			Recursive:      aws.Bool(true),
		})
		if err != nil {
			return nil, fmt.Errorf("getParametersByPath: %w", err)
		}

		// trim the prefix from the names so we can use them as filenames
		for _, param := range out.Parameters {
			name := strings.TrimPrefix(*param.Name, prefix)
			if prefix != "/" {
				name = strings.TrimPrefix(name, "/")
			}

			*param.Name = name
		}

		paramList = append(paramList, out.Parameters...)

		token = out.NextToken
		if token == nil {
			break
		}
	}

	// no such thing as empty directories in SSM PS, they're artificial
	if len(paramList) == 0 {
		return nil, fmt.Errorf("%w (or not a dir): %q", fs.ErrNotExist, prefix)
	}

	return paramList, nil
}

// list assignes a sorted list of the children of this directory to f.children
func (f *awssmpFile) list() error {
	paramList, err := f.listParameters()
	if err != nil {
		return fmt.Errorf("listParameters: %w", err)
	}

	// a set of files that we've already seen - we don't want to add duplicates
	seen := map[string]struct{}{}

	for _, entry := range paramList {
		name, _, found := strings.Cut(*entry.Name, "/")

		if _, ok := seen[name]; ok {
			continue
		}

		if name == "" {
			continue
		}

		seen[name] = struct{}{}

		child := awssmpFile{
			ctx:    f.ctx,
			name:   name,
			root:   path.Join(f.root, f.name),
			client: f.client,
		}

		if found {
			// given that directories are artificial, they have a zero time
			child.fi = internal.DirInfo(name, time.Time{})
		} else {
			fi, err := child.Stat()
			if err != nil {
				return err
			}

			child.fi = fi
		}

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
func (f *awssmpFile) ReadDir(n int) ([]fs.DirEntry, error) {
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
