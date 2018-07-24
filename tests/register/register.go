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

package register

import (
	"fmt"
	"github.com/coreos/ignition/tests/types"
	"strings"

	json "github.com/ajeddeloh/go-json"
	"github.com/coreos/go-semver/semver"
	types1 "github.com/coreos/ignition/config/v1/types"
	types20 "github.com/coreos/ignition/config/v2_0/types"
	types21 "github.com/coreos/ignition/config/v2_1/types"
	types22 "github.com/coreos/ignition/config/v2_2/types"
	types23 "github.com/coreos/ignition/config/v2_3_experimental/types"
	typesInternal "github.com/coreos/ignition/internal/config/types"
)

type TestType int

const (
	NegativeTest TestType = iota
	PositiveTest
)

var Tests map[TestType][]types.Test

func init() {
	Tests = make(map[TestType][]types.Test)
}

var count = 0

func register(tType TestType, t types.Test) {
	Tests[tType] = append(Tests[tType], t)
	count++
	fmt.Printf("Count at: %d\n", count)
}

// Registers t multiple times with different, compatible config versions
func Register(tType TestType, t types.Test) {
	// update this 2-D array with new config versions
	configVersions := [][]semver.Version{
		{semver.Version{}}, // place holder
		{types1.MaxVersion},
		{types20.MaxVersion, types21.MaxVersion, types22.MaxVersion, types23.MaxVersion},
	}

	// todo: reformat a filesystem
	// todo: "Appending to the Config with a Remote Config from OEM", preemeption, //

	var config typesInternal.Config
	err := json.Unmarshal([]byte(t.Config), &config)
	version, semverErr := semver.NewVersion(config.Ignition.Version) // todo: how to handle errors - someone which forget to put version in config and leave configminversion blank or test might purposely leave it out

	register(tType, t)
	//todo: what happends if 3.0 is maxTypes.MaxVersion? also document this under update all relevenat places to use the new experiemental package
	// if t.ConfigMinVersion if blank, config needs to have version already; this means tests should be ran only as that specific version
	if err == nil && semverErr == nil && version != nil && t.ConfigMinVersion != "" {
		initalVersion := *version

		for *version != configVersions[version.Major][len(configVersions[version.Major])-1] { // todo: if version.String() != configVersions, set version = configVersions
			test := types.DeepCopy(t) // replace test with t for ease of reading

			// version.BumpMinor()
			// if version.Minor == 3 { // todo: make this less hardcoded
			// 	*version = types23.MaxVersion
			// }

			version.BumpMinor()
			version = &configVersions[version.Major][version.Minor]

			test.Name += " " + version.String()
			test.Config = strings.Replace(test.Config, initalVersion.String(), version.String(), 1)
			register(tType, test)
		}
	}
}
