package vaultauth

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/vault/api"
)

// NewTokenAuth authenticates with the given token, or if none is provided,
// attempts to read from the $VAULT_TOKEN environment variable, or the
// $HOME/.vault-token file.
//
// When using this method, the token is not managed by vaultfs, and will not be
// revoked when files are closed. It is the responsibility of the caller to
// manage the token.
//
// See also https://www.vaultproject.io/docs/auth/token
func NewTokenAuth(token string) api.AuthMethod {
	return &tokenAuthMethod{token: token, fsys: os.DirFS("/")}
}

type tokenAuthMethod struct {
	fsys  fs.FS
	token string
}

func (m *tokenAuthMethod) Login(_ context.Context, _ *api.Client) (*api.Secret, error) {
	if m.token != "" {
		return &api.Secret{Auth: &api.SecretAuth{ClientToken: m.token}}, nil
	}

	// maybe $VAULT_TOKEN is set?
	if token := os.Getenv("VAULT_TOKEN"); token != "" {
		return &api.Secret{Auth: &api.SecretAuth{ClientToken: token}}, nil
	}

	// ok, let's try $HOME/.vault-token
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	p := path.Join(homeDir, ".vault-token")
	p = strings.TrimPrefix(p, "/")

	b, err := fs.ReadFile(m.fsys, p)
	if err != nil {
		return nil, fmt.Errorf("readFile %q: %w", p, err)
	}

	return &api.Secret{Auth: &api.SecretAuth{ClientToken: string(b)}}, nil
}

// Logout implements the vaultfs.authLogouter interface because we need to keep
// the token unmanaged.
func (m *tokenAuthMethod) Logout(_ context.Context, client *api.Client) {
	// just clear the client's token, nothing else needs to be done here
	client.ClearToken()
}
