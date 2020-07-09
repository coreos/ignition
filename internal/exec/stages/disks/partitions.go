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
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/coreos/ignition/v2/config/v3_2_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/sgdisk"
)

var (
	ErrBadSgdiskOutput = errors.New("sgdisk had unexpected output")
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
			return s.partitionDisk(dev, devAlias)
		}, "partitioning %q", devAlias)
		if err != nil {
			return err
		}
	}

	return nil
}

// partitionMatches determines if the existing partition matches the spec given. See doc/operator notes for what
// what it means for an existing partition to match the spec. spec must have non-zero Start and Size. existing must
// also have non-zero start and size and non-nil start and size and label.
// n.b. existing.{Size,Start}MiB must be converted to sectors first (yes the variable name becomes misleading)
func partitionMatches(existing util.PartitionInfo, spec types.Partition) error {
	if spec.Number != existing.Number {
		return fmt.Errorf("partition numbers did not match (specified %d, got %d). This should not happen, please file a bug.", spec.Number, existing.Number)
	}
	if spec.StartMiB != nil && int64(*spec.StartMiB) != existing.StartSector {
		return fmt.Errorf("starting sector did not match (specified %d, got %d)", *spec.StartMiB, existing.StartSector)
	}
	if spec.SizeMiB != nil && int64(*spec.SizeMiB) != existing.SizeInSectors {
		return fmt.Errorf("size did not match (specified %d, got %d)", *spec.SizeMiB, existing.SizeInSectors)
	}
	if spec.GUID != nil && *spec.GUID != "" && strings.ToLower(*spec.GUID) != strings.ToLower(existing.GUID) {
		return fmt.Errorf("GUID did not match (specified %q, got %q)", *spec.GUID, existing.GUID)
	}
	if spec.TypeGUID != nil && *spec.TypeGUID != "" && strings.ToLower(*spec.TypeGUID) != strings.ToLower(existing.TypeGUID) {
		return fmt.Errorf("type GUID did not match (specified %q, got %q)", *spec.TypeGUID, existing.TypeGUID)
	}
	if spec.Label != nil && *spec.Label != existing.Label {
		return fmt.Errorf("label did not match (specified %q, got %q)", *spec.Label, existing.Label)
	}
	return nil
}

// partitionShouldBeInspected returns if the partition has zeroes that need to be resolved to sectors.
func partitionShouldBeInspected(part types.Partition) bool {
	if part.Number == 0 {
		return false
	}
	return (part.StartMiB != nil && *part.StartMiB == 0) ||
		(part.SizeMiB != nil && *part.SizeMiB == 0)
}

func convertMiBToSectors(mib *int, sectorSize uint64) {
	if mib != nil {
		*mib = int(uint64(*mib) * (1024 * 1024 / sectorSize))
	}
}

// getRealStartAndSize returns a map of partition numbers to a struct that contains what their real start
// and end sector should be. It runs sgdisk --pretend to determine what the partitions would look like if
// everything specified were to be (re)created.
// It also converts everything to sectors so the StartMiB/SizeMiB will NOT be in MiB after this call
func (s stage) getRealStartAndSize(dev types.Disk, devAlias string, diskInfo util.DiskInfo) ([]types.Partition, error) {
	op := sgdisk.Begin(s.Logger, devAlias)
	for _, part := range dev.Partitions {
		convertMiBToSectors(part.SizeMiB, diskInfo.LogicalSectorSize)
		convertMiBToSectors(part.StartMiB, diskInfo.LogicalSectorSize)

		if info, exists := diskInfo.GetPartition(part.Number); exists {
			// delete all existing partitions
			op.DeletePartition(part.Number)
			if part.StartMiB == nil && (part.WipePartitionEntry == nil || !*part.WipePartitionEntry) {
				// don't care means keep the same if we can't wipe, otherwise stick it at start 0
				// This is not type safe, but should be mostly ok. JSON needs to learn about int size...
				i := int(info.StartSector)
				part.StartMiB = &i
			}
			if part.SizeMiB == nil && (part.WipePartitionEntry == nil || !*part.WipePartitionEntry) {
				// This is not type safe, but should be mostly ok. JSON needs to learn about int size...
				i := int(info.SizeInSectors)
				part.SizeMiB = &i
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
	for _, part := range dev.Partitions {
		if partitionShouldBeInspected(part) {
			op.Info(part.Number)
			partitionsToInspect = append(partitionsToInspect, part.Number)
		}
	}

	output, err := op.Pretend()
	if err != nil {
		return nil, err
	}

	realDimensions, err := parseSgdiskPretend(output, partitionsToInspect)
	if err != nil {
		return nil, err
	}

	result := []types.Partition{}
	for _, part := range dev.Partitions {
		if dims, ok := realDimensions[part.Number]; ok {
			if part.StartMiB != nil {
				part.StartMiB = &dims.start
			}
			if part.SizeMiB != nil {
				part.SizeMiB = &dims.size
			}
		}
		result = append(result, part)
	}
	return result, nil
}

type sgdiskOutput struct {
	start int
	size  int
}

// parseLine takes a regexp that captures an int and a string to match on. On success it returns
// the captured int and nil. If the regexp does not match it returns -1 and nil. If it encountered
// an error it returns 0 and the error.
func parseLine(r *regexp.Regexp, line string) (int, error) {
	matches := r.FindStringSubmatch(line)
	switch len(matches) {
	case 0:
		return -1, nil
	case 2:
		return strconv.Atoi(matches[1])
	default:
		return 0, ErrBadSgdiskOutput
	}
}

// parseSgdiskPretend parses the output of running sgdisk pretend with --info specified for each partition
// number specified in partitionNumbers. E.g. if paritionNumbers is [1,4,5], it is expected that the sgdisk
// output was from running `sgdisk --pretend <commands> --info=1 --info=4 --info=5`. It assumes the the
// partition labels are well behaved (i.e. contain no control characters). It returns a list of partitions
// matching the partition numbers specified, but with the start and size information as determined by sgdisk.
// The partition numbers need to passed in because sgdisk includes them in its output.
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
	current := sgdiskOutput{}
	i := 0

	lines := strings.Split(sgdiskOut, "\n")
	for _, line := range lines {
		switch state {
		case START:
			start, err := parseLine(startRegex, line)
			if err != nil {
				return nil, err
			}
			if start != -1 {
				current.start = start
				state = END
			}
		case END:
			end, err := parseLine(endRegex, line)
			if err != nil {
				return nil, err
			}
			if end != -1 {
				current.size = 1 + end - current.start
				output[partitionNumbers[i]] = current
				i++
				if i == len(partitionNumbers) {
					state = FAIL_ON_START_END
				} else {
					current = sgdiskOutput{}
					state = START
				}
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

// getPartitionMap returns a map of partitions on device, indexed by partition number
func (s stage) getPartitionMap(device string) (util.DiskInfo, error) {
	info := util.DiskInfo{}
	err := s.Logger.LogOp(
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

// partitionDisk partitions devAlias according to the spec given by dev
func (s stage) partitionDisk(dev types.Disk, devAlias string) error {
	if dev.WipeTable != nil && *dev.WipeTable {
		op := sgdisk.Begin(s.Logger, devAlias)
		s.Logger.Info("wiping partition table requested on %q", devAlias)
		op.WipeTable(true)
		op.Commit()
	}

	// Ensure all partitions with number 0 are last
	sort.Stable(PartitionList(dev.Partitions))

	op := sgdisk.Begin(s.Logger, devAlias)

	diskInfo, err := s.getPartitionMap(devAlias)
	if err != nil {
		return err
	}

	// get a list of parititions that have size and start 0 replaced with the real sizes
	// that would be used if all specified partitions were to be created anew.
	// Also change all of the start/size values into sectors.
	resolvedPartitions, err := s.getRealStartAndSize(dev, devAlias, diskInfo)
	if err != nil {
		return err
	}

	for _, part := range resolvedPartitions {
		shouldExist := partitionShouldExist(part)
		info, exists := diskInfo.GetPartition(part.Number)
		var matchErr error
		if exists {
			matchErr = partitionMatches(info, part)
		}
		matches := exists && matchErr == nil
		wipeEntry := part.WipePartitionEntry != nil && *part.WipePartitionEntry

		// This is a translation of the matrix in the operator notes.
		switch {
		case !exists && !shouldExist:
			s.Logger.Info("partition %d specified as nonexistant and no partition was found. Success.", part.Number)
		case !exists && shouldExist:
			op.CreatePartition(part)
		case exists && !shouldExist && !wipeEntry:
			return fmt.Errorf("partition %d exists but is specified as nonexistant and wipePartitionEntry is false", part.Number)
		case exists && !shouldExist && wipeEntry:
			op.DeletePartition(part.Number)
		case exists && shouldExist && matches:
			s.Logger.Info("partition %d found with correct specifications", part.Number)
		case exists && shouldExist && !wipeEntry && !matches:
			return fmt.Errorf("Partition %d didn't match: %v", part.Number, matchErr)
		case exists && shouldExist && wipeEntry && !matches:
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
