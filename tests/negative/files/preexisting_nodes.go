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
	register.Register(register.NegativeTest, ForceFileCreation())
	register.Register(register.NegativeTest, ForceDirCreation())
	register.Register(register.NegativeTest, ForceLinkCreation())
	register.Register(register.NegativeTest, ForceHardLinkCreation())
	register.Register(register.NegativeTest, ForceFileCreationOverNonemptyDir())
	register.Register(register.NegativeTest, ForceLinkCreationOverNonemptyDir())
}

func ForceFileCreation() types.Test {
	name := "files.create.overwrite.false"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents"
	      },
		  "overwrite": false
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceDirCreation() types.Test {
	name := "directories.create.overwrite.false"
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
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: "hello, world",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceLinkCreation() types.Test {
	name := "links.sym.create.overwrite.false"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/target",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents"
	      }
	    }],
	    "links": [{
	      "path": "/foo/bar",
	      "target": "/foo/target"
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: "asdf\nfdsa",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceHardLinkCreation() types.Test {
	name := "links.hard.create.overwrite.false"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/target",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents"
	      }
	    }],
	    "links": [{
	      "path": "/foo/bar",
	      "target": "/foo/target",
		  "hard": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: "asdf\nfdsa",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceFileCreationOverNonemptyDir() types.Test {
	name := "files.creates.overwrite.false.overdir"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents"
	      },
		  "overwrite": false
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo/bar",
				Name:      "baz",
			},
			Contents: "asdf\nfdsa",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ForceLinkCreationOverNonemptyDir() types.Test {
	name := "links.sym.create.overwrite.false.overdir"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/target",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents"
	      }
	    }],
	    "links": [{
	      "path": "/foo/bar",
	      "target": "/foo/target"
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo/bar",
				Name:      "baz",
			},
			Contents: "asdf\nfdsa",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
