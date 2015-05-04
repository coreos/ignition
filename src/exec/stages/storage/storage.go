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

package disks

import (
	"fmt"
	"os/exec"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/src/exec/stages"
	"github.com/coreos/ignition/src/exec/util"
	"github.com/coreos/ignition/src/log"
	"github.com/coreos/ignition/src/sgdisk"
	"github.com/coreos/ignition/src/systemd"
)

const (
	name = "disks"
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
	subStage string
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config config.Config) bool {

	if err := s.createPartitions(config); err != nil {
		s.logger.Crit(fmt.Sprintf("create partitions failed: %v", err))
		return false
	}

	if err := s.createRaids(config); err != nil {
		s.logger.Crit(fmt.Sprintf("failed to create raids: %v", err))
		return false
	}

	if err := s.createFilesystems(config); err != nil {
		s.logger.Crit(fmt.Sprintf("failed to create filesystems: %v", err))
		return false
	}

	return true
}

func (s stage) createPartitions(config config.Config) error {
	if !s.enterSubStage(len(config.Storage.Disks) != 0, "createPartitions") {
		return nil
	}

	devs := []string{}
	for _, disk := range config.Storage.Disks {
		devs = append(devs, string(disk.Device))
	}

	if err := s.logOp(func() error { return systemd.WaitOnDevices(devs, "disks") }, "waiting for devices %v", devs); err != nil {
		return fmt.Errorf("failed to wait on disk devs: %v", err)
	}

	for _, dev := range config.Storage.Disks {
		err := s.logOp(func() error {
			op := sgdisk.Begin(string(dev.Device))
			if dev.WipeTable {
				s.logger.Info(fmt.Sprintf("wiping partition table on %q", dev.Device))
				op.WipeTable(true)
			}

			for _, part := range dev.Partitions {
				err := op.CreatePartition(sgdisk.Partition{
					Number: part.Number,
					Length: part.Size.Value(), // TODO(vc): normalize units... sectors? do something sane.
					Offset: part.Start.Value(),
					Label:  string(part.Label),
				})
				if err != nil {
					return fmt.Errorf("create failure: %v", err)
				}
			}

			if err := op.Commit(); err != nil {
				return fmt.Errorf("commit failure: %v", err)
			}
			return nil
		}, "partitioning %q", dev.Device)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s stage) createRaids(config config.Config) error {
	if !s.enterSubStage(len(config.Storage.Arrays) != 0, "createRaids") {
		return nil
	}

	devs := []string{}
	for _, array := range config.Storage.Arrays {
		for _, dev := range array.Devices {
			devs = append(devs, string(dev))
		}

		for _, dev := range array.Spares {
			devs = append(devs, string(dev))
		}
	}

	if err := s.logOp(func() error { return systemd.WaitOnDevices(devs, "raids") }, "waiting for devices %v", devs); err != nil {
		return fmt.Errorf("failed to wait on raids: %v", err)
	}

	for _, md := range config.Storage.Arrays {
		// FIXME(vc): this is utterly flummoxed by a preexisting md.Name, the magic of device-resident md metadata really interferes with us.
		// It's as if what ignition really needs is to turn off automagic md probing/running before getting started.
		args := []string{
			"--create", md.Name,
			"--force",
			"--run",
			"--level", md.Level,
			"--raid-devices", fmt.Sprintf("%d", len(md.Devices)),
			// FIXME(vc): md.Spares, this doesn't map well with mdadm's cli, change config.
		}
		for _, dev := range md.Devices {
			args = append(args, string(dev))
		}

		cmd := exec.Command("/sbin/mdadm", args...)
		if err := s.logOp(cmd.Run, "creating %q", md); err != nil {
			return fmt.Errorf("mdadm failed: %v", err)
		}
	}

	return nil
}

func (s stage) createFilesystems(config config.Config) error {
	if !s.enterSubStage(len(config.Storage.Filesystems) != 0, "createFilesystems") {
		return nil
	}

	devs := []string{}
	for _, fs := range config.Storage.Filesystems {
		devs = append(devs, string(fs.Device))
	}

	if err := s.logOp(func() error { return systemd.WaitOnDevices(devs, "filesystem") }, "waiting for devices %v", devs); err != nil {
		return fmt.Errorf("failed to wait on filesystem devices: %v", err)
	}

	for _, fs := range config.Storage.Filesystems {
		mkfs := ""
		args := []string(fs.Options)
		switch fs.Format {
		case "btrfs":
			mkfs = "/sbin/mkfs.btrfs"
			args = append(args, "--force")
		case "ext4":
			mkfs = "/sbin/mkfs.ext4"
			args = append(args, "-F")
		default:
			return fmt.Errorf("unsupported filesystem format: %q", fs.Format)
		}

		args = append(args, string(fs.Device))
		cmd := exec.Command(mkfs, args...)

		if err := s.logOp(cmd.Run, "creating %q filesystem on %q", fs.Format, string(fs.Device)); err != nil {
			return fmt.Errorf("failed to run %q: %v %v", mkfs, err, args)
		}

		// TODO(vc): apply fs.Files, which requires mounting somewhere...
	}

	return nil
}

func (s *stage) enterSubStage(ok bool, subStage string) bool {
	if !ok {
		return false
	}
	s.subStage = subStage
	return true
}

// TODO(vc): move these somewhere all of ignition can use, and instead of passing a bare logger interface around make these available.
// logOp executes and logs the start/finish/failure of an arbitrary function
func (s stage) logOp(op func() error, format string, a ...interface{}) error {
	s.logStart(format, a...)
	if err := op(); err != nil {
		s.logFail(format, a...)
		return err
	}
	s.logFinish(format, a...)
	return nil
}

func (s stage) logStart(format string, a ...interface{}) {
	s.logger.Info(fmt.Sprintf("%s: [start]  ", s.subStage) + fmt.Sprintf(format, a...))
}

func (s stage) logFail(format string, a ...interface{}) {
	s.logger.Crit(fmt.Sprintf("%s: [fail]   ", s.subStage) + fmt.Sprintf(format, a...))
}

func (s stage) logFinish(format string, a ...interface{}) {
	s.logger.Info(fmt.Sprintf("%s: [finish] ", s.subStage) + fmt.Sprintf(format, a...))
}
