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
	register.Register(register.PositiveTest, OEMSearchPath())
}

func OEMSearchPath() types.Test {
	name := "Read files from multiple locations in OEM search path"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {"version": "2.0.0"},
		"storage": {
			"files": [
				{
					"filesystem": "root",
					"path": "/ignition/out-1",
					"contents": {"source": "oem:///source-1"}
				},
				{
					"filesystem": "root",
					"path": "/ignition/out-2",
					"contents": {"source": "oem:///source-2"}
				},
				{
					"filesystem": "root",
					"path": "/ignition/out-3",
					"contents": {"source": "oem:///source-3"}
				}
			]}
	}`
	oemDirs := [][]types.File{
		{
			{
				Node: types.Node{
					Name: "source-1",
				},
				Contents: "source-a",
			},
		},
		{
			{
				Node: types.Node{
					Name: "source-2",
				},
				Contents: "source-b",
			},
		},
	}
	in[0].Partitions.AddFiles("OEM", []types.File{
		{
			Node: types.Node{
				Name: "source-3",
			},
			Contents: "source-c",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "out-1",
				Directory: "ignition",
			},
			Contents: "source-a",
		},
		{
			Node: types.Node{
				Name:      "out-2",
				Directory: "ignition",
			},
			Contents: "source-b",
		},
		{
			Node: types.Node{
				Name:      "out-3",
				Directory: "ignition",
			},
			Contents: "source-c",
		},
	})

	return types.Test{
		Name:    name,
		In:      in,
		Out:     out,
		OEMDirs: oemDirs,
		Config:  config,
	}
}
