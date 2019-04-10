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

package general

import (
	"strings"

	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/servers"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	// TODO: Add S3 tests
	register.Register(register.PositiveTest, ReformatFilesystemAndWriteFile())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigHTTP())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigHTTP())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigTFTP())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigTFTP())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigData())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigData())
	register.Register(register.PositiveTest, VersionOnlyConfig())
	register.Register(register.PositiveTest, EmptyUserdata())
}

func ReformatFilesystemAndWriteFile() types.Test {
	name := "Reformat Filesystem to ext4 & drop file in /ignition/test"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "EFI-SYSTEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
		"ignition": {"version": "$version"},
		"storage": {
			"filesystems": [{
				"path": "/tmp0",
				"device": "$DEVICE",
				"format": "ext4",
				"wipeFilesystem": true
			}],
			"files": [{
				"path": "/tmp0/test",
				"contents": {"source": "data:,asdf"}
			}]}
	}`
	configMinVersion := "3.0.0"

	out[0].Partitions.GetPartition("EFI-SYSTEM").FilesystemType = "ext4"
	out[0].Partitions.GetPartition("EFI-SYSTEM").Files = []types.File{
		{
			Node: types.Node{
				Name:      "test",
				Directory: "/",
			},
			Contents: "asdf",
		},
	}

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ReplaceConfigWithRemoteConfigHTTP() types.Test {
	name := "Replacing the Config with a Remote Config from HTTP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "replace": {
	        "source": "http://127.0.0.1:8080/config",
			"verification": { "hash": "sha512-HASH" }
	      }
	    }
	  }
	}`, "HASH", servers.ConfigHash, 1)
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
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

func ReplaceConfigWithRemoteConfigTFTP() types.Test {
	name := "Replacing the Config with a Remote Config from TFTP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
          "ignition": {
            "version": "$version",
            "config": {
              "replace": {
                "source": "tftp://127.0.0.1:69/config",
                        "verification": { "hash": "sha512-HASH" }
              }
            }
          }
        }`, "HASH", servers.ConfigHash, 1)
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
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

func AppendConfigWithRemoteConfigHTTP() types.Test {
	name := "Appending to the Config with a Remote Config from HTTP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "merge": [{
	        "source": "http://127.0.0.1:8080/config",
			"verification": { "hash": "sha512-HASH" }
	      }]
	    }
	  },
      "storage": {
        "files": [{
          "path": "/foo/bar2",
          "contents": { "source": "data:,another%20example%20file%0A" }
        }]
      }
	}`, "HASH", servers.ConfigHash, 1)
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
		},
		{
			Node: types.Node{
				Name:      "bar2",
				Directory: "foo",
			},
			Contents: "another example file\n",
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

func AppendConfigWithRemoteConfigTFTP() types.Test {
	name := "Appending to the Config with a Remote Config from TFTP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
          "ignition": {
            "version": "$version",
            "config": {
              "merge": [{
                "source": "tftp://127.0.0.1:69/config",
                        "verification": { "hash": "sha512-HASH" }
              }]
            }
          },
      "storage": {
        "files": [{
          "path": "/foo/bar2",
          "contents": { "source": "data:,another%20example%20file%0A" }
        }]
      }
        }`, "HASH", servers.ConfigHash, 1)
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
		},
		{
			Node: types.Node{
				Name:      "bar2",
				Directory: "foo",
			},
			Contents: "another example file\n",
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

func ReplaceConfigWithRemoteConfigData() types.Test {
	name := "Replacing the Config with a Remote Config from Data"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
          "ignition": {
            "version": "$version",
            "config": {
              "replace": {
				  "source": "data:,%7B%22ignition%22%3A%7B%22version%22%3A%20%223.0.0%22%7D%2C%22storage%22%3A%20%7B%22files%22%3A%20%5B%7B%22filesystem%22%3A%20%22root%22%2C%22path%22%3A%20%22%2Ffoo%2Fbar%22%2C%22contents%22%3A%7B%22source%22%3A%22data%3A%2Canother%2520example%2520file%250A%22%7D%7D%5D%7D%7D%0A"
              }
            }
          }
        }`
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "another example file\n",
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

func AppendConfigWithRemoteConfigData() types.Test {
	name := "Appending to the Config with a Remote Config from Data"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
          "ignition": {
            "version": "$version",
            "config": {
              "merge": [{
				  "source": "data:,%7B%22ignition%22%3A%7B%22version%22%3A%20%223.0.0%22%7D%2C%22storage%22%3A%20%7B%22files%22%3A%20%5B%7B%22filesystem%22%3A%20%22root%22%2C%22path%22%3A%20%22%2Ffoo%2Fbar%22%2C%22contents%22%3A%7B%22source%22%3A%22data%3A%2Canother%2520example%2520file%250A%22%7D%7D%5D%7D%7D%0A"
              }]
            }
          }
        }`
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "another example file\n",
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

func VersionOnlyConfig() types.Test {
	name := "Version Only Config"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {"version": "$version"}
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func EmptyUserdata() types.Test {
	name := "Empty Userdata"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := ``

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigShouldBeBad: true,
	}
}
