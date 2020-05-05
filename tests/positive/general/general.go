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
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigHTTPCompressed())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigHTTPUsingHeaders())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigHTTPUsingHeadersWithRedirect())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigHTTPUsingOverwrittenHeaders())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigHTTP())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigHTTPCompressed())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigHTTPUsingHeaders())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigHTTPUsingHeadersWithRedirect())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigHTTPUsingOverwrittenHeaders())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigTFTP())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigTFTP())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigData())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigData())
	register.Register(register.PositiveTest, VersionOnlyConfig())
	register.Register(register.PositiveTest, EmptyUserdata())
}

func ReformatFilesystemAndWriteFile() types.Test {
	name := "genernal.reformat.withfile"
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
	name := "config.replace.http"
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

func ReplaceConfigWithRemoteConfigHTTPCompressed() types.Test {
	name := "config.replace.http.compressed"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "replace": {
	        "compression": "gzip",
	        "source": "http://127.0.0.1:8080/config_compressed",
			"verification": { "hash": "sha512-HASH" }
	      }
	    }
	  }
	}`, "HASH", servers.ConfigHash, 1)
	configMinVersion := "3.1.0"
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

func ReplaceConfigWithRemoteConfigHTTPUsingHeaders() types.Test {
	name := "config.replace.http.headers"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "replace": {
			"source": "http://127.0.0.1:8080/config_headers",
			"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
			"verification": { "hash": "sha512-HASH" }
	      }
	    }
	  }
	}`, "HASH", servers.ConfigHash, 1)
	configMinVersion := "3.1.0"
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

func ReplaceConfigWithRemoteConfigHTTPUsingHeadersWithRedirect() types.Test {
	name := "config.replace.http.headers.redirect"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "replace": {
			"source": "http://127.0.0.1:8080/config_headers_redirect",
			"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
			"verification": { "hash": "sha512-HASH" }
	      }
	    }
	  }
	}`, "HASH", servers.ConfigHash, 1)
	configMinVersion := "3.1.0"
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

func ReplaceConfigWithRemoteConfigHTTPUsingOverwrittenHeaders() types.Test {
	name := "config.replace.http.headers.overwrite"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "replace": {
			"source": "http://127.0.0.1:8080/config_headers_overwrite",
			"httpHeaders": [
				{"name": "Keep-Alive", "value": "1000"},
				{"name": "Accept", "value": "application/json"},
				{"name": "Accept-Encoding", "value": "identity, compress"},
				{"name": "User-Agent", "value": "MyUA"}
			],
			"verification": { "hash": "sha512-HASH" }
	      }
	    }
	  }
	}`, "HASH", servers.ConfigHash, 1)
	configMinVersion := "3.1.0"
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
	name := "config.replace.tftp"
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
	name := "config.merge.http"
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

func AppendConfigWithRemoteConfigHTTPCompressed() types.Test {
	name := "config.merge.http.compressed"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "merge": [{
	        "compression": "gzip",
	        "source": "http://127.0.0.1:8080/config_compressed",
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
	configMinVersion := "3.1.0"
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

func AppendConfigWithRemoteConfigHTTPUsingHeaders() types.Test {
	name := "config.merge.http.headers"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "merge": [{
			"source": "http://127.0.0.1:8080/config_headers",
			"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
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
	configMinVersion := "3.1.0"
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

func AppendConfigWithRemoteConfigHTTPUsingHeadersWithRedirect() types.Test {
	name := "config.merge.http.headers.redirect"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "merge": [{
			"source": "http://127.0.0.1:8080/config_headers_redirect",
			"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
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
	configMinVersion := "3.1.0"
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

func AppendConfigWithRemoteConfigHTTPUsingOverwrittenHeaders() types.Test {
	name := "config.merge.http.headers.overwrite"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "merge": [{
			"source": "http://127.0.0.1:8080/config_headers_overwrite",
			"httpHeaders": [
				{"name": "Keep-Alive", "value": "1000"},
				{"name": "Accept", "value": "application/json"},
				{"name": "Accept-Encoding", "value": "identity, compress"},
				{"name": "User-Agent", "value": "MyUA"}
			],
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
	configMinVersion := "3.1.0"
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
	name := "config.merge.tftp"
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
	name := "config.replace.dataurl"
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
	name := "config.merge.dataurl"
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
	name := "general.versiononly"
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
	name := "general.empty"
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
