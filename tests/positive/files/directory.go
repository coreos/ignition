// Copyright 2017 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package files

import (
	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, CreateDirectoryOnRoot())
	register.Register(register.PositiveTest, ForceDirCreation())
	register.Register(register.PositiveTest, ForceDirCreationOverNonemptyDir())
	register.Register(register.PositiveTest, DirCreationOverNonemptyDir())
	register.Register(register.PositiveTest, CheckOrdering())
	register.Register(register.PositiveTest, ApplyDefaultDirectoryPermissions())
	register.Register(register.PositiveTest, CreateDirectoryFromTAR())
}

func CreateDirectoryOnRoot() types.Test {
	name := "directories.create"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [{
	      "path": "/foo/bar"
	    }]
	  }
	}`
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
		},
	})
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceDirCreation() types.Test {
	name := "directories.create.force"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [{
	      "path": "/foo/bar",
	      "overwrite": true
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: "hello, world",
		},
	})
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
		},
	})
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func DirCreationOverNonemptyDir() types.Test {
	name := "directories.match"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [{
	      "path": "/foo/bar",
	      "mode": 511
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo/bar",
				Name:      "baz",
			},
			Contents: "hello, world",
		},
	})
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo/bar",
				Name:      "baz",
			},
			Contents: "hello, world",
		},
	})
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Mode: 0777,
		},
	})
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceDirCreationOverNonemptyDir() types.Test {
	name := "directories.match.overwrite"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [{
	      "path": "/foo/bar",
	      "overwrite": true
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo/bar",
				Name:      "baz",
			},
			Contents: "hello, world",
		},
	})
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
		},
	})
	configMinVersion := "3.0.0"
	// TODO: add ability to ensure that foo/bar/baz doesn't exist here.

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func CheckOrdering() types.Test {
	name := "directories.ordering"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [{
	      "path": "/foo/bar/baz",
	      "mode": 511,
	      "overwrite": false
	    },
	    {
	      "path": "/baz/quux",
	      "mode": 493,
	      "overwrite": false
	    }]
	  }
	}`
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Target: "/",
		},
	})
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "baz",
			},
			Mode: 0777,
		},
		{
			Node: types.Node{
				Directory: "baz",
				Name:      "quux",
			},
			Mode: 0755,
		},
	})
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
func ApplyDefaultDirectoryPermissions() types.Test {
	name := "directories.defaultperms"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [{
	      "filesystem": "root",
	      "path": "/foo/bar"
	    }]
	  }
	}`
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Mode: 0755,
		},
	})
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func CreateDirectoryFromTAR() types.Test {
	name := "directories.tar"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()

	handleErr := func(err error) {
		if err != nil {
			panic(fmt.Sprintf("error generating test archive: %v", err))
		}
	}

	type filespec struct {
		tar.Header
		Contents string
	}

	usr, err := user.Lookup("bin")
	handleErr(err)
	binUID, err := strconv.Atoi(usr.Uid)
	handleErr(err)

	grp, err := user.Lookup("daemon")
	handleErr(err)
	daemonGID, err := strconv.Atoi(grp.Gid)
	handleErr(err)

	files := []filespec{
		{
			Header: tar.Header{
				Typeflag: tar.TypeReg,
				Name:     "reg",
			},
			Contents: "Hello, world\n",
		},
		{
			Header: tar.Header{
				Typeflag: tar.TypeDir,
				Name:     "dir",
			},
		},
		{
			Header: tar.Header{
				Typeflag: tar.TypeSymlink,
				Name:     "symlink",
				Linkname: "reg",
			},
		},
		{
			Header: tar.Header{
				Typeflag: tar.TypeSymlink,
				Name:     "dir/symlink",
				Linkname: "../reg",
			},
		},
		{
			Header: tar.Header{
				Typeflag: tar.TypeLink,
				Name:     "link",
				Linkname: "/reg",
			},
		},
		{
			// Verify that hard link resolution is done relative to target
			Header: tar.Header{
				Typeflag: tar.TypeLink,
				Name:     "dir/link",
				Linkname: "../reg",
			},
		},
		{
			Header: tar.Header{
				Typeflag: tar.TypeReg,
				Name:     "xattrs",
				Xattrs: map[string]string{
					"security.capability": "\x00\x00\x00\x02\xc2\x10\x2c\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", // some permitted-only capabilities
					"user.foo":            "bar",
				},
			},
		},
		{
			Header: tar.Header{
				Typeflag: tar.TypeReg,
				Name:     "uidgid",
				Uid:      1,
				Gid:      2,
			},
		},
		{
			Header: tar.Header{
				Typeflag: tar.TypeReg,
				Name:     "usergroup",
				Uname:    "bin",
				Gname:    "daemon",
				Uid:      binUID,
				Gid:      daemonGID,
			},
		},
		{
			Header: tar.Header{
				Typeflag: tar.TypeReg,
				Name:     "mode",
				Mode:     01234,
			},
		},
	}

	var archive strings.Builder

	enc := base64.NewEncoder(base64.StdEncoding, &archive)
	gz := gzip.NewWriter(enc)
	ar := tar.NewWriter(gz)

	for _, file := range files {
		node := types.Node{
			Name:      filepath.Base(file.Name),
			Directory: filepath.Join("dir", filepath.Dir(file.Name)),
			User:      file.Uid,
			Group:     file.Gid,
		}

		switch file.Typeflag {
		case tar.TypeReg:
			if file.Size == 0 {
				file.Size = int64(len(file.Contents))
			}

			// types.File.Mode is actually an os.FileMode more than a file mode
			// as defined by stat, so we have to convert between the upper bits
			// and the portable Go definitions of setuid/setgid/sticky.
			mode := os.FileMode(file.Mode & 0777)
			if file.Mode&unix.S_ISUID != 0 {
				mode |= os.ModeSetuid
			}
			if file.Mode&unix.S_ISGID != 0 {
				mode |= os.ModeSetgid
			}
			if file.Mode&unix.S_ISVTX != 0 {
				mode |= os.ModeSticky
			}

			out[0].Partitions.AddFiles("ROOT", []types.File{{
				Node:     node,
				Contents: file.Contents,
				Mode:     int(mode),
			}})
		case tar.TypeDir:
			out[0].Partitions.AddDirectories("ROOT", []types.Directory{{
				Node: node,
			}})
		case tar.TypeLink:
			target := file.Linkname
			if filepath.IsAbs(target) {
				target = filepath.Join("dir", target)
			} else {
				target = filepath.Join(node.Directory, file.Linkname)
			}
			out[0].Partitions.AddLinks("ROOT", []types.Link{{
				Node:   node,
				Target: target,
				Hard:   true,
			}})
		case tar.TypeSymlink:
			out[0].Partitions.AddLinks("ROOT", []types.Link{{
				Node:   node,
				Target: file.Linkname,
			}})
		}

		handleErr(ar.WriteHeader(&file.Header))
		if file.Typeflag == tar.TypeReg {
			_, err := io.WriteString(ar, file.Contents)
			handleErr(err)
		}
	}

	handleErr(ar.Close())
	handleErr(gz.Close())
	handleErr(enc.Close())

	config := fmt.Sprintf(`{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [
	      {
	        "path": "/dir",
	        "overwrite": true,
	        "contents": {
	          "archive": "tar",
	          "source": "data:;base64,%v",
	          "compression": "gzip"
	        }
	      }
	    ]
	  }
	}`, archive.String())

	configMinVersion := "3.4.0-experimental"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
