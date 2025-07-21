package vaultauth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/hashicorp/vault/api"
)

// withAuthMethoder is an fs.FS that can be configured with a custom AuthMethod
type withAuthMethoder interface {
	WithAuthMethod(auth api.AuthMethod) fs.FS
}

// WithAuthMethod configures the given FS to authenticate with auth, if the
// filesystem supports it.
//
// Note that this is not required if $VAULT_TOKEN is set.
func WithAuthMethod(auth api.AuthMethod, fsys fs.FS) fs.FS {
	if afsys, ok := fsys.(withAuthMethoder); ok {
		return afsys.WithAuthMethod(auth)
	}

	return fsys
}

func remoteAuth(ctx context.Context, client *api.Client, mount, extra string, vars map[string]any) (*api.Secret, error) {
	p := path.Join("auth", mount, "login", extra)

	secret, err := client.Logical().WriteWithContext(ctx, p, vars)
	if err != nil {
		return nil, fmt.Errorf("vault write to %s failed: %w", p, vaultFSError(err))
	}

	return secret, nil
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

// CompositeAuthMethod returns an AuthMethod that will try each of the given
// methods in order, until one succeeds.
func CompositeAuthMethod(methods ...api.AuthMethod) api.AuthMethod {
	return &compositeAuthMethod{methods: methods}
}

type compositeAuthMethod struct {
	chosen  api.AuthMethod
	methods []api.AuthMethod
}

func (m *compositeAuthMethod) Login(ctx context.Context, client *api.Client) (secret *api.Secret, err error) {
	if m.chosen == nil {
		for _, auth := range m.methods {
			if auth == nil {
				continue
			}

			secret, err = auth.Login(ctx, client)
			if err == nil {
				m.chosen = auth

				break
			}
		}
	}

	if m.chosen == nil {
		return nil, fmt.Errorf("unable to authenticate with vault by any configured method. Last error was: %w", err)
	}

	return secret, nil
}

func (m *compositeAuthMethod) Logout(ctx context.Context, client *api.Client) {
	if m.chosen == nil {
		return
	}

	// some auth methods (like the token method) manage their own logout, to
	// avoid revoking the token, which shouldn't be managed here
	if lauth, ok := m.chosen.(interface {
		Logout(ctx context.Context, client *api.Client)
	}); ok {
		lauth.Logout(ctx, client)
	} else {
		_, _ = client.Logical().WriteWithContext(ctx, "auth/token/revoke-self", nil)
		client.ClearToken()
	}
}
