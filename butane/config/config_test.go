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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/coreos/ignition/v2/butane/config/common"
	ignition "github.com/coreos/ignition/v2/config/v3_5"
	ignitionExp "github.com/coreos/ignition/v2/config/v3_7_experimental"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestOpenShiftVersions(t *testing.T) {
	tests := []struct {
		version         string
		ignitionVersion string
	}{
		{"4.22.0", "3.5.0"},
		{"4.23.0-experimental", "3.7.0-experimental"},
		{"5.0.0", "3.5.0"},
		{"5.1.0-experimental", "3.7.0-experimental"},
	}
	for _, test := range tests {
		for _, raw := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/raw=%v", test.version, raw), func(t *testing.T) {
				input := []byte(fmt.Sprintf(`variant: openshift
version: %s
metadata:
  name: version-test
  labels:
    machineconfiguration.openshift.io/role: worker
storage:
  files:
    - path: /etc/version-test
      mode: 0644
      contents:
        inline: hello
`, test.version))
				out, r, err := TranslateBytes(input, common.TranslateBytesOptions{Raw: raw})
				if !assert.NoError(t, err) || !assert.Empty(t, r.Entries) {
					return
				}
				if !raw {
					var mc struct {
						APIVersion string `yaml:"apiVersion"`
						Kind       string `yaml:"kind"`
						Spec       struct {
							Config map[string]interface{} `yaml:"config"`
						} `yaml:"spec"`
					}
					if !assert.NoError(t, yaml.Unmarshal(out, &mc)) {
						return
					}
					assert.Equal(t, "machineconfiguration.openshift.io/v1", mc.APIVersion)
					assert.Equal(t, "MachineConfig", mc.Kind)
					out, err = json.Marshal(mc.Spec.Config)
					if !assert.NoError(t, err) {
						return
					}
				}
				var cfg struct {
					Ignition struct {
						Version string `json:"version"`
					} `json:"ignition"`
				}
				if !assert.NoError(t, json.Unmarshal(out, &cfg)) {
					return
				}
				assert.Equal(t, test.ignitionVersion, cfg.Ignition.Version)
				if test.ignitionVersion == "3.5.0" {
					// OCP 5.0's MCO uses this parser, not the newest vendored spec.
					parsed, r, err := ignition.ParseCompatibleVersion(out)
					if !assert.NoError(t, err) || !assert.Empty(t, r.Entries) || !assert.Len(t, parsed.Storage.Files, 1) {
						return
					}
					assert.Equal(t, "/etc/version-test", parsed.Storage.Files[0].Path)
				} else {
					parsed, r, err := ignitionExp.ParseCompatibleVersion(out)
					if !assert.NoError(t, err) || !assert.Empty(t, r.Entries) || !assert.Len(t, parsed.Storage.Files, 1) {
						return
					}
					assert.Equal(t, "/etc/version-test", parsed.Storage.Files[0].Path)
				}
			})
		}
	}
}

func TestOpenShiftUnknownVersions(t *testing.T) {
	for _, version := range []string{"4.23.0", "5.0.0-experimental", "5.0.1", "5.1.0", "5.2.0-experimental"} {
		t.Run(version, func(t *testing.T) {
			for _, raw := range []bool{false, true} {
				out, _, err := TranslateBytes([]byte("variant: openshift\nversion: "+version+"\n"), common.TranslateBytesOptions{Raw: raw})
				assert.IsType(t, common.ErrUnknownVersion{}, err)
				assert.Empty(t, out)
			}
		})
	}
}

// Both development lines currently share their feature set and Ignition target.
func TestOpenShiftExperimentalParity(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"minimal", ""},
		{"grub", `grub:
  users:
    - name: root
      password_hash: grub.pbkdf2.sha512.10000.874A958E526409...
`},
		{"openshift fields", `openshift:
  kernel_arguments: [console=ttyS0]
  extensions: [usbguard]
  fips: false
  kernel_type: realtime
`},
		{"unsupported file mode", `storage:
  files:
    - path: /etc/test
      mode: 04755
`},
	}
	for _, test := range tests {
		for _, raw := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/raw=%v", test.name, raw), func(t *testing.T) {
				input := `variant: openshift
version: %s
metadata:
  name: parity-test
  labels:
    machineconfiguration.openshift.io/role: worker
` + test.config
				options := common.TranslateBytesOptions{Raw: raw}
				before, beforeReport, beforeErr := TranslateBytes([]byte(fmt.Sprintf(input, "4.23.0-experimental")), options)
				after, afterReport, afterErr := TranslateBytes([]byte(fmt.Sprintf(input, "5.1.0-experimental")), options)
				assert.Equal(t, before, after)
				assert.Equal(t, beforeReport, afterReport)
				assert.Equal(t, beforeErr, afterErr)
			})
		}
	}
}

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
          start_mib: 17179869184
`)
	out, r, err := TranslateBytes(input, common.TranslateBytesOptions{})
	assert.NoError(t, err)
	assert.False(t, r.IsFatal())

	strOut := string(out)
	assert.Contains(t, strOut, "sizeMiB: 17179869184")
	assert.Contains(t, strOut, "startMiB: 17179869184")
	assert.NotContains(t, strOut, "e+")
}
