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
	types1 "github.com/coreos/ignition/config/v1/types"
	types20 "github.com/coreos/ignition/config/v2_0/types"
	types21 "github.com/coreos/ignition/config/v2_1/types"
	types22 "github.com/coreos/ignition/config/v2_2/types"
	types23 "github.com/coreos/ignition/config/v2_3/types"
	types24 "github.com/coreos/ignition/config/v2_4_experimental/types"
	"github.com/coreos/ignition/tests/types"
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
		{semver.Version{}}, // place holder
		{types1.MaxVersion},
		{types20.MaxVersion, types21.MaxVersion, types22.MaxVersion, types23.MaxVersion, types24.MaxVersion},
	}

	test := types.DeepCopy(t)
	version, semverErr := semver.NewVersion(test.ConfigMinVersion)
	test.ReplaceAllVersionVars(test.ConfigMinVersion)
	register(tType, test) // some tests purposefully don't have config version

	if semverErr == nil && version != nil && t.ConfigMinVersion != "" {
		for _, v := range configVersions[version.Major] {
			if version.LessThan(v) {
				test = types.DeepCopy(t)
				test.ReplaceAllVersionVars(v.String())
				register(tType, test)
			}
		}
	}
}
