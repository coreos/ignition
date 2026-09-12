package gcpsmfs

import (
	"context"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
)

// SecretIterator matches the Next method of *secretmanager.SecretIterator
type SecretIterator interface {
	Next() (*secretmanagerpb.Secret, error)
}

type SecretManagerClient interface {
	AccessSecretVersion(
		ctx context.Context,
		req *secretmanagerpb.AccessSecretVersionRequest,
		opts ...gax.CallOption,
	) (*secretmanagerpb.AccessSecretVersionResponse, error)
	GetSecretVersion(
		ctx context.Context,
		req *secretmanagerpb.GetSecretVersionRequest,
		opts ...gax.CallOption,
	) (*secretmanagerpb.SecretVersion, error)
	ListSecrets(ctx context.Context, req *secretmanagerpb.ListSecretsRequest, opts ...gax.CallOption) SecretIterator
}

// clientAdapter adapts the real GCP Secret Manager client to the SecretManagerClient interface, since we need to accommodate the iterator.
type clientAdapter struct {
	*secretmanager.Client
}

func (c *clientAdapter) ListSecrets(ctx context.Context, req *secretmanagerpb.ListSecretsRequest, opts ...gax.CallOption) SecretIterator {
	return c.Client.ListSecrets(ctx, req, opts...)
}
