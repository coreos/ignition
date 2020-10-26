// Copyright 2020 Red Hat, Inc.
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
	register.Register(register.PositiveTest, VerifyMergingOfMultipleBaseConfigs())
}

var (
	baseConfig = []byte(`{
				"ignition": { "version": "3.0.0" },
				"storage": {
					"files": [{
						"path": "/foo/bar",
						"contents": { "source": "data:,base%20config%0A" }
					}]
				}
			}`)

	baseConfig2 = []byte(`{
				"ignition": { "version": "3.0.0" },
				"storage": {
					"files": [{
						"path": "/foo/bar2",
						"contents": { "source": "data:,base%20config2%0A" }
					}]
				}
			}`)
	baseConfig3 = []byte(`{
				"ignition": { "version": "3.0.0" },
				"storage": {
					"files": [{
						"path": "/foo/bar",
						"contents": { "source": "data:,base%20config3%0A" }
					}]
				}
			}`)
	platformConfig = []byte(`{
				"ignition": { "version": "3.0.0" },
				"storage": {
					"files": [{
						"path": "/foo/bar3",
						"contents": { "source": "data:,platform%20config%0A" }
					}]
				}
			}`)
	platformConfig2 = []byte(`{
				"ignition": { "version": "3.0.0" },
				"storage": {
					"files": [{
						"path": "/foo/bar4",
						"contents": { "source": "data:,platform%20config2%0A" }
					}]
				}
			}`)
	platformConfig3 = []byte(`{
				"ignition": { "version": "3.0.0" },
				"storage": {
					"files": [{
						"path": "/foo/bar2",
						"contents": { "source": "data:,platform%20config3%0A" }
					}]
				}
			}`)
)

// VerifyMergingOfMultipleBaseConfigs checks if multiple
// base configs and/or platform configs are merged in the
// right order.
func VerifyMergingOfMultipleBaseConfigs() types.Test {
	name := "merge.multiple.base.configs"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {"version": "$version"}
	}`
	configMinVersion := "3.0.0"
	var systemFiles []types.File

	systemFiles = append(systemFiles, types.File{
		Node: types.Node{
			Name:      "40-base.ign",
			Directory: "base.d",
		},
		Contents: string(baseConfig),
	}, types.File{
		Node: types.Node{
			Name:      "50-base2.ign",
			Directory: "base.d",
		},
		Contents: string(baseConfig2),
	}, types.File{
		Node: types.Node{
			Name:      "60-base3.ign",
			Directory: "base.d",
		},
		Contents: string(baseConfig3),
	}, types.File{
		Node: types.Node{
			Name:      "10-plaform.ign",
			Directory: "base.platform.d/file",
		},
		Contents: string(platformConfig),
	}, types.File{
		Node: types.Node{
			Name:      "20-plaform2.ign",
			Directory: "base.platform.d/file",
		},
		Contents: string(platformConfig2),
	}, types.File{
		Node: types.Node{
			Name:      "30-plaform3.ign",
			Directory: "base.platform.d/file",
		},
		Contents: string(platformConfig3),
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "base config3\n",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar2",
				Directory: "foo",
			},
			Contents: "platform config3\n",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar3",
				Directory: "foo",
			},
			Contents: "platform config\n",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar4",
				Directory: "foo",
			},
			Contents: "platform config2\n",
		},
	})
	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		SystemDirFiles:   systemFiles,
		ConfigMinVersion: configMinVersion,
	}
}
