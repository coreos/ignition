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
	"os"

	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.PositiveTest, CreateDirectoryOnRoot())
	register.Register(register.PositiveTest, ForceDirCreation())
	register.Register(register.PositiveTest, ForceDirCreationOverNonemptyDir())
	register.Register(register.PositiveTest, CheckOrdering())
	register.Register(register.PositiveTest, ApplyDefaultDirectoryPermissions())
}

func CreateDirectoryOnRoot() types.Test {
	name := "Create a Directory on the Root Filesystem"
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
	configMinVersion := "3.0.0-experimental"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceDirCreation() types.Test {
	name := "Force Directory Creation"
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
	configMinVersion := "3.0.0-experimental"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceDirCreationOverNonemptyDir() types.Test {
	name := "Force Directory Creation Over Nonempty Directory"
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
	configMinVersion := "3.0.0-experimental"
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
	name := "TODO"
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
			Mode: 0777 | int(os.ModeDir),
		},
		{
			Node: types.Node{
				Directory: "baz",
				Name:      "quux",
			},
			Mode: 0755 | int(os.ModeDir),
		},
	})
	configMinVersion := "3.0.0-experimental"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
func ApplyDefaultDirectoryPermissions() types.Test {
	name := "Apply Default Directory Permissions"
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
			Mode: 0755 | int(os.ModeDir),
		},
	})
	configMinVersion := "3.0.0-experimental"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
