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
	"errors"
	"fmt"
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	cutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/sgdisk"
	iutil "github.com/coreos/ignition/v2/internal/util"
)

var (
	ErrBadSgdiskOutput = errors.New("sgdisk had unexpected output")
)

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
func partitionMatches(existing util.PartitionInfo, spec sgdisk.Partition) error {
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
func partitionMatchesResize(existing util.PartitionInfo, spec sgdisk.Partition) bool {
	return cutil.IsTrue(spec.Resize) && partitionMatchesCommon(existing, spec) == nil
}

// partitionMatchesCommon handles the common tests (excluding the partition size) to determine
// if the existing partition matches the spec given.
func partitionMatchesCommon(existing util.PartitionInfo, spec sgdisk.Partition) error {
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

func convertMiBToSectors(mib *int, sectorSize int) *int64 {
	if mib != nil {
		v := int64(*mib) * (1024 * 1024 / int64(sectorSize))
		return &v
	} else {
		return nil
	}
}

// getRealStartAndSize returns a copy of the given partition configuration with the real partition
// numbers, start sectors, and end sectors filled in. It runs sgdisk --pretend to determine what the
// partitions would look like if everything specified were to be (re)created.
func (s stage) getRealStartAndSize(dev types.Disk, devAlias string, diskInfo util.DiskInfo) ([]sgdisk.Partition, error) {
	used := map[int]bool{}

	// Determine which partition numbers are already used.
	for _, part := range diskInfo.Partitions {
		used[part.Number] = true
	}

	partitions := []sgdisk.Partition{}
	for _, cpart := range dev.Partitions {
		partitions = append(partitions, sgdisk.Partition{
			Partition:     cpart,
			StartSector:   convertMiBToSectors(cpart.StartMiB, diskInfo.LogicalSectorSize),
			SizeInSectors: convertMiBToSectors(cpart.SizeMiB, diskInfo.LogicalSectorSize),
		})
	}

	op := sgdisk.Begin(s.Logger, devAlias)
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
		if part.Number > 0 {
			// Mark the partition number as used or not.
			used[part.Number] = partitionShouldExist(part)
		}
		if partitionShouldExist(part) {
			// Clear the label. sgdisk doesn't escape control characters. This makes parsing easier
			part.Label = nil
			op.CreatePartition(part)
		}
	}

	free := 1
	partitionsToInspect := []int{}
	for i := range partitions {
		part := &partitions[i]
		if partitionShouldExist(*part) {
			// Find the next free partition number and use it.
			if part.Number == 0 {
				for used[free] {
					free++
				}
				part.Number = free
				free++
			}
			// We only care to examine partitions that have start or size 0.
			if part.StartSector == nil || *part.StartSector == 0 ||
				part.SizeInSectors == nil || *part.SizeInSectors == 0 {
				op.Info(part.Number)
				partitionsToInspect = append(partitionsToInspect, part.Number)
			}
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

	for i := range partitions {
		part := &partitions[i]
		if dims, ok := realDimensions[part.Number]; ok {
			part.StartSector = &dims.start
			part.SizeInSectors = &dims.size
		}
	}
	return partitions, nil
}

type sgdiskOutput struct {
	start int64
	size  int64
}

// parseLine takes a regexp that captures an int64 and a string to match on. On success it returns
// the captured int64 and nil. If the regexp does not match it returns -1 and nil. If it encountered
// an error it returns 0 and the error.
func parseLine(r *regexp.Regexp, line string) (int64, error) {
	matches := r.FindStringSubmatch(line)
	switch len(matches) {
	case 0:
		return -1, nil
	case 2:
		return strconv.ParseInt(matches[1], 10, 64)
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
	startRegex := regexp.MustCompile(`^First sector: (\d*) \(.*\)$`)
	endRegex := regexp.MustCompile(`^Last sector: (\d*) \(.*\)$`)
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
func partitionShouldExist(part sgdisk.Partition) bool {
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

// blockDevDMName returns the device mapper name for the given block device,
// or an empty string if it is not a device mapper device.
func blockDevDMName(blockDevResolved string) string {
	blockDevNode := filepath.Base(blockDevResolved)
	dmNameBytes, err := os.ReadFile(fmt.Sprintf("/sys/class/block/%s/dm/name", blockDevNode))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(dmNameBytes))
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
func blockDevPartitions(blockDevResolved string, dmName string) ([]string, error) {
	blockDevNode := filepath.Base(blockDevResolved)

	if dmName != "" {
		// For device mapper (e.g. multipath), partition devices are
		// separate DM nodes. Find them via dm-name symlinks.
		matches, _ := filepath.Glob(fmt.Sprintf("/dev/disk/by-id/dm-name-%sp[0-9]*", dmName))
		var partitions []string
		for _, m := range matches {
			resolved, err := filepath.EvalSymlinks(m)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %q: %v", m, err)
			}
			partitions = append(partitions, resolved)
		}
		return partitions, nil
	}

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
func blockDevInUse(blockDevResolved string, dmName string, skipPartitionCheck bool) (bool, []string, error) {
	// Note: This ignores swap and LVM usage
	inUse := false
	held := false
	if dmName == "" {
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
	partitions, err := blockDevPartitions(blockDevResolved, dmName)
	if err != nil {
		return false, nil, fmt.Errorf("failed to retrieve partitions of %q: %v", blockDevResolved, err)
	}
	var activePartitions []string
	for _, partition := range partitions {
		partInUse, _, err := blockDevInUse(partition, dmName, true)
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

// partitionDevPath returns the expected device path for the given partition
// number on the given disk. For device mapper devices (e.g. multipath), the
// partition devices are separate DM nodes, so we locate them via
// /dev/disk/by-id/dm-name-<name>p<N> symlinks. The returned path may or may
// not exist on disk.
func partitionDevPath(blockDevResolved string, dmName string, prefix string, partNum int) string {
	if dmName == "" {
		return fmt.Sprintf("%s%s%d", blockDevResolved, prefix, partNum)
	}
	return fmt.Sprintf("/dev/disk/by-id/dm-name-%sp%d", dmName, partNum)
}

// dmPartitionStartAndSize returns the start sector and size (in 512-byte
// sectors) of a device mapper partition device by parsing `dmsetup table`.
// The table for a linear DM partition looks like:
//
//	0 <size> linear <major:minor> <start>
func dmPartitionStartAndSize(partDev string) (int64, int64, error) {
	out, err := exec.Command(distro.DmsetupCmd(), "table", partDev).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("dmsetup table failed for %q: %v", partDev, err)
	}
	// Parse: "0 <size> linear <dev> <start>"
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 5 || fields[2] != "linear" {
		return 0, 0, fmt.Errorf("unexpected dmsetup table output for %q: %q", partDev, string(out))
	}
	size, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse size from dmsetup table for %q: %v", partDev, err)
	}
	start, err := strconv.ParseInt(fields[4], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse start from dmsetup table for %q: %v", partDev, err)
	}
	return start, size, nil
}

// partitionDisk partitions devAlias according to the spec given by dev
func (s stage) partitionDisk(dev types.Disk, devAlias string) error {
	blockDevResolved, err := filepath.EvalSymlinks(devAlias)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %v", devAlias, err)
	}

	dmName := blockDevDMName(blockDevResolved)

	inUse, activeParts, err := blockDevInUse(blockDevResolved, dmName, false)
	if err != nil {
		return fmt.Errorf("failed usage check on %q: %v", devAlias, err)
	}
	if inUse && len(activeParts) == 0 {
		return fmt.Errorf("refusing to operate on directly active disk %q", devAlias)
	}
	if cutil.IsTrue(dev.WipeTable) {
		op := sgdisk.Begin(s.Logger, devAlias)
		s.Info("wiping partition table requested on %q", devAlias)
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

	op := sgdisk.Begin(s.Logger, devAlias)

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

	var partxAdd []int
	var partxDelete []int
	var partxUpdate []int

	for _, part := range resolvedPartitions {
		shouldExist := partitionShouldExist(part)
		info, exists := diskInfo.GetPartition(part.Number)
		var matchErr error
		if exists {
			matchErr = partitionMatches(info, part)
		}
		matches := exists && matchErr == nil
		wipeEntry := cutil.IsTrue(part.WipePartitionEntry)

		partDevForCheck := partitionDevPath(blockDevResolved, dmName, prefix, part.Number)
		if resolved, err := filepath.EvalSymlinks(partDevForCheck); err == nil {
			partDevForCheck = resolved
		}
		partInUse := iutil.StrSliceContains(activeParts, partDevForCheck)

		var modification bool

		// This is a translation of the matrix in the operator notes.
		switch {
		case !exists && !shouldExist:
			s.Info("partition %d specified as nonexistant and no partition was found. Success.", part.Number)
		case !exists && shouldExist:
			op.CreatePartition(part)
			modification = true
			partxAdd = append(partxAdd, part.Number)
		case exists && !shouldExist && !wipeEntry:
			return fmt.Errorf("partition %d exists but is specified as nonexistant and wipePartitionEntry is false", part.Number)
		case exists && !shouldExist && wipeEntry:
			op.DeletePartition(part.Number)
			modification = true
			partxDelete = append(partxDelete, part.Number)
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
				partxUpdate = append(partxUpdate, part.Number)
			} else {
				return fmt.Errorf("partition %d didn't match: %v", part.Number, matchErr)
			}
		case exists && shouldExist && wipeEntry && !matches:
			s.Info("partition %d did not meet specifications, wiping partition entry and recreating", part.Number)
			op.DeletePartition(part.Number)
			op.CreatePartition(part)
			modification = true
			partxUpdate = append(partxUpdate, part.Number)
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
	runPartxCommand := func(op string, partitions iter.Seq[int]) {
		for partNr := range partitions {
			// Don't use LogCmd here because we don't want to treat failure as
			// critical and this command will never produce anything on Stdout.
			cmd := exec.Command(distro.PartxCmd(), "--"+op, "--nr", fmt.Sprint(partNr), blockDevResolved)
			s.Info("triggering partition %d %s on %q", partNr, op, devAlias)
			s.Debug("executing: %q", cmd.Args)
			_, err := cmd.Output()
			if err, ok := err.(*exec.ExitError); ok {
				s.Notice("%v: Cmd: %q Stderr: %q", err, cmd.Args, err.Stderr)
			}
		}
	}
	runPartxCommand("delete", slices.Values(partxDelete))
	runPartxCommand("update", slices.Values(partxUpdate))
	runPartxCommand("add", slices.Values(partxAdd))

	// It's best to wait here for the /dev/ABC entries to be
	// (re)created, not only for other parts of the initramfs but
	// also because s.waitOnDevices() can still race with udev's
	// partition entry recreation.
	if err := s.waitForUdev(devAlias); err != nil {
		return fmt.Errorf("failed to wait for udev on %q after partitioning: %v", devAlias, err)
	}

	for _, part := range resolvedPartitions {
		partDev := partitionDevPath(blockDevResolved, dmName, prefix, part.Number)

		if slices.Contains(partxDelete, part.Number) {
			_, err := os.Stat(partDev)
			if err == nil {
				return fmt.Errorf("%q unexpectedly exists after partitioning", partDev)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("failed to stat %q after partitioning: %v", partDev, err)
			}
		} else if slices.Contains(partxAdd, part.Number) || slices.Contains(partxUpdate, part.Number) {
			var kernelStart, kernelSize int64

			// sysfs always reports in 512-byte sectors; convert our expected
			// values from logical sectors to 512-byte sectors for comparison
			logicalTo512 := int64(diskInfo.LogicalSectorSize) / 512

			if dmName != "" {
				// DM partition devices don't have a "start" entry in
				// sysfs. Use dmsetup table to get start and size.
				kernelStart, kernelSize, err = dmPartitionStartAndSize(partDev)
				if err != nil {
					return fmt.Errorf("failed to get DM table for %q: %v", partDev, err)
				}
			} else {
				partDevResolved, err := filepath.EvalSymlinks(partDev)
				if err != nil {
					return fmt.Errorf("failed to resolve %q: %v", partDev, err)
				}
				sysBlockDir := fmt.Sprintf("/sys/class/block/%s/", filepath.Base(partDevResolved))

				startStr, err := os.ReadFile(sysBlockDir + "start")
				if err != nil {
					return fmt.Errorf("failed to read start of %q from sysfs: %v", partDev, err)
				}
				kernelStart, err = strconv.ParseInt(strings.TrimSpace(string(startStr)), 10, 64)
				if err != nil {
					return fmt.Errorf("failed to parse start of %q from sysfs: %v", partDev, err)
				}

				sizeStr, err := os.ReadFile(sysBlockDir + "size")
				if err != nil {
					return fmt.Errorf("failed to read size of %q from sysfs: %v", partDev, err)
				}
				kernelSize, err = strconv.ParseInt(strings.TrimSpace(string(sizeStr)), 10, 64)
				if err != nil {
					return fmt.Errorf("failed to parse size of %q from sysfs: %v", partDev, err)
				}
			}

			if part.StartSector != nil {
				expectedStart := *part.StartSector * logicalTo512
				if kernelStart != expectedStart {
					return fmt.Errorf("kernel partition start for %q does not match expected (%d != %d 512-byte sectors)", partDev, kernelStart, expectedStart)
				}
			}
			if part.SizeInSectors != nil {
				expectedSize := *part.SizeInSectors * logicalTo512
				if kernelSize != expectedSize {
					return fmt.Errorf("kernel partition size for %q does not match expected (%d != %d 512-byte sectors)", partDev, kernelSize, expectedSize)
				}
			}
		}
	}

	return nil
}
