// Package autofs provides the ability to look up all filesystems supported by
// this module. Using this package will compile a great many dependencies into
// the resulting binary, so unless you need to support all supported filesystems,
// use fsimpl.NewMux instead.
package autofs

import (
	"io/fs"
	"net/url"
	"sync"

	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/awsimdsfs"
	"github.com/hairyhenderson/go-fsimpl/awssmfs"
	"github.com/hairyhenderson/go-fsimpl/awssmpfs"
	"github.com/hairyhenderson/go-fsimpl/blobfs"
	"github.com/hairyhenderson/go-fsimpl/consulfs"
	"github.com/hairyhenderson/go-fsimpl/filefs"
	"github.com/hairyhenderson/go-fsimpl/gcpmetafs"
	"github.com/hairyhenderson/go-fsimpl/gcpsmfs"
	"github.com/hairyhenderson/go-fsimpl/gitfs"
	"github.com/hairyhenderson/go-fsimpl/httpfs"
	"github.com/hairyhenderson/go-fsimpl/vaultfs"
)

// Lookup returns an appropriate filesystem for the given URL.
// If a filesystem can't be found for the provided URL's scheme, an error will
// be returned.
func Lookup(u string) (fs.FS, error) {
	return initMux().Lookup(u)
}

// FS is used to register this filesystem with an fsimpl.FSMux
//
//nolint:gochecknoglobals
var FS = &autoFS{}

type autoFS struct{}

var _ fsimpl.FSProvider = (*autoFS)(nil)

func (c *autoFS) Schemes() []string {
	return initMux().Schemes()
}

func (c *autoFS) New(u *url.URL) (fs.FS, error) {
	return initMux().New(u)
}

func initMux() fsimpl.FSMux {
	return sync.OnceValue(func() fsimpl.FSMux {
		mux := fsimpl.NewMux()
		mux.Add(awsimdsfs.FS)
		mux.Add(awssmfs.FS)
		mux.Add(awssmpfs.FS)
		mux.Add(blobfs.FS)
		mux.Add(consulfs.FS)
		mux.Add(filefs.FS)
		mux.Add(gcpmetafs.FS)
		mux.Add(gcpsmfs.FS)
		mux.Add(gitfs.FS)
		mux.Add(httpfs.FS)
		mux.Add(vaultfs.FS)

		return mux
	})()
}
