// Package httpfs provides a read-only filesystem that reads from an HTTP
// server.
//
// This filesystem only supports single-file operations since there is no
// facility in HTTP for directory listings. As such, this filesystem's behaviour
// does not comply with [testing/fstest.TestFS].
//
// # Usage
//
// To use this filesystem, call [New] with a base URL. All reads from the
// filesystem are relative to this base URL. Only the schemes "http" and "https"
// are supported.
//
// To scope the filesystem to a specific path, use that path on the URL. For
// example, for a filesystem that can only read sub-paths of
// "https://example.com/foo/bar", you could use a URL like:
//
//	https://example.com/foo/bar/
//
// Note: when scoping URLs to specific paths, the URL should end in "/".
//
// # Setting the Context
//
// This filesystem supports setting a context with the [fsimpl.WithContextFS]
// extension. This can be useful in order to control timeouts, or for
// distributed trace propagation.
//
// For example, to inject a context with a timeout of 3 seconds:
//
//	fsys, _ := httpfs.New(url)
//
//	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
//	defer cancel()
//
//	fsys = fsimpl.WithContextFS(fsys, ctx)
//
// # Adding custom HTTP headers
//
// This filesystem supports adding custom HTTP headers with the [fsimpl.WithHeaderFS]
// extension. This can be useful for setting authentication headers, or for
// setting a user-agent.
//
// For example, to set the user-agent to "my-app":
//
//	fsys, _ := httpfs.New(url)
//
//	fsys = fsimpl.WithHeaderFS(fsys, http.Header{
//		"User-Agent": []string{"my-app"},
//	})
//
// # Using your own HTTP client
//
// By default, this filesystem uses Go's [net/http.DefaultClient], but sometimes
// you may want to use a different HTTP client. The [fsimpl.WithHTTPClientFS]
// extension allows you to do this.
//
// For example, to use a client with a custom transport:
//
//	fsys, _ := httpfs.New(url)
//
//	client := &http.Client{Transport: myCustomTransport}
//
//	fsys = fsimpl.WithHTTPClientFS(fsys, client)
package httpfs
