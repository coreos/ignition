package awssmfs

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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/awsimdsfs"
	"github.com/hairyhenderson/go-fsimpl/internal"
)

// withSMClienter is an fs.FS that can be configured to use the given Secrets
// Manager client.
type withSMClienter interface {
	WithSMClient(smclient SecretsManagerClient) fs.FS
}

// WithSMClientFS overrides the AWS Secrets Manager client used by fs, if the
// filesystem supports it (i.e. has a WithSMClient method). This can be used for
// configuring specialized client options.
//
// Note that this should not be used together with WithHTTPClient. If you wish
// only to override the HTTP client, use WithHTTPClient alone.
func WithSMClientFS(smclient SecretsManagerClient, fsys fs.FS) fs.FS {
	if fsys, ok := fsys.(withSMClienter); ok {
		return fsys.WithSMClient(smclient)
	}

	return fsys
}

type awssmFS struct {
	ctx        context.Context
	base       *url.URL
	httpclient *http.Client
	smclient   SecretsManagerClient
	imdsfs     fs.FS
	root       string
}

// New provides a filesystem (an fs.FS) backed by the AWS Secrets Manager,
// rooted at the given URL. Note that the URL may be either a regular
// hierarchical URL (like "aws+sm:///foo/bar") or an opaque URI (like
// "aws+sm:foo/bar"), depending on how secrets are organized in Secrets Manager.
//
// A context can be given by using WithContextFS.
func New(u *url.URL) (fs.FS, error) {
	if u.Scheme != "aws+sm" {
		return nil, fmt.Errorf("invalid URL scheme %q", u.Scheme)
	}

	root := u.Path
	if root == "" {
		root = u.Opaque
	}

	iu, _ := url.Parse("aws+imds:")

	imdsfs, err := awsimdsfs.New(iu)
	if err != nil {
		return nil, fmt.Errorf("couldn't create IMDS filesystem: %w", err)
	}

	return &awssmFS{
		ctx:    context.Background(),
		base:   u,
		root:   root,
		imdsfs: imdsfs,
	}, nil
}

// FS is used to register this filesystem with an fsimpl.FSMux
//
//nolint:gochecknoglobals
var FS = fsimpl.FSProviderFunc(New, "aws+sm")

var (
	_ fs.FS                     = (*awssmFS)(nil)
	_ fs.ReadFileFS             = (*awssmFS)(nil)
	_ fs.ReadDirFS              = (*awssmFS)(nil)
	_ fs.SubFS                  = (*awssmFS)(nil)
	_ internal.WithContexter    = (*awssmFS)(nil)
	_ internal.WithHTTPClienter = (*awssmFS)(nil)
	_ withSMClienter            = (*awssmFS)(nil)
	_ internal.WithIMDSFSer     = (*awssmFS)(nil)
)

func (f awssmFS) URL() string {
	return f.base.String()
}

func (f *awssmFS) WithContext(ctx context.Context) fs.FS {
	if ctx == nil {
		return f
	}

	fsys := *f
	fsys.ctx = ctx

	return &fsys
}

func (f *awssmFS) WithHTTPClient(client *http.Client) fs.FS {
	if client == nil {
		return f
	}

	fsys := *f
	fsys.httpclient = client

	return &fsys
}

func (f *awssmFS) WithSMClient(smclient SecretsManagerClient) fs.FS {
	if smclient == nil {
		return f
	}

	fsys := *f
	fsys.smclient = smclient

	return &fsys
}

func (f *awssmFS) WithIMDSFS(imdsfs fs.FS) fs.FS {
	if imdsfs == nil {
		return f
	}

	fsys := *f
	fsys.imdsfs = imdsfs

	return &fsys
}

func (f *awssmFS) getClient(ctx context.Context) (SecretsManagerClient, error) {
	if f.smclient != nil {
		return f.smclient, nil
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

	optFns := []func(*secretsmanager.Options){}

	// setting a host in the URL is only intended for test purposes
	if f.base.Host != "" {
		optFns = append(optFns, func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String("http://" + f.base.Host)
		})
	}

	f.smclient = secretsmanager.NewFromConfig(cfg, optFns...)

	return f.smclient, nil
}

func (f *awssmFS) Sub(name string) (fs.FS, error) {
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

func (f *awssmFS) Open(name string) (fs.File, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	smclient, err := f.getClient(f.ctx)
	if err != nil {
		return nil, err
	}

	file := &awssmFile{
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

func (f *awssmFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	smclient, err := f.getClient(f.ctx)
	if err != nil {
		return nil, err
	}

	dir := &awssmFile{
		ctx:    f.ctx,
		name:   name,
		root:   f.root,
		client: smclient,
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
func (f *awssmFS) ReadFile(name string) ([]byte, error) {
	if !internal.ValidPath(name) {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: fs.ErrInvalid}
	}

	smclient, err := f.getClient(f.ctx)
	if err != nil {
		return nil, err
	}

	secret, err := smclient.GetSecretValue(f.ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(path.Join(f.root, name)),
	})
	if err != nil {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: convertAWSError(err)}
	}

	if secret.SecretString != nil {
		return []byte(*secret.SecretString), nil
	}

	return secret.SecretBinary, nil
}

type awssmFile struct {
	ctx    context.Context
	fi     fs.FileInfo
	client SecretsManagerClient
	body   io.Reader
	name   string
	root   string

	children []*awssmFile
	diroff   int
}

var _ fs.ReadDirFile = (*awssmFile)(nil)

func (f *awssmFile) Close() error {
	// no-op - no state is kept
	return nil
}

func (f *awssmFile) Read(p []byte) (int, error) {
	if f.body == nil {
		err := f.getSecret()
		if err != nil {
			return 0, &fs.PathError{Op: "read", Path: f.name, Err: err}
		}
	}

	return f.body.Read(p)
}

func (f *awssmFile) Stat() (fs.FileInfo, error) {
	if f.fi != nil {
		return f.fi, nil
	}

	err := f.getSecret()
	if err == nil {
		return f.fi, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: err}
	}

	// may be a directory, so attempt to list one child
	// no need for special handling for opaque paths, as "." will never hit this
	// code path (Open sets f.fi to a DirInfo)
	filters := []smtypes.Filter{{Key: "name", Values: []string{path.Join(f.root, f.name) + "/"}}}
	params := secretsmanager.ListSecretsInput{MaxResults: aws.Int32(1), Filters: filters}

	secrets, err := f.client.ListSecrets(f.ctx, &params)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: convertAWSError(err)}
	}

	if len(secrets.SecretList) == 0 {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: fs.ErrNotExist}
	}

	f.fi = internal.DirInfo(f.name, time.Time{})

	return f.fi, nil
}

// convertAWSError converts an AWS error to an error suitable for returning
// from the package. We don't want to leak SDK error types.
func convertAWSError(err error) error {
	// We can't find the resource that you asked for.
	var rnfErr *smtypes.ResourceNotFoundException
	if errors.As(err, &rnfErr) {
		return fmt.Errorf("%w: %s", fs.ErrNotExist, rnfErr.ErrorMessage())
	}

	// Secrets Manager can't decrypt the protected secret text using the provided KMS key.
	var dcErr *smtypes.DecryptionFailure
	if errors.As(err, &dcErr) {
		return fmt.Errorf("%w: %s: %s", fs.ErrPermission, dcErr.ErrorCode(), dcErr.ErrorMessage())
	}

	// An error occurred on the server side.
	var internalErr *smtypes.InternalServiceError
	if errors.As(err, &internalErr) {
		return fmt.Errorf("internal error: %s: %s", internalErr.ErrorCode(), internalErr.ErrorMessage())
	}

	// You provided an invalid value for a parameter.
	var paramErr *smtypes.InvalidParameterException
	if errors.As(err, &paramErr) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, paramErr.ErrorMessage())
	}

	// You provided a parameter value that is not valid for the current state of the resource.
	var reqErr *smtypes.InvalidRequestException
	if errors.As(err, &reqErr) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, reqErr.ErrorMessage())
	}

	return err
}

// getSecret gets the secret value from AWS Secrets Manager and populates body
// and fi. SDK errors will not be leaked, instead they will be converted to more
// general errors.
func (f *awssmFile) getSecret() error {
	fullPath := path.Join(f.root, f.name)

	secret, err := f.client.GetSecretValue(f.ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &fullPath,
	})
	if err != nil {
		return fmt.Errorf("getSecretValue: %w", convertAWSError(err))
	}

	body := secret.SecretBinary

	// May be a string
	if secret.SecretString != nil {
		body = []byte(*secret.SecretString)
	}

	seclen := int64(len(body))
	f.body = bytes.NewReader(body)

	// secret versions are immutable, so the created date for this version
	// is also the last modified date
	modTime := secret.CreatedDate
	if modTime == nil {
		modTime = &time.Time{}
	}

	// populate fi
	f.fi = internal.FileInfo(f.name, seclen, 0o444, *modTime, "")

	return nil
}

// listPrefix returns the secret prefix for this directory
func (f *awssmFile) listPrefix() string {
	// when listing "." at the root (or opaque root), avoid returning "//"
	if f.name == "." && (f.root == "" || f.root == "/") {
		return f.root
	}

	return path.Join(f.root, f.name) + "/"
}

func (f *awssmFile) listSecrets() ([]smtypes.SecretListEntry, error) {
	prefix := f.listPrefix()

	filter := prefix
	if filter == "" {
		// the opaque root (aws+sm:) is a special case - we need to list
		// everything that doesn't start with /
		filter = "!/"
	}

	secretList := []smtypes.SecretListEntry{}

	for token := (*string)(nil); ; {
		params := secretsmanager.ListSecretsInput{
			Filters:   []smtypes.Filter{{Key: "name", Values: []string{filter}}},
			NextToken: token,
		}

		secrets, err := f.client.ListSecrets(f.ctx, &params)
		if err != nil {
			return nil, fmt.Errorf("listSecrets: %w", err)
		}

		// trim the prefix from the names so we can use them as filenames
		for _, secret := range secrets.SecretList {
			name := strings.TrimPrefix(*secret.Name, prefix)
			if prefix != "/" {
				name = strings.TrimPrefix(name, "/")
			}

			*secret.Name = name
		}

		secretList = append(secretList, secrets.SecretList...)

		token = secrets.NextToken
		if token == nil {
			break
		}
	}

	// no such thing as empty directories in SM, they're artificial
	if len(secretList) == 0 {
		return nil, fmt.Errorf("%w (or empty): %q", fs.ErrNotExist, prefix)
	}

	return secretList, nil
}

// list assignes a sorted list of the children of this directory to f.children
func (f *awssmFile) list() error {
	secretList, err := f.listSecrets()
	if err != nil {
		return err
	}

	// a set of files that we've already seen - we don't want to add duplicates
	seen := map[string]struct{}{}

	for _, entry := range secretList {
		parts := strings.SplitN(*entry.Name, "/", 2)
		name := parts[0]

		if _, ok := seen[name]; ok {
			continue
		}

		seen[name] = struct{}{}

		child := awssmFile{
			ctx:    f.ctx,
			name:   name,
			root:   path.Join(f.root, f.name),
			client: f.client,
		}

		if len(parts) > 1 {
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
func (f *awssmFile) ReadDir(n int) ([]fs.DirEntry, error) {
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
