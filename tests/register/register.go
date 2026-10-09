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
	"strings"

	"github.com/coreos/go-semver/semver"
	types30 "github.com/coreos/ignition/v2/config/v3_0/types"
	types31 "github.com/coreos/ignition/v2/config/v3_1/types"
	types32 "github.com/coreos/ignition/v2/config/v3_2/types"
	types33 "github.com/coreos/ignition/v2/config/v3_3/types"
	types34 "github.com/coreos/ignition/v2/config/v3_4/types"
	types35 "github.com/coreos/ignition/v2/config/v3_5/types"
	types36 "github.com/coreos/ignition/v2/config/v3_6/types"
	types_exp "github.com/coreos/ignition/v2/config/v3_7_experimental/types"
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

// partitionerBackends enumerates the disk partitioner backends that
// partition-exercising tests are run against. Mirroring the way Register
// fans tests out across config versions, every test whose config drives
// Ignition's disks stage is registered once per backend so both the sgdisk
// and sfdisk code paths are covered by the existing fixtures.
var partitionerBackends = []string{"sgdisk", "sfdisk"}

func register(tType TestType, t types.Test) {
	if !testExercisesPartitioner(t) {
		Tests[tType] = append(Tests[tType], t)
		return
	}
	// Fan the test out across the partitioner backends. The backend is
	// selected in Ignition via the IGNITION_PARTITIONER environment variable
	// (honored by distro.PartitionerBackend) and recorded in the test name so
	// each variant is individually addressable with -test.run.
	for _, backend := range partitionerBackends {
		variant := types.DeepCopy(t)
		// DeepCopy does not copy Env, so build a fresh slice to avoid the
		// variants aliasing a shared backing array under t.Parallel.
		variant.Env = append(append([]string(nil), t.Env...), "IGNITION_PARTITIONER="+backend)
		variant.Name = t.Name + "/" + backend
		Tests[tType] = append(Tests[tType], variant)
	}
}

// testExercisesPartitioner reports whether running t causes Ignition to invoke
// a partitioner. Ignition only runs a partitioner when the config's
// storage.disks section is non-empty, so tests without disks (files, users,
// systemd units, ...) are registered once and need no backend fan-out.
func testExercisesPartitioner(t types.Test) bool {
	return strings.Contains(t.Config, `"disks"`)
}

// Registers t for every version, inside the same major version,
// that is equal to or greater than the specified ConfigMinVersion.
func Register(tType TestType, t types.Test) {
	// update confgiVersions with new config versions
	configVersions := [][]semver.Version{
		{semver.Version{}}, // place holder 0
		{semver.Version{}}, // place holder 1
		{semver.Version{}}, // place holder 2
		{types30.MaxVersion, types31.MaxVersion, types32.MaxVersion, types33.MaxVersion, types34.MaxVersion, types35.MaxVersion, types36.MaxVersion, types_exp.MaxVersion},
	}

	test := types.DeepCopy(t)
	version, semverErr := semver.NewVersion(test.ConfigMinVersion)
	test.ReplaceAllVersionVars(test.ConfigMinVersion)
	test.ConfigVersion = test.ConfigMinVersion
	register(tType, test) // some tests purposefully don't have config version

	if semverErr == nil && version != nil && t.ConfigMinVersion != "" {
		for _, v := range configVersions[version.Major] {
			if version.LessThan(v) {
				// Check if the test is limited to a max version
				if test.ConfigMaxVersion != "" {
					maximumVersion, maxVersionSemverErr := semver.NewVersion(test.ConfigMinVersion)
					// If a valid max version is given and the next deep copy is greater than the max version then we dont register it
					if maxVersionSemverErr == nil && maximumVersion.LessThan(v) {
						continue
					}
				}

				test = types.DeepCopy(t)
				test.ReplaceAllVersionVars(v.String())
				test.ConfigVersion = v.String()
				register(tType, test)
			}
		}
	}
}
