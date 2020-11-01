// Copyright 2020 CoreOS, Inc.
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
	register.Register(register.PositiveTest, CreateInstantiatedService())
	register.Register(register.PositiveTest, CreateEmptyDropinsUnit())
	register.Register(register.PositiveTest, TestUnmaskUnit())
	register.Register(register.PositiveTest, TestMaskUnit())
}

func CreateInstantiatedService() types.Test {
	name := "instantiated.unit.create"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": { "version": "$version" },
		"systemd": {
			"units": [
			  {
				"name": "echo@.service",
				"contents": "[Unit]\nDescription=f\n[Service]\nType=oneshot\nExecStart=/bin/echo %i\nRemainAfterExit=yes\n[Install]\nWantedBy=multi-user.target\n"
			  },
			  {
				"enabled": true,
				"name": "echo@bar.service"
			  },
			  {
				"enabled": true,
				"name": "echo@foo.service"
			  }
			]
		  }
		}`
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "echo@.service",
				Directory: "etc/systemd/system",
			},
			Contents: "[Unit]\nDescription=f\n[Service]\nType=oneshot\nExecStart=/bin/echo %i\nRemainAfterExit=yes\n[Install]\nWantedBy=multi-user.target\n",
		},
		{
			Node: types.Node{
				Name:      "20-ignition.preset",
				Directory: "etc/systemd/system-preset",
			},
			Contents: "enable echo@.service bar foo\n",
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

// CreateEmptyDropinsUnit writes an empty dropin to the disk.
func CreateEmptyDropinsUnit() types.Test {
	name := "unit.create.emptydropin"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": { "version": "$version" },
		"systemd": {
			"units": [
			  {
				"name": "zincati.service",
				"dropins": [
					{
						"name": "empty.conf",
						"contents": ""
					}
				]
			  }
			]
		  }
		}`
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "empty.conf",
				Directory: "etc/systemd/system/zincati.service.d",
			},
			Contents: "",
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

// TestUnmaskUnit tests unmasking systemd units. In the case that a
// systemd unit is masked already it will unmask it. In the case it is
// not masked it will do nothing. In the code below we have foo.service,
// which is masked in the input, and bar.service, which is not masked in
// the input. We test that foo.service is properly unmasked and that the
// unmasking of bar.service is skipped (because it wasn't masked to begin
// with) and that bar.service gets enabled.
func TestUnmaskUnit() types.Test {
	name := "unit.unmask"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": { "version": "$version" },
		"systemd": {
			"units": [
				{
					"name": "foo.service",
					"mask": false
				},
				{
					"name": "bar.service",
					"enabled": true,
					"mask": false
				}
			]
        }
		}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "etc/systemd/system",
				Name:      "foo.service",
			},
			Target: "/dev/null",
			Hard:   false,
		},
	})
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "foo.service",
				Directory: "usr/lib/systemd/system",
			},
			Contents: "[Unit]\nDescription=f\n[Service]\nType=oneshot\nExecStart=/usr/bin/true\n[Install]\nWantedBy=multi-user.target\n",
		},
		{
			Node: types.Node{
				Name:      "bar.service",
				Directory: "usr/lib/systemd/system",
			},
			Contents: "[Unit]\nDescription=f\n[Service]\nType=oneshot\nExecStart=/usr/bin/true\n[Install]\nWantedBy=multi-user.target\n",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "20-ignition.preset",
				Directory: "etc/systemd/system-preset",
			},
			Contents: "enable bar.service\n",
		},
	})
	out[0].Partitions.AddRemovedNodes("ROOT", []types.Node{
		{
			Directory: "etc/systemd/system",
			Name:      "foo.service",
		},
		{
			Directory: "etc/systemd/system",
			Name:      "bar.service",
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

// TestMaskUnit tests masking systemd units. In this case foo.service
// verified to be masked by checking it's symlinked to /dev/null.
// We'll also mask a non-existent bar.service.
func TestMaskUnit() types.Test {
	name := "unit.mask"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": { "version": "$version" },
		"systemd": {
			"units": [
				{
					"name": "foo.service",
					"mask": true
				},
				{
					"name": "bar.service",
					"mask": true
				}
			]
        }
		}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "foo.service",
				Directory: "usr/lib/systemd/system",
			},
			Contents: "[Unit]\nDescription=f\n[Service]\nType=oneshot\nExecStart=/usr/bin/true\n[Install]\nWantedBy=multi-user.target\n",
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "etc/systemd/system",
				Name:      "foo.service",
			},
			Target: "/dev/null",
			Hard:   false,
		},
		{
			Node: types.Node{
				Directory: "etc/systemd/system",
				Name:      "bar.service",
			},
			Target: "/dev/null",
			Hard:   false,
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
