// Package gcpmetafs provides an interface to [GCP VM Metadata Service],
// which is a service available to Google Cloud Platform VM instances that provides
// information about the instance, project, and custom metadata.
//
// This filesystem's behaviour complies with fstest.TestFS.
//
// # Usage
//
// To use this filesystem, call [New] with a base URL. All reads from the
// filesystem are relative to this base URL. Only the scheme "gcp+meta" is
// supported.
//
// The GCP VM Metadata Service provides two main categories of data:
// - /instance: instance metadata (specific to the VM)
// - /project: project metadata (shared across all VMs in the project)
//
// Within these paths, data is organized into categories. See [VM metadata documentation]
// for more information.
//
// To scope the filesystem to a specific path, use that path on the URL. For
// example, for a filesystem that can only read instance metadata from the GCP VM
// Metadata Service, you would use a URL like:
//
//	gcp+meta:///instance/
//
// # Configuration
//
// By default, the filesystem uses the standard GCP VM Metadata Service endpoint
// at http://metadata.google.internal/computeMetadata/v1/ and includes the required
// "Metadata-Flavor: Google" header with all requests.
//
// If you require more customized configuration, you can override the default
// HTTP client with the [fsimpl.WithHTTPClientFS] function.
//
// [GCP VM Metadata Service]: https://cloud.google.com/compute/docs/metadata/overview
// [VM metadata documentation]: https://cloud.google.com/compute/docs/metadata/default-metadata-values
package gcpmetafs
