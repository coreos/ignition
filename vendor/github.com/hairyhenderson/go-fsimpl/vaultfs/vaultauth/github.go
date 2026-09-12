package vaultauth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
)

// NewGitHubAuth authenticates to Vault with the GitHub auth method.
//
// Use [WithGitHubMountPath] to specify the mount path for the GitHub auth
// method. If not specified, the default is "github".
//
// See also https://www.vaultproject.io/docs/auth/github
func NewGitHubAuth(token *GitHubToken, opts ...GitHubLoginOption) (api.AuthMethod, error) {
	err := token.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid github token: %w", err)
	}

	a := &gitHubAuthMethod{
		fsys:      os.DirFS("/"),
		mountPath: "github",
		token:     token,
	}

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, fmt.Errorf("error from GitHub login option: %w", err)
		}
	}

	return a, nil
}

type GitHubLoginOption func(a *gitHubAuthMethod) error

func WithGitHubMountPath(mountPath string) GitHubLoginOption {
	return func(a *gitHubAuthMethod) error {
		a.mountPath = mountPath

		return nil
	}
}

// GitHubToken is a struct that allows you to specify where your application is
// storing the token required for login to the GitHub auth method.
type GitHubToken struct {
	FromFile   string
	FromString string
	FromEnv    string
}

//nolint:gocyclo
func (token *GitHubToken) validate() error {
	if token == nil {
		return errors.New("github auth method requires a token")
	}

	if token.FromFile == "" && token.FromEnv == "" && token.FromString == "" {
		return errors.New("token for GitHub auth must be provided with a source file, environment variable, or plaintext string")
	}

	if token.FromFile != "" && (token.FromEnv != "" || token.FromString != "") {
		return errors.New("only one source for the token should be specified")
	}

	if token.FromEnv != "" && (token.FromFile != "" || token.FromString != "") {
		return errors.New("only one source for the token should be specified")
	}

	return nil
}

type gitHubAuthMethod struct {
	fsys      fs.FS
	token     *GitHubToken
	mountPath string
}

func (a *gitHubAuthMethod) Login(ctx context.Context, client *api.Client) (*api.Secret, error) {
	token := ""

	switch {
	case a.token.FromFile != "":
		t, err := a.readSecretIDFromFile()
		if err != nil {
			return nil, fmt.Errorf("error reading GitHub token from file: %w", err)
		}

		token = t
	case a.token.FromEnv != "":
		token = os.Getenv(a.token.FromEnv)
		if token == "" {
			return nil, fmt.Errorf("GitHub token environment variable %q not set", a.token.FromEnv)
		}
	default:
		token = a.token.FromString
	}

	secret, err := remoteAuth(ctx, client, a.mountPath, "",
		map[string]any{"token": token})
	if err != nil {
		return nil, fmt.Errorf("github login failed: %w", err)
	}

	return secret, nil
}

func (a *gitHubAuthMethod) readSecretIDFromFile() (string, error) {
	f, err := a.fsys.Open(a.token.FromFile)
	if err != nil {
		return "", fmt.Errorf("unable to open file containing GitHub token: %w", err)
	}
	defer f.Close()

	// limit the read since GH tokens are ~short
	b, err := io.ReadAll(io.LimitReader(f, 512))
	if err != nil {
		return "", fmt.Errorf("unable to read GitHub token: %w", err)
	}

	// trim any accidentally-added leading or trailing whitespace
	return strings.TrimSpace(string(b)), nil
}
