// Package vaultauth provides an interface to a few custom Vault auth methods
// for use with [github.com/hairyhenderson/go-fsimpl/vaultfs], but which can
// also be used directly with a [*github.com/hashicorp/vault/api.Client].
//
// See also these auth methods provided with the Vault API:
//   - [github.com/hashicorp/vault/api/auth/approle]
//   - [github.com/hashicorp/vault/api/auth/aws]
//   - [github.com/hashicorp/vault/api/auth/azure]
//   - [github.com/hashicorp/vault/api/auth/gcp]
//   - [github.com/hashicorp/vault/api/auth/kubernetes]
//   - [github.com/hashicorp/vault/api/auth/ldap]
//   - [github.com/hashicorp/vault/api/auth/userpass]
package vaultauth
