// Package awssmpfs provides an interface to the AWS Systems Manager (SSM)
// Parameter Store which allows you to interact with the SSM Parameter Store API
// as a standard filesystem.
//
// This filesystem's behaviour complies with [testing/fstest.TestFS].
//
// # Usage
//
// To use this filesystem, call [New] with a base URL. All reads from the
// filesystem are relative to this base URL. Only the scheme "aws+smp" is
// supported. The URL may not be opaque (with no leading "/" in the path), as
// AWS SSM Parameter Store does not support this ([1]).
//
// To scope the filesystem to a specific path, use that path on the URL. For
// example, for a filesystem that can only read parameters with names starting
// with "/prod/foo/", you would use a URL like:
//
//	aws+smp:///prod/foo/
//
// For use with alternate endpoints (e.g. [localstack]), you can set a host on
// the URL. For example, for a filesystem that reads parameters from a
// localstack instance running on localhost, you could use a URL like:
//
//	aws+smp://localhost:4566
//
// To retrieve a specific version of a parameter, suffix the path with a colon
// and the version number. For example, to retrieve version 2 of the parameter
// "/prod/keys/foo" you would use "foo:2" in the path:
//
//	fsys, err := awssmpfs.New("aws+smp:///prod")
//	f, err := fsys.Open("keys/foo:2")
//	// etc...
//
// # Authentication
//
// To authenticate, the default [credential chain] is used. This means that
// credentials can be provided with environment variables, shared credentials
// files, or EC2 instance metadata. See the [credential chain] documentation for
// details.
//
// # Extensions
//
// The filesystem may be configured with a few standard and awssmpfs-specific
// extensions. See the documentation for each extension for more details:
//
//   - [fsimpl.WithContextFS]
//   - [fsimpl.WithHTTPClientFS]
//   - [WithClientFS]
//
// [1]: https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-paramstore-su-create.html#sysman-parameter-name-constraints
// [credential chain]: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
// [localstack]: https://localstack.cloud
package awssmpfs
