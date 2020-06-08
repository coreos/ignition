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
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, CreateHardLinkOnRoot())
	register.Register(register.PositiveTest, MatchHardLinkOnRoot())
	register.Register(register.PositiveTest, CreateSymlinkOnRoot())
	register.Register(register.PositiveTest, MatchSymlinkOnRoot())
	register.Register(register.PositiveTest, ForceLinkCreation())
	register.Register(register.PositiveTest, ForceHardLinkCreation())
	register.Register(register.PositiveTest, CreateDeepHardLinkToFile())
	register.Register(register.PositiveTest, WriteOverSymlink())
	register.Register(register.PositiveTest, WriteOverBrokenSymlink())
	register.Register(register.PositiveTest, CreateHardLinkToSymlink())
}

func CreateHardLinkOnRoot() types.Test {
	name := "links.hard.create"
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
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "target",
			},
			Contents: "asdf\nfdsa",
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Target: "/foo/target",
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

func MatchHardLinkOnRoot() types.Test {
	name := "links.hard.match"
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
			Target: "/target",
			Hard:   true,
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "target",
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "existing",
			},
			Target: "/target",
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

func CreateSymlinkOnRoot() types.Test {
	name := "links.sym.create"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "links": [{
	      "path": "/foo/bar",
	      "target": "/foo/target",
	      "hard": false
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Target: "/foo/target",
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

func MatchSymlinkOnRoot() types.Test {
	name := "links.sym.match"
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
	in[0].Partitions.AddFiles("ROOT", []types.File{
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
			Target: "/target",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "target",
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "existing",
			},
			Target: "/target",
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

func ForceLinkCreation() types.Test {
	name := "links.sym.create.force"
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
			Contents: "asdf\nfdsa",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "target",
			},
			Contents: "asdf\nfdsa",
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Target: "/foo/target",
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

func ForceHardLinkCreation() types.Test {
	name := "links.hard.create.force"
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
	      "hard": true,
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
			Contents: "asdf\nfdsa",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "target",
			},
			Contents: "asdf\nfdsa",
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Target: "/foo/target",
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

// CreateDeepHardLinkToFile checks if Ignition can create a hard
// link to a file that's deeper than the hard link. For more
// information: https://github.com/coreos/ignition/issues/800
func CreateDeepHardLinkToFile() types.Test {
	name := "links.hard.deep.create.file"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar/baz",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents"
	      }
	    }],
	    "links": [{
	      "path": "/foo/quux",
	      "target": "/foo/bar/baz",
	      "hard": true
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo/bar",
				Name:      "baz",
			},
			Contents: "asdf\nfdsa",
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "quux",
			},
			Target: "/foo/bar/baz",
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

func WriteOverSymlink() types.Test {
	name := "links.sym.writeover"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/etc/file",
	      "mode": 420,
	      "overwrite": true,
	      "contents": { "source": "" }
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
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "rofile",
				Directory: "usr",
			},
			Contents: "",
			Mode:     420,
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "rofile",
				Directory: "usr",
			},
			Contents: "",
			Mode:     420,
		},
		{
			Node: types.Node{
				Name:      "file",
				Directory: "etc",
			},
			Contents: "",
			Mode:     420,
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

func WriteOverBrokenSymlink() types.Test {
	name := "links.sym.writeover.broken"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/etc/file",
	      "mode": 420,
	      "overwrite": true,
	      "contents": { "source": "" }
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
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "file",
				Directory: "etc",
			},
			Contents: "",
			Mode:     420,
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

func CreateHardLinkToSymlink() types.Test {
	name := "links.hard.create.tosym"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "links": [{
	      "path": "/foo",
	      "target": "/bar",
	      "hard": true
	    }]
	  }
	}`
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "bar",
			},
			Target: "nonexistent",
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "/",
				Name:      "bar",
			},
			Target: "nonexistent",
		},
		{
			Node: types.Node{
				Directory: "/",
				Name:      "foo",
			},
			Target: "/bar",
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
