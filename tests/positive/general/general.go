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
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.PositiveTest, ReformatFilesystemAndWriteFile())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigHTTP())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigHTTP())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigTFTP())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigTFTP())
	register.Register(register.PositiveTest, ReplaceConfigWithRemoteConfigOEM())
	register.Register(register.PositiveTest, AppendConfigWithRemoteConfigOEM())
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
		"ignition": {"version": "2.0.0"},
		"storage": {
			"filesystems": [{
				"mount": {
					"device": "$DEVICE",
					"format": "ext4",
					"create": {
						"force": true
					}},
				 "name": "test"}],
			"files": [{
				"filesystem": "test",
				"path": "/ignition/test",
				"contents": {"source": "data:,asdf"}
			}]}
	}`

	out[0].Partitions.GetPartition("EFI-SYSTEM").FilesystemType = "ext4"
	out[0].Partitions.GetPartition("EFI-SYSTEM").Files = []types.File{
		{
			Node: types.Node{
				Name:      "test",
				Directory: "ignition",
			},
			Contents: "asdf",
		},
	}

	return types.Test{
		Name:       name,
		In:         in,
		Out:        out,
		MntDevices: mntDevices,
		Config:     config,
	}
}

func ReplaceConfigWithRemoteConfigHTTP() types.Test {
	name := "Replacing the Config with a Remote Config from HTTP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": {
	    "version": "2.0.0",
	    "config": {
	      "replace": {
	        "source": "http://127.0.0.1:8080/config",
			"verification": { "hash": "sha512-41d9a1593dd4cbcacc966dce574523ffe3780ec2710716fab28b46f0f24d20b5ec49f307a9e9d331af958e508f472f32135c740d1214c5f02fc36016b538e7ff" }
	      }
	    }
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

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func ReplaceConfigWithRemoteConfigTFTP() types.Test {
	name := "Replacing the Config with a Remote Config from TFTP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
          "ignition": {
            "version": "2.1.0",
            "config": {
              "replace": {
                "source": "tftp://127.0.0.1:69/config",
                        "verification": { "hash": "sha512-fa00083efe3f00eb984e6dc27cc8673585cce4319e39099ce014103619ae7ab7dc3555e51401c7df472bdd125c552e528f54d717b8147129c99836d3dedc9760" }
              }
            }
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

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func ReplaceConfigWithRemoteConfigOEM() types.Test {
	name := "Replacing the Config with a Remote Config from OEM"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
          "ignition": {
            "version": "2.1.0",
            "config": {
              "replace": {
                "source": "oem:///config",
                        "verification": { "hash": "sha512-73395ffef4b1aefac56b6406f7aed307199d960cc8ad9317e0e8b6497a64f879b33fd59eca533f5f139aa4237f7d81de08c6f7f17db9dd2c072e9ecccb0fed42" }
              }
            }
          }
        }`
	in[0].Partitions.AddFiles("OEM", []types.File{
		{
			Node: types.Node{
				Name: "config",
			},
			Contents: `{
	"ignition": { "version": "2.1.0" },
	"storage": {
		"files": [{
		  "filesystem": "root",
		  "path": "/foo/bar",
		  "contents": { "source": "data:,example%20file%0A" }
		}]
	}
}`,
		},
	})
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
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func AppendConfigWithRemoteConfigHTTP() types.Test {
	name := "Appending to the Config with a Remote Config from HTTP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": {
	    "version": "2.0.0",
	    "config": {
	      "append": [{
	        "source": "http://127.0.0.1:8080/config",
			"verification": { "hash": "sha512-41d9a1593dd4cbcacc966dce574523ffe3780ec2710716fab28b46f0f24d20b5ec49f307a9e9d331af958e508f472f32135c740d1214c5f02fc36016b538e7ff" }
	      }]
	    }
	  },
      "storage": {
        "files": [{
          "filesystem": "root",
          "path": "/foo/bar2",
          "contents": { "source": "data:,another%20example%20file%0A" }
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
		{
			Node: types.Node{
				Name:      "bar2",
				Directory: "foo",
			},
			Contents: "another example file\n",
		},
	})

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func AppendConfigWithRemoteConfigTFTP() types.Test {
	name := "Appending to the Config with a Remote Config from TFTP"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
          "ignition": {
            "version": "2.1.0",
            "config": {
              "append": [{
                "source": "tftp://127.0.0.1:69/config",
                        "verification": { "hash": "sha512-fa00083efe3f00eb984e6dc27cc8673585cce4319e39099ce014103619ae7ab7dc3555e51401c7df472bdd125c552e528f54d717b8147129c99836d3dedc9760" }
              }]
            }
          },
      "storage": {
        "files": [{
          "filesystem": "root",
          "path": "/foo/bar2",
          "contents": { "source": "data:,another%20example%20file%0A" }
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
		{
			Node: types.Node{
				Name:      "bar2",
				Directory: "foo",
			},
			Contents: "another example file\n",
		},
	})

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func AppendConfigWithRemoteConfigOEM() types.Test {
	name := "Appending to the Config with a Remote Config from OEM"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
          "ignition": {
            "version": "2.1.0",
            "config": {
              "append": [{
                "source": "oem:///config",
                        "verification": { "hash": "sha512-73395ffef4b1aefac56b6406f7aed307199d960cc8ad9317e0e8b6497a64f879b33fd59eca533f5f139aa4237f7d81de08c6f7f17db9dd2c072e9ecccb0fed42" }
              }]
            }
          },
      "storage": {
        "files": [{
          "filesystem": "root",
          "path": "/foo/bar2",
          "contents": { "source": "data:,another%20example%20file%0A" }
        }]
      }
        }`
	in[0].Partitions.AddFiles("OEM", []types.File{
		{
			Node: types.Node{
				Name: "config",
			},
			Contents: `{
	"ignition": { "version": "2.1.0" },
	"storage": {
		"files": [{
		  "filesystem": "root",
		  "path": "/foo/bar",
		  "contents": { "source": "data:,example%20file%0A" }
		}]
	}
}`,
		},
	})
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
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func VersionOnlyConfig() types.Test {
	name := "Version Only Config"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {"version": "2.1.0"}
	}`

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}

func EmptyUserdata() types.Test {
	name := "Empty Userdata"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := ``

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}
