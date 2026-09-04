// Copyright 2020 Red Hat, Inc
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

package config

import (
	"testing"

	"github.com/coreos/ignition/v2/butane/config/common"

	"github.com/stretchr/testify/assert"
)

// Ensures that large sizeMiB values are rendered as normal integers in
// the generated MachineConfig YAML, rather than being formatted with
// scientific notation (e.g. "1e+06" instead of "1000000").
// See: https://github.com/coreos/ignition/issues/2309
func TestOpenShiftSizeMiBFormatting(t *testing.T) {
	input := []byte(`
variant: openshift
version: 4.19.0
metadata:
  name: test
  labels:
    machineconfiguration.openshift.io/role: worker
storage:
  disks:
    - device: /dev/sda
      partitions:
        - number: 1
          size_mib: 17179869184
`)
	out, r, err := TranslateBytes(input, common.TranslateBytesOptions{})
	assert.NoError(t, err)
	assert.False(t, r.IsFatal())

	strOut := string(out)
	assert.Contains(t, strOut, "sizeMiB: 17179869184")
	assert.NotContains(t, strOut, "e+")
}
