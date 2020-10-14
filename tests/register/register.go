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
	"github.com/coreos/go-semver/semver"
	types30 "github.com/coreos/ignition/v2/config/v3_0/types"
	types31 "github.com/coreos/ignition/v2/config/v3_1/types"
	types32 "github.com/coreos/ignition/v2/config/v3_2/types"
	types_exp "github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/tests/types"
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

func register(tType TestType, t types.Test) {
	Tests[tType] = append(Tests[tType], t)
}

// Registers t for every version, inside the same major version,
// that is equal to or greater than the specified ConfigMinVersion.
func Register(tType TestType, t types.Test) {
	// update confgiVersions with new config versions
	configVersions := [][]semver.Version{
		{semver.Version{}}, // place holder 0
		{semver.Version{}}, // place holder 1
		{semver.Version{}}, // place holder 2
		{types30.MaxVersion, types31.MaxVersion, types32.MaxVersion, types_exp.MaxVersion},
	}

	test := types.DeepCopy(t)
	version, semverErr := semver.NewVersion(test.ConfigMinVersion)
	test.ReplaceAllVersionVars(test.ConfigMinVersion)
	test.ConfigVersion = test.ConfigMinVersion
	register(tType, test) // some tests purposefully don't have config version

	if semverErr == nil && version != nil && t.ConfigMinVersion != "" {
		for _, v := range configVersions[version.Major] {
			if version.LessThan(v) {
				test = types.DeepCopy(t)
				test.ReplaceAllVersionVars(v.String())
				test.ConfigVersion = v.String()
				register(tType, test)
			}
		}
	}
}
