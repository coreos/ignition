// Copyright 2022 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseVariantVersion extracts the variant and version from Butane config bytes.
//
// This function only parses the minimal metadata needed to identify which
// translator to use. It does not validate the full config structure.
//
// Returns an error if the variant or version fields are missing or invalid.
func ParseVariantVersion(input []byte) (variant, version string, err error) {
	var cf commonFields
	if err := yaml.Unmarshal(input, &cf); err != nil {
		return "", "", fmt.Errorf("failed to parse config: %w", err)
	}

	if cf.Variant == "" {
		return "", "", fmt.Errorf("missing 'variant' field in config")
	}

	return cf.Variant, cf.Version.String(), nil
}
