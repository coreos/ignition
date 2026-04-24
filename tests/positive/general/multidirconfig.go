// Copyright 2024 Red Hat, Inc.
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
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, UserIgnFromLocalDir())
	register.Register(register.PositiveTest, UserIgnRuntimeOverridesLocal())
	register.Register(register.PositiveTest, BaseDFragmentsMergedAcrossDirs())
}

// UserIgnFromLocalDir verifies that user.ign is found in the local
// config directory when no higher-priority directory contains it.
func UserIgnFromLocalDir() types.Test {
	name := "multidir.user.ign.from.local"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()

	config := `{
		"ignition": {"version": "$version"}
	}`
	configMinVersion := "3.0.0"

	userConfig := `{
		"ignition": {"version": "3.0.0"},
		"storage": {
			"files": [{
				"path": "/ignition/source",
				"contents": {"source": "data:,local"},
				"overwrite": true
			}]
		}
	}`

	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "source",
				Directory: "ignition",
			},
			Contents: "unset",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "source",
				Directory: "ignition",
			},
			Contents: "local",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
		SystemLocalDirFiles: []types.File{
			{
				Node: types.Node{
					Name: "user.ign",
				},
				Contents: userConfig,
			},
		},
		ConfigShouldBeBad: true,
	}
}

// UserIgnRuntimeOverridesLocal verifies that user.ign in the runtime
// directory takes precedence over the local and vendor directories.
func UserIgnRuntimeOverridesLocal() types.Test {
	name := "multidir.user.ign.runtime.overrides"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()

	config := `{
		"ignition": {"version": "$version"}
	}`
	configMinVersion := "3.0.0"

	makeUserConfig := func(source string) string {
		return `{
			"ignition": {"version": "3.0.0"},
			"storage": {
				"files": [{
					"path": "/ignition/source",
					"contents": {"source": "data:,` + source + `"},
					"overwrite": true
				}]
			}
		}`
	}

	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "source",
				Directory: "ignition",
			},
			Contents: "unset",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "source",
				Directory: "ignition",
			},
			Contents: "runtime",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
		SystemDirFiles: []types.File{
			{
				Node:     types.Node{Name: "user.ign"},
				Contents: makeUserConfig("vendor"),
			},
		},
		SystemLocalDirFiles: []types.File{
			{
				Node:     types.Node{Name: "user.ign"},
				Contents: makeUserConfig("local"),
			},
		},
		SystemRuntimeDirFiles: []types.File{
			{
				Node:     types.Node{Name: "user.ign"},
				Contents: makeUserConfig("runtime"),
			},
		},
		ConfigShouldBeBad: true,
	}
}

// BaseDFragmentsMergedAcrossDirs verifies that base.d fragments are
// merged across directories, with higher-priority dirs winning for
// same-named files.
func BaseDFragmentsMergedAcrossDirs() types.Test {
	name := "multidir.base.d.merged.across.dirs"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()

	config := `{
		"ignition": {"version": "$version"}
	}`
	configMinVersion := "3.0.0"

	vendorBase := `{
		"ignition": {"version": "3.0.0"},
		"storage": {
			"files": [{
				"path": "/foo/from-vendor",
				"contents": {"source": "data:,vendor"}
			}]
		}
	}`

	localBase := `{
		"ignition": {"version": "3.0.0"},
		"storage": {
			"files": [{
				"path": "/foo/from-local",
				"contents": {"source": "data:,local"}
			}]
		}
	}`

	// Same filename as vendor; runtime should win
	runtimeOverride := `{
		"ignition": {"version": "3.0.0"},
		"storage": {
			"files": [{
				"path": "/foo/from-vendor",
				"contents": {"source": "data:,runtime-override"}
			}]
		}
	}`

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "from-vendor",
				Directory: "foo",
			},
			Contents: "runtime-override",
		},
		{
			Node: types.Node{
				Name:      "from-local",
				Directory: "foo",
			},
			Contents: "local",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
		SystemDirFiles: []types.File{
			{
				Node: types.Node{
					Name:      "40-base.ign",
					Directory: "base.d",
				},
				Contents: vendorBase,
			},
		},
		SystemLocalDirFiles: []types.File{
			{
				Node: types.Node{
					Name:      "50-local.ign",
					Directory: "base.d",
				},
				Contents: localBase,
			},
		},
		SystemRuntimeDirFiles: []types.File{
			{
				Node: types.Node{
					Name:      "40-base.ign",
					Directory: "base.d",
				},
				Contents: runtimeOverride,
			},
		},
	}
}
