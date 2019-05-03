// Copyright 2019 Red Hat, Inc.
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

package filesystems

import (
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.NegativeTest, UsesSymlinks())
}

func UsesSymlinks() types.Test {
	name := "filesystem.pathusessymlinks"
	in := types.GetBaseDisk()
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "tmp0",
				Directory: "/",
			},
			Hard:   false,
			Target: "/",
		},
	})
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	out := in
	config := `{
		"ignition": {"version": "$version"},
		"storage": {
			"filesystems": [{
				"format": "ext4",
				"path": "/tmp0",
				"device": "$DEVICE"
			}]
		}
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		MntDevices:       mntDevices,
		ConfigMinVersion: configMinVersion,
	}
}
