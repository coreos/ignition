package consulfs

import (
	"io/fs"

	"github.com/hashicorp/consul/api/v2"
)

type withConfiger interface {
	WithConfig(config *api.Config) fs.FS
}

type withQueryOptionser interface {
	WithQueryOptions(opts *api.QueryOptions) fs.FS
}

type withTokener interface {
	WithToken(token string) fs.FS
}

// WithTokenFS sets the Consul token used by fsys, if the filesystem supports it
// (i.e. has a WithToken method).
//
// Note that calling WithTokenFS will reset the underlying client, which may
// introduce a performance penalty.
func WithTokenFS(token string, fsys fs.FS) fs.FS {
	if fsys, ok := fsys.(withTokener); ok {
		return fsys.WithToken(token)
	}

	return fsys
}

// WithConfigFS configures the Consul client used by fsys, if the filesystem
// supports it (i.e. has a WithConfig method). This can be used to set a custom
// HTTP client, TLS configuration, etc... This can not be used to set the
// address of the Consul server, as that is set by the base URL or the
// CONSUL_HTTP_ADDR environment variable.
//
// If you only want to set the token, use [WithTokenFS] instead.
//
// Note that calling WithConfigFS will reset the underlying client, which may
// introduce a performance penalty.
func WithConfigFS(config *api.Config, fsys fs.FS) fs.FS {
	if fsys, ok := fsys.(withConfiger); ok {
		return fsys.WithConfig(config)
	}

	return fsys
}

// WithQueryOptionsFS sets Consul KV query options used by fsys, if the
// filesystem supports it (i.e. has a WithQueryOptions method). This can not be
// used to set the request context (use [fsimpl.WithContextFS] instead).
//
// For options that can also be set in the config, those will be overridden
// temporarily by the options set here. The underlying configuration and client
// will be preserved on the filesystem.
func WithQueryOptionsFS(opts *api.QueryOptions, fsys fs.FS) fs.FS {
	if fsys, ok := fsys.(withQueryOptionser); ok {
		return fsys.WithQueryOptions(opts)
	}

	return fsys
}
