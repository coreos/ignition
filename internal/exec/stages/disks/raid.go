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
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/coreos/ignition/v2/config/v3_2_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/util"
)

func (s stage) createRaids(config types.Config) error {
	raids := config.Storage.Raid

	if len(raids) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createRaids")
	defer s.Logger.PopPrefix()

	results := make(chan error)
	for _, md := range raids {
		if md.Spares == nil {
			zero := 0
			md.Spares = &zero
		}

		args := []string{
			"--create", md.Name,
			"--force",
			"--run",
			"--homehost", "any",
			"--level", md.Level,
			"--raid-devices", fmt.Sprintf("%d", len(md.Devices)-*md.Spares),
		}

		if *md.Spares > 0 {
			args = append(args, "--spare-devices", fmt.Sprintf("%d", *md.Spares))
		}

		for _, o := range md.Options {
			args = append(args, string(o))
		}

		devs := []string{}
		for _, dev := range md.Devices {
			args = append(args, util.DeviceAlias(string(dev)))
			devs = append(devs, string(dev))
		}

		go func(md types.Raid) {
			if err := s.waitOnDevicesAndCreateAliases(devs, "raids"); err != nil {
				results <- err
			} else {
				if _, err := s.Logger.LogCmd(
					exec.Command(distro.MdadmCmd(), args...),
					"creating %q", md.Name,
				); err != nil {
					results <- fmt.Errorf("mdadm failed: %v", err)
				} else {
					results <- nil
				}
			}
		}(md)
	}

	// Return combined errors
	var errs []string
	for range raids {
		if err := <-results; err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}
