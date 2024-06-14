// Copyright 2023 CoreOS, Inc.
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

package luks

import (
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, LuksWithKeyfileKey())
	register.Register(register.PositiveTest, LuksWithTPM2())

}

func LuksWithKeyfileKey() types.Test {
	name := "luks.formattedDevice.wipeVolume.keyfile"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
		"ignition": { "version": "$version" },
		"storage": {
		  "luks": [
			{
			  "device": "$DEVICE",
			  "name": "$uuid1",
			  "keyFile": {
				"compression": "",
				"source": "data:,REPLACE-THIS-WITH-YOUR-KEY-MATERIAL"
			  },
			  "wipeVolume": true
			}
		  ]
		}
	}`
	configMinVersion := "3.2.0"
	in[0].Partitions.GetPartition("OEM").FilesystemType = "ext4"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "crypto_LUKS"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func LuksWithTPM2() types.Test {
	name := "luks.formattedDevice.wipeVolume.tpm2"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
		"ignition": { "version": "$version" },
		"storage": {
		  "luks": [
			{
			  "clevis": {
				"tpm2": true
			  },	
			  "device": "$DEVICE",
			  "name": "$uuid1",
			  "wipeVolume": true
			}
		  ]
		}
	}`
	configMinVersion := "3.2.0"
	in[0].Partitions.GetPartition("OEM").FilesystemType = "ext4"
	out[0].Partitions.GetPartition("OEM").FilesystemType = "crypto_LUKS"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
