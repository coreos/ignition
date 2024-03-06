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
// createRaids creates the raid arrays described in config.Storage.Raid.

package disks

import (
	"fmt"
	"os/exec"
	"strings"

	cutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_5_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/util"
)

func (s stage) createRaids(config types.Config) error {
	if len(config.Storage.Raid) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createRaids")
	defer s.Logger.PopPrefix()

	devs := []string{}
	for _, array := range config.Storage.Raid {
		for _, dev := range array.Devices {
			devs = append(devs, string(dev))
		}
	}

	if err := s.waitOnDevicesAndCreateAliases(devs, "raids"); err != nil {
		return err
	}

	for _, md := range config.Storage.Raid {
		devName := md.Name
		if !strings.HasPrefix(devName, "/dev") {
			devName = "/dev/md/" + md.Name
		}
		if cutil.IsTrue(md.Assemble) {
			args := []string{
				"--assemble",
				"--run",
				"--homehost", "any",
				devName,
			}

			for _, dev := range md.Devices {
				args = append(args, util.DeviceAlias(string(dev)))
			}

			if _, err := s.Logger.LogCmd(
				exec.Command(distro.MdadmCmd(), args...),
				"assembling %q", md.Name,
			); err != nil {
				s.Logger.Info("mdadm assemble failed: %v", err)
			} else {
				if err := s.waitOnDevices([]string{devName}, "raids"); err != nil {
					s.Logger.Info("mdadm assemble failed: %v", err)
				} else {
					if err := s.waitOnDevices([]string{devName}, "raids"); err != nil {
						return err
					}
					return nil
				}
			}
		}

		if md.Spares == nil {
			zero := 0
			md.Spares = &zero
		}
		args := []string{
			"--create", md.Name,
			"--force",
			"--run",
			"--homehost", "any",
			"--level", *md.Level,
			"--raid-devices", fmt.Sprintf("%d", len(md.Devices)-*md.Spares),
		}

		if *md.Spares > 0 {
			args = append(args, "--spare-devices", fmt.Sprintf("%d", *md.Spares))
		}

		for _, o := range md.Options {
			args = append(args, string(o))
		}

		for _, dev := range md.Devices {
			args = append(args, util.DeviceAlias(string(dev)))
		}

		if _, err := s.Logger.LogCmd(
			exec.Command(distro.MdadmCmd(), args...),
			"creating %q", md.Name,
		); err != nil {
			return fmt.Errorf("mdadm failed: %v", err)
		}

		// Wait for the created device node to show up, no udev
		// race prevention required because this node did not
		// exist before.
		if err := s.waitOnDevices([]string{devName}, "raids"); err != nil {
			return err
		}
	}

	return nil
}
