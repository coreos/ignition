// Copyright 2015 CoreOS, Inc.
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
)

type Unit struct {
	Name     UnitName     `json:"name,omitempty"     yaml:"name"`
	Enable   bool         `json:"enable,omitempty"   yaml:"enable"`
	Mask     bool         `json:"mask,omitempty"     yaml:"mask"`
	Contents string       `json:"contents,omitempty" yaml:"contents"`
	DropIns  []UnitDropIn `json:"dropins,omitempty"  yaml:"dropins"`
}

type UnitDropIn struct {
	Name     UnitName `json:"name,omitempty"     yaml:"name"`
	Contents string   `json:"contents,omitempty" yaml:"contents"`
}

type UnitName string

func (n *UnitName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return n.unmarshal(unmarshal)
}

func (n *UnitName) UnmarshalJSON(data []byte) error {
	return n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
}

type unitName UnitName

func (n *UnitName) unmarshal(unmarshal func(interface{}) error) error {
	tn := unitName(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = UnitName(tn)
	return n.assertValid()
}

func (n UnitName) assertValid() error {
	return nil
}
