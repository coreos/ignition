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
	register.Register(register.PositiveTest, CreateFileOnRootFromBase64())
	register.Register(register.PositiveTest, UserGroupByID())
	register.Register(register.PositiveTest, ForceFileCreation())
	register.Register(register.PositiveTest, AppendToAFile())
	register.Register(register.PositiveTest, AppendToExistingFile())
	register.Register(register.PositiveTest, AppendToNonexistentFile())
	register.Register(register.PositiveTest, ApplyDefaultFilePermissions())
	register.Register(register.PositiveTest, ApplyCustomFilePermissions())
	register.Register(register.PositiveTest, ApplyCustomFilePermissionsBeforeBugfix())
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

func CreateFileOnRootFromBase64() types.Test {
	name := "files.create.base64"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": { "source": "data:;base64,ZXhhbXBsZSBmaWxlCg==" }
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

func ApplyCustomFilePermissions() types.Test {
	name := "files.customperms"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "passwd": {
	    "users": [
	      {
	        "name": "auser",
	        "uid": 1001,
	        "shouldExist": false
	      }
	    ],
	    "groups": [
	      {
	        "name": "auser",
	        "gid": 1001,
	        "shouldExist": false
	      }
	    ]
	  },
	  "storage": {
	    "files": [
	      {
	        "path": "/foo/setuidsetgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 3565
	      },
	      {
	        "path": "/foo/auser/setuidsetgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 3565,
	        "user": { "id": 1001 },
	        "group": { "id": 1001 }
	      },
	      {
	        "path": "/foo/root/setuidsetgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 3565,
	        "user": { "id": 0 },
	        "group": { "id": 0 }
	      },
	      {
	        "path": "/foo/setuid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 2541
	      },
	      {
	        "path": "/foo/auser/setuid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 2541,
	        "user": { "id": 1001 },
	        "group": { "id": 1001 }
	      },
	      {
	        "path": "/foo/root/setuid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 2541,
	        "user": { "id": 0 },
	        "group": { "id": 0 }
	      },
	      {
	        "path": "/foo/setgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 1517
	      },
	      {
	        "path": "/foo/auser/setgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 1517,
	        "user": { "id": 1001 },
	        "group": { "id": 1001 }
	      },
	      {
	        "path": "/foo/root/setgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 1517,
	        "user": { "id": 0 },
	        "group": { "id": 0 }
	      }
	    ]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "setuidsetgid",
			},
			Contents: "hello world\n",
			Mode:     06755,
		},
		{
			Node: types.Node{
				Directory: "foo/auser",
				Name:      "setuidsetgid",
				User:      1001,
				Group:     1001,
			},
			Contents: "hello world\n",
			Mode:     06755,
		},
		{
			Node: types.Node{
				Directory: "foo/root",
				Name:      "setuidsetgid",
				User:      0,
				Group:     0,
			},
			Contents: "hello world\n",
			Mode:     06755,
		},
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "setuid",
			},
			Contents: "hello world\n",
			Mode:     04755,
		},
		{
			Node: types.Node{
				Directory: "foo/auser",
				Name:      "setuid",
				User:      1001,
				Group:     1001,
			},
			Contents: "hello world\n",
			Mode:     04755,
		},
		{
			Node: types.Node{
				Directory: "foo/root",
				Name:      "setuid",
				User:      0,
				Group:     0,
			},
			Contents: "hello world\n",
			Mode:     04755,
		},
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "setgid",
			},
			Contents: "hello world\n",
			Mode:     02755,
		},
		{
			Node: types.Node{
				Directory: "foo/auser",
				Name:      "setgid",
				User:      1001,
				Group:     1001,
			},
			Contents: "hello world\n",
			Mode:     02755,
		},
		{
			Node: types.Node{
				Directory: "foo/root",
				Name:      "setgid",
				User:      0,
				Group:     0,
			},
			Contents: "hello world\n",
			Mode:     02755,
		},
	})
	configMinVersion := "3.6.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ApplyCustomFilePermissionsBeforeBugfix() types.Test {
	// This test is put here to ensure that we don't accidentally fix configs that were not working before the fix.
	// Ensure all versions before the bugfix are still losing special bits.
	// See: https://github.com/coreos/ignition/issues/2042
	name := "files.customperms.pre-#2042-fix"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "passwd": {
	    "users": [
	      {
	        "name": "auser",
	        "uid": 1001,
	        "shouldExist": false
	      }
	    ],
	    "groups": [
	      {
	        "name": "auser",
	        "gid": 1001,
	        "shouldExist": false
	      }
	    ]
	  },
	  "storage": {
	    "files": [
	      {
	        "path": "/foo/setuidsetgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 3565
	      },
	      {
	        "path": "/foo/auser/setuidsetgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 3565,
	        "user": { "id": 1001 },
	        "group": { "id": 1001 }
	      },
	      {
	        "path": "/foo/root/setuidsetgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 3565,
	        "user": { "id": 0 },
	        "group": { "id": 0 }
	      },
	      {
	        "path": "/foo/setuid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 2541
	      },
	      {
	        "path": "/foo/auser/setuid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 2541,
	        "user": { "id": 1001 },
	        "group": { "id": 1001 }
	      },
	      {
	        "path": "/foo/root/setuid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 2541,
	        "user": { "id": 0 },
	        "group": { "id": 0 }
	      },
	      {
	        "path": "/foo/setgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 1517
	      },
	      {
	        "path": "/foo/auser/setgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 1517,
	        "user": { "id": 1001 },
	        "group": { "id": 1001 }
	      },
	      {
	        "path": "/foo/root/setgid",
	        "contents": { "source": "data:,hello%20world%0A" },
	        "mode": 1517,
	        "user": { "id": 0 },
	        "group": { "id": 0 }
	      }
	    ]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "setuidsetgid",
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
		{
			Node: types.Node{
				Directory: "foo/auser",
				Name:      "setuidsetgid",
				User:      1001,
				Group:     1001,
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
		{
			Node: types.Node{
				Directory: "foo/root",
				Name:      "setuidsetgid",
				User:      0,
				Group:     0,
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "setuid",
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
		{
			Node: types.Node{
				Directory: "foo/auser",
				Name:      "setuid",
				User:      1001,
				Group:     1001,
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
		{
			Node: types.Node{
				Directory: "foo/root",
				Name:      "setuid",
				User:      0,
				Group:     0,
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "setgid",
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
		{
			Node: types.Node{
				Directory: "foo/auser",
				Name:      "setgid",
				User:      1001,
				Group:     1001,
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
		{
			Node: types.Node{
				Directory: "foo/root",
				Name:      "setgid",
				User:      0,
				Group:     0,
			},
			Contents: "hello world\n",
			Mode:     0755,
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMaxVersion: "3.5.0",
		ConfigMinVersion: "3.2.0",
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
