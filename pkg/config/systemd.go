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
	"errors"
	"strings"
)

type Systemd struct {
	Units []Unit
}

type Unit struct {
	Name     ServiceName
	Enable   bool
	Mask     bool
	Contents string
	DropIns  []UnitDropIn
}

type UnitDropIn struct {
	Name     UnitName
	Contents string
}

type UnitName string

func (n *UnitName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	*n = UnitName(name)

	return nil
}

type ServiceName string

func (n *ServiceName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name UnitName
	if err := unmarshal(&name); err != nil {
		return err
	}
	*n = ServiceName(name)

	if strings.HasSuffix(string(name), ".service") {
		return nil
	}
	return errors.New("invalid systemd service name")
}
