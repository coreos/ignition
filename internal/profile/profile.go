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

package profile

import (
	"encoding/json"
	"fmt"
	"os"
)

type Profile struct {
	OEM OEM `json:"oem"`
}

type OEM struct {
	// Device link where oem partition is found.
	Device string `json:"device,omitempty"`
	// OEM directories within root fs to consider before mounting.
	SearchDirectories []string `json:"search-dirs,omitempty"`
}

var defaultProfile = Profile{
	OEM: OEM{
		Device:            "/dev/disk/by-label/OEM",
		SearchDirectories: []string{"/usr/share/oem"},
	},
}

func New(filename string) (Profile, error) {
	profile := defaultProfile

	if filename == "" {
		return profile, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return Profile{}, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&profile); err != nil {
		return Profile{}, fmt.Errorf("decoding %s: %v", filename, err)
	}

	return profile, nil
}
