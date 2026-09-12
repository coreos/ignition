package gitfs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/hairyhenderson/go-fsimpl/internal/env"
)

// withAuthenticatorer is an fs.FS that can be configured to authenticate to a
// git repository using a specific method
type withAuthenticatorer interface {
	WithAuthenticator(auth Authenticator) fs.FS
}

// Authenticator provides an AuthMethod for a given URL. If the URL is not
// appropriate for the given AuthMethod, an error will be returned.
type Authenticator interface {
	Authenticate(u *url.URL) (AuthMethod, error)
}

type authenticatorFunc func(u *url.URL) (AuthMethod, error)

func (a authenticatorFunc) Authenticate(u *url.URL) (AuthMethod, error) {
	return a(u)
}

// AuthMethod is an HTTP or SSH authentication method that can be used to
// authenticate to a git repository.
// See the github.com/go-git/go-git module for details.
type AuthMethod interface {
	fmt.Stringer
	Name() string
}

// WithAuthenticator configures the given FS to authenticate with auth, if the
// filesystem supports it.
func WithAuthenticator(auth Authenticator, fsys fs.FS) fs.FS {
	if afsys, ok := fsys.(withAuthenticatorer); ok {
		return afsys.WithAuthenticator(auth)
	}

	return fsys
}

var (
	_ Authenticator = (*authenticatorFunc)(nil)
	_ Authenticator = (*basicAuthenticator)(nil)
)

// AutoAuthenticator is an Authenticator that chooses the first available
// authenticator based on the given URL (when appropriate) and the environment
// variables, in this order of precedence:
//
//	BasicAuthenticator
//	TokenAuthenticator
//	PublicKeyAuthenticator
//	SSHAgentAuthenticator
//	NoopAuthenticator
func AutoAuthenticator() Authenticator {
	return &autoAuthenticator{
		authenticators: []Authenticator{
			BasicAuthenticator("", ""),
			TokenAuthenticator(""),
			PublicKeyAuthenticator("", []byte{}, ""),
			SSHAgentAuthenticator(""),
			NoopAuthenticator(),
		},
	}
}

type autoAuthenticator struct {
	authenticators []Authenticator
}

func (a *autoAuthenticator) Authenticate(u *url.URL) (AuthMethod, error) {
	for _, auth := range a.authenticators {
		if method, err := auth.Authenticate(u); err == nil {
			return method, nil
		}
	}

	return nil, fmt.Errorf("no authentication method available for %s", u)
}

// NoopAuthenticator is an Authenticator that will not attempt any
// authentication methods. Can only be used with 'git', 'file', 'http', and
// 'https' schemes.
//
// Useful when desiring no authentication at all (e.g. for local repositories,
// or to ensure that target repositories are public).
func NoopAuthenticator() Authenticator {
	return authenticatorFunc(func(u *url.URL) (AuthMethod, error) {
		if u.Scheme != "git" && u.Scheme != "file" && u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("no-op authentication not supported for scheme %q", u.Scheme)
		}

		return nil, nil
	})
}

// BasicAuthenticator is an Authenticator that provides HTTP Basic
// Authentication. Use only with HTTP/HTTPS repositories.
//
// A username or password provided in the URL will override the credentials
// provided here.
// If password is omitted, the environment variable GIT_HTTP_PASSWORD will be
// used.
// If GIT_HTTP_PASSWORD_FILE is set, the password will be read from the
// referenced file on the local filesystem.
//
// For authenticating with GitHub, Bitbucket, GitLab, and other popular git
// hosts, use this with a personal access token, with username 'git'.
func BasicAuthenticator(username, password string) Authenticator {
	return &basicAuthenticator{
		envfsys: os.DirFS("/"), username: username, password: password,
	}
}

type basicAuthenticator struct {
	envfsys            fs.FS
	username, password string
}

func (a *basicAuthenticator) Authenticate(u *url.URL) (AuthMethod, error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("basic authentication not supported for scheme %q", u.Scheme)
	}

	username := u.User.Username()
	if username == "" {
		username = a.username
	}

	// password can come from URL, environment, or the provided one
	password, _ := u.User.Password()
	if password == "" {
		password = a.password
	}

	if password == "" {
		password = env.GetenvFS(a.envfsys, "GIT_HTTP_PASSWORD")
	}

	if username == "" && password == "" {
		return nil, nil
	}

	return &githttp.BasicAuth{Username: username, Password: password}, nil
}

// TokenAuthenticator is an Authenticator that uses HTTP token authentication
// (also known as bearer authentication).
//
// If token is omitted, the environment variable GIT_HTTP_TOKEN will be	used.
// If GIT_HTTP_TOKEN_FILE is set, the token will be read from the referenced
// file on the local filesystem.
//
// Note: If you are looking to use OAuth tokens with popular servers (e.g.
// GitHub, Bitbucket, GitLab), use BasicAuthenticator instead. These servers
// use HTTP Basic Authentication, with the OAuth token as user or password.
func TokenAuthenticator(token string) Authenticator {
	return &tokenAuthenticator{envfsys: os.DirFS("/"), token: token}
}

type tokenAuthenticator struct {
	envfsys fs.FS
	token   string
}

func (a *tokenAuthenticator) Authenticate(u *url.URL) (AuthMethod, error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("token authentication not supported for scheme %q", u.Scheme)
	}

	token := a.token
	if token == "" {
		token = env.GetenvFS(a.envfsys, "GIT_HTTP_TOKEN")
	}

	if token == "" {
		return nil, errors.New("token may not be empty for token authentication")
	}

	return &githttp.TokenAuth{Token: token}, nil
}

// PublicKeyAuthenticator provides an Authenticator that uses SSH public key
// authentication. Use only with SSH repositories.
//
// The privkey is a PEM-encoded private key. Set keyPass if privKey is a
// password-encrypted PEM block, otherwise leave it empty.
//
// If privKey is omitted, the GIT_SSH_KEY environment variable will be used. For
// ease of use, the variable may optionally be base64-encoded. If
// GIT_SSH_KEY_FILE is set, the key will be read from the referenced file on the
// local filesystem.
//
// Supports PKCS#1 (RSA), PKCS#8 (RSA, ECDSA, ed25519), SEC 1 (ECDSA),
// DSA (OpenSSL), and OpenSSH private keys.
func PublicKeyAuthenticator(username string, privKey []byte, keyPass string) Authenticator {
	return &publicKeyAuthenticator{
		envfsys: os.DirFS("/"), username: username, privKey: privKey, keyPass: keyPass,
	}
}

type publicKeyAuthenticator struct {
	envfsys  fs.FS
	username string
	keyPass  string
	privKey  []byte
}

func (a *publicKeyAuthenticator) Authenticate(u *url.URL) (AuthMethod, error) {
	if u.Scheme != "ssh" {
		return nil, fmt.Errorf("public key authentication not supported for scheme %q", u.Scheme)
	}

	username := u.User.Username()
	if username == "" {
		username = a.username
	}

	k := a.privKey
	if len(k) == 0 {
		var err error

		envKey := env.GetenvFS(a.envfsys, "GIT_SSH_KEY")

		k, err = base64.StdEncoding.DecodeString(envKey)
		if err != nil {
			// if it can't be base64-decoded, assume it was a non-base64 key
			// from a file
			k = []byte(envKey)
		}
	}

	if len(k) == 0 {
		return nil, errors.New("private key may not be empty for public key authentication")
	}

	return ssh.NewPublicKeys(username, k, a.keyPass)
}

// SSHAgentAuthenticator is an Authenticator that uses the ssh-agent protocol.
// Use only with SSH repositories.
//
// If username is not provided or present in the URL, the user will be the same
// as the current user.
//
// This method depends on the SSH_AUTH_SOCK environment variable being correctly
// configured by the SSH agent. See ssh-agent(1) for details.
func SSHAgentAuthenticator(username string) Authenticator {
	return &sshAgentAuthenticator{envfsys: os.DirFS("/"), username: username}
}

type sshAgentAuthenticator struct {
	envfsys  fs.FS
	username string
}

func (a *sshAgentAuthenticator) Authenticate(u *url.URL) (AuthMethod, error) {
	if u.Scheme != "ssh" {
		return nil, fmt.Errorf("ssh-agent authentication not supported for scheme %q", u.Scheme)
	}

	username := u.User.Username()
	if username == "" {
		username = a.username
	}

	return ssh.NewSSHAgentAuth(username)
}
