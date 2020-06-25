// Copyright 2020 Red Hat, Inc.
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
//
// +build s390x

// Derived from partitions.go this file has a similar structure but has functions
// very specific to partitioning and formatting DASD disks.

package disks

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/coreos/ignition/v2/config/v3_2_experimental/types"
	"github.com/coreos/ignition/v2/internal/dasdfmt"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/fdasd"
)

var (
	ErrBadDasdfmtOutput = errors.New("dasdfmt had unexpected output")
	ErrBadFdasdOutput   = errors.New("fdasd had unexpected output")
)

type DasdDiskInfo struct {
	BytesPerBlock int
	Partitions    []DasdPartitionInfo

	//DASD specific variables
	Cylinders         int
	TracksPerCylinder int
	BlocksPerTrack    int
}

type DasdPartitionInfo struct {
	util.PartitionInfo
	// DASD specific variables
	StartTrack   int64
	SizeInTracks int64
	ShouldExist  bool
}

func (d DasdDiskInfo) GetDasdPartition(n int) (DasdPartitionInfo, bool) {
	for _, part := range d.Partitions {
		if part.Number == n {
			return part, true
		}
	}
	return DasdPartitionInfo{}, false
}

func dasdPartitionMatches(existing DasdPartitionInfo, spec fdasd.Partition) error {
	if spec.Number != existing.Number {
		return fmt.Errorf("partition numbers did not match (specified %d, got %d). This should not happen, please file a bug.", spec.Number, existing.Number)
	}
	if spec.StartTrack != existing.StartTrack {
		return fmt.Errorf("starting track did not match (specified %d, got %d)", spec.StartTrack, existing.StartTrack)
	}
	if spec.SizeInTracks != existing.SizeInTracks {
		return fmt.Errorf("size did not match (specified %d, got %d)", spec.SizeInTracks, existing.SizeInTracks)
	}
	return nil
}

func checkInvalidFields(spec types.Partition) error {
	if spec.Label != nil || spec.GUID != nil || spec.TypeGUID != nil {
		return fmt.Errorf("One or more invalid fields present in partition configuration")
	}

	if spec.Number > 3 {
		return fmt.Errorf("Partition number cannot exceed 3 for DASDs - only 3 partitions allowed")
	}
	return nil
}

// Instead of sectors, DASD cylinders have tracks which are in CKD format and each track has info about the start, size
// and data contained in it(https://www.ibm.com/support/knowledgecenter/zosbasics/com.ibm.zos.zconcepts/zconc_datasetstruct.htm)
func convertMiBToTracks(mib *int, blocksPerTrack int, sectorSize int) int64 {
	return int64(math.Ceil(float64((*mib)*1024*1024) / float64(blocksPerTrack*sectorSize)))
}

func mergePartitions(startPart int, endPart int, diskInfo DasdDiskInfo) ([]fdasd.Partition, error) {
	partitions := diskInfo.Partitions
	remainingParts := []fdasd.Partition{}
	for index := startPart; index < endPart; index++ {
		if index >= len(partitions) {
			return []fdasd.Partition{}, fmt.Errorf("Can't find existing partition %d. Please ensure there are no gaps in partition numbering", index)
		}
		addPart := fdasd.Partition{
			Number:             partitions[index].Number,
			StartTrack:         partitions[index].StartTrack,
			SizeInTracks:       partitions[index].SizeInTracks,
			ShouldExist:        true,
			WipePartitionEntry: false,
		}
		remainingParts = append(remainingParts, addPart)
	}
	return remainingParts, nil
}

func AddPartition(partition types.Partition, diskInfo DasdDiskInfo) fdasd.Partition {
	newPart := fdasd.Partition{}
	newPart.Number = partition.Number
	if partition.SizeMiB != nil {
		newPart.SizeInTracks = convertMiBToTracks(partition.SizeMiB, diskInfo.BlocksPerTrack, diskInfo.BytesPerBlock)
	}
	if partition.StartMiB != nil {
		newPart.StartTrack = convertMiBToTracks(partition.StartMiB, diskInfo.BlocksPerTrack, diskInfo.BytesPerBlock)
	}
	newPart.ShouldExist = (partition.ShouldExist == nil || *partition.ShouldExist)
	newPart.WipePartitionEntry = partition.WipePartitionEntry != nil && *partition.WipePartitionEntry
	if info, exists := diskInfo.GetDasdPartition(partition.Number); exists {
		if partition.StartMiB == nil && (partition.WipePartitionEntry == nil || !*partition.WipePartitionEntry) {
			newPart.StartTrack = info.StartTrack
		}
		if partition.SizeMiB == nil && (partition.WipePartitionEntry == nil || !*partition.WipePartitionEntry) {
			newPart.SizeInTracks = info.SizeInTracks
		}
	}
	return newPart
}

// getDasdRealStartAndSize returns a map of partition numbers to a struct that contains what their real start
// and end track should be. Because there is no `pretend` equivalent for fdasd, it becomes a little
// cumbersome and ugly to correctly determine the partition layout in all cases. So, there are some differences
// to how partitioning with DASDs will work:
//       - existing partitions will not be merged when creating new partitions unless explicitly specified
//       - any existing partitions specified will be wiped and recreated.
// It also converts everything to tracks so the StartMiB/SizeMiB will NOT be in MiB after this call
func getDasdRealStartAndSize(dev types.Disk, diskInfo DasdDiskInfo) ([]fdasd.Partition, error) {
	result := []fdasd.Partition{}
	lastPart := 0

	// Merge partitions and arrange
	for _, part := range dev.Partitions {
		if err := checkInvalidFields(part); err != nil {
			return []fdasd.Partition{}, err
		}
		// try to merge existing partitions if there are gaps
		if part.Number-lastPart > 1 {
			// there is a gap in the partition
			missingPartitions, err := mergePartitions(lastPart, part.Number-1, diskInfo)
			if err != nil {
				return []fdasd.Partition{}, err
			}
			result = append(result, missingPartitions...)
		}

		partition := AddPartition(part, diskInfo)
		result = append(result, partition)
		lastPart = part.Number
	}

	//add rest of the partitions if any
	remainingParts, _ := mergePartitions(lastPart, len(diskInfo.Partitions), diskInfo)
	result = append(result, remainingParts...)

	// Fill in missing values
	nextTrack := int64(0)
	for index := range result {
		if result[index].StartTrack == 0 {
			if nextTrack == 0 {
				result[index].StartTrack = 2 // DASD partitions start at 2 for CDL
			} else {
				result[index].StartTrack = nextTrack
			}
		}

		if result[index].SizeInTracks == 0 {
			if index+1 < len(result) && result[index+1].ShouldExist {
				if result[index+1].StartTrack > 0 {
					result[index].SizeInTracks = result[index+1].StartTrack - result[index].StartTrack
				}
			} else {
				result[index].SizeInTracks = int64(diskInfo.Cylinders*diskInfo.TracksPerCylinder) - result[index].StartTrack
			}
		}

		if result[index].ShouldExist {
			if result[index].SizeInTracks == 0 {
				return []fdasd.Partition{}, fmt.Errorf("Current partition %d at max size. cannot add more", index)
			}
			nextTrack = result[index].StartTrack + result[index].SizeInTracks
		}
	}
	return result, nil
}

// parseDasdDiskLine takes a regexp that captures an int and a string to match on. On success it returns
// the captured string, int and nil. If the regexp does not match it returns -1 and nil. If it encountered
// an error it returns 0 and the error.
func parseDasdDiskLine(r *regexp.Regexp, line string) (string, int, error) {
	matches := r.FindStringSubmatch(line)
	switch len(matches) {
	case 0:
		return "", -1, nil
	case 3:
		num, err := strconv.Atoi(matches[2])
		return matches[1], num, err
	default:
		return "", 0, ErrBadFdasdOutput
	}
}

// parseDasdDiskAndPartInfo parses device specific information needed for converting the sizes into
// tracks and also gets the information about the partitions. Option example present in:
// https://www.ibm.com/support/knowledgecenter/linuxonibm/com.ibm.linux.z.lgdd/lgdd_r_fasdusingoptions.html
func parseDasdDiskAndPartInfo(infostr string) (DasdDiskInfo, error) {
	diskInfo := DasdDiskInfo{}
	dasdDiskInfoRegex := regexp.MustCompile("\\s+([a-z\\ ]+)\\s\\.+\\:\\s+(\\d+)")
	scanner := bufio.NewScanner(strings.NewReader(infostr))
	for scanner.Scan() {
		line := scanner.Text()
		attr, value, err := parseDasdDiskLine(dasdDiskInfoRegex, line)
		if err != nil {
			return diskInfo, err
		}
		if value != -1 {
			switch attr {
			case "cylinders":
				diskInfo.Cylinders = value
				continue
			case "tracks per cylinder":
				diskInfo.TracksPerCylinder = value
				continue
			case "blocks per track":
				diskInfo.BlocksPerTrack = value
				continue
			case "bytes per block":
				diskInfo.BytesPerBlock = value
				continue
			default:
				break
			}
		}

		dasdPartInfoRegex := regexp.MustCompile("([a-zA-Z0-9/.-]+)\\s+(\\d+)\\s+(\\d+)\\s+(\\d+)\\s+(\\d+).+")
		matches := dasdPartInfoRegex.FindStringSubmatch(line)
		if len(matches) == 6 {
			partnum, _ := strconv.Atoi(matches[5])
			startTrack, _ := strconv.ParseInt(matches[2], 10, 64)
			sizeInTracks, _ := strconv.ParseInt(matches[4], 10, 64)

			part := DasdPartitionInfo{
				PartitionInfo: util.PartitionInfo{Number: partnum},
				StartTrack:    startTrack,
				SizeInTracks:  sizeInTracks,
			}
			diskInfo.Partitions = append(diskInfo.Partitions, part)
		}
	}

	if scanner.Err() != nil {
		return DasdDiskInfo{}, fmt.Errorf("reading fdasd output failed: %v", scanner.Err())
	}
	if diskInfo.Cylinders == 0 || diskInfo.TracksPerCylinder == 0 || diskInfo.BlocksPerTrack == 0 {
		return DasdDiskInfo{}, fmt.Errorf("reading fdasd output failed: could not read disk info - disk may need to be formatted")
	}

	return diskInfo, nil
}

// Allow sorting partitions (must be a stable sort) so partitions are ordered by number
type DasdPartitionList []types.Partition

func (p DasdPartitionList) Len() int {
	return len(p)
}

func (p DasdPartitionList) Less(i, j int) bool {
	return p[i].Number < p[j].Number
}

func (p DasdPartitionList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// partitionDasdDisk partitions devAlias according to the spec given by dev
// since the creation of partitions through fdasd invalidates existing partitions,
// there are some options will are not necessary/ will not make sense for DASD:
//  - wipePartitionEntry : all partitions will be recreated through fdasd anyway,
//                         so there is no point to this option.
//  - matching of partitions to check if it is the same as the original is also
//    not useful as the partition will be newly recreated anyway.
func (s stage) partitionDasdDisk(dev types.Disk, devAlias string) error {
	if dev.WipeTable != nil && *dev.WipeTable {
		s.Logger.Info("wiping partition table requested on %q", devAlias)
		if err := dasdfmt.Format(s.Logger, devAlias); err != nil {
			return err
		}
	} else {
		s.Logger.Info("Note: Adding any new partitions will NOT preserve existing partitions")
	}

	// Ensure all partitions with number 0 are last
	sort.Stable(DasdPartitionList(dev.Partitions))

	op := fdasd.Begin(s.Logger, devAlias)

	infostr, err := op.GetDiskAndPartitionsInfo()
	if err != nil {
		return err
	}

	// use fdasd to get information about the device and the partitions
	diskInfo, err := parseDasdDiskAndPartInfo(infostr)
	if err != nil {
		return err
	}

	// get a list of parititions that have size and start 0 replaced with the real sizes
	// that would be used if all specified partitions were to be created anew.
	// Also change all of the start/size values into tracks.
	resolvedPartitions, err := getDasdRealStartAndSize(dev, diskInfo)
	if err != nil {
		return err
	}

	for _, part := range resolvedPartitions {
		shouldExist := part.ShouldExist
		info, exists := diskInfo.GetDasdPartition(part.Number)
		var matchErr error
		if exists {
			matchErr = dasdPartitionMatches(info, part)
		}
		matches := exists && matchErr == nil
		wipeEntry := part.WipePartitionEntry

		// This is a translation of the matrix in the operator notes.
		switch {
		case !exists && !shouldExist:
			s.Logger.Info("partition %d specified as nonexistant and no partition was found. Success.", part.Number)
		case !exists && shouldExist:
			if err := op.CreatePartition(part.StartTrack, part.SizeInTracks); err != nil {
				return fmt.Errorf("partition %d could not be added %v", part.Number, err)
			}
		case exists && !shouldExist && !wipeEntry:
			return fmt.Errorf("partition %d exists but is specified as nonexistant and wipePartitionEntry is false", part.Number)
		case exists && !shouldExist && wipeEntry:
			s.Logger.Info("partition %d should not exist - will not be created", part.Number)
		case exists && shouldExist && matches:
			s.Logger.Info("partition %d found, will be recreated", part.Number)
			if err := op.CreatePartition(part.StartTrack, part.SizeInTracks); err != nil {
				return fmt.Errorf("partition %d could not be added %v", part.Number, err)
			}
		case exists && shouldExist && !wipeEntry && !matches:
			return fmt.Errorf("Partition %d didn't match: %v", part.Number, matchErr)
		case exists && shouldExist && wipeEntry && !matches:
			s.Logger.Info("partition %d did not meet specifications, wiping partition entry and recreating", part.Number)
			if err := op.CreatePartition(part.StartTrack, part.SizeInTracks); err != nil {
				return fmt.Errorf("partition %d could not be added %v", part.Number, err)
			}
		default:
			// unfortunatey, golang doesn't check that all cases are handled exhaustively
			panic(fmt.Sprintf("Unreachable code reached when processing partition %d. golang--", part.Number))
		}
	}

	if err := op.Commit(); err != nil {
		return fmt.Errorf("commit failure: %v", err)
	}
	return nil
}
