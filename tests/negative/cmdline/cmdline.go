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
	register.Register(register.NegativeTest, ConfigFileNotFound())
	register.Register(register.NegativeTest, DeviceNotFound())
	register.Register(register.NegativeTest, DeviceWithoutPath())
	register.Register(register.NegativeTest, PathWithoutDevice())
}

// ConfigFileNotFound verifies that Ignition fails when the device exists
// but the referenced config file path does not.
func ConfigFileNotFound() types.Test {
	name := "cmdline.device.config.notfound"
	in := types.GetBaseDisk()
	out := in

	config := `{
		"ignition": { "version": "$version" }
	}`
	configMinVersion := "3.0.0"

	// Add a labeled disk but do NOT put any config file on it.
	in = append(in, types.Disk{
		Alignment: types.IgnitionAlignment,
		Partitions: types.Partitions{
			{
				Label:           "IGNCONFIG",
				Number:          1,
				Length:          65536,
				FilesystemType:  "ext4",
				FilesystemLabel: "IGNCONFIG",
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
			},
		},
	})

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigMinVersion:  configMinVersion,
		SkipCriticalCheck: true,
		Env: []string{
			"IGNITION_KERNEL_CMDLINE_PATH=$SYSTEM_CONFIG_DIR/cmdline",
		},
		SystemDirFiles: []types.File{
			{
				Node: types.Node{
					Name:      "cmdline",
					Directory: "/",
				},
				Contents: "ignition.config.device=IGNCONFIG ignition.config.path=/nonexistent.ign",
			},
		},
	}
}

// DeviceNotFound verifies that Ignition fails when the specified device
// label does not exist (times out waiting for the device).
func DeviceNotFound() types.Test {
	name := "cmdline.device.notfound"
	in := types.GetBaseDisk()
	out := in

	config := `{
		"ignition": { "version": "$version" }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigMinVersion:  configMinVersion,
		SkipCriticalCheck: true,
		Env: []string{
			"IGNITION_KERNEL_CMDLINE_PATH=$SYSTEM_CONFIG_DIR/cmdline",
		},
		SystemDirFiles: []types.File{
			{
				Node: types.Node{
					Name:      "cmdline",
					Directory: "/",
				},
				Contents: "ignition.config.device=DOESNOTEXIST ignition.config.path=/config.ign",
			},
		},
	}
}

// DeviceWithoutPath verifies that Ignition fails when only
// ignition.config.device is set without ignition.config.path.
func DeviceWithoutPath() types.Test {
	name := "cmdline.device.without.path"
	in := types.GetBaseDisk()
	out := in

	config := `{
		"ignition": { "version": "$version" }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigMinVersion:  configMinVersion,
		SkipCriticalCheck: true,
		Env: []string{
			"IGNITION_KERNEL_CMDLINE_PATH=$SYSTEM_CONFIG_DIR/cmdline",
		},
		SystemDirFiles: []types.File{
			{
				Node: types.Node{
					Name:      "cmdline",
					Directory: "/",
				},
				Contents: "ignition.config.device=IGNCONFIG",
			},
		},
	}
}

// PathWithoutDevice verifies that Ignition fails when only
// ignition.config.path is set without ignition.config.device.
func PathWithoutDevice() types.Test {
	name := "cmdline.path.without.device"
	in := types.GetBaseDisk()
	out := in

	config := `{
		"ignition": { "version": "$version" }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigMinVersion:  configMinVersion,
		SkipCriticalCheck: true,
		Env: []string{
			"IGNITION_KERNEL_CMDLINE_PATH=$SYSTEM_CONFIG_DIR/cmdline",
		},
		SystemDirFiles: []types.File{
			{
				Node: types.Node{
					Name:      "cmdline",
					Directory: "/",
				},
				Contents: "ignition.config.path=/config.ign",
			},
		},
	}
}
