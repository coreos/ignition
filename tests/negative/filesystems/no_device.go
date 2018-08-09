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

package filesystems

import (
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.NegativeTest, NoDevice())
	register.Register(register.NegativeTest, NoDeviceWithForce())
	register.Register(register.NegativeTest, NoDeviceWithWipeFilesystemTrue())
	register.Register(register.NegativeTest, NoDeviceWithWipeFilesystemFalse())
}

func NoDevice() types.Test {
	name := "No Device"
	in := types.GetBaseDisk()
	out := in
	config := `{
		"ignition": {"version": "$version"},
		"storage": {
			"filesystems": [{
				"mount": {
					"format": "ext4"
				},
				"name": "foobar"
			}]
		}
	}`
	configMinVersion := "2.0.0"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigShouldBeBad: true,
		ConfigMinVersion:  configMinVersion,
	}
}

func NoDeviceWithForce() types.Test {
	name := "No Device w/ Force"
	in := types.GetBaseDisk()
	out := in
	config := `{
		"ignition": {"version": "$version"},
		"storage": {
			"filesystems": [{
				"mount": {
					"format": "ext4",
					"create": {
						"force": true
					}
				},
				"name": "foobar"
			}]
		}
	}`
	configMinVersion := "2.0.0"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigShouldBeBad: true,
		ConfigMinVersion:  configMinVersion,
	}
}

func NoDeviceWithWipeFilesystemTrue() types.Test {
	name := "No Device w/ wipeFilesystem true"
	in := types.GetBaseDisk()
	out := in
	config := `{
		"ignition": {"version": "$version"},
		"storage": {
			"filesystems": [{
				"mount": {
					"format": "ext4",
					"wipeFilesystem": true
				},
				"name": "foobar"
			}]
		}
	}`
	configMinVersion := "2.1.0"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigShouldBeBad: true,
		ConfigMinVersion:  configMinVersion,
	}
}

func NoDeviceWithWipeFilesystemFalse() types.Test {
	name := "No Device w/ wipeFilesystem false"
	in := types.GetBaseDisk()
	out := in
	config := `{
		"ignition": {"version": "$version"},
		"storage": {
			"filesystems": [{
				"mount": {
					"format": "ext4",
					"wipeFilesystem": false
				},
				"name": "foobar"
			}]
		}
	}`
	configMinVersion := "2.1.0"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigShouldBeBad: true,
		ConfigMinVersion:  configMinVersion,
	}
}
