// Copyright 2026 Red Hat, Inc.
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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenShiftDocs(t *testing.T) {
	dir := t.TempDir()
	if !assert.NoError(t, generate(dir)) {
		return
	}
	tests := []struct {
		file            string
		version         string
		ignitionVersion string
		experimental    bool
	}{
		{"config-openshift-v4_22.md", "4.22.0", "3.5.0", false},
		{"config-openshift-v4_23-exp.md", "4.23.0-experimental", "3.7.0-experimental", true},
		{"config-openshift-v5_0.md", "5.0.0", "3.5.0", false},
		{"config-openshift-v5_1-exp.md", "5.1.0-experimental", "3.7.0-experimental", true},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, test.file))
			if !assert.NoError(t, err) {
				return
			}
			doc := string(data)
			assert.Contains(t, doc, "# OpenShift Specification v"+test.version)
			assert.Contains(t, doc, "generates Ignition configs with version `"+test.ignitionVersion+"`")
			assert.Contains(t, doc, "Setuid/setgid/sticky bits are not supported.")
			if test.experimental {
				assert.Contains(t, doc, "**_grub_** (object): describes the desired GRUB bootloader configuration.")
				assert.Contains(t, doc, "**_file_mode_** (integer): Custom permissions to apply to files")
				assert.Contains(t, doc, "Ownership, file modes (using `file_mode`)")
			} else {
				assert.Contains(t, doc, "**_grub_** (object): Unsupported")
				assert.Contains(t, doc, "Ownership is not preserved.")
				assert.NotContains(t, doc, "`file_mode`")
				assert.NotContains(t, doc, "**_file_mode_**")
				assert.NotContains(t, doc, "**_dir_mode_**")
				assert.NotContains(t, doc, "**_quadlets_**")
			}
			// Regenerating must not delete the retained experimental spec or
			// change the newly generated stable documentation.
			if !assert.NoError(t, generate(dir)) {
				return
			}
			again, err := os.ReadFile(filepath.Join(dir, test.file))
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, data, again)
		})
	}
}
