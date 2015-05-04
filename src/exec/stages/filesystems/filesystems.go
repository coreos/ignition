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

package filesystems

import (
	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/exec/stages"
	"github.com/coreos/ignition/src/exec/util"
	"github.com/coreos/ignition/src/log"
)

const (
	name = "filesystems"
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger log.Logger, root string) stages.Stage {
	return &stage{
		DestDir: util.DestDir(root),
		logger:  logger,
	}
}

func (creator) Name() string {
	return name
}

type stage struct {
	logger log.Logger
	util.DestDir
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config config.Config) (stages.StageStatus, string) {
	if len(config.Storage.Filesystems) == 0 {
		return stages.StageRunRequest, "prepivot"
	}

	// TODO(vc): inject a systemd unit for running ignition's filesystem_formats stage which depends on device units for all filesystems in config
	// TODO(vc): we must escape the device paths according to systemd-escape --path rules, I've added this escaping to go-systemd.

	// Note that our "success" is where we've scheduled the next stage, just tell the caller and we'll deal with it in the engine.
	return stages.StageScheduled, "filesystem_formats"
}
