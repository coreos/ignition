package vaultfs

import (
	"sync/atomic"

	"github.com/hashicorp/vault/api"
)

func newRefCountedClient(c *api.Client) *refCountedClient {
	return &refCountedClient{Client: c}
}

// refCountedClient is a wrapper for the Vault *api.Client that tracks
// references so that the token can be shared across multiple open files and
// directories within the filesystem.
//
// This allows the vaultFile.Close method to avoid revoking the token when
// other files are still open.
type refCountedClient struct {
	*api.Client
	refs atomic.Uint64
}

// var _ VaultClient = (*refCountedClient)(nil)

func (c *refCountedClient) AddRef() {
	c.refs.Add(1)
}

func (c *refCountedClient) RemoveRef() {
	if c.Refs() == 0 {
		// if this panics, it's a programming error - given this is called on
		// close, and close should only succeed once, this should never happen.
		panic("refCountedClient.RemoveRef called when refs already 0. " +
			"This indicates a programming error in vaultfs and should be reported!")
	}

	// decrements the ref count by overflowing (see atomic.AddUint64 doc)
	c.refs.Add(^uint64(0))
}

func (c *refCountedClient) Refs() uint64 {
	return c.refs.Load()
}
