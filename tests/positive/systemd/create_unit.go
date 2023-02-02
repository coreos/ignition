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

package systemd

import (
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, CreateSystemdService())
	register.Register(register.PositiveTest, CreateSystemdUserService())
}

func CreateSystemdService() types.Test {
	name := "systemd.unit.create"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": { "version": "$version" },
		"systemd": {
			"units": [{
				"name": "example.service",
				"enabled": true,
				"contents": "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"
			}]
		}
	}`
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "example.service",
				Directory: "etc/systemd/system",
			},
			Contents: "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target",
		},
		{
			Node: types.Node{
				Name:      "20-ignition.preset",
				Directory: "etc/systemd/system-preset",
			},
			Contents: "enable example.service\n",
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

func CreateSystemdUserService() types.Test {
	name := "systemd.unit.userunit.create"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": { "version": "$version" },
		"systemd": {
			"units": [{
				"name": "example.service",
				"enabled": true,
				"scope": "user",
				"users": ["tester1", "tester2"],
				"contents": "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"
			},
			{
				"contents": "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target",
				"enabled": true,
				"name": "example.service",
				"scope": "global"
			}]
		}
	}`
	configMinVersion := "3.4.0-experimental"

	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "passwd",
				Directory: "etc",
			},
			Contents: "tester1:x:44:4242::/var/users/tester1:/bin/false\ntester2:x:45:4242::/home/tester2:/bin/false",
		},
		{
			Node: types.Node{
				Name:      "nsswitch.conf",
				Directory: "etc",
			},
			Contents: "passwd: files\ngroup: files\nshadow: files\ngshadow: files\n",
		},
	})

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "example.service",
				Directory: "var/users/tester1/.config/systemd/user",
			},
			Contents: "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target",
		},
		{
			Node: types.Node{
				Name:      "example.service",
				Directory: "home/tester2/.config/systemd/user",
			},
			Contents: "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target",
		},
		{
			Node: types.Node{
				Name:      "21-ignition.preset",
				Directory: "etc/systemd/user-preset",
			},
			Contents: "enable example.service\n",
		},

		{
			Node: types.Node{
				Name:      "example.service",
				Directory: "etc/systemd/user",
			},
			Contents: "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target",
		},
		{
			Node: types.Node{
				Name:      "20-ignition.preset",
				Directory: "etc/systemd/user-preset",
			},
			Contents: "enable example.service\n",
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
