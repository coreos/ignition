# URL Schemes and Filesystems

## URL Format

All supported filesystems can be requested with a base [URL][]. As a refresher,
a URL has the following components:

```pre
  foo://userinfo@example.com:8042/over/there?name=ferret#nose
  \_/   \_______________________/\_________/ \_________/ \__/
   |           |                    |            |        |
scheme     authority               path        query   fragment
```

For our purposes, the _scheme_ and the _path_ components are especially
important, though the other components are used by certain filesystems for
particular purposes.

- _scheme:_ Identifies which filesystem to use. Some filesystems allow multiple
    different schemes to clarify access modes, such as `consul+https`.
- _authority:_ Used by remote (network-based) filesystems, and can be omitted in
    some of those cases. Consists of _userinfo_ (`user:pass`), _host_, and _port_.
- _path:_ Can be omitted. Used as the root of the filesystem. For example, given
    a base of `http://example.com/foo`, calling `Open` with a path of
    `bar/baz.txt` will effectively resolve to
    `http://example.com/foo/bar/baz.txt`.
- _query:_ Used to alter filesystem-specific behaviour. For example an S3 bucket
    region can be specified with a `region` query:
    `s3://my-bucket?region=us-west-1`.
- _fragment:_ Used rarely

### Opaque URIs

For some filesystems, opaque URIs can be used (rather than a hierarchical URL):

```pre
scheme                   path        query   fragment
   |   _____________________|__   _______|_   _|
  / \ /                        \ /         \ /  \
  urn:example:animal:ferret:nose?name=ferret#nose
```

The semantics of the different URI components are essentially the same as for
hierarchical URLs (see above), but the _path_ component may not start with a `/`
character.

### Non-standard URL conventions

For the most part, this module uses standard URL conventions as much as
possible, however there are certain special cases.

#### Composite paths

Some filesystems can use _composite_ paths to add extra location context to the
path. This is marked by the presence of a double-slash (`//`) in the path
component.

For example, in a git URL like
`git+https://github.com/hairyhenderson/go-which.git//cmd/which`, the `//`
sequence is used to separate the repository from the path. In this example, the
filesystem will be rooted at `/cmd/which` inside the `go-which` repo.

## Filesystem-specific URL Considerations

### `aws+sm`

The _scheme_ and _path_ components are used by this filesystem. This may be an
[_opaque_ URI](#opaque-uris) (rather than a hierarchical URL) when the secret
name or prefix does not begin with a `/` character (e.g. `aws+sm:prod/env1`).

- _scheme_ must be `aws+sm`
- _path_ is used optionally to specify the root secret heirarchy (this may be a
hierarchical path beginning with `/`, or an opaque path without a leading `/`)

#### Examples

- `aws+sm:///` - filesystem that makes all accessible secrets available which
    are prefixed with a `/` character.
- `aws+sm:///scoped/secrets` - filesystem that makes available only secrets
    whose names begin with `/scoped/secrets/`.
- `aws+sm:` - filesystem that makes available all accessible secrets which are
    not prefixed with a `/` character.
- `aws+sm:prod/env1` - filesystem that makes available only secrets whose names
    begin with `prod/env1/`.

### `aws+smp`

The _scheme_ and _path_ components are used by this filesystem.

- _scheme_ must be `aws+smp`
- _path_ is used optionally to specify the root parameter heirarchy

#### Examples

- `aws+smp:///` - filesystem that makes all accessible parameters available
- `aws+smp:///scoped/params` - filesystem that makes available only parameters
    with a prefix of `/scoped/params`.

### `azblob`

The _scheme_, _authority_, _path_, and _query_ components are used by this
filesystem.

- _scheme_ must be `azblob`
- _authority_ is used to specify the bucket name
- _path_ is used to specify the path to root the filesystem at
- _query_ can be used to provide parameters to configure the connection:
  - `domain`: The domain name used to access the Azure Blob storage (e.g.
    `blob.core.windows.net`). Overrides any setting provided by
    `AZURE_STORAGE_DOMAIN`

#### Examples

- `azblob://mybucket/`- filesystem rooted at the root of the `mybucket` bucket
- `azblob://mybucket/configs/` - filesystem rooted at the `/configs` prefix in
    the `mybucket` bucket
- `azblob://mybucket/configs?domain=foo.example.com` - same as the previous
    example, except the domain is overridden to `foo.example.com`.

#### Azure Blob Store Environment Variables

The following optional environment variables are understood by the `azblob`
filesystem:

| name | usage |
|------|-------|
| `AZURE_STORAGE_ACCOUNT` | The Azure storage account name. Required. |
| `AZURE_STORAGE_DOMAIN` | (optional) The domain name used to access the Azure Blob storage |
| `AZURE_STORAGE_KEY` | Azure storage account key. Either this or `AZURE_STORAGE_SAS_TOKEN` must be set |
| `AZURE_STORAGE_SAS_TOKEN` | Azure shared access signature (SAS) token. Either this or `AZURE_STORAGE_KEY` must be set |

### `consul`

The _scheme_, _authority_, and _path_ components are used by this filesystem.

- _scheme_ can be `consul`, `consul+http`, or `consul+https`. The
    first two are equivalent, while the third instructs the client to connect to
    Consul over an encrypted HTTPS connection. Encryption can alternately be
    enabled by use of the `$CONSUL_HTTP_SSL` environment variable.
- _authority_ is used to specify the server to connect to (e.g.
    `consul://localhost:8500`), but if not specified, the `$CONSUL_HTTP_ADDR`
    environment variable will be used.
- _path_ is used optionally to specify the root key-space

#### Consul Environment Variables

The following optional environment variables are understood by the Consul
filesystem:

| name | usage |
|------|-------|
| `CONSUL_HTTP_ADDR` | Hostname and optional port for connecting to Consul. Defaults to `http://localhost:8500` |
| `CONSUL_TIMEOUT` | Timeout (in seconds) when communicating to Consul. Defaults to 10 seconds. |
| `CONSUL_HTTP_TOKEN` | The Consul token to use when connecting to the server. |
| `CONSUL_HTTP_AUTH` | Should be specified as `<username>:<password>`. Used to authenticate to the server. |
| `CONSUL_HTTP_SSL` | Force HTTPS if set to `true` value. Disables if set to `false`. Any value acceptable to [`strconv.ParseBool`](https://golang.org/pkg/strconv/#ParseBool) can be provided. |
| `CONSUL_TLS_SERVER_NAME` | The server name to use as the SNI host when connecting to Consul via TLS. |
| `CONSUL_CACERT` | Path to CA file for verifying Consul server using TLS. |
| `CONSUL_CAPATH` | Path to directory of CA files for verifying Consul server using TLS. |
| `CONSUL_CLIENT_CERT` | Client certificate file for certificate authentication. If this is set, `$CONSUL_CLIENT_KEY` must also be set. |
| `CONSUL_CLIENT_KEY` | Client key file for certificate authentication. If this is set, `$CONSUL_CLIENT_CERT` must also be set. |
| `CONSUL_HTTP_SSL_VERIFY` | Set to `false` to disable Consul TLS certificate checking. Any value acceptable to [`strconv.ParseBool`](https://golang.org/pkg/strconv/#ParseBool) can be provided. <br/> _Recommended only for testing and development scenarios!_ |
| `CONSUL_VAULT_ROLE` | Set to the name of the role to use for authenticating to Consul with [Vault's Consul secret backend](https://www.vaultproject.io/docs/secrets/consul/index.html). |
| `CONSUL_VAULT_MOUNT` | Used to override the mount-point when using Vault's Consul secret back-end for authentication. Defaults to `consul`. |

#### Authentication

Instead of using a non-authenticated Consul connection, you can authenticate with these methods:

- provide an [ACL Token](https://www.consul.io/docs/guides/acl.html#acl-tokens) in the `CONSUL_HTTP_TOKEN` environment variable
- use HTTP Basic Auth by setting the `CONSUL_HTTP_AUTH` environment variable
- dynamically generate an ACL token with Vault. This requires Vault to be configured to use the [Consul secret backend](https://www.vaultproject.io/docs/secrets/consul/index.html) and is enabled by passing the name of the role to use in the `CONSUL_VAULT_ROLE` environment variable.

#### Examples

- `consul:///` - filesytem that accesses Consul at `http://localhost:8500`,
    making all accessible keys available.
- `consul+https://my-consul-server.com:8533/foo` - filesystem that accesses
    the server running at `https://my-consul-server.com:8533`, making only keys
    prefixed by `/foo/` available.
- `consul:///foo/` - filesystem that accesses the server running at
    `http://localhost:8500`, making only keys prefixed by `/foo/` available.

### `file`

The _scheme_ and _path_ components are used by this filesystem.

- _scheme_ must be `file`
- _authority_ can be used on Windows when a UNC is being referenced
- _path_ can be set to root the filesystem at a given directory

#### Examples

- `file:///` - provides full access to the local filesystem. Equivalent to
    using [`os.DirFS("/")`](https://pkg.go.dev/os#DirFS). On Windows, the
    filesystem is rooted at the "current" volume.
- `file:///tmp` - provides access to the local filesystem, rooted at the `/tmp`
    directory.
- `file:///D:/` - _(Windows-specific)_ provides full access to the local
    filesystem rooted at the `D:\` volume.
- `file://./C:/Program%20Files` - _(Windows-specific)_ provides access to the
    `C:\Program Files\` directory. Note that this is equivalent to using a UNC
    in the local namespace (like `\\.\C:\...`)
- `file://remoteserver/sharename/foo` - _(Windows-specific)_ a filesystem rooted
    at the UNC `\\remoteserver\sharename\foo`

### `git`

Note that this filesystem accesses the git state, and so for local filesystem
repositories, any files not committed to a branch (i.e. "dirty" or modified
files) will not be visible.

The _scheme_, _authority_ (with _userinfo_), _path_, and _fragment_ are used by
this filesystem.

- _scheme_ may be one of these values:
  - `git`: uses the [classic Git protocol](https://git-scm.com/book/en/v2/Git-on-the-Server-The-Protocols#_the_git_protocol)
    (as served by `git daemon`)
  - `git+file`: uses the local filesystem (repo can be bare or not)
  - `git+http`, `git+https`: uses the [Smart HTTP protocol](https://git-scm.com/book/en/v2/Git-on-the-Server-The-Protocols#_the_http_protocols)
  - `git+ssh`: uses the [SSH protocol](https://git-scm.com/book/en/v2/Git-on-the-Server-The-Protocols#_the_ssh_protocol)
- _authority_ points to the remote git server hostname (and optional port, if
    applicable). The _userinfo_ subcomponent can be used for authenticated
    schemes like `git+https` and `git+ssh`.
- _path_ is a composite of the path to the repository, and the path to the
    directory being referenced within. The `//` sequence (double forward-slash)
    is used to separate the repository from the path. If no `//` is present in
    the URL, the filesystem will be rooted at the root directory of the repository.
- _fragment_ can be used to specify which branch or tag to reference. By
    default, the repository's default branch will be chosen.
  - branches can be referenced by short name or by the long form. Valid
    fragments are `#main`, `#master`, `#develop`, `#refs/heads/mybranch`, etc...
  - tags must use the long form prefixed by `refs/tags/`, i.e. `#refs/tags/v1`
    for the `v1` tag

#### Authentication

The `git` and `git+file` schemes are always unauthenticated,
`git+http`/`git+https` can _optionally_ be authenticated, and `git+ssh` _must_
be authenticated.

Authenticating with both HTTP and SSH requires the user to be set (like
`git+ssh://user@example.com`), but the credentials vary otherwise.

##### HTTP(S) Authentication

Note that because HTTP connections are unencypted, and HTTP authentication is
performed with headers, it is strongly recommended to _only_ use HTTPS
(`git+https`) connections when accessing authenticated repositories.

###### Basic Auth

The most common form. The password can be specified as part of the URL, or
provided through the `GIT_HTTP_PASSWORD` environment variable, or in a file
referenced by the `GIT_HTTP_PASSWORD_FILE` environment variable.

For authenticating with GitHub, Bitbucket, GitLab and other popular git hosts,
use this method with a _personal access token_, and the user set to `git`.

###### Token Auth

Some servers require the use of a bearer token. To use this method, a user is
_not_ required, and the token must be set in the `GIT_HTTP_TOKEN` environment
variable, or in a file referenced by the `GIT_HTTP_TOKEN_FILE` environment
variable.

##### SSH Authentication

Only public key based authentication is supported for `git+ssh` connections. The
key can be provided directly, or via the SSH Agent (or Pageant on Windows).

To provide a key directly, set the `GIT_SSH_KEY` to the contents of the key, or
point `GIT_SSH_KEY_FILE` to a file containing the key. Because the file may
contain newline characters that may be difficult to provide in an environment
variable, it can also be Base64-encoded.

If neither `GIT_SSH_KEY` nor `GIT_SSH_KEY_FILE` are set, gomplate will attempt
to use the SSH Agent.

**Note:** password-protected SSH keys are currently not supported. If you have a
password-protected key, use the SSH Agent.

#### Examples

- `git+https://github.com/hairyhenderson/gomplate//docs-src/content/functions` -
    filesystem rooted at the `github.com/hairyhenderson/gomplate` repo, rooted
    in the `/docs-src/content/functions` directory.
- `git+file:///repos/go-which` - filesystem rooted at the root of the repo
    located at `/repos/go-which` on the local filesystem.
- `git+https://github.com/hairyhenderson/go-which//cmd/which#refs/tags/v0.1.0` -
    filesystem rooted at a directory, on the `v0.1.0` tag.
- `git+ssh://git@github.com/hairyhenderson/go-which.git` - filesystem rooted
    at the root of the repo, using the SSH agent for authentication

### `gcp+sm`

The _scheme_ and _path_ components are used by this filesystem.

- _scheme_ must be `gcp+sm`
- _path_ is used optionally to specify the project (must start with `/projects/`)

#### Examples

- `gcp+sm:///projects/my-project` - filesystem rooted at the `my-project` project. Files are accessed by their secret name (e.g. `my-secret`).
- `gcp+sm:///` - filesystem with no project context. Files must be accessed by their full resource name (e.g. `projects/my-project/secrets/my-secret`).

#### Authentication

The `gcp+sm` filesystem uses Google Cloud [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/provide-credentials-adc) to find credentials. ADC can obtain credentials from several sources, including:

- User credentials configured with the `gcloud` CLI
- Service account credentials from the metadata server on GCP (GCE, GKE, Cloud Run, etc.)
- Workload Identity / Workload Identity Federation
- A service account key file referenced by the `GOOGLE_APPLICATION_CREDENTIALS` environment variable

To use the last option, set `GOOGLE_APPLICATION_CREDENTIALS` to the path of a service account JSON credentials file.

See Google Cloud's [Getting Started with Authentication](https://cloud.google.com/docs/authentication/getting-started)
and [Provide credentials for Application Default Credentials](https://cloud.google.com/docs/authentication/provide-credentials-adc)
documentation for details.

### `gs`

The _scheme_, _authority_, _path_, and _query_ components are used by this
filesystem.

- _scheme_ must be `gs`
- _authority_ is used to specify the bucket name
- _path_ is used to specify the path to root the filesystem at
- _query_ can be used to provide parameters to configure the connection:
  - `access_id`: (optional) Usually unnecessary. Sets the `GoogleAccessID` (see
    https://godoc.org/cloud.google.com/go/storage#SignedURLOptions)
  - `private_key_path`: (optional) Usually unnecessary. Sets the path to the
    Google service account private key (see
    https://godoc.org/cloud.google.com/go/storage#SignedURLOptions)

#### Authentication

Most `gs` buckets need credentials, provided by the
`GOOGLE_APPLICATION_CREDENTIALS` environment variable. This should point to an
authentication configuration JSON file.

See Google Cloud's [Getting Started with Authentication](https://cloud.google.com/docs/authentication/getting-started) 
documentation for details.

Some buckets can be accessed anonymously. To do this, set the `GOOGLE_ANON`
environment variable to `true`. Note that this is a non-standard environment
variable, unique to this module.

#### Examples

- `gs://mybucket/foo/` - filesystem rooted at `/foo` in the `mybucket` bucket.
- `gs://mybucket/`- filesystem rooted at the root of the `mybucket` bucket.

### `http`

Note that HTTP does not support directory listings, and so this filesystem does
not implement the `ReadDirFS` interface.

The _scheme_, _authority_, _path_, and _query_ components are used by this
filesystem.

- _scheme_ must be `http` or `https`
- _authority_ must be provided, and all parts are supported
- _path_ is used to specify the path to root the filesystem at
- _query_ can be used to provide parameters to the remote HTTP server

#### Examples

- `http://localhost/` - filesystem rooted at `/` on the HTTP server running at
    http://localhost:80
- `https://example.com/foo/bar?baz=42` - filesystem rooted at `/foo/bar` on the
    server running at https://example.com. All requests will be sent with the
    query string `baz=42`.

### `s3`

The _scheme_, _authority_, _path_, and _query_ components are used by this
filesystem.

- _scheme_ must be `s3`
- _authority_ is used to specify the s3 bucket name
- _path_ is used optionally to specify the root
- _query_ can be used to provide parameters to configure the connection:
  - `region`: The AWS region for requests. Defaults to the value from the
    `AWS_REGION` or `AWS_DEFAULT_REGION` environment variables, or the EC2
    region if used in AWS EC2.
  - `profile`: The shared config profile name to load from the shared
    AWS configuration files. Defaults to the value from the `AWS_PROFILE` or
    `AWS_DEFAULT_PROFILE` environment variables, or "default" if none are set.
  - `accelerate`: A value of `true` uses the [S3 Transfer Accleration](https://aws.amazon.com/s3/transfer-acceleration/) endpoints.
  - `disable_https`: A value of `true` disables the use of HTTPS when sending
    requests. Use only for test scenarios!
  - `use_path_style`: Allows you to enable the client to use path-style
    addressing, i.e., `https://s3.amazonaws.com/BUCKET/KEY`. By default, the S3
    client will use virtual hosted bucket addressing when possible
    (`https://BUCKET.s3.amazonaws.com/KEY`). This is necessary for some S3
    compatible object storage servers.
  - `anonymous`: _Experimental: May be renamed in future releases._ A value of
    `true` configures the client to not sign the request with AWS credentials.
    This is necessary for accessing public S3 buckets.
  - `dualstack`: A value of `true` configures the use of dualstack endpoint for
    a bucket. See the [AWS documentation](https://docs.aws.amazon.com/AmazonS3/latest/API/dual-stack-endpoints.html) 
    for more information.
  - `endpoint`: The endpoint (fully qualified URI). Useful for using a different
    S3-compatible object storage server. You can also set the `AWS_S3_ENDPOINT`
    environment variable.
  - `fips`: A value of `true` configures the use of the FIPS endpoint for a
    bucket. See the [AWS documentation](https://aws.amazon.com/compliance/fips/)
    for more information.
  - `rate_limiter_capacity`: An integer value configures the capacity of a token
    bucket used in client-side rate limits. If no value is set, client-side rate
    limiting is disabled. See the [AWS documentation](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/retries-timeouts/#client-side-rate-limiting).

#### Examples

- `s3://mybucket/`- filesystem rooted at the root of the `mybucket` bucket.
    The region will be inferred.
- `s3://mybucket/configs/` - filesystem rooted at the `/configs` prefix in the
    `mybucket` bucket. The region will be inferred.
- `s3://mybucket/configs?region=eu-west-1` - same as the previous example,
    except the bucket's region is overridden to `eu-west-1`.
- `s3://mybucket/configs?endpoint=localhost:5432&disableSSL=true&s3ForcePathStyle=true` -
    this example is typical of a scenario where an S3-compatible server (such as
    [Minio][], [Zenko CloudServer][], or testing-focused servers such as
    [gofakes3][]) is being used. The endpoint is overridden to be a server
    running on `localhost`, and encryption is disabled since the endpoint is
    local. Also, "path-style" access is used - typical for local servers, or
    scenarios where modifying DNS is impossible or impractical.

### `vault`

The _scheme_, _authority_, _path_, and _query_ components are used by this
filesystem.

- _scheme_ must be one of `vault`, `vault+https` (same as `vault`), or
    `vault+http`. The latter can be used to access
    [dev mode](https://www.vaultproject.io/docs/concepts/dev-server.html) Vault
    servers, for test purposes. Otherwise, all connections to Vault are
    encrypted.
- _authority_ can optionally be used to specify the Vault server's hostname and
    port. This overrides the value of `$VAULT_ADDR`.
- _path_ is used to specify the path to root the filesystem at
- _query_ is used to provide parameters to dynamic secret back-ends that require
    these. The values are included in the JSON body of the `PUT` request.

#### Examples

- `vault:///` - filesystem providing full access to accessible secrets on the
    server specified by `$VAULT_ADDR`.
- `vault://vault.example.com:8200` - filesystem providing full access to 
    accessible secrets on the server running at `vault.example.com` over HTTPS
    at port `8200`
- `vault:///ssh/creds/?ip=10.1.2.3&username=user` - filesystem that allows
    reading dynamic secrets with the parameters `ip` and `username` provided
    in the body
- `vault:///secret/configs/` - filesystem rooted at `/secret/configs` on the
    server at `$VAULT_ADDR`

#### Vault Authentication

This table describes the currently-supported authentication mechanisms and how to use them, in order of precedence:

| auth back-end | configuration |
|-------------:|---------------|
| [`approle`](https://www.vaultproject.io/docs/auth/approle.html) | Environment variables `$VAULT_ROLE_ID` and `$VAULT_SECRET_ID` must be set to the appropriate values.<br/> If the back-end is mounted to a different location, set `$VAULT_AUTH_APPROLE_MOUNT`. |
| [`github`](https://www.vaultproject.io/docs/auth/github.html) | Environment variable `$VAULT_AUTH_GITHUB_TOKEN` must be set to an appropriate value.<br/> If the back-end is mounted to a different location, set `$VAULT_AUTH_GITHUB_MOUNT`. |
| [`userpass`](https://www.vaultproject.io/docs/auth/userpass.html) | Environment variables `$VAULT_AUTH_USERNAME` and `$VAULT_AUTH_PASSWORD` must be set to the appropriate values.<br/> If the back-end is mounted to a different location, set `$VAULT_AUTH_USERPASS_MOUNT`. |
| [`token`](https://www.vaultproject.io/docs/auth/token.html) | Determined from either the `$VAULT_TOKEN` environment variable, or read from the file `~/.vault-token` |
| [`aws`](https://www.vaultproject.io/docs/auth/aws.html) | The env var  `$VAULT_AUTH_AWS_ROLE` defines the [role](https://www.vaultproject.io/api/auth/aws/index.html#role-4) to log in with - defaults to the AMI ID of the EC2 instance. Usually a [Client Nonce](https://www.vaultproject.io/docs/auth/aws.html#client-nonce) should be used as well. Set `$VAULT_AUTH_AWS_NONCE` to the nonce value. The nonce can be generated and stored by setting `$VAULT_AUTH_AWS_NONCE_OUTPUT` to a path on the local filesystem.<br/>If the back-end is mounted to a different location, set `$VAULT_AUTH_AWS_MOUNT`.|
| [`app-id`](https://www.vaultproject.io/docs/auth/app-id.html) | **(Deprecated - use `approle` instead)** |

_**Note:**_ The secret values listed in the above table can either be set in
environment variables or provided in files for increased security. To use files,
specify the filename by appending `_FILE` to the environment variable, (e.g.
`VAULT_SECRET_ID_FILE`). If the non-file variable is set, this will override any
`_FILE` variable and the secret file will be ignored.

#### Vault Permissions

The correct capabilities must be allowed for the [authenticated](#vault-authentication) credentials. See the [Vault documentation](https://www.vaultproject.io/docs/concepts/policies.html#capabilities) for full details.

- regular secret read operations require the `read` capability
- dynamic secret generation requires the `create` and `update` capabilities
- list support requires the `list` capability

#### Vault Environment variables

In addition to the variables documented [above](#vault-authentication), a number of environment variables are interpreted by the Vault client, and are documented in the [official Vault documentation](https://www.vaultproject.io/docs/commands/index.html#environment-variables).


[URL]: https://tools.ietf.org/html/rfc3986

[Minio]: https://min.io
[Zenko CloudServer]: https://www.zenko.io/cloudserver/
[gofakes3]: https://github.com/johannesboyne/gofakes3
