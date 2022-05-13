// Copyright 2021 Red Hat, Inc.
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

package networkmanager

import (
	"fmt"

	"github.com/Wifx/gonetworkmanager"
)

type DHCPOptions map[string]string

// GetDHCPOptions returns a map where keys are the interface names, for the
// network interfaces where DHCP is enabled, and values the maps of DHCP options
//
// This uses the NetworkManager DBus API.
func GetDHCPOptions() (map[string]DHCPOptions, error) {
	// Create a new instance of gonetworkmanager
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return nil, err
	}

	// Get network devices
	devices, err := nm.GetPropertyDevices()
	if err != nil {
		return nil, err
	}

	res := make(map[string]DHCPOptions)

	for _, device := range devices {
		interfaceName, err := device.GetPropertyInterface()
		if err != nil {
			return nil, err
		}
		dhcpd, err := device.GetPropertyDHCP4Config()
		if err != nil {
			return nil, err
		}
		if dhcpd == nil {
			continue
		}

		dhcpo, err := dhcpd.GetPropertyOptions()
		if err != nil {
			return nil, err
		}
		res[interfaceName] = make(DHCPOptions)
		for k, v := range dhcpo {
			res[interfaceName][k] = fmt.Sprintf("%v", v)
		}
	}

	return res, nil
}
