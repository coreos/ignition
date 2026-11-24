// Copyright 2026 CoreOS, Inc.
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

package types

import (
	"os"
	"strings"

	"github.com/coreos/ignition/v2/tests/fixtures"
)

// ReadFixture returns the embedded fixture named name (from tests/fixtures)
// with the @VERSION@ placeholder replaced by version. The fixture is baked
// into the test binary at build time, so it works even when the binary runs
// without the source repository present. Pass "$version" to defer substitution
// to the test framework's per-version registration.
func ReadFixture(name, version string) string {
	data, err := fixtures.FS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return strings.ReplaceAll(string(data), "@VERSION@", version)
}

// WriteVersionedFixture writes ReadFixture(name, version) to a temporary host
// file and returns its absolute path. This is useful for configs that Ignition
// fetches directly (e.g. via a file:// URL), which must contain a concrete spec
// version and a real path. The file persists for the lifetime of the test
// process.
func WriteVersionedFixture(name, version string) string {
	f, err := os.CreateTemp("", "ignition-"+name)
	if err != nil {
		panic(err)
	}
	if _, err := f.WriteString(ReadFixture(name, version)); err != nil {
		_ = f.Close()
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	return f.Name()
}
