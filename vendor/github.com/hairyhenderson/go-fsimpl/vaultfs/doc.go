// Package vaultfs provides an interface to [Hashicorp Vault] which allows you to
// interact with the Vault API as a standard filesystem.
//
// This filesystem's behaviour complies with [testing/fstest.TestFS].
//
// # Usage
//
// To use this filesystem, call [New] with a base URL. All reads from the
// filesystem are relative to this base URL. The schemes "vault", "vault+https",
// "https", "vault+http", and "http" are supported, though the http schemes
// should only be used in development/test environments. If no authority part
// (host:port) is present in the URL, the $VAULT_ADDR environment variable will
// be used as the Vault server's address.
//
// To scope the filesystem to a specific path, use that path on the URL. For
// example, for a filesystem that can only read from the K/V engine mounted on
// "secret" with a prefix of "dev", you could use a URL like:
//
//	vault:///secret/dev/
//
// Note: when scoping URLs to specific paths, the URL must end in "/".
//
// Reading secrets and credentials from all Vault Secret Engines is supported,
// though some may require parameters to be set. To set parameters, append a
// URL query string to the path when reading (in the form
// "?param=value&param2=value2"). When parameters are set in this way, vaultfs
// will send a POST request to the Vault API, except when the K/V Version 2
// secret engine is in use.
//
// In general, data is read from Vault in the same way as with the Vault CLI -
// that is, the "/v1" prefix is not needed, and with the K/V Version 2 secret
// engine the "data" prefix should not be provided.
//
// When reading from K/V v2 secret engines, specific versions of the secret can
// be read by providing a "version" query parameter. For example, to read the
// fifth version of the secret at "secret/mysecret", you could use a URL like:
//
//	vault:///secret/mysecret?version=5
//
// See the [Vault Secret Engine Docs] for more details.
//
// # Authentication
//
// A number of authentication methods are supported and documented in detail
// by the [vaultauth] package. Use the [vaultauth.WithAuthMethod]
// function to set your desired auth method. Custom auth methods can be created
// by implementing the [github.com/hashicorp/vault/api.AuthMethod] interface.
//
// By default, the $VAULT_TOKEN environment variable will be used as the Vault
// token, falling back to the content of $HOME/.vault-token.
//
// When multiple files are opened simultaneously, the same authentication token
// will be used for each of them, and will only be revoked when the last file is
// closed. This ensures that a minimal number of tokens are acquired, however
// this also means that tokens may be leaked if all opened files are not closed.
//
// See the [vaultauth] docs for details on each auth method.
//
// For help in deciding which auth method to use, consult the [Vault Auth Docs].
//
// # Permissions
//
// The correct capabilities must be allowed for the authenticated credentials.
// Regular secret read operations require the "read" capability, dynamic secret
// generation requires "create" and "update", and listing (ReadDir) requires the
// "list" capability.
//
// See [Vault Capabilities Docs] for more details on how to configure these on
// your Vault server.
//
// # Environment Variables
//
// A number of environment variables are understood by the Go Vault client that
// vaultfs uses internally. See [Vault Client Environment Variable Docs] for
// detail.
//
// [Vault Auth Docs]: https://vaultproject.io/docs/auth
// [Vault Capabilities Docs]: https://vaultproject.io/docs/concepts/policies#capabilities
// [Vault Client Environment Variable Docs]: https://vaultproject.io/docs/commands#environment-variables
// [Vault Secret Engine Docs]: https://vaultproject.io/docs/secrets
// [Hashicorp Vault]: https://vaultproject.io
package vaultfs
