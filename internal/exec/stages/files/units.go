// Copyright 2018 CoreOS, Inc.
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

package files

import (
	"github.com/coreos/ignition/internal/config/types"
	"github.com/coreos/ignition/internal/exec/util"
)

// createUnits creates the units listed under systemd.units and networkd.units.
func (s stage) createUnits(config types.Config) error {
	for _, unit := range config.Systemd.Units {
		if err := s.writeSystemdUnit(unit); err != nil {
			return err
		}
		if unit.Enable {
			s.Logger.Warning("the enable field has been deprecated in favor of enabled")
			if err := s.Logger.LogOp(
				func() error { return s.EnableUnit(unit) },
				"enabling unit %q", unit.Name,
			); err != nil {
				return err
			}
		}
		if unit.Enabled != nil {
			if *unit.Enabled {
				if err := s.Logger.LogOp(
					func() error { return s.EnableUnit(unit) },
					"enabling unit %q", unit.Name,
				); err != nil {
					return err
				}
			} else {
				if err := s.Logger.LogOp(
					func() error { return s.DisableUnit(unit) },
					"disabling unit %q", unit.Name,
				); err != nil {
					return err
				}
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
func (s stage) writeSystemdUnit(unit types.Unit) error {
	return s.Logger.LogOp(func() error {
		for _, dropin := range unit.Dropins {
			if dropin.Contents == "" {
				continue
			}

			f, err := util.FileFromSystemdUnitDropin(unit, dropin)
			if err != nil {
				s.Logger.Crit("error converting systemd dropin: %v", err)
				return err
			}
			if err := s.Logger.LogOp(
				func() error { return s.PerformFetch(f) },
				"writing systemd drop-in %q at %q", dropin.Name, f.Path,
			); err != nil {
				return err
			}
		}

		if unit.Contents == "" {
			return nil
		}

		f, err := util.FileFromSystemdUnit(unit)
		if err != nil {
			s.Logger.Crit("error converting unit: %v", err)
			return err
		}
		if err := s.Logger.LogOp(
			func() error { return s.PerformFetch(f) },
			"writing unit %q at %q", unit.Name, f.Path,
		); err != nil {
			return err
		}

		return nil
	}, "processing unit %q", unit.Name)
}

// writeNetworkdUnit creates the specified unit and any dropins for that unit.
// If the contents of the unit or are empty, the unit is not created. The same
// applies to the unit's dropins.
func (s stage) writeNetworkdUnit(unit types.Networkdunit) error {
	return s.Logger.LogOp(func() error {
		for _, dropin := range unit.Dropins {
			if dropin.Contents == "" {
				continue
			}

			f, err := util.FileFromNetworkdUnitDropin(unit, dropin)
			if err != nil {
				s.Logger.Crit("error converting networkd dropin: %v", err)
				return err
			}
			if err := s.Logger.LogOp(
				func() error { return s.PerformFetch(f) },
				"writing networkd drop-in %q at %q", dropin.Name, f.Path,
			); err != nil {
				return err
			}
		}
		if unit.Contents == "" {
			return nil
		}

		f, err := util.FileFromNetworkdUnit(unit)
		if err != nil {
			s.Logger.Crit("error converting unit: %v", err)
			return err
		}
		if err := s.Logger.LogOp(
			func() error { return s.PerformFetch(f) },
			"writing unit %q at %q", unit.Name, f.Path,
		); err != nil {
			return err
		}

		return nil
	}, "processing unit %q", unit.Name)
}
