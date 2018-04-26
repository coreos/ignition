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

// The storage stage is responsible for partitioning disks, creating RAID
// arrays, formatting partitions, writing files, writing systemd units, and
// writing network units.

package disks

import (
	"fmt"

	"github.com/coreos/ignition/internal/config/types"
	"github.com/coreos/ignition/internal/exec/util"
	"github.com/coreos/ignition/internal/sgdisk"
)

// createPartitions creates the partitions described in config.Storage.Disks.
func (s stage) createPartitions(config types.Config) error {
	if len(config.Storage.Disks) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createPartitions")
	defer s.Logger.PopPrefix()

	devs := []string{}
	for _, disk := range config.Storage.Disks {
		devs = append(devs, string(disk.Device))
	}

	if err := s.waitOnDevicesAndCreateAliases(devs, "disks"); err != nil {
		return err
	}

	for _, dev := range config.Storage.Disks {
		devAlias := util.DeviceAlias(string(dev.Device))

		err := s.Logger.LogOp(func() error {
			op := sgdisk.Begin(s.Logger, devAlias)
			if dev.WipeTable {
				s.Logger.Info("wiping partition table requested on %q", devAlias)
				op.WipeTable(true)
			}

			for _, part := range dev.Partitions {
				op.CreatePartition(part)
			}

			if err := op.Commit(); err != nil {
				return fmt.Errorf("commit failure: %v", err)
			}
			return nil
		}, "partitioning %q", devAlias)
		if err != nil {
			return err
		}
	}

	return nil
}
