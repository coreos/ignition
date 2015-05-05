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

package disk_partitions

import (
	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/exec/stages"
	"github.com/coreos/ignition/src/exec/util"
	"github.com/coreos/ignition/src/log"
)

const (
	name = "disk_partitions"
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

// The disk_partitions stage creates all partitions for the disks in the config.
// We expect all the disk devices to be plugged/ready, the "disks" stage ensures this for us.
func (s stage) Run(config config.Config) (stages.StageStatus, string) {

	if err := s.createPartitions(config); err != nil {
		return stages.StageFailed, ""
	}
	return stages.StageRunRequest, "raid"
}

func (s stage) createPartitions(config config.Config) error {
	if len(config.Storage.Filesystems) == 0 {
		return nil
	}

	// TODO(vc): create the partitions for all disks in config.Storage.Disks
	// we assume the devices are available now, the normal use will schedule this stage predicated on the devices plugging via systemd.
	return nil
}
