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
// limitations under the License.

package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestUnmarshalJSONForYAMLIntegers(t *testing.T) {
	jsonCfg := []byte(`{"sizeMiB":8389000,"startMiB":2048,"values":[1000000],"ratio":1.5}`)
	v, err := unmarshalJSONForYAML(jsonCfg)
	assert.NoError(t, err)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	assert.NoError(t, enc.Encode(v))
	assert.NoError(t, enc.Close())
	out := buf.String()

	assert.Contains(t, out, "sizeMiB: 8389000")
	assert.Contains(t, out, "startMiB: 2048")
	assert.Contains(t, out, "- 1000000")
	assert.Contains(t, out, "ratio: 1.5")
	assert.False(t, strings.Contains(out, "e+"), "YAML must not use scientific notation: %s", out)
}
