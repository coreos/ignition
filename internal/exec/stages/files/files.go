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

package files

import (
	"errors"

	"github.com/coreos/ignition/internal/config/types"
	"github.com/coreos/ignition/internal/exec/stages"
	"github.com/coreos/ignition/internal/exec/util"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/resource"
)

const (
	name = "files"
)

var (
	ErrFilesystemUndefined = errors.New("the referenced filesystem was not defined")
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger *log.Logger, root string, f resource.Fetcher) stages.Stage {
	return &stage{
		Util: util.Util{
			DestDir: root,
			Logger:  logger,
			Fetcher: f,
		},
	}
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

func (s stage) Run(config types.Config) bool {
	if err := s.createPasswd(config); err != nil {
		s.Logger.Crit("failed to create users/groups: %v", err)
		return false
	}

	if err := s.createFilesystemsEntries(config); err != nil {
		s.Logger.Crit("failed to create files: %v", err)
		return false
	}

	if err := s.createUnits(config); err != nil {
		s.Logger.Crit("failed to create units: %v", err)
		return false
	}

	return true
}
