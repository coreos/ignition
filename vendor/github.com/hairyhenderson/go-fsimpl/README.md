# hairyhenderson/go-fsimpl

[![GoDoc][godoc-image]][godocs]
[![Build][gh-actions-image]][gh-actions-url]

This module contains a collection of Go _filesystem implementations_ that can
be discovered dynamically by URL scheme.

These filesystems implement the [`fs.FS`](https://pkg.go.dev/io/fs#FS) interface
[introduced in Go 1.16](https://go.dev/doc/go1.16#fs). This means that currently all implementations are
read-only, however this may change in the future (see
[golang/go#45757](https://github.com/golang/go/issues/45757) for progress).

Most implementations implement the [`fs.ReadDirFS`](https://pkg.go.dev/io/fs#ReadDirFS)
interface, though the `httpfs` filesystem does not.

Some extensions are available to help add specific functionality to certain
filesystems:
- `WithContextFS` - injects a context into a filesystem, for propagating
	cancellation in filesystems that support it.
- `WithHeaderFS` - sets the `http.Header` for all HTTP requests used by the
	filesystem. This can be useful for authentication, or for requesting
	specific content types.
- `WithHTTPClientFS` - sets the `*http.Client` for all HTTP requests to be made
	with.

Many of the filesystem packages also have their own extensions.

This module also provides `ContentType`, an extension to the
[`fs.FileInfo`](https://pkg.go.dev/io/fs#FileInfo) type to help identify an
appropriate MIME content type for a given file. For filesystems that support it,
the HTTP `Content-Type` header is used for this. Otherwise, the type is guessed
from the file extension.

## History & Project Status

This module is _in active development_, and the API is still subject to breaking
changes.

The filesystem packages should operate correctly, based on the tests, but there
may be edge cases that are not covered. Please open an issue if you find one!

Most of these filesystems are based on code from [gomplate](https://github.com/hairyhenderson/gomplate),
which supports all of these as datasources. This module is intended to 
eventually be used within gomplate.

## Supported Filesystems

Here's the list of filesystems & URL schemes supported by this module:

| Package    | Scheme(s) | Description |
|------------|-----------|-------------|
| [awsimdsfs]| `aws+imds` | [AWS IMDS][] |
| [awssmfs]  | `aws+sm` | [AWS Secrets Manager][] |
| [awssmpfs] | `aws+smp` | [AWS Systems Manager Parameter Store][AWS SMP] |
| [blobfs]   | `azblob` | [Azure Blob Storage][] |
| [blobfs]   | `gs` | [Google Cloud Storage][] |
| [blobfs]   | `s3` | [Amazon S3][] |
| [consulfs] | `consul`, `consul+http`, `consul+https` | [HashiCorp Consul][] |
| [filefs]   | `file` | local filesystem |
| [gcpmetafs] | `gcp+meta` | [GCP Metadata][] |
| [gcpsmfs]   | `gcp+sm`   | [Google Secret Manager] |
| [gitfs]    | `git`, `git+file`, `git+http`, `git+https`, `git+ssh` | local/remote git repository |
| [httpfs]   | `http`, `https` | HTTP server |
| [tracefs]  | n/a | a filesystem that instruments other filesystems for tracing with [OpenTelemetry][] |
| [vaultfs]  | `vault`, `vault+http`, `vault+https` | [HashiCorp Vault][] |

See the individual package documentation for more details.

## Installation

Use `go get` to install the latest version of `go-fsimpl`:

```console
$ go get -u github.com/hairyhenderson/go-fsimpl
```

## Usage

If you know that you want an HTTP filesystem, for example:

```go
import (
	"net/url"

	"github.com/hairyhenderson/go-fsimpl/httpfs"
)

func main() {
	base, _ := url.Parse("https://example.com")
	fsys, _ := httpfs.New(base)

	f, _ := fsys.Open("hello.txt")
	defer f.Close()

	// now do what you like with the file...
}
```

If you're not sure what filesystem type you need (for example, if you're dealing
with a user-provided URL), you can use a filesystem mux:

```go
import (
	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/blobfs"
	"github.com/hairyhenderson/go-fsimpl/filefs"
	"github.com/hairyhenderson/go-fsimpl/gitfs"
	"github.com/hairyhenderson/go-fsimpl/httpfs"
)

func main() {
	mux := fsimpl.NewMux()
	mux.Add(filefs.FS)
	mux.Add(httpfs.FS)
	mux.Add(blobfs.FS)
	mux.Add(gitfs.FS)

	// for example, a URL that points to a subdirectory at a specific tag in a
	// given git repo, hosted on GitHub and authenticated with SSH...
	fsys, err := mux.Lookup("git+ssh://git@github.com/foo/bar.git//baz#refs/tags/v1.0.0")
	if err != nil {
		log.Fatal(err)
	}

	f, _ := fsys.Open("hello.txt")
	defer f.Close()

	// now do what you like with the file...
}
```

## Developing

You will require `git` including `git daemon` and `consul` executables on your path for running the tests.

## License

[The MIT License](http://opensource.org/licenses/MIT)

Copyright (c) 2021-2024 Dave Henderson

[godocs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl
[godoc-image]: https://pkg.go.dev/badge/github.com/hairyhenderson/go-fsimpl
[gh-actions-image]: https://github.com/hairyhenderson/go-fsimpl/actions/workflows/build.yml/badge.svg
[gh-actions-url]: https://github.com/hairyhenderson/go-fsimpl/actions?workflow=Build&branch=main

[AWS IMDS]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html
[AWS SMP]: https://aws.amazon.com/systems-manager/features#Parameter_Store
[AWS Secrets Manager]: https://aws.amazon.com/secrets-manager
[HashiCorp Consul]: https://consul.io
[HashiCorp Vault]: https://vaultproject.io
[Amazon S3]: https://aws.amazon.com/s3/
[Google Cloud Storage]: https://cloud.google.com/storage/
[Google Secret Manager]: https://cloud.google.com/security/products/secret-manager
[Azure Blob Storage]: https://azure.microsoft.com/en-us/services/storage/blobs/
[OpenTelemetry]: https://opentelemetry.io
[GCP Metadata]: https://cloud.google.com/compute/docs/metadata/overview

[awsimdsfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/awsimdsfs
[awssmfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/awssmfs
[awssmpfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/awssmpfs
[blobfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/blobfs
[consulfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/consulfs
[filefs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/filefs
[gitfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/gitfs
[blobfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/blobfs
[httpfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/httpfs
[blobfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/blobfs
[tracefs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/tracefs
[vaultfs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/vaultfs
[gcpmetafs]: https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl/gcpmetafs
