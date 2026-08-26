// Package awsimdsfs provides an interface to [AWS IMDS] (Instance Metadata
// Service), which is a service available to EC2 instances that provides
// information about the instance, and user data.
//
// This filesystem's behaviour complies with fstest.TestFS.
//
// # Usage
//
// To use this filesystem, call [New] with a base URL. All reads from the
// filesystem are relative to this base URL. Only the scheme "aws+imds" is
// supported.
//
// IMDS supports three kinds of data: [instance metadata], [user data], and
// [dynamic data]. This filesystem supports all three, available at the
// following paths:
// - /meta-data: instance metadata
// - /user-data: user data
// - /dynamic: dynamic data
//
// Within the "/meta-data" and "/dynamic" paths, data is organized into
// categories. See [Instance metadata categories] for more information.
//
// To scope the filesystem to a specific path, use that path on the URL. For
// example, for a filesystem that can only read dynamic data from the IMDS,
// you would use a URL like:
//
//	aws+imds:///dynamic/
//
// # Configuration
//
// The AWS Secrets Manager client is configured using the default credential
// chain (see https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
// for more information).
//
// If you require more customized configuration, you can override the default
// client with the [WithIMDSClientFS] function.
//
// [AWS IMDS]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html
// [instance metadata]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
// [user data]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-add-user-data.html
// [dynamic data]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-dynamic-data-retrieval.html
// [Instance metadata categories]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-categories.html
package awsimdsfs
