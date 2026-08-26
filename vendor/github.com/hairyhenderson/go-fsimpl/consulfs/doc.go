// Package consulfs provides an interface to Hashicorp Consul which allows you
// to interact with the Consul K/V store as a standard filesystem.
//
// This filesystem's behaviour complies with [testing/fstest.TestFS].
//
// # Usage
//
// To use this filesystem, call [New] with a base URL. All reads from the
// filesystem are relative to this base URL. The schemes "consul",
// "consul+https", "https", "consul+http", and "http" are supported, though the
// http schemes should only be used in development/test environments. If no
// authority part (host:port) is present in the URL, the CONSUL_HTTP_ADDR
// environment variable will be used as the Consul server's address.
//
// To scope the filesystem to a specific path, use that path on the URL. For
// example, for a filesystem that can only read from keys prefixed with "foo",
// you could use a URL like "consul://consulserver.local/foo/".
//
// Note: when scoping URLs to specific paths, the URL must end in "/".
//
// In general, data is read from Consul in the same way as with the Consul CLI -
// that is, the "/kv" prefix is not needed.
//
// See the [Consul KV Store docs] for more details.
//
// # Authentication
//
// To authenticate with Consul, an [ACL Token] will need to be set. You can set
// the token in a few ways: with the CONSUL_HTTP_TOKEN environment variable, by
// providing a custom Consul client configuration with the [WithConfigFS]
// extension, by providing custom Consul query options with the
// [WithQueryOptionsFS] extension, or by providing a token directly with the
// [WithTokenFS] extension.
//
// Usually, you'll want to use the [WithTokenFS] extension.
//
// # Extensions
//
// The filesystem may be configured with a few standard and consulfs-specific
// extensions. See the documentation for each extension for more details:
//
//   - [fsimpl.WithContextFS]
//   - [fsimpl.WithHeaderFS]
//   - [WithTokenFS]
//   - [WithConfigFS]
//   - [WithQueryOptionsFS]
//
// [Consul KV Store docs]: https://www.consul.io/docs/dynamic-app-config/kv
// [ACL Token]: https://www.consul.io/docs/security/acl/acl-tokens
package consulfs
