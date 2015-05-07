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
	return &stage{
		DestDir: util.DestDir(root),
		logger:  logger,
	}
}

func (creator) Name() string {
	return name
}

type stage struct {
	logger *log.Logger
	util.DestDir
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config config.Config) bool {
	for _, unit := range config.Systemd.Units {
		if !s.writeSystemdUnit(unit) {
			return false
		}
		if unit.Enable {
			err := s.logger.LogOp(func() error { return s.EnableUnit(unit) }, "enabling unit %q", unit.Name)
			if err != nil {
				return false
			}
		}
		if unit.Mask {
			err := s.logger.LogOp(func() error { return s.MaskUnit(unit) }, "masking unit %q", unit.Name)
			if err != nil {
				return false
			}
		}
	}
	for _, unit := range config.Networkd.Units {
		if !s.writeNetworkdUnit(unit) {
			return false
		}
	}
	return true
}

func (s stage) writeSystemdUnit(unit config.Unit) bool {
	return s.writeUnit(unit, util.FileFromSystemdUnit)
}

func (s stage) writeNetworkdUnit(unit config.Unit) bool {
	return s.writeUnit(unit, util.FileFromNetworkdUnit)
}

func (s stage) writeUnit(unit config.Unit, fileFromUnit func(unit config.Unit) *config.File) bool {
	err := s.logger.LogOp(func() error {
		for _, dropin := range unit.DropIns {
			if dropin.Contents == "" {
				continue
			}

			f := util.FileFromUnitDropin(unit, dropin)
			err := s.logger.LogOp(func() error { return s.WriteFile(f) }, "writing dropin %q at %q", dropin.Name, f.Path)
			if err != nil {
				return err
			}
		}

		if unit.Contents == "" {
			return nil
		}

		f := fileFromUnit(unit)
		err := s.logger.LogOp(func() error { return s.WriteFile(f) }, "writing unit %q at %q", unit.Name, f.Path)
		if err != nil {
			return err
		}

		return nil
	}, "writing unit %q", unit.Name)

	return err == nil
}
