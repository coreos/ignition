package vaultfs

import (
	"io/fs"

	"github.com/hashicorp/vault/api"
)

type withClienter interface {
	WithClient(client *api.Client) fs.FS
}

// WithClient injects a Vault client into the filesystem fs, if the
// filesystem supports it (i.e. is a [FS], or some other type with a
// WithClient method). The current client will be replaced. It is the
// caller's responsibility to ensure the client is configured correctly.
func WithClient(client *api.Client, fsys fs.FS) fs.FS {
	if cfsys, ok := fsys.(withClienter); ok {
		return cfsys.WithClient(client)
	}

	return fsys
}

type withConfiger interface {
	WithConfig(config *api.Config) fs.FS
}

// WithConfig injects a Vault configuration into the filesystem fs, if the
// filesystem supports it (i.e. is a [FS], or some other type with a
// WithConfig method). The current client will be replaced. If the
// configuration is invalid, an error will be logged and nil will be returned.
func WithConfig(config *api.Config, fsys fs.FS) fs.FS {
	if cfsys, ok := fsys.(withConfiger); ok {
		return cfsys.WithConfig(config)
	}

	return fsys
}
