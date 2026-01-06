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
	// Register only the specified minimum config version (no expansion across versions)
	test := types.DeepCopy(t)
	test.ReplaceAllVersionVars(test.ConfigMinVersion)
	test.ConfigVersion = test.ConfigMinVersion
	register(tType, test)
}
