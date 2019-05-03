// Copyright 2018 CoreOS, Inc.
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
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.NegativeTest, WriteOverBrokenSymlink())
	register.Register(register.NegativeTest, SymlinkResolutionCausesConflicts())
	register.Register(register.NegativeTest, FailMatchHardLinkOnRoot())
	register.Register(register.NegativeTest, FailMatchSymlinkOnRoot())
}

func WriteOverBrokenSymlink() types.Test {
	name := "links.sym.overwritelink"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/etc/file",
	      "overwrite": false,
	      "mode": 420
	    }]
	  }
	}`
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "file",
				Directory: "etc",
			},
			Target: "/usr/rofile",
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

func SymlinkResolutionCausesConflicts() types.Test {
	name := "links.sym.resovledconflicts"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar/baz"
	    },
	    {
	      "path": "/bar/baz"
	    }]
	  }
	}`
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "foo",
				Directory: "/",
			},
			Target: "/",
			Hard:   false,
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

func FailMatchHardLinkOnRoot() types.Test {
	name := "links.hard.badmatch"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "links": [{
	      "path": "/existing",
	      "target": "/target",
	      "hard": true
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "WrongTarget",
			},
		},
		{
			Node: types.Node{
				Directory: "/",
				Name:      "target",
			},
		},
	})
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "existing",
			},
			Target: "/WrongTarget",
			Hard:   true,
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

func FailMatchSymlinkOnRoot() types.Test {
	name := "links.sym.badmatch"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "links": [{
	      "path": "/existing",
	      "target": "/target"
	    }]
	  }
	}`
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "existing",
			},
			Target: "/WrongTarget",
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
