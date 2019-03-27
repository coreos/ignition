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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/coreos/ignition/config/v3_0/types"
	"github.com/coreos/ignition/internal/distro"
	"github.com/coreos/ignition/internal/exec/util"
)

// createUnits creates the units listed under systemd.units.
func (s *stage) createUnits(config types.Config) error {
	enabledOneUnit := false
	for _, unit := range config.Systemd.Units {
		if err := s.writeSystemdUnit(unit, false); err != nil {
			return err
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
			enabledOneUnit = true
		}
		if unit.Mask != nil && *unit.Mask {
			if err := s.Logger.LogOp(
				func() error { return s.MaskUnit(unit) },
				"masking unit %q", unit.Name,
			); err != nil {
				return err
			}
		}
	}
	// and relabel the preset file itself if we enabled/disabled something
	if enabledOneUnit {
		s.relabel(util.PresetPath)
	}
	return nil
}

// writeSystemdUnit creates the specified unit and any dropins for that unit.
// If the contents of the unit or are empty, the unit is not created. The same
// applies to the unit's dropins.
func (s *stage) writeSystemdUnit(unit types.Unit, runtime bool) error {
	// use a different DestDir if it's runtime so it affects our /run (but not
	// if we're running locally through blackbox tests)
	u := s.Util
	if runtime && !distro.BlackboxTesting() {
		u.DestDir = "/"
	}

	return s.Logger.LogOp(func() error {
		relabeledDropinDir := false
		for _, dropin := range unit.Dropins {
			if dropin.Contents == nil || *dropin.Contents == "" {
				continue
			}
			f, err := u.FileFromSystemdUnitDropin(unit, dropin, runtime)
			if err != nil {
				s.Logger.Crit("error converting systemd dropin: %v", err)
				return err
			}
			relabelPath := f.Node.Path
			if !runtime {
				// trim off prefix since this needs to be relative to the sysroot
				if !strings.HasPrefix(f.Node.Path, s.DestDir) {
					panic(fmt.Sprintf("Dropin path %s isn't under prefix %s", f.Node.Path, s.DestDir))
				}
				relabelPath = f.Node.Path[len(s.DestDir):]
			}
			if err := s.Logger.LogOp(
				func() error { return u.PerformFetch(f) },
				"writing systemd drop-in %q at %q", dropin.Name, f.Node.Path,
			); err != nil {
				return err
			}
			if !relabeledDropinDir {
				s.relabel(filepath.Dir(relabelPath))
				relabeledDropinDir = true
			}
		}

		if unit.Contents == nil || *unit.Contents == "" {
			return nil
		}

		f, err := u.FileFromSystemdUnit(unit, runtime)
		if err != nil {
			s.Logger.Crit("error converting unit: %v", err)
			return err
		}
		relabelPath := f.Node.Path
		if !runtime {
			// trim off prefix since this needs to be relative to the sysroot
			if !strings.HasPrefix(f.Node.Path, s.DestDir) {
				panic(fmt.Sprintf("Unit path %s isn't under prefix %s", f.Node.Path, s.DestDir))
			}
			relabelPath = f.Node.Path[len(s.DestDir):]
		}
		if err := s.Logger.LogOp(
			func() error { return u.PerformFetch(f) },
			"writing unit %q at %q", unit.Name, f.Node.Path,
		); err != nil {
			return err
		}
		s.relabel(relabelPath)

		return nil
	}, "processing unit %q", unit.Name)
}
