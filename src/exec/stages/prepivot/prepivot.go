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

package prepivot

import (
	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/exec/stages"
	"github.com/coreos/ignition/src/exec/util"
	"github.com/coreos/ignition/src/log"
)

const (
	name = "prepivot"
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger *log.Logger, root string) stages.Stage {
	return &stage{util.Util{
		DestDir: root,
		Logger:  logger,
	}}
}

func (creator) Name() string {
	return name
}

type stage struct {
	util.Util
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config config.Config) bool {
	if err := s.createUnits(config); err != nil {
		s.Logger.Crit("failed to create units: %v", err)
		return false
	}
	return true
}

// createUnits creates the units listed under systemd.units and networkd.units.
func (s stage) createUnits(config config.Config) error {
	for _, unit := range config.Systemd.Units {
		if err := s.writeSystemdUnit(unit); err != nil {
			return err
		}
		if unit.Enable {
			if err := s.Logger.LogOp(
				func() error { return s.EnableUnit(unit) },
				"enabling unit %q", unit.Name,
			); err != nil {
				return err
			}
		}
		if unit.Mask {
			if err := s.Logger.LogOp(
				func() error { return s.MaskUnit(unit) },
				"masking unit %q", unit.Name,
			); err != nil {
				return err
			}
		}
	}
	for _, unit := range config.Networkd.Units {
		if err := s.writeNetworkdUnit(unit); err != nil {
			return err
		}
	}
	return nil
}

// writeSystemdUnit creates the specified unit and any dropins for that unit.
// If the contents of the unit or are empty, the unit is not created. The same
// applies to the unit's dropins.
func (s stage) writeSystemdUnit(unit config.SystemdUnit) error {
	return s.Logger.LogOp(func() error {
		for _, dropin := range unit.DropIns {
			if dropin.Contents == "" {
				continue
			}

			f := util.FileFromUnitDropin(unit, dropin)
			if err := s.Logger.LogOp(
				func() error { return s.WriteFile(f) },
				"writing dropin %q at %q", dropin.Name, f.Path,
			); err != nil {
				return err
			}
		}

		if unit.Contents == "" {
			return nil
		}

		f := util.FileFromSystemdUnit(unit)
		if err := s.Logger.LogOp(
			func() error { return s.WriteFile(f) },
			"writing unit %q at %q", unit.Name, f.Path,
		); err != nil {
			return err
		}

		return nil
	}, "writing unit %q", unit.Name)
}

// writeNetworkdUnit creates the specified unit. If the contents of the unit or
// are empty, the unit is not created.
func (s stage) writeNetworkdUnit(unit config.NetworkdUnit) error {
	return s.Logger.LogOp(func() error {
		if unit.Contents == "" {
			return nil
		}

		f := util.FileFromNetworkdUnit(unit)
		if err := s.Logger.LogOp(
			func() error { return s.WriteFile(f) },
			"writing unit %q at %q", unit.Name, f.Path,
		); err != nil {
			return err
		}

		return nil
	}, "writing unit %q", unit.Name)
}
