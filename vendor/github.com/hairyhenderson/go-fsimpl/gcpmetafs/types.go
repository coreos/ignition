package gcpmetafs

import (
	"context"
)

// MetadataClient is an interface that wraps the basic functionality needed
// to interact with the GCP VM Metadata Service. It matches the signature of
// metadata.Client.GetWithContext for direct compatibility.
type MetadataClient interface {
	// GetWithContext retrieves metadata for the specified path
	GetWithContext(ctx context.Context, path string) (string, error)
}
