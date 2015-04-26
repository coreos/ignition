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
	"fmt"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/exec/stages"
	"github.com/coreos/ignition/exec/util"
	"github.com/coreos/ignition/log"
)

const (
	name = "prepivot"
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger log.Logger, root string) stages.Stage {
	return &stage{
		root:   root,
		logger: logger,
	}
}

func (creator) Name() string {
	return name
}

type stage struct {
	logger log.Logger
	root   string
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config config.Config) bool {
	for _, unit := range config.Systemd.Units {
		if !s.writeUnit(unit) {
			return false
		}
	}
	for _, unit := range config.Networkd.Units {
		if !s.writeUnit(unit) {
			return false
		}
	}
	return true
}

func (s stage) writeUnit(unit config.Unit) bool {
	s.logger.Info(fmt.Sprintf("writing unit %q", unit.Name))
	defer s.logger.Info(fmt.Sprintf("done writing unit %q", unit.Name))

	for _, dropin := range unit.DropIns {
		if dropin.Contents == "" {
			continue
		}

		f := util.FileFromUnitDropin(s.root, unit, dropin)
		s.logger.Info(fmt.Sprintf("writing dropin %q at %q", dropin.Name, f.Path))
		if err := util.WriteFile(f); err != nil {
			s.logger.Err(fmt.Sprintf("failed to write dropin %q: %v", dropin.Name, err))
			return false
		}
	}

	if unit.Contents == "" {
		return true
	}

	f := util.FileFromUnit(s.root, unit)
	s.logger.Info(fmt.Sprintf("writing unit %q at %q", unit.Name, f.Path))
	if err := util.WriteFile(f); err != nil {
		s.logger.Err(fmt.Sprintf("failed to write unit %q: %v", unit.Name, err))
		return false
	}

	return true
}
