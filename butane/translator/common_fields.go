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

	"github.com/coreos/go-semver/semver"
)

type commonFields struct {
	Variant string         `yaml:"variant"`
	Version semver.Version `yaml:"version"`
}

func (c *commonFields) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain commonFields
	var raw plain

	if err := unmarshal(&raw); err != nil {
		return err
	}

	if raw.Variant == "" {
		return fmt.Errorf("variant cannot be empty")
	}

	*c = commonFields(raw)
	return nil
}

func (c *commonFields) asKey() string {
	return fmt.Sprintf("%s+%s", c.Variant, c.Version.String())
}

func newCF(variant, version string) (commonFields, error) {
	if variant == "" {
		return commonFields{}, fmt.Errorf("variant cannot be empty")
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return commonFields{}, fmt.Errorf("invalid version: %w", err)
	}

	return commonFields{
		Variant: variant,
		Version: *v,
	}, nil
}
