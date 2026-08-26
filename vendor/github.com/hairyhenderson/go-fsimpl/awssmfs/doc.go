// Package awssmfs provides an interface to AWS Secrets Manager which allows you
// to interact with the Secrets Manager API as a standard filesystem.
//
// This filesystem's behaviour complies with fstest.TestFS.
//
// # Usage
//
// To use this filesystem, call New with a base URL. All reads from the
// filesystem are relative to this base URL. Only the scheme "aws+sm" is
// supported. The URL may be an opaque URI (with no leading "/" in the path), in
// which case secrets with names starting with "/" are ignored. If the URL path
// does begin with "/", secrets with names not starting with "/" are instead
// ignored.
//
// To scope the filesystem to a specific path, use that path on the URL. For
// example, for a filesystem that can only read secrets with names starting with
// "/prod/foo/", you would use a URL like:
//
//	aws+sm:///prod/foo/
//
// And for a filesystem that can only read secrets with names starting with
// "prod/bar/", you would use the following opaque URI:
//
//	aws+sm:prod/bar/
//
// # Configuration
//
// The AWS Secrets Manager client is configured using the default credential
// chain (see https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
// for more information).
//
// If you require more customized configuration, you can override the default
// client with the WithSMClientFS function.
package awssmfs
