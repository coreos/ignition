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
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coreos/ignition/v2/config/shared/errors"
	cutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/systemd"
)

// Preset holds the information about
// a given systemd unit.
type Preset struct {
	unit           string
	enabled        bool
	instantiatable bool
	instances      []string
	scope          util.UnitScope
}

// warnOnOldSystemdVersion checks the version of Systemd
// in a given system and prints a warning if older than 240.
func (s *stage) warnOnOldSystemdVersion() error {
	systemdVersion, err := systemd.GetSystemdVersion(context.Background())
	if err != nil {
		return err
	}
	if systemdVersion < 240 {
		s.Logger.Warning("The version of systemd (%q) is less than 240. Enabling/disabling instantiated units may not work. See https://github.com/coreos/ignition/issues/586 for more information.", systemdVersion)
	}
	return nil
}

// createUnits creates the units listed under systemd.units.
func (s *stage) createUnits(config types.Config) error {
	presets := make(map[string]*Preset)
	for _, unit := range config.Systemd.Units {
		if err := s.writeSystemdUnit(unit); err != nil {
			return err
		}
		if unit.Enabled != nil {
			// identifier keyword is used to distinguish systemd units
			// which are either enabled or disabled. Appending
			// it to a unitName will avoid overwriting the existing
			// unitName's instance if the state of the unit is different.
			identifier := "disabled"
			if *unit.Enabled {
				identifier = "enabled"
			}
			if strings.Contains(unit.Name, "@") {
				unitName, instance, err := parseInstanceUnit(unit)
				if err != nil {
					return err
				}
				key := fmt.Sprintf("%s.%s-%s", util.GetUnitScope(unit), unitName, identifier)
				if _, ok := presets[key]; ok {
					presets[key].instances = append(presets[key].instances, instance)
				} else {
					presets[key] = &Preset{unitName, *unit.Enabled, true, []string{instance}, util.GetUnitScope(unit)}
				}
			} else {
				key := fmt.Sprintf("%s-%s", unit.Key(), identifier)
				if _, ok := presets[key]; !ok {
					presets[key] = &Preset{unit.Name, *unit.Enabled, false, []string{}, util.GetUnitScope(unit)}
				} else {
					return fmt.Errorf("%q key is already present in the presets map", key)
				}
			}
		}
		if unit.Mask != nil {
			if *unit.Mask { // mask: true
				if err := s.Logger.LogOp(
					func() error {
						var err error = s.MaskUnit(unit)
						return err
					},
					"masking unit %q for scope %q", unit.Name, string(util.GetUnitScope(unit)),
				); err != nil {
					return err
				}

			} else { // mask: false
				masked, err := s.IsUnitMasked(unit)
				if err != nil {
					return err
				}
				if masked {
					if err := s.Logger.LogOp(
						func() error {
							return s.UnmaskUnit(unit)
						},
						"unmasking unit %q for scope %q", unit.Name, string(util.GetUnitScope(unit)),
					); err != nil {
						return err
					}
				}
			}
		}
	}
	// if we have presets then create the systemd preset file.
	if len(presets) != 0 {
		if err := s.createSystemdPresetFiles(presets); err != nil {
			return err
		}
	}

	return nil
}

// parseInstanceUnit extracts the name and a corresponding instance
// for a given instantiated unit.
// e.g: echo@bar.service ==> unitName=echo@.service & instance=bar
func parseInstanceUnit(unit types.Unit) (string, string, error) {
	at := strings.Index(unit.Name, "@")
	if at == -1 {
		return "", "", errors.ErrInvalidInstantiatedUnit
	}
	dot := strings.LastIndex(unit.Name, ".")
	if dot == -1 {
		return "", "", errors.ErrNoSystemdExt
	}
	instance := unit.Name[at+1 : dot]
	serviceInstance := unit.Name[0:at+1] + unit.Name[dot:len(unit.Name)]
	return serviceInstance, instance, nil
}

// createSystemdPresetFile creates the presetfile for enabled/disabled
// systemd units.
func (s *stage) createSystemdPresetFiles(presets map[string]*Preset) error {
	hasInstanceUnit := false
	// sort the units before writing to the systemd presets file to ensure
	// the file is written in a consistent order across multiple runs
	unitNames := make([]string, 0, len(presets))
	for unit := range presets {
		unitNames = append(unitNames, unit)
	}
	sort.Strings(unitNames)

	for _, name := range unitNames {
		value := presets[name]
		unitString := value.unit
		if err := s.relabelPath(filepath.Join(s.DestDir, s.SystemdPresetPath(value.scope))); err != nil {
			return err
		}
		if value.instantiatable {
			hasInstanceUnit = true
			// Let's say we have two instantiated enabled units listed under
			// the systemd units i.e. echo@foo.service, echo@bar.service
			// then the unitString will look like "echo@.service foo bar"
			unitString = fmt.Sprintf("%s %s", unitString, strings.Join(value.instances, " "))
		}
		if value.enabled {
			if err := s.Logger.LogOp(
				func() error { return s.EnableUnit(unitString, value.scope) },
				"setting %q preset to enabled for %q", value.scope, unitString,
			); err != nil {
				return err
			}
		} else {
			if err := s.Logger.LogOp(
				func() error { return s.DisableUnit(unitString, value.scope) },
				"setting %q preset to disabled for %q", value.scope, unitString,
			); err != nil {
				return err
			}
		}
	}
	// Print the warning if there's an instantiated unit present under
	// the systemd units and the version of systemd in a given system
	// is older than 240.
	if hasInstanceUnit {
		if err := s.warnOnOldSystemdVersion(); err != nil {
			return err
		}
	}
	return nil
}

// writeSystemdUnit creates the specified unit and any dropins for that unit.
// If the contents of the unit or are empty, the unit is not created. The same
// applies to the unit's dropins.
func (s *stage) writeSystemdUnit(unit types.Unit) error {
	return s.Logger.LogOp(func() error {
		for _, dropin := range unit.Dropins {
			if dropin.Contents == nil {
				continue
			}
			fetchops, err := s.FilesFromSystemdUnitDropin(unit, dropin)
			if err != nil {
				s.Logger.Crit("error converting systemd dropin: %v", err)
				return err
			}
			for _, f := range fetchops {
				relabeledDropinDir := false
				// trim off prefix since this needs to be relative to the sysroot
				if !strings.HasPrefix(f.Node.Path, s.DestDir) {
					panic(fmt.Sprintf("Dropin path %s isn't under prefix %s", f.Node.Path, s.DestDir))
				}
				relabelPath := f.Node.Path[len(s.DestDir):]
				if err := s.Logger.LogOp(
					func() error { return s.PerformFetch(f) },
					"writing systemd drop-in %q at %q", dropin.Name, f.Node.Path,
				); err != nil {
					return err
				}
				if !relabeledDropinDir {
					s.relabel(filepath.Dir(relabelPath))
					relabeledDropinDir = true
				}
			}
		}

		if cutil.NilOrEmpty(unit.Contents) {
			return nil
		}

		fetchops, err := s.FilesFromSystemdUnit(unit)
		if err != nil {
			s.Logger.Crit("error converting unit: %v", err)
			return err
		}

		for _, f := range fetchops {
			// trim off prefix since this needs to be relative to the sysroot
			if !strings.HasPrefix(f.Node.Path, s.DestDir) {
				panic(fmt.Sprintf("Unit path %s isn't under prefix %s", f.Node.Path, s.DestDir))
			}
			relabelPath := f.Node.Path[len(s.DestDir):]
			if err := s.Logger.LogOp(
				func() error { return s.PerformFetch(f) },
				"writing unit %q at %q", unit.Name, f.Node.Path,
			); err != nil {
				return err
			}

			s.relabel(relabelPath)
		}

		return nil
	}, "processing unit %q for scope %q", unit.Name, string(util.GetUnitScope(unit)))
}
