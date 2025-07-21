package vaultauth

import (
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
	"github.com/hashicorp/vault/api/auth/userpass"
)

// EnvAuthMethod configures the auth method based on environment variables. It
// will attempt to authenticate with the following four methods, in order of
// precedence:
//
// # approle
//
// The [github.com/hashicorp/vault/api/auth/approle.NewAppRoleAuth] is called,
// using the roleID from $VAULT_ROLE_ID and the secretID from $VAULT_SECRET_ID.
// The default mount path can be overridden with $VAULT_AUTH_APPROLE_MOUNT.
//
// # github
//
// The [NewGitHubAuth] is called, using the token from $VAULT_AUTH_GITHUB_TOKEN.
// The default mount path can be overridden with $VAULT_AUTH_GITHUB_MOUNT.
//
// # userpass
//
// The [github.com/hashicorp/vault/api/auth/userpass.NewUserpassAuth] is called,
// using the username from $VAULT_AUTH_USERNAME and the password from
// $VAULT_AUTH_PASSWORD. The default mount path can be overridden with
// $VAULT_AUTH_USERPASS_MOUNT.
//
// # token
//
// The [NewTokenAuth] is called, using the token from $VAULT_TOKEN, or the
// token contained in $HOME/.vault-token.
//
// Note that this auth method is provided as a convenience, and is not intended
// to be heavily depended upon. It is recommended that you use the auth methods
// directly, and configure them with the appropriate options.
func EnvAuthMethod() api.AuthMethod {
	return CompositeAuthMethod(
		envAppRoleAdapter(),
		envGitHubAdapter(),
		envUserPassAdapter(),
		NewTokenAuth(""),
	)
}

// envAppRoleAdapter builds an AppRoleAuth from environment variables, for use
// only with [EnvAuthMethod]
func envAppRoleAdapter() api.AuthMethod {
	roleID := os.Getenv("VAULT_ROLE_ID")
	if roleID == "" {
		return nil
	}

	secretID := &approle.SecretID{FromEnv: "VAULT_SECRET_ID"}

	var opts []approle.LoginOption

	mountPath := os.Getenv("VAULT_AUTH_APPROLE_MOUNT")
	if mountPath != "" {
		opts = []approle.LoginOption{approle.WithMountPath(mountPath)}
	}

	a, err := approle.NewAppRoleAuth(roleID, secretID, opts...)
	if err != nil {
		return nil
	}

	return a
}

// envGitHubAdapter builds a GitHubAuth from environment variables, for use only
// with [EnvAuthMethod]
func envGitHubAdapter() api.AuthMethod {
	var opts []GitHubLoginOption

	mountPath := os.Getenv("VAULT_AUTH_GITHUB_MOUNT")
	if mountPath != "" {
		opts = []GitHubLoginOption{WithGitHubMountPath(mountPath)}
	}

	token := &GitHubToken{FromEnv: "VAULT_AUTH_GITHUB_TOKEN"}

	a, err := NewGitHubAuth(token, opts...)
	if err != nil {
		return nil
	}

	return a
}

// envUserPassAdapter builds a UserPassAuth from environment variables, for use
// only with [EnvAuthMethod]
func envUserPassAdapter() api.AuthMethod {
	username := os.Getenv("VAULT_AUTH_USERNAME")
	if username == "" {
		return nil
	}

	password := &userpass.Password{FromEnv: "VAULT_AUTH_PASSWORD"}

	var opts []userpass.LoginOption

	mountPath := os.Getenv("VAULT_AUTH_USERPASS_MOUNT")
	if mountPath != "" {
		opts = []userpass.LoginOption{userpass.WithMountPath(mountPath)}
	}

	a, err := userpass.NewUserpassAuth(username, password, opts...)
	if err != nil {
		return nil
	}

	return a
}
