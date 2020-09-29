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
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/util"
)

var (
	ErrBadFilesystem = errors.New("filesystem is not of the correct type")
)

// createFilesystems creates the filesystems described in config.Storage.Filesystems.
func (s stage) createFilesystems(config types.Config) error {
	fss := config.Storage.Filesystems

	if len(fss) == 0 {
		return nil
	}

	s.Logger.PushPrefix("createFilesystems")
	defer s.Logger.PopPrefix()

	devs := []string{}
	for _, fs := range fss {
		devs = append(devs, string(fs.Device))
	}

	if err := s.waitOnDevicesAndCreateAliases(devs, "filesystems"); err != nil {
		return err
	}

	// Create filesystems concurrently up to GOMAXPROCS
	concurrency := runtime.GOMAXPROCS(-1)
	work := make(chan types.Filesystem, len(fss))
	results := make(chan error)

	for i := 0; i < concurrency; i++ {
		go func() {
			for fs := range work {
				results <- s.createFilesystem(fs)
			}
		}()
	}

	for _, fs := range fss {
		work <- fs
	}
	close(work)

	// Return combined errors
	var errs []string
	for range fss {
		if err := <-results; err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func (s stage) createFilesystem(fs types.Filesystem) error {
	if fs.Format == nil {
		return nil
	}
	devAlias := util.DeviceAlias(string(fs.Device))

	var info util.FilesystemInfo
	err := s.Logger.LogOp(
		func() error {
			var err error
			info, err = util.GetFilesystemInfo(devAlias, false)
			if err != nil {
				// Try again, allowing multiple filesystem
				// fingerprints this time.  If successful,
				// log a warning and continue.
				var err2 error
				info, err2 = util.GetFilesystemInfo(devAlias, true)
				if err2 == nil {
					s.Logger.Warning("%v", err)
				}
				err = err2
			}
			return err
		},
		"determining filesystem type of %q", fs.Device,
	)
	if err != nil {
		return err
	}
	s.Logger.Info("found %s filesystem at %q with uuid %q and label %q", info.Type, fs.Device, info.UUID, info.Label)

	if fs.WipeFilesystem == nil || !*fs.WipeFilesystem {
		// If the filesystem isn't forcefully being created, then we need
		// to check if it is of the correct type or that no filesystem exists.
		if info.Type == *fs.Format &&
			(fs.Label == nil || info.Label == *fs.Label) &&
			(fs.UUID == nil || canonicalizeFilesystemUUID(info.Type, info.UUID) == canonicalizeFilesystemUUID(*fs.Format, *fs.UUID)) {
			s.Logger.Info("filesystem at %q is already correctly formatted. Skipping mkfs...", fs.Device)
			return nil
		} else if info.Type != "" {
			s.Logger.Err("filesystem at %q is not of the correct type, label, or UUID (found %s, %q, %s) and a filesystem wipe was not requested", fs.Device, info.Type, info.Label, info.UUID)
			return ErrBadFilesystem
		}
	}

	if _, err := s.Logger.LogCmd(
		exec.Command(distro.WipefsCmd(), "-a", devAlias),
		"wiping filesystem signatures from %q",
		devAlias,
	); err != nil {
		return fmt.Errorf("wipefs failed: %v", err)
	}

	mkfs := ""
	args := translateOptionSliceToStringSlice(fs.Options)
	switch *fs.Format {
	case "btrfs":
		mkfs = distro.BtrfsMkfsCmd()
		args = append(args, "--force")
		if fs.UUID != nil {
			args = append(args, "-U", canonicalizeFilesystemUUID(*fs.Format, *fs.UUID))
		}
		if fs.Label != nil {
			args = append(args, "-L", *fs.Label)
		}
	case "ext4":
		mkfs = distro.Ext4MkfsCmd()
		args = append(args, "-F")
		if fs.UUID != nil {
			args = append(args, "-U", canonicalizeFilesystemUUID(*fs.Format, *fs.UUID))
		}
		if fs.Label != nil {
			args = append(args, "-L", *fs.Label)
		}
	case "xfs":
		mkfs = distro.XfsMkfsCmd()
		args = append(args, "-f")
		if fs.UUID != nil {
			args = append(args, "-m", "uuid="+canonicalizeFilesystemUUID(*fs.Format, *fs.UUID))
		}
		if fs.Label != nil {
			args = append(args, "-L", *fs.Label)
		}
	case "swap":
		mkfs = distro.SwapMkfsCmd()
		args = append(args, "-f")
		if fs.UUID != nil {
			args = append(args, "-U", canonicalizeFilesystemUUID(*fs.Format, *fs.UUID))
		}
		if fs.Label != nil {
			args = append(args, "-L", *fs.Label)
		}
	case "vfat":
		mkfs = distro.VfatMkfsCmd()
		// There is no force flag for mkfs.vfat, it always destroys any data on
		// the device at which it is pointed.
		if fs.UUID != nil {
			args = append(args, "-i", canonicalizeFilesystemUUID(*fs.Format, *fs.UUID))
		}
		if fs.Label != nil {
			args = append(args, "-n", *fs.Label)
		}
	default:
		return fmt.Errorf("unsupported filesystem format: %q", *fs.Format)
	}

	args = append(args, devAlias)
	if _, err := s.Logger.LogCmd(
		exec.Command(mkfs, args...),
		"creating %q filesystem on %q",
		*fs.Format, devAlias,
	); err != nil {
		return fmt.Errorf("mkfs failed: %v", err)
	}

	return nil
}

// golang--
func translateOptionSliceToStringSlice(opts []types.FilesystemOption) []string {
	newOpts := make([]string, len(opts))
	for i, o := range opts {
		newOpts[i] = string(o)
	}
	return newOpts
}

// canonicalizeFilesystemUUID does the minimum amount of canonicalization
// required to make two valid equivalent UUIDs compare equal, but doesn't
// attempt to fully validate the UUID.
func canonicalizeFilesystemUUID(format, uuid string) string {
	uuid = strings.ToLower(uuid)
	if format == "vfat" {
		// FAT uses a 32-bit volume ID instead of a UUID. blkid
		// (and the rest of the world) formats it as A1B2-C3D4, but
		// mkfs.fat doesn't permit the dash, so strip it. Older
		// versions of Ignition would fail if the config included
		// the dash, so we need to support omitting it.
		if len(uuid) >= 5 && uuid[4] == '-' {
			uuid = uuid[0:4] + uuid[5:]
		}
	}
	return uuid
}
