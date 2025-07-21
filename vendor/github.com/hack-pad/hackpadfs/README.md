# hackpadfs  [![Go Reference](https://pkg.go.dev/badge/github.com/hack-pad/hackpadfs.svg)](https://pkg.go.dev/github.com/hack-pad/hackpadfs) [![CI](https://github.com/hack-pad/hackpadfs/actions/workflows/ci.yml/badge.svg)](https://github.com/hack-pad/hackpadfs/actions/workflows/ci.yml) [![Coverage Status](https://coveralls.io/repos/github/hack-pad/hackpadfs/badge.svg?branch=main)](https://coveralls.io/github/hack-pad/hackpadfs?branch=main)

File systems, interfaces, and helpers for Go.

Want to get started? Check out the [guides](#getting-started) below.

## File systems

`hackpadfs` includes several implemented file systems, ready for use in a wide variety of applications:

* [`os.FS`](https://pkg.go.dev/github.com/hack-pad/hackpadfs/os) - The familiar `os` package. Implements all of the familiar behavior from the standard library using new interface design.
* [`mem.FS`](https://pkg.go.dev/github.com/hack-pad/hackpadfs/mem) - In-memory file system.
* [`indexeddb.FS`](https://pkg.go.dev/github.com/hack-pad/hackpadfs/indexeddb) - WebAssembly compatible file system, uses [IndexedDB](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API) under the hood.
* [`tar.ReaderFS`](https://pkg.go.dev/github.com/hack-pad/hackpadfs/tar) - A streaming tar FS for memory and time-constrained programs.
* [`mount.FS`](https://pkg.go.dev/github.com/hack-pad/hackpadfs/mount) - Composable file system. Capable of mounting file systems on top of each other.
* [`keyvalue.FS`](https://pkg.go.dev/github.com/hack-pad/hackpadfs/keyvalue) - Generic key-value file system. Excellent for quickly writing your own file system. `mem.FS` and `indexeddb.FS` are built upon it.

Looking for custom file system inspiration? Examples include:

* [`s3.FS`](https://pkg.go.dev/github.com/hack-pad/hackpadfs/examples/s3)

Each of these file systems runs through the rigorous [`hackpadfs/fstest` suite](fstest/fstest.go) to ensure both correctness and compliance with the standard library's `os` package behavior. If you're implementing your own FS, we recommend using `fstest` in your own tests as well.

### Interfaces

Based upon the groundwork laid in Go 1.16's [`io/fs` package](https://golang.org/doc/go1.16#fs), `hackpadfs` defines many essential file system interfaces.

Here's a few of the interfaces defined by `hackpadfs`:

```go
type FS interface {
    Open(name string) (File, error)
}

type CreateFS interface {
    FS
    Create(name string) (File, error)
}

type MkdirFS interface {
    FS
    Mkdir(name string, perm FileMode) error
}

type StatFS interface {
    FS
    Stat(name string) (FileInfo, error)
}
```

See the [reference docs](https://pkg.go.dev/github.com/hack-pad/hackpadfs) for full documentation.

Using these interfaces, you can create and compose your own file systems. The interfaces are small, enabling custom file systems to implement only the required pieces.

## Getting started

There's many ways to use `hackpadfs`. Jump to one of these guides:

* [Quick start](#quick-start)
* [Working with interfaces](#working-with-interfaces)


### Quick start

If you've used the standard library's `os` package, you probably understand most of how `os.FS` works!

In this example, we create a new `os.FS` and print the contents of `/tmp/hello.txt`.

```go
import (
    "fmt"

    "github.com/hack-pad/hackpadfs"
    "github.com/hack-pad/hackpadfs/os"
)

filePath := "tmp/hello.txt"
fs, _ := os.NewFS()
file, _ := fs.Open(filePath)
defer file.Close()

buffer := make([]byte, 1024)
n, _ := file.Read(buffer)
fmt.Println("Contents of hello.txt:", string(buffer[:n]))
```

#### Relative file paths

Relative paths are not allowed in Go's `io/fs` specification, so we must use absolute paths (without the first `/`).
To simulate relative paths, use the `SubFS` interface to create a new "rooted" FS like this:

```go
import (
    goOS "os"
    "github.com/hack-pad/hackpadfs/os"
)

fs := os.NewFS()
workingDirectory, _ := goOS.Getwd()                    // Get current working directory
workingDirectory, _ = fs.FromOSPath(workingDirectory)  // Convert to an FS path
workingDirFS, _ := fs.Sub(workingDirectory)            // Run all file system operations rooted at the current working directory
```

#### Path separators (slashes)

Following the [`io/fs` specification](https://pkg.go.dev/io/fs@go1.17.1#ValidPath):
> Path names passed to open are UTF-8-encoded, unrooted, slash-separated sequences of path elements, like “x/y/z”. Path names must not contain an element that is “.” or “..” or the empty string, except for the special case that the root directory is named “.”. Paths must not start or end with a slash: “/x” and “x/” are invalid.
>
> Note that paths are slash-separated on all systems, even Windows. Paths containing other characters such as backslash and colon are accepted as valid, but those characters must never be interpreted by an FS implementation as path element separators.

In `hackpadfs`, this means:
* All path separators are "forward slashes" or `/`. Even on Windows, slashes are converted under the hood.
* A path starting or ending with `/` is invalid
* To reference the root file path, use `.`
* All paths are unrooted (not [relative paths](#relative-file-paths))
* Paths are not necessarily cleaned when containing relative-path elements (e.g. `mypath/.././myotherpath`). Some FS implementations resolve these, but it is not guaranteed. File systems should reject these paths via `io/fs.ValidPath()`.

### Working with interfaces

It's a good idea to use interfaces -- file systems should be no different. Swappable file systems enable powerful combinations.

However, directly handling interface values can be difficult to deal with. Luckily, we have several helpers available.

In the below example, we use `hackpadfs`'s package-level functions to do interface checks and call the appropriate methods for us:

```go
import (
    "github.com/hack-pad/hackpadfs"
    "github.com/hack-pad/hackpadfs/mem"
)

func main() {
    fs, _ := mem.NewFS()
    helloWorld(fs)
}

// helloWorld uses a generic hackpadfs.FS to create a file containing "world".
// Returns an error if 'fs' does not support creating or writing to files.
func helloWorld(fs hackpadfs.FS) error {
    // hackpadfs.Create(...) checks to see if 'fs' implements Create, with a few fallback interfaces as well.
    file, err := hackpadfs.Create(fs, "hello.txt")
    if err != nil {
	return err
    }
    // Same here for hackpadfs.WriteFile(...). If the file doesn't support writing, a "not implemented" error is returned.
    _, err = hackpadfs.WriteFile(file, []byte("world"))
    return err
}
```

Notice the package-level function calls to `hackpadfs.Create(...)` and `hackpadfs.WriteFile(...)`.
Since the interface we're using doesn't know about those methods, we use these helpers to detect support and run those operations in one call.

Now whenever we need to reuse `helloWorld()` with a completely different file system, it's ready to go!
