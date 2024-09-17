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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	cutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/partitioners"
	"github.com/coreos/ignition/v2/internal/partitioners/sfdisk"
	"github.com/coreos/ignition/v2/internal/partitioners/sgdisk"
	iutil "github.com/coreos/ignition/v2/internal/util"
)

func getDeviceManager(logger *log.Logger, dev string) partitioners.DeviceManager {
	// To be replaced with build tag support or something similar.
	if false {
		return sgdisk.Begin(logger, dev)
	}
	return sfdisk.Begin(logger, dev)
}

// createPartitions creates the partitions described in config.Storage.Disks.
func (s stage) createPartitions(config types.Config) error {
	if len(config.Storage.Disks) == 0 {
		return nil
	}
	s.PushPrefix("createPartitions")
	defer s.PopPrefix()

	devs := []string{}
	for _, disk := range config.Storage.Disks {
		devs = append(devs, string(disk.Device))
	}

	if err := s.waitOnDevicesAndCreateAliases(devs, "disks"); err != nil {
		return err
	}

	for _, dev := range config.Storage.Disks {
		devAlias := util.DeviceAlias(string(dev.Device))

		err := s.LogOp(func() error {
			return s.partitionDisk(dev, devAlias)
		}, "partitioning %q", devAlias)
		if err != nil {
			return err
		}
	}

	return nil
}

// partitionMatches determines if the existing partition matches the spec given. See doc/operator notes for what
// what it means for an existing partition to match the spec. spec must have non-zero Start and Size.
func partitionMatches(existing util.PartitionInfo, spec partitioners.Partition) error {
	if err := partitionMatchesCommon(existing, spec); err != nil {
		return err
	}
	if spec.SizeInSectors != nil && *spec.SizeInSectors != existing.SizeInSectors {
		return fmt.Errorf("size did not match (specified %d, got %d)", *spec.SizeInSectors, existing.SizeInSectors)
	}
	return nil
}

// partitionMatchesResize returns if the existing partition should be resized by evaluating if
// `resize`field is true and partition matches in all respects except size.
func partitionMatchesResize(existing util.PartitionInfo, spec partitioners.Partition) bool {
	return cutil.IsTrue(spec.Resize) && partitionMatchesCommon(existing, spec) == nil
}

// partitionMatchesCommon handles the common tests (excluding the partition size) to determine
// if the existing partition matches the spec given.
func partitionMatchesCommon(existing util.PartitionInfo, spec partitioners.Partition) error {
	if spec.Number != existing.Number {
		return fmt.Errorf("partition numbers did not match (specified %d, got %d). This should not happen, please file a bug", spec.Number, existing.Number)
	}
	if spec.StartSector != nil && *spec.StartSector != existing.StartSector {
		return fmt.Errorf("starting sector did not match (specified %d, got %d)", *spec.StartSector, existing.StartSector)
	}
	if cutil.NotEmpty(spec.GUID) && !strings.EqualFold(*spec.GUID, existing.GUID) {
		return fmt.Errorf("GUID did not match (specified %q, got %q)", *spec.GUID, existing.GUID)
	}
	if cutil.NotEmpty(spec.TypeGUID) && !strings.EqualFold(*spec.TypeGUID, existing.TypeGUID) {
		return fmt.Errorf("type GUID did not match (specified %q, got %q)", *spec.TypeGUID, existing.TypeGUID)
	}
	if spec.Label != nil && *spec.Label != existing.Label {
		return fmt.Errorf("label did not match (specified %q, got %q)", *spec.Label, existing.Label)
	}
	return nil
}

// partitionShouldBeInspected returns if the partition has zeroes that need to be resolved to sectors.
func partitionShouldBeInspected(part partitioners.Partition) bool {
	if part.Number == 0 {
		return false
	}
	return (part.StartSector != nil && *part.StartSector == 0) ||
		(part.SizeInSectors != nil && *part.SizeInSectors == 0)
}

func convertMiBToSectors(mib *int, sectorSize int) *int64 {
	if mib != nil {
		v := int64(*mib) * (1024 * 1024 / int64(sectorSize))
		return &v
	} else {
		return nil
	}
}

// getRealStartAndSize returns a map of partition numbers to a struct that contains what their real start
// and end sector should be. It runs sgdisk --pretend to determine what the partitions would look like if
// everything specified were to be (re)created.
func (s stage) getRealStartAndSize(dev types.Disk, devAlias string, diskInfo util.DiskInfo) ([]partitioners.Partition, error) {
	partitions := []partitioners.Partition{}
	for _, cpart := range dev.Partitions {
		partitions = append(partitions, partitioners.Partition{
			Partition:     cpart,
			StartSector:   convertMiBToSectors(cpart.StartMiB, diskInfo.LogicalSectorSize),
			SizeInSectors: convertMiBToSectors(cpart.SizeMiB, diskInfo.LogicalSectorSize),
		})
	}

	op := getDeviceManager(s.Logger, devAlias)
	for _, part := range partitions {
		if info, exists := diskInfo.GetPartition(part.Number); exists {
			// delete all existing partitions
			op.DeletePartition(part.Number)
			if part.StartSector == nil && !cutil.IsTrue(part.WipePartitionEntry) {
				// don't care means keep the same if we can't wipe, otherwise stick it at start 0
				part.StartSector = &info.StartSector
			}
			if part.SizeInSectors == nil && !cutil.IsTrue(part.WipePartitionEntry) {
				part.SizeInSectors = &info.SizeInSectors
			}
		}
		if partitionShouldExist(part) {
			// Clear the label. sgdisk doesn't escape control characters. This makes parsing easier
			part.Label = nil
			op.CreatePartition(part)
		}
	}

	// We only care to examine partitions that have start or size 0.
	partitionsToInspect := []int{}
	for _, part := range partitions {
		if partitionShouldBeInspected(part) {
			op.Info(part.Number)
			partitionsToInspect = append(partitionsToInspect, part.Number)
		}
	}

	output, err := op.Pretend()
	if err != nil {
		return nil, err
	}
	realDimensions, err := op.ParseOutput(output, partitionsToInspect)
	if err != nil {
		return nil, err
	}

	result := []partitioners.Partition{}
	for _, part := range partitions {
		if dims, ok := realDimensions[part.Number]; ok {
			if part.StartSector != nil {
				part.StartSector = &dims.Start
			}
			if part.SizeInSectors != nil {
				part.SizeInSectors = &dims.Size
			}
		}
		result = append(result, part)
	}
	return result, nil
}

// partitionShouldExist returns whether a bool is indicating if a partition should exist or not.
// nil (unspecified in json) is treated the same as true.
func partitionShouldExist(part partitioners.Partition) bool {
	return !cutil.IsFalse(part.ShouldExist)
}

// getPartitionMap returns a map of partitions on device, indexed by partition number
func (s stage) getPartitionMap(device string) (util.DiskInfo, error) {
	info := util.DiskInfo{}
	err := s.LogOp(
		func() error {
			var err error
			info, err = util.DumpDisk(device)
			return err
		}, "reading partition table of %q", device)
	if err != nil {
		return util.DiskInfo{}, err
	}
	return info, nil
}

// Allow sorting partitions (must be a stable sort) so partition number 0 happens last
// regardless of where it was in the list.
type PartitionList []types.Partition

func (p PartitionList) Len() int {
	return len(p)
}

// We only care about partitions with number 0 being considered the "largest" elements
// so they are processed last.
func (p PartitionList) Less(i, j int) bool {
	return p[i].Number != 0 && p[j].Number == 0
}

func (p PartitionList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func isBlockDevMapper(blockDevResolved string) bool {
	blockDevNode := filepath.Base(blockDevResolved)
	dmName := fmt.Sprintf("/sys/class/block/%s/dm/name", blockDevNode)
	_, err := os.Stat(dmName)
	return err == nil
}

// Expects a /dev/xyz path
func blockDevHeld(blockDevResolved string) (bool, error) {
	_, blockDevNode := filepath.Split(blockDevResolved)

	holdersDir := fmt.Sprintf("/sys/class/block/%s/holders/", blockDevNode)
	entries, err := os.ReadDir(holdersDir)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve holders of %q: %v", blockDevResolved, err)
	}
	return len(entries) > 0, nil
}

// Expects a /dev/xyz path
func blockDevMounted(blockDevResolved string) (bool, error) {
	mounts, err := os.Open("/proc/mounts")
	if err != nil {
		return false, fmt.Errorf("failed to open /proc/mounts: %v", err)
	}
	scanner := bufio.NewScanner(mounts)
	for scanner.Scan() {
		mountSource := strings.Split(scanner.Text(), " ")[0]
		if strings.HasPrefix(mountSource, "/") {
			mountSourceResolved, err := filepath.EvalSymlinks(mountSource)
			if err != nil {
				return false, fmt.Errorf("failed to resolve %q: %v", mountSource, err)
			}
			if mountSourceResolved == blockDevResolved {
				return true, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to check mounts for %q: %v", blockDevResolved, err)
	}
	return false, nil
}

// Expects a /dev/xyz path
func blockDevPartitions(blockDevResolved string) ([]string, error) {
	_, blockDevNode := filepath.Split(blockDevResolved)

	// This also works for extended MBR partitions
	sysDir := fmt.Sprintf("/sys/class/block/%s/", blockDevNode)
	entries, err := os.ReadDir(sysDir)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sysfs entries of %q: %v", blockDevResolved, err)
	}
	var partitions []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), blockDevNode) {
			partitions = append(partitions, "/dev/"+entry.Name())
		}
	}

	return partitions, nil
}

// Expects a /dev/xyz path
func blockDevInUse(blockDevResolved string, skipPartitionCheck bool) (bool, []string, error) {
	// Note: This ignores swap and LVM usage
	inUse := false
	isDevMapper := isBlockDevMapper(blockDevResolved)
	held := false
	if !isDevMapper {
		var err error
		held, err = blockDevHeld(blockDevResolved)
		if err != nil {
			return false, nil, fmt.Errorf("failed to check if %q is held: %v", blockDevResolved, err)
		}
	}
	mounted, err := blockDevMounted(blockDevResolved)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check if %q is mounted: %v", blockDevResolved, err)
	}
	inUse = held || mounted
	if skipPartitionCheck {
		return inUse, nil, nil
	}
	partitions, err := blockDevPartitions(blockDevResolved)
	if err != nil {
		return false, nil, fmt.Errorf("failed to retrieve partitions of %q: %v", blockDevResolved, err)
	}
	var activePartitions []string
	for _, partition := range partitions {
		partInUse, _, err := blockDevInUse(partition, true)
		if err != nil {
			return false, nil, fmt.Errorf("failed to check if partition %q is in use: %v", partition, err)
		}
		if partInUse {
			activePartitions = append(activePartitions, partition)
			inUse = true
		}
	}
	return inUse, activePartitions, nil
}

// Expects a /dev/xyz path
func partitionNumberPrefix(blockDevResolved string) string {
	lastChar := blockDevResolved[len(blockDevResolved)-1]
	if '0' <= lastChar && lastChar <= '9' {
		return "p"
	}
	return ""
}

// partitionDisk partitions devAlias according to the spec given by dev
func (s stage) partitionDisk(dev types.Disk, devAlias string) error {
	blockDevResolved, err := filepath.EvalSymlinks(devAlias)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %v", devAlias, err)
	}

	inUse, activeParts, err := blockDevInUse(blockDevResolved, false)
	if err != nil {
		return fmt.Errorf("failed usage check on %q: %v", devAlias, err)
	}
	if inUse && len(activeParts) == 0 {
		return fmt.Errorf("refusing to operate on directly active disk %q", devAlias)
	}
	if cutil.IsTrue(dev.WipeTable) {
		op := getDeviceManager(s.Logger, devAlias)
		s.Logger.Info("wiping partition table requested on %q", devAlias)
		if len(activeParts) > 0 {
			return fmt.Errorf("refusing to wipe active disk %q", devAlias)
		}
		op.WipeTable(true)
		if err := op.Commit(); err != nil {
			// `sgdisk --zap-all` will exit code 2 if the table was corrupted; retry it
			// https://github.com/coreos/fedora-coreos-tracker/issues/1596
			s.Info("potential error encountered while wiping table... retrying")
			if err := op.Commit(); err != nil {
				return err
			}
		}
	}

	// Ensure all partitions with number 0 are last
	sort.Stable(PartitionList(dev.Partitions))

	op := getDeviceManager(s.Logger, devAlias)

	diskInfo, err := s.getPartitionMap(devAlias)
	if err != nil {
		return err
	}

	prefix := partitionNumberPrefix(blockDevResolved)

	// get a list of parititions that have size and start 0 replaced with the real sizes
	// that would be used if all specified partitions were to be created anew.
	// Also calculate sectors for all of the start/size values.
	resolvedPartitions, err := s.getRealStartAndSize(dev, devAlias, diskInfo)
	if err != nil {
		return err
	}

	var partxAdd []uint64
	var partxDelete []uint64
	var partxUpdate []uint64

	for _, part := range resolvedPartitions {
		shouldExist := partitionShouldExist(part)
		info, exists := diskInfo.GetPartition(part.Number)
		var matchErr error
		if exists {
			matchErr = partitionMatches(info, part)
		}
		matches := exists && matchErr == nil
		wipeEntry := cutil.IsTrue(part.WipePartitionEntry)
		partInUse := iutil.StrSliceContains(activeParts, fmt.Sprintf("%s%s%d", blockDevResolved, prefix, part.Number))

		var modification bool

		// This is a translation of the matrix in the operator notes.
		switch {
		case !exists && !shouldExist:
			s.Info("partition %d specified as nonexistant and no partition was found. Success.", part.Number)
		case !exists && shouldExist:
			op.CreatePartition(part)
			modification = true
			partxAdd = append(partxAdd, uint64(part.Number))
		case exists && !shouldExist && !wipeEntry:
			return fmt.Errorf("partition %d exists but is specified as nonexistant and wipePartitionEntry is false", part.Number)
		case exists && !shouldExist && wipeEntry:
			op.DeletePartition(part.Number)
			modification = true
			partxDelete = append(partxDelete, uint64(part.Number))
		case exists && shouldExist && matches:
			s.Info("partition %d found with correct specifications", part.Number)
		case exists && shouldExist && !wipeEntry && !matches:
			if partitionMatchesResize(info, part) {
				s.Info("resizing partition %d", part.Number)
				op.DeletePartition(part.Number)
				part.Number = info.Number
				part.GUID = &info.GUID
				part.TypeGUID = &info.TypeGUID
				part.Label = &info.Label
				part.StartSector = &info.StartSector
				op.CreatePartition(part)
				modification = true
				partxUpdate = append(partxUpdate, uint64(part.Number))
			} else {
				return fmt.Errorf("partition %d didn't match: %v", part.Number, matchErr)
			}
		case exists && shouldExist && wipeEntry && !matches:
			s.Info("partition %d did not meet specifications, wiping partition entry and recreating", part.Number)
			op.DeletePartition(part.Number)
			op.CreatePartition(part)
			modification = true
			partxUpdate = append(partxUpdate, uint64(part.Number))
		default:
			// unfortunatey, golang doesn't check that all cases are handled exhaustively
			return fmt.Errorf("unreachable code reached when processing partition %d. golang--", part.Number)
		}

		if partInUse && modification {
			return fmt.Errorf("refusing to modify active partition %d on %q", part.Number, devAlias)
		}
	}

	if err := op.Commit(); err != nil {
		return fmt.Errorf("commit failure: %v", err)
	}

	// In contrast to similar tools, sgdisk does not trigger the update of the
	// kernel partition table with BLKPG but only uses BLKRRPART which fails
	// as soon as one partition of the disk is mounted
	if len(activeParts) > 0 {
		runPartxCommand := func(op string, partitions []uint64) error {
			for _, partNr := range partitions {
				cmd := exec.Command(distro.PartxCmd(), "--"+op, "--nr", strconv.FormatUint(partNr, 10), blockDevResolved)
				if _, err := s.LogCmd(cmd, "triggering partition %d %s on %q", partNr, op, devAlias); err != nil {
					return fmt.Errorf("partition %s failed: %v", op, err)
				}
			}
			return nil
		}
		if err := runPartxCommand("delete", partxDelete); err != nil {
			return err
		}
		if err := runPartxCommand("update", partxUpdate); err != nil {
			return err
		}
		if err := runPartxCommand("add", partxAdd); err != nil {
			return err
		}
	}

	// It's best to wait here for the /dev/ABC entries to be
	// (re)created, not only for other parts of the initramfs but
	// also because s.waitOnDevices() can still race with udev's
	// partition entry recreation.
	if err := s.waitForUdev(devAlias); err != nil {
		return fmt.Errorf("failed to wait for udev on %q after partitioning: %v", devAlias, err)
	}

	return nil
}
