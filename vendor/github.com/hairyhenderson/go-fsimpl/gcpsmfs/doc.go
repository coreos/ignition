// Package gcpsmfs provides an interface to Google Cloud Secret Manager which allows you
// to interact with the Secret Manager API as a standard filesystem.
//
// This filesystem's behaviour complies with fstest.TestFS.
//
// # Usage
//
// To use this filesystem, call New with a base URL. All reads from the
// filesystem are relative to this base URL. Only the scheme "gcp+sm" is
// supported.
//
//	gcp+sm:///projects/my-project-id
//
// Reads from the filesystem are mapped to the latest version of the secret with the
// corresponding name.
//
//	fs.ReadFile(fsys, "my-secret") // reads projects/my-project-id/secrets/my-secret/versions/latest
//
// # Configuration
//
// The GCP Secret Manager client is configured using the default credential
// chain.
//
// If you require more customized configuration, you can override the default
// client with the WithSMClientFS function.
package gcpsmfs
