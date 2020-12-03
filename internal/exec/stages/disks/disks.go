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

// The storage stage is responsible for partitioning disks, creating RAID
// arrays, formatting partitions, writing files, writing systemd units, and
// writing network units.

package disks

import (
	"fmt"
	"os/exec"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/systemd"
)

const (
	name = "disks"
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

func (s stage) Run(config types.Config) error {
	// Interacting with disks/partitions/raids/filesystems in general can cause
	// udev races. If we do not need to  do anything, we also do not need to
	// do the udevadm settle and can just return here.
	if len(config.Storage.Disks) == 0 &&
		len(config.Storage.Raid) == 0 &&
		len(config.Storage.Filesystems) == 0 &&
		len(config.Storage.Luks) == 0 {
		return nil
	}

	if err := s.createPartitions(config); err != nil {
		return fmt.Errorf("create partitions failed: %v", err)
	}

	if err := s.createRaids(config); err != nil {
		return fmt.Errorf("failed to create raids: %v", err)
	}

	if err := s.createLuks(config); err != nil {
		return fmt.Errorf("failed to create luks: %v", err)
	}

	if err := s.createFilesystems(config); err != nil {
		return fmt.Errorf("failed to create filesystems: %v", err)
	}

	// udevd registers an IN_CLOSE_WRITE inotify watch on block device
	// nodes, and synthesizes udev "change" events when the watch fires.
	// mkfs.btrfs triggers multiple such events, the first of which
	// occurs while there is no recognizable filesystem on the
	// partition. Thus, if an existing partition is reformatted as
	// btrfs while keeping the same filesystem label, there will be a
	// synthesized uevent that deletes the /dev/disk/by-label symlink
	// and a second one that restores it. If we didn't account for this,
	// a systemd unit that depended on the by-label symlink (e.g.
	// systemd-fsck-root.service) could have the symlink deleted out
	// from under it.
	//
	// There's no way to fix this completely. We can't wait for the
	// restoring uevent to propagate, since we can't determine which
	// specific uevents were triggered by the mkfs. We can wait for
	// udev to settle, though it's conceivable that the deleting uevent
	// has already been processed and the restoring uevent is still
	// sitting in the inotify queue. In practice the uevent queue will
	// be the slow one, so this should be good enough.
	//
	// Test case: boot failure in coreos.ignition.*.btrfsroot kola test.
	//
	// Additionally, partitioning (and possibly creating raid) suffers
	// the same problem. To be safe, always settle.
	if _, err := s.Logger.LogCmd(
		exec.Command(distro.UdevadmCmd(), "settle"),
		"waiting for udev to settle",
	); err != nil {
		return fmt.Errorf("udevadm settle failed: %v", err)
	}

	return nil
}

// waitOnDevices waits for the devices enumerated in devs as a logged operation
// using ctxt for the logging and systemd unit identity.
func (s stage) waitOnDevices(devs []string, ctxt string) error {
	if err := s.LogOp(
		func() error { return systemd.WaitOnDevices(devs, ctxt) },
		"waiting for devices %v", devs,
	); err != nil {
		return fmt.Errorf("failed to wait on %s devs: %v", ctxt, err)
	}

	return nil
}

// createDeviceAliases creates device aliases for every device in devs.
func (s stage) createDeviceAliases(devs []string) error {
	for _, dev := range devs {
		target, err := util.CreateDeviceAlias(dev)
		if err != nil {
			return fmt.Errorf("failed to create device alias for %q: %v", dev, err)
		}
		s.Logger.Info("created device alias for %q: %q -> %q", dev, util.DeviceAlias(dev), target)
	}

	return nil
}

// waitOnDevicesAndCreateAliases simply wraps waitOnDevices and createDeviceAliases.
func (s stage) waitOnDevicesAndCreateAliases(devs []string, ctxt string) error {
	if err := s.waitOnDevices(devs, ctxt); err != nil {
		return err
	}

	if err := s.createDeviceAliases(devs); err != nil {
		return err
	}

	return nil
}
