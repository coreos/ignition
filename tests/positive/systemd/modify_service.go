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
	register.Register(register.PositiveTest, ModifySystemdService())
	register.Register(register.PositiveTest, MaskSystemdServices())
}

func ModifySystemdService() types.Test {
	name := "systemd.unit.modify"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "systemd": {
	    "units": [{
	      "name": "systemd-journald.service",
	      "dropins": [{
	        "name": "debug.conf",
	        "contents": "[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug"
	      }]
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "debug.conf",
				Directory: "etc/systemd/system/systemd-journald.service.d",
			},
			Contents: "[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug",
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

func MaskSystemdServices() types.Test {
	name := "systemd.unit.mask"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "systemd": {
	    "units": [{
	      "name": "systemd-journald.service",
		  "mask": true
	    }]
	  }
	}`
	configMinVersion := "3.0.0"
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "systemd-journald.service",
				Directory: "etc/systemd/system",
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
