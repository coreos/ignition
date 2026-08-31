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

	"github.com/clarketm/json"
)

// unmarshalJSONForYAML decodes JSON into a generic structure suitable for
// YAML encoding. Integers are preserved as int64 so yaml.v3 does not emit
// scientific notation for values >= 1e6 (e.g. sizeMiB: 8.389e+06).
func unmarshalJSONForYAML(jsonCfg []byte) (interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(jsonCfg))
	dec.UseNumber()
	var ifaceCfg interface{}
	if err := dec.Decode(&ifaceCfg); err != nil {
		return nil, err
	}
	return convertJSONNumbers(ifaceCfg), nil
}

// convertJSONNumbers walks a decoded JSON tree and replaces json.Number
// values with int64 when possible, otherwise float64. json.Unmarshal into
// interface{} otherwise uses float64, and yaml.v3 then formats values >= 1e6
// with strconv.FormatFloat 'g' as scientific notation.
func convertJSONNumbers(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		for k, val := range x {
			x[k] = convertJSONNumbers(val)
		}
		return x
	case []interface{}:
		for i, val := range x {
			x[i] = convertJSONNumbers(val)
		}
		return x
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		f, err := x.Float64()
		if err != nil {
			return x.String()
		}
		return f
	default:
		return v
	}
}
