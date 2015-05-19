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
	"fmt"

	"github.com/coreos/ignition/Godeps/_workspace/src/github.com/alecthomas/units"
)

type Partition struct {
	Label  PartitionLabel     `json:"label,omitempty" yaml:"label"`
	Number int                `json:"number"          yaml:"number"`
	Size   PartitionDimension `json:"size"            yaml:"size"`
	Start  PartitionDimension `json:"start"           yaml:"start"`
}

type PartitionLabel string

func (n *PartitionLabel) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return n.unmarshal(unmarshal)
}

func (n *PartitionLabel) UnmarshalJSON(data []byte) error {
	return n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
}

type partitionLabel PartitionLabel

func (n *PartitionLabel) unmarshal(unmarshal func(interface{}) error) error {
	tn := partitionLabel(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = PartitionLabel(tn)
	return n.assertValid()
}

func (n PartitionLabel) assertValid() error {
	return nil
}

type PartitionDimension uint64

func (n *PartitionDimension) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// In YAML we allow human-readable dimensions like GiB/TiB etc.
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	b2b, err := units.ParseBase2Bytes(str) // TODO(vc): replace the units package
	if err != nil {
		return err
	}
	if b2b < 0 {
		return fmt.Errorf("negative value inappropriate: %q", str)
	}

	// Translate bytes into sectors
	sectors := (b2b / 512)
	if b2b%512 != 0 {
		sectors++
	}
	*n = PartitionDimension(uint64(sectors))
	return nil
}

func (n *PartitionDimension) UnmarshalJSON(data []byte) error {
	// In JSON we expect plain integral sectors.
	// The YAML->JSON conversion is responsible for normalizing human units to sectors.
	var pd uint64
	if err := json.Unmarshal(data, &pd); err != nil {
		return err
	}
	*n = PartitionDimension(pd)
	return nil
}
