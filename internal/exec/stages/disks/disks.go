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
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/exec/stages"
	"github.com/coreos/ignition/internal/exec/util"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/resource"
	"github.com/coreos/ignition/internal/sgdisk"
	"github.com/coreos/ignition/internal/systemd"
	putil "github.com/coreos/ignition/internal/util"
)

const (
	name = "disks"
)

var (
	ErrBadFilesystem   = errors.New("filesystem is not of the correct type")
	ErrBadSgdiskOutput = errors.New("sgdisk had unexpected output")
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

	client *resource.HttpClient
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config types.Config) bool {
	// Interacting with disks/paritions/raids/filesystems in general can cause
	// udev races. If we do not need to  do anything, we also do not need to
	// do the udevadm settle and can just return here.
	if len(config.Storage.Disks) == 0 &&
		len(config.Storage.Raid) == 0 &&
		len(config.Storage.Filesystems) == 0 {
		return true
	}

	if err := s.createPartitions(config); err != nil {
		s.Logger.Crit("create partitions failed: %v", err)
		return false
	}

	if err := s.createRaids(config); err != nil {
		s.Logger.Crit("failed to create raids: %v", err)
		return false
	}

	if err := s.createFilesystems(config); err != nil {
		s.Logger.Crit("failed to create filesystems: %v", err)
		return false
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
		exec.Command("/bin/udevadm", "settle"),
		"waiting for udev to settle",
	); err != nil {
		s.Logger.Crit("udevadm settle failed: %v", err)
		return false
	}

	return true
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

// partitionMatches determines if the existing partition matches the spec given. See doc/operator notes for what
// what it means for an existing partition to match the spec. spec must have non-zero Start an Size. existing must
// also have non-zero start and size and non-nil start and size and label.
func partitionMatches(existing, spec types.Partition) error {
	if spec.Number != existing.Number {
		// sanity check
		return fmt.Errorf("partition numbers did not match (specified %q, got %q). This should not happen.", spec.Number, existing.Number)
	}
	if spec.Start != nil && *spec.Start != *existing.Start {
		return fmt.Errorf("starting sector did not match (specified %q, got %q)", *spec.Start, *existing.Start)
	}
	if spec.Size != nil && *spec.Size != *existing.Size {
		return fmt.Errorf("size did not match (specified %q, got %q)", *spec.Size, *existing.Size)
	}
	if spec.GUID != "" && spec.GUID != existing.GUID {
		return fmt.Errorf("GUID did not match (specified %q, got %q)", spec.GUID, existing.GUID)
	}
	if spec.TypeGUID != "" && spec.TypeGUID != existing.TypeGUID {
		return fmt.Errorf("type GUID did not match (specified %q, got %q)", spec.TypeGUID, existing.TypeGUID)
	}
	if spec.Label != nil && *spec.Label != *existing.Label {
		return fmt.Errorf("label did not match (specified %q, got %q)", *spec.Label, *existing.Label)
	}
	return nil
}

func getPartitionDeviceName(diskDev string, part int) string {
	asRunes := []rune(diskDev)
	if unicode.IsDigit(asRunes[len(asRunes)-1]) {
		return fmt.Sprintf("%sp%d", diskDev, part)
	} else {
		return fmt.Sprintf("%s%d", diskDev, part)
	}
}

// getReadStartAnsSize returns a map of partition numbers to a struct that contains what their real start
// and end sector should be. It runs sgdisk --pretend to determine what the partitions would look like if
// everything specified were to be (re)created.
func (s stage) getRealStartAndSize(dev types.Disk, devAlias string, existanceMap map[int]types.Partition) (map[int]sgdiskOutput, error) {
	op := sgdisk.Begin(s.Logger, devAlias)
	for _, part := range dev.Partitions {
		_, exists := existanceMap[part.Number]
		if exists {
			// delete all existing partitions
			s.Logger.Info("Deleting partition %v", part.Number)
			op.DeletePartition(part.Number)
		}
		if partitionShouldExist(part) {
			// Clear the label so it doesn't interfere with the sgdisk parsing in case
			// it has control characters in it
			part.Label = nil
			op.CreatePartition(part)
		}
	}

	// We only care to examine partitions that have start or size 0
	partitionsToInspect := []int{}
	for _, part := range dev.Partitions {
		if ((part.Start != nil && *part.Start == 0) || (part.Size != nil && *part.Size == 0)) && part.Number != 0 {
			op.Info(part.Number)
			partitionsToInspect = append(partitionsToInspect, part.Number)
		}
	}

	output, err := op.Pretend()
	if err != nil {
		return nil, err
	}
	return parseSgdiskPretend(output, partitionsToInspect)
}

type sgdiskOutput struct {
	start int
	end   int
}

// parseSgdiskPretend parses the output of running sgdisk pretend with --info specified for each partition
// number specified in partitionNumbers. E.g. if paritionNumbers is [1,4,5], it is expected that the sgdisk
// output was from running `sgdisk --pretend <commands> --info=1 --info=4 --info=5`. It assumes the the
// partition labels are well behaved (i.e. contain no control characters). It returns a list of partitions
// matching the partition numbers specified, but with the start and size information as determined by sgdisk.
func parseSgdiskPretend(sgdiskOut string, partitionNumbers []int) (map[int]sgdiskOutput, error) {
	if len(partitionNumbers) == 0 {
		return nil, nil
	}
	startRegex := regexp.MustCompile("^First sector: (\\d*) \\(.*\\)$")
	endRegex := regexp.MustCompile("^Last sector: (\\d*) \\(.*\\)$")
	const (
		START             = iota
		END               = iota
		FAIL_ON_START_END = iota
	)

	output := map[int]sgdiskOutput{}
	state := START
	i := 0
	current := sgdiskOutput{}
	var err error

	lines := strings.Split(sgdiskOut, "\n")
	for _, line := range lines {
		switch state {
		case START:
			matches := startRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				if current.start, err = strconv.Atoi(matches[1]); err != nil {
					return nil, err
				}
				state = END
			} else if len(matches) != 0 {
				return nil, ErrBadSgdiskOutput
			}
		case END:
			matches := endRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				if current.end, err = strconv.Atoi(matches[1]); err != nil {
					return nil, err
				}
				output[partitionNumbers[i]] = current

				i++
				if i == len(partitionNumbers) {
					// No more partitions to get info on. If the sgdisk output has more something went wrong.
					state = FAIL_ON_START_END
				} else {
					// Setup the next types.Partition
					current = sgdiskOutput{}
					state = START
				}
			} else if len(matches) != 0 {
				return nil, ErrBadSgdiskOutput
			}
		case FAIL_ON_START_END:
			if len(startRegex.FindStringSubmatch(line)) != 0 ||
				len(endRegex.FindStringSubmatch(line)) != 0 {
				return nil, ErrBadSgdiskOutput
			}
		}
	}

	if state != FAIL_ON_START_END {
		// We stopped parsing in the middle of a info block. Something is wrong
		return nil, ErrBadSgdiskOutput
	}

	return output, nil
}

// partitionShouldExist returns whether a bool is indicating if a partition should exist or not.
// nil (unspecified in json) is treated the same as true.
func partitionShouldExist(part types.Partition) bool {
	return part.ShouldExist == nil || *part.ShouldExist
}

// partitionDisk partitions devAlias according to the spec given by dev
func (s stage) partitionDisk(dev types.Disk, devAlias string) error {
	if dev.WipeTable {
		op := sgdisk.Begin(s.Logger, devAlias)
		s.Logger.Info("wiping partition table requested on %q", devAlias)
		op.WipeTable(true)
		op.Commit()
	}
	op := sgdisk.Begin(s.Logger, devAlias)

	originalParts, err := s.getPartitionMap(devAlias)
	if err != nil {
		return err
	}

	// get a list of parititions that have size and start 0 replaced with the real sizes
	// that would be used if all specified partitions were to be created anew.
	partsWithRealSizes, err := s.getRealStartAndSize(dev, devAlias, originalParts)
	if err != nil {
		return err
	}

	for _, part := range dev.Partitions {
		shouldExist := partitionShouldExist(part)
		info, exists := originalParts[part.Number]
		var matchErr error
		matches := false

		// replace start/size 0 with the actual values they would be
		if startAndEnd, ok := partsWithRealSizes[part.Number]; ok {
			part.Start = putil.IntToPtr(startAndEnd.start)
			part.Size = putil.IntToPtr(1 + startAndEnd.end - startAndEnd.start)
		}

		if exists {
			matchErr = partitionMatches(info, part)
			matches = (matchErr == nil)
		}

		// This is a translation of the matrix in the operator notes.
		switch {
		case !exists && !shouldExist:
			s.Logger.Info("partition %d specified as nonexistant and no partition was found. Success.", part.Number)
		case !exists && shouldExist:
			op.CreatePartition(part)
		case exists && !shouldExist && !part.WipePartitionEntry:
			return fmt.Errorf("partition %d exists but is specified as nonexistant and wipePartitionEntry is false", part.Number)
		case exists && !shouldExist && part.WipePartitionEntry:
			op.DeletePartition(part.Number)
		case exists && shouldExist && matches:
			s.Logger.Info("partition %d found with correct specifications", part.Number)
		case exists && shouldExist && !part.WipePartitionEntry && !matches:
			return fmt.Errorf("Partition %d didn't match: %v", part.Number, matchErr)
		case exists && shouldExist && part.WipePartitionEntry && !matches:
			s.Logger.Info("partition %d did not meet specifications, wiping partition entry and recreating", part.Number)
			op.DeletePartition(part.Number)
			op.CreatePartition(part)
		default:
			// unfortunatey, golang doesn't check that all cases are handled exhaustively
			return fmt.Errorf("Unreachable code reached when processing partition %d. golang--", part.Number)
		}
	}

	if err := op.Commit(); err != nil {
		return fmt.Errorf("commit failure: %v", err)
	}
	return nil
}

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
			return s.partitionDisk(dev, devAlias)
		}, "partitioning %q", devAlias)
		if err != nil {
			return err
		}
	}

	return nil
}

// createRaids creates the raid arrays described in config.Storage.Raid.
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
		args := []string{
			"--create", md.Name,
			"--force",
			"--run",
			"--homehost", "any",
			"--level", md.Level,
			"--raid-devices", fmt.Sprintf("%d", len(md.Devices)-md.Spares),
		}

		if md.Spares > 0 {
			args = append(args, "--spare-devices", fmt.Sprintf("%d", md.Spares))
		}

		for _, dev := range md.Devices {
			args = append(args, util.DeviceAlias(string(dev)))
		}

		if _, err := s.Logger.LogCmd(
			exec.Command("/sbin/mdadm", args...),
			"creating %q", md.Name,
		); err != nil {
			return fmt.Errorf("mdadm failed: %v", err)
		}
	}

	return nil
}

// createFilesystems creates the filesystems described in config.Storage.Filesystems.
func (s stage) createFilesystems(config types.Config) error {
	fss := make([]types.Mount, 0, len(config.Storage.Filesystems))
	for _, fs := range config.Storage.Filesystems {
		if fs.Mount != nil {
			fss = append(fss, *fs.Mount)
		}
	}

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

	for _, fs := range fss {
		if err := s.createFilesystem(fs); err != nil {
			return err
		}
	}

	return nil
}

func (s stage) createFilesystem(fs types.Mount) error {
	info, err := s.readFilesystemInfo(fs)
	if err != nil {
		return err
	}

	if fs.Create != nil {
		// If we are using 2.0.0 semantics...

		if !fs.Create.Force && info.format != "" {
			s.Logger.Err("filesystem detected at %q (found %s) and force was not requested", fs.Device, info.format)
			return ErrBadFilesystem
		}
	} else if !fs.WipeFilesystem {
		// If the filesystem isn't forcefully being created, then we need
		// to check if it is of the correct type or that no filesystem exists.

		if info.format == fs.Format &&
			(fs.Label == nil || info.label == *fs.Label) &&
			(fs.UUID == nil || canonicalizeFilesystemUUID(info.format, info.uuid) == canonicalizeFilesystemUUID(fs.Format, *fs.UUID)) {
			s.Logger.Info("filesystem at %q is already correctly formatted. Skipping mkfs...", fs.Device)
			return nil
		} else if info.format != "" {
			s.Logger.Err("filesystem at %q is not of the correct type, label, or UUID (found %s, %q, %s) and a filesystem wipe was not requested", fs.Device, info.format, info.label, info.uuid)
			return ErrBadFilesystem
		}
	}

	mkfs := ""
	var args []string
	if fs.Create == nil {
		args = translateMountOptionSliceToStringSlice(fs.Options)
	} else {
		args = translateCreateOptionSliceToStringSlice(fs.Create.Options)
	}
	switch fs.Format {
	case "btrfs":
		mkfs = "/sbin/mkfs.btrfs"
		args = append(args, "--force")
		if fs.UUID != nil {
			args = append(args, []string{"-U", canonicalizeFilesystemUUID(fs.Format, *fs.UUID)}...)
		}
		if fs.Label != nil {
			args = append(args, []string{"-L", *fs.Label}...)
		}
	case "ext4":
		mkfs = "/sbin/mkfs.ext4"
		args = append(args, "-F")
		if fs.UUID != nil {
			args = append(args, []string{"-U", canonicalizeFilesystemUUID(fs.Format, *fs.UUID)}...)
		}
		if fs.Label != nil {
			args = append(args, []string{"-L", *fs.Label}...)
		}
	case "xfs":
		mkfs = "/sbin/mkfs.xfs"
		args = append(args, "-f")
		if fs.UUID != nil {
			args = append(args, []string{"-m", "uuid=" + canonicalizeFilesystemUUID(fs.Format, *fs.UUID)}...)
		}
		if fs.Label != nil {
			args = append(args, []string{"-L", *fs.Label}...)
		}
	case "swap":
		mkfs = "/sbin/mkswap"
		args = append(args, "-f")
		if fs.UUID != nil {
			args = append(args, []string{"-U", canonicalizeFilesystemUUID(fs.Format, *fs.UUID)}...)
		}
		if fs.Label != nil {
			args = append(args, []string{"-L", *fs.Label}...)
		}
	case "vfat":
		mkfs = "/sbin/mkfs.vfat"
		// There is no force flag for mkfs.vfat, it always destroys any data on
		// the device at which it is pointed.
		if fs.UUID != nil {
			args = append(args, []string{"-i", canonicalizeFilesystemUUID(fs.Format, *fs.UUID)}...)
		}
		if fs.Label != nil {
			args = append(args, []string{"-n", *fs.Label}...)
		}
	default:
		return fmt.Errorf("unsupported filesystem format: %q", fs.Format)
	}

	devAlias := util.DeviceAlias(string(fs.Device))
	args = append(args, devAlias)
	if _, err := s.Logger.LogCmd(
		exec.Command(mkfs, args...),
		"creating %q filesystem on %q",
		fs.Format, devAlias,
	); err != nil {
		return fmt.Errorf("mkfs failed: %v", err)
	}

	return nil
}

// golang--
func translateMountOptionSliceToStringSlice(opts []types.MountOption) []string {
	newOpts := make([]string, len(opts))
	for i, o := range opts {
		newOpts[i] = string(o)
	}
	return newOpts
}

// golang--
func translateCreateOptionSliceToStringSlice(opts []types.CreateOption) []string {
	newOpts := make([]string, len(opts))
	for i, o := range opts {
		newOpts[i] = string(o)
	}
	return newOpts
}

type filesystemInfo struct {
	format string
	uuid   string
	label  string
}

func (s stage) readFilesystemInfo(fs types.Mount) (filesystemInfo, error) {
	res := filesystemInfo{}
	err := s.Logger.LogOp(
		func() error {
			var err error
			res.format, err = util.FilesystemType(fs.Device)
			if err != nil {
				return err
			}
			res.uuid, err = util.FilesystemUUID(fs.Device)
			if err != nil {
				return err
			}
			res.label, err = util.FilesystemLabel(fs.Device)
			if err != nil {
				return err
			}
			s.Logger.Info("found %s filesystem at %q with uuid %q and label %q", res.format, fs.Device, res.uuid, res.label)
			return nil
		},
		"determining filesystem type of %q", fs.Device,
	)

	return res, err
}

// getPartitionMap returns a map of partitions on device, indexed by partition number
func (s stage) getPartitionMap(device string) (map[int]types.Partition, error) {
	parts := []types.Partition{}
	err := s.Logger.LogOp(
		func() error {
			p, err := util.DumpPartitionTable(device)
			if err != nil {
				return err
			}
			parts = p
			return nil
		}, "reading partition table of %q", device)
	if err != nil {
		return nil, err
	}
	m := map[int]types.Partition{}
	for _, part := range parts {
		m[part.Number] = part
	}
	return m, nil
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
