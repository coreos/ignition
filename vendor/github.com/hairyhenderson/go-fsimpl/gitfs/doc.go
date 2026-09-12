// Package gitfs provides a read-only filesystem backed by a git repository.
//
// This filesystem accesses the git state, and so for local repositories, files
// not committed to a branch (i.e. "dirty" or modified files) will not be
// visible.
//
// This filesystem's behaviour complies with fstest.TestFS.
//
// # Usage
//
// To use this filesystem, call New with a base URL. All reads from the
// filesystem are relative to this base URL. Valid schemes are 'git', 'file',
// 'http', 'https', 'ssh', and the same prefixed with 'git+' (e.g.
// 'git+ssh://example.com').
//
// # URL Format
//
// The scheme, authority (with userinfo), path, and fragment are used by this
// filesystem.
//
// Scheme may be one of:
//
// - 'git': use the classic Git protocol, as served by 'git daemon'
//
// - 'file': use the local filesystem (repo can be bare or not)
//
// - 'http'/'https': use the Smart HTTP protocol
//
// - 'ssh': use the SSH protocol
//
// See https://git-scm.com/book/en/v2/Git-on-the-Server-The-Protocols for more
// on these protocols.
//
// Authority points to the remote git server hostname (and optional port, if
// applicable). The userinfo subcomponent (i.e. 'user:password@...') can be used for
// authenticated schemes like 'https' and 'ssh'.
//
// Path is a composite of the path to the repository and the path to a directory
// referenced within. The '//' sequence (double forward-slash) is used to
// separate the repository from the path. If no '//' is present in the path, the
// filesystem will be rooted at the root directory of the repository.
//
// Fragment is used to specify which branch or tag to reference. When not
// specified, the repository's default branch will be chosen.
// Branches are referenced by short name (such as '#main') or by the long form
// prefixed with '#refs/heads/'. Valid examples are '#develop',
// '#refs/heads/mybranch', etc...
// Tags are referenced by long form, prefixed with 'refs/tags/'. Valid examples
// are '#refs/tags/v1', '#refs/tags/mytag', etc...
//
// Here are a few more examples of URLs valid for this filesystem:
//
//	git+https://github.com/hairyhenderson/gomplate//docs-src/content/functions
//	git+file:///repos/go-which
//	git+https://github.com/hairyhenderson/go-which//cmd/which#refs/tags/v0.1.0
//	git+ssh://git@github.com/hairyhenderson/go-which.git
//
// # Authentication
//
// The authentication mechanisms used by gitfs are dependent on the URL scheme.
// A number of Authenticators are provided in this package. See the
// documentation for the Authenticator type for more information.
//
// # Environment Variables
//
// The Authenticators in this package optionally support the use of environment
// variables to provide credentials. These are:
//
// - GIT_HTTP_PASSWORD: the password to use for HTTP Basic Authentication
//
// - GIT_HTTP_PASSWORD_FILE: the path to a file containing the password to use
// for HTTP Basic Authentication
//
// - GIT_HTTP_TOKEN: the token to use for HTTP token authentication
//
// - GIT_HTTP_TOKEN_FILE: the path to a file containing the token to use for
// HTTP token authentication
//
// - GIT_SSH_KEY: the (optionally Base64-encoded) PEM-encoded private key to use
// for SSH public key authentication
//
// - GIT_SSH_KEY_FILE: the path to a file containing the PEM-encoded private key
// to use for SSH public key authentication
package gitfs
