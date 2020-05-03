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
	register.Register(register.PositiveTest, CreateFileOnRoot())
	register.Register(register.PositiveTest, UserGroupByID())
	register.Register(register.PositiveTest, ForceFileCreation())
	register.Register(register.PositiveTest, AppendToAFile())
	register.Register(register.PositiveTest, AppendToExistingFile())
	register.Register(register.PositiveTest, AppendToNonexistentFile())
	register.Register(register.PositiveTest, ApplyDefaultFilePermissions())
	register.Register(register.PositiveTest, CreateFileFromCompressedDataURL())
	// TODO: Investigate why ignition's C code hates our environment
	// register.Register(register.PositiveTest, UserGroupByName())
}

func CreateFileOnRoot() types.Test {
	name := "files.create"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": { "source": "data:,example%20file%0A" }
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
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

func UserGroupByID() types.Test {
	name := "files.owner.byid"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": { "source": "data:,example%20file%0A" },
		  "user": {"id": 500},
		  "group": {"id": 500}
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
			Contents: "example file\n",
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

func UserGroupByName() types.Test {
	name := "files.owner.byname"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": { "source": "data:,example%20file%0A" },
		  "user": {"name": "core"},
		  "group": {"name": "core"}
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
			Contents: "example file\n",
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

func ForceFileCreation() types.Test {
	name := "files.create.force"
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
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: "asdf\nfdsa",
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

func AppendToAFile() types.Test {
	name := "files.append.withcreate"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": { "source": "data:,example%20file%0A" },
	      "user": {"id": 500},
	      "group": {"id": 500},
	      "append": [{ "source": "data:,hello%20world%0A" }]
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
			Contents: "example file\nhello world\n",
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

func AppendToExistingFile() types.Test {
	name := "files.append.existing"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "append": [{ "source": "data:,hello%20world%0A" }]
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
			Contents: "example file\n",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
			Contents: "example file\nhello world\n",
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

func AppendToNonexistentFile() types.Test {
	name := "files.append.nonexistent"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "append": [{ "source": "data:,hello%20world%0A" }],
	      "group": {"id": 500}
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				Group:     500,
			},
			Contents: "hello world\n",
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

func ApplyDefaultFilePermissions() types.Test {
	name := "files.defaultperms"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
	      "contents": { "source": "data:,hello%20world%0A" }
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: "hello world\n",
			Mode:     0644,
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

func CreateFileFromCompressedDataURL() types.Test {
	name := "files.create.compressed"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
	        "compression": "gzip",
	        "source": "data:,%1F%8B%08%08%90e%AB%5E%02%03z%00K%ADH%CC-%C8IUH%CB%CCI%E5%02%00tp%A6%CB%0D%00%00%00"
	      }
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
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
