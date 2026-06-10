// Copyright 2026 CoreOS, Inc.
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

package cmdline

import (
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, FetchConfigFromDevice())
}

func FetchConfigFromDevice() types.Test {
	name := "cmdline.device.fetch"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()

	// Config that will be placed on the labeled device partition.
	// This is what Ignition will actually read via the cmdline provider.
	deviceConfig := `{
		"ignition": { "version": "3.4.0" },
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": { "source": "data:,example%20file%0A" }
			}]
		}
	}`

	// Config for the test framework's validation. Uses $version so the
	// test is registered across spec versions. The file platform won't
	// be consulted because the cmdline provider takes priority.
	config := `{
		"ignition": { "version": "$version" },
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": { "source": "data:,example%20file%0A" }
			}]
		}
	}`
	configMinVersion := "3.0.0"

	// Add a second disk with a labeled partition containing the config file.
	// The cmdline provider will mount this partition and read config.ign.
	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:           "IGNCONFIG",
				Number:          1,
				Length:          65536,
				FilesystemType:  "ext4",
				FilesystemLabel: "IGNCONFIG",
				Files: []types.File{
					{
						Node: types.Node{
							Name:      "config.ign",
							Directory: "/",
						},
						Contents: deviceConfig,
					},
				},
			},
		},
	})
	out = append(out, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:           "IGNCONFIG",
				Number:          1,
				Length:          65536,
				FilesystemType:  "ext4",
				FilesystemLabel: "IGNCONFIG",
				Files: []types.File{
					{
						Node: types.Node{
							Name:      "config.ign",
							Directory: "/",
						},
						Contents: deviceConfig,
					},
				},
			},
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
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
		Env: []string{
			"IGNITION_KERNEL_CMDLINE_PATH=$SYSTEM_CONFIG_DIR/cmdline",
		},
		SystemDirFiles: []types.File{
			{
				Node: types.Node{
					Name:      "cmdline",
					Directory: "/",
				},
				Contents: "ignition.config.device=IGNCONFIG ignition.config.path=/config.ign",
			},
		},
	}
}
