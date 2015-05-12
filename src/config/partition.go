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

	//	"github.com/coreos/ignition/Godeps/_workspace/src/github.com/GoogleCloudPlatform/kubernetes/pkg/api/resource"
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
	// in YAML we may be flexible allowing human-readable dimensions like GiB/TiB etc.
	// TODO(vc)
	return n.unmarshal(unmarshal)
}

func (n *PartitionDimension) UnmarshalJSON(data []byte) error {
	// in JSON we only want sectors.
	// TODO(vc)
	return n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
}

type partitionDimension PartitionDimension

func (n *PartitionDimension) unmarshal(unmarshal func(interface{}) error) error {
	tn := partitionDimension(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = PartitionDimension(tn)
	return n.assertValid()
}

func (n PartitionDimension) assertValid() error {
	return nil
}
