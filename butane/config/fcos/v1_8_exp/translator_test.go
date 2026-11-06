// Copyright 2026 Red Hat, Inc
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
// limitations under the License.)

package v1_8_exp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coreos/ignition/v2/butane/config/common"
	"github.com/coreos/ignition/v2/butane/translator"

	"github.com/stretchr/testify/assert"
)

const configHeader = `variant: fcos
version: 1.8.0-experimental
`

func TestRegistryTranslationParity(t *testing.T) {
	filesDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(filesDir, "contents"), []byte("local contents"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		input   string
		options common.TranslateBytesOptions
	}{
		{
			name:  "minimal",
			input: configHeader,
		},
		{
			name:  "pretty",
			input: configHeader,
			options: common.TranslateBytesOptions{
				Pretty: true,
			},
		},
		{
			name: "unused key",
			input: configHeader + `storage:
  files:
    - path: /etc/example
      unused: true
`,
		},
		{
			name: "source validation",
			input: configHeader + `storage:
  files:
    - path: /etc/example
      contents:
        source: https://example.com
        inline: example
`,
		},
		{
			name: "source warning",
			input: configHeader + `storage:
  files:
    - path: /etc/example
      mode: 420
`,
		},
		{
			name: "translation warning",
			input: configHeader + `storage:
  disks:
    - device: /dev/vda
      partitions:
        - label: root
          number: 5
          size_mib: 8192
`,
		},
		{
			name: "local file",
			input: configHeader + `storage:
  files:
    - path: /etc/example
      contents:
        local: contents
`,
			options: common.TranslateBytesOptions{
				TranslateOptions: common.TranslateOptions{
					FilesDir: filesDir,
				},
			},
		},
		{
			name: "missing files dir",
			input: configHeader + `storage:
  files:
    - path: /etc/example
      contents:
        local: contents
`,
		},
		{
			name: "disabled auto compression",
			input: configHeader + `storage:
  files:
    - path: /etc/example
      contents:
        inline: ` + strings.Repeat("z", 2048) + "\n",
			options: common.TranslateBytesOptions{
				TranslateOptions: common.TranslateOptions{
					NoResourceAutoCompression: true,
				},
			},
		},
		{
			name: "generated validation",
			input: configHeader + `storage:
  files:
    - path: relative
`,
		},
		{
			name: "duplicate generated keys",
			input: configHeader + `storage:
  files:
    - path: /etc/example
    - path: /etc/example
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expectedOutput, expectedReport, expectedErr := ToIgn3_7Bytes([]byte(test.input), test.options)
			result, err := translator.Global.Translate(context.Background(), []byte(test.input), translator.Options{
				FilesDir:                  test.options.FilesDir,
				NoResourceAutoCompression: test.options.NoResourceAutoCompression,
				DebugPrintTranslations:    test.options.DebugPrintTranslations,
				Pretty:                    test.options.Pretty,
				Raw:                       test.options.Raw,
			})

			assert.Equal(t, expectedOutput, result.Output)
			assert.Equal(t, expectedReport, result.Report)
			if expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, expectedErr.Error())
			}
		})
	}
}

func TestSpecTranslatorMetadata(t *testing.T) {
	registered, err := translator.Global.Get("fcos", "1.8.0-experimental")
	if err != nil {
		t.Fatal(err)
	}

	metadata := registered.Metadata()
	assert.Equal(t, "fcos", metadata.Variant)
	assert.Equal(t, "1.8.0-experimental", metadata.Version.String())
	assert.Equal(t, "3.7.0-experimental", metadata.IgnitionVersion.String())
	assert.Equal(t, "Fedora CoreOS", metadata.Description)
	assert.True(t, metadata.Experimental)
}

func TestSpecTranslatorRejectsUnexpectedType(t *testing.T) {
	implementation := specTranslator{}
	_, err := implementation.Validate(Config{})
	assert.Error(t, err)
	_, _, err = implementation.Translate(Config{}, translator.Options{})
	assert.Error(t, err)
}
