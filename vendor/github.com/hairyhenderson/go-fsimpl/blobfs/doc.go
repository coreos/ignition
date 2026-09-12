// Package blobfs provides an interface to blob stores such as Google Cloud
// Storage, Azure Blob Storage, or AWS S3, allowing you to interact with the
// store as a standard filesystem.
//
// This filesystem's behaviour complies with fstest.TestFS.
//
// # Usage
//
// To use this filesystem, call New with a base URL. All reads from the
// filesystem are relative to this base URL. The schemes "s3", "gs", and "azblob"
// are supported.
package blobfs
