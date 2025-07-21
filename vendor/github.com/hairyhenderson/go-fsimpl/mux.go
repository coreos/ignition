package fsimpl

import (
	"fmt"
	"io/fs"
	"net/url"
	"sort"
	"strings"
)

// FSMux allows you to dynamically look up a registered filesystem for a given
// URL. All filesystems provided in this module can be registered, and
// additional filesystems can be registered given an implementation of
// FSProvider.
// FSMux is itself an FSProvider, which provides the superset of all registered
// filesystems.
type FSMux map[string]func(*url.URL) (fs.FS, error)

var _ FSProvider = (FSMux)(nil)

// NewMux returns an FSMux ready for use.
func NewMux() FSMux {
	return FSMux(map[string]func(*url.URL) (fs.FS, error){})
}

// Add registers the given filesystem provider for its supported URL schemes. If
// any of its schemes are already registered, they will be overridden.
func (m FSMux) Add(fs FSProvider) {
	for _, scheme := range fs.Schemes() {
		m[scheme] = fs.New
	}
}

// Lookup returns an appropriate filesystem for the given URL. Use Add to
// register providers.
func (m FSMux) Lookup(u string) (fs.FS, error) {
	base, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	return m.New(base)
}

// Schemes - implements FSProvider
func (m FSMux) Schemes() []string {
	schemes := make([]string, 0, len(m))
	for scheme := range m {
		schemes = append(schemes, scheme)
	}

	sort.Strings(schemes)

	return schemes
}

// New - implements FSProvider
func (m FSMux) New(u *url.URL) (fs.FS, error) {
	f, ok := m[u.Scheme]
	if !ok {
		return nil, fmt.Errorf("no filesystem registered for scheme %q", u.Scheme)
	}

	return f(u)
}

// FSProvider provides a filesystem for a set of defined schemes
type FSProvider interface {
	// Schemes returns the valid URL schemes for this filesystem
	Schemes() []string

	// New returns a filesystem from the given URL
	New(u *url.URL) (fs.FS, error)
}

// FSProviderFunc -
func FSProviderFunc(f func(*url.URL) (fs.FS, error), schemes ...string) FSProvider {
	return fsp{f, schemes}
}

type fsp struct {
	newFunc func(*url.URL) (fs.FS, error)
	schemes []string
}

func (p fsp) Schemes() []string {
	return p.schemes
}

func (p fsp) New(u *url.URL) (fs.FS, error) {
	return p.newFunc(u)
}

// WrappedFSProvider is an FSProvider that returns the given fs.FS.
// When given a URL with a non-root path (i.e. not '/'), fs.Sub will be used to
// return a filesystem appropriate for the URL.
func WrappedFSProvider(fsys fs.FS, schemes ...string) FSProvider {
	return fsp{
		newFunc: func(u *url.URL) (fs.FS, error) {
			dir := u.Path
			if dir == "/" {
				dir = "."
			} else if strings.HasPrefix(dir, "/") {
				dir = strings.TrimLeft(dir, "/")
			}

			return fs.Sub(fsys, dir)
		},
		schemes: schemes,
	}
}
