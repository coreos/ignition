// Copyright 2024 Red Hat
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

package sfdisk

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	sharedErrors "github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/partitioners"
)

type Operation struct {
	logger    *log.Logger
	dev       string
	wipe      bool
	parts     []partitioners.Partition
	deletions []int
	infos     []int
}

// Begin begins an sfdisk operation
func Begin(logger *log.Logger, dev string) *Operation {
	return &Operation{logger: logger, dev: dev}
}

// CreatePartition adds the supplied partition to the list of partitions to be created as part of an operation
func (op *Operation) CreatePartition(p partitioners.Partition) {
	op.parts = append(op.parts, p)
}

func (op *Operation) DeletePartition(num int) {
	op.deletions = append(op.deletions, num)
}

func (op *Operation) Info(num int) {
	op.infos = append(op.infos, num)
}

// WipeTable toggles if the table is to be wiped first when committing this operation.
func (op *Operation) WipeTable(wipe bool) {
	op.wipe = wipe
}

// readExistingPartitions reads the current partition table using sfdisk --dump
func (op *Operation) readExistingPartitions() (map[int]partitioners.Partition, error) {
	cmd := exec.Command(distro.SfdiskCmd(), "--dump", op.dev)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the disk has no partition table, that's okay
		return make(map[int]partitioners.Partition), nil
	}

	partitions := make(map[int]partitioners.Partition)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, comments, and the label line
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "label:") ||
			strings.HasPrefix(line, "label-id:") || strings.HasPrefix(line, "device:") ||
			strings.HasPrefix(line, "unit:") || strings.HasPrefix(line, "first-lba:") ||
			strings.HasPrefix(line, "last-lba:") || strings.HasPrefix(line, "sector-size:") {
			continue
		}

		// Parse partition line format: "/dev/sdX1 : start=XXX, size=XXX, type=XXX, uuid=XXX, name=XXX"
		// Or numbered format: "1 : start=XXX, size=XXX, type=XXX, uuid=XXX, name=XXX"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		// Extract partition number from the first part
		numStr := strings.TrimSpace(parts[0])
		// Handle /dev/sdX1 format or just "1" format
		var partNum int
		if strings.Contains(numStr, "/dev/") {
			// Extract number from device name (e.g., /dev/sda1 -> 1, /dev/nvme0n1p2 -> 2)
			re := regexp.MustCompile(`\d+$`)
			match := re.FindString(numStr)
			if match == "" {
				continue
			}
			partNum, err = strconv.Atoi(match)
			if err != nil {
				continue
			}
		} else {
			partNum, err = strconv.Atoi(numStr)
			if err != nil {
				continue
			}
		}

		// Parse the attributes
		attrs := strings.TrimSpace(parts[1])
		p := partitioners.Partition{
			Partition: types.Partition{
				Number: partNum,
			},
		}

		// Parse key=value pairs
		kvPairs := strings.Split(attrs, ",")
		for _, kv := range kvPairs {
			kv = strings.TrimSpace(kv)
			kvParts := strings.SplitN(kv, "=", 2)
			if len(kvParts) != 2 {
				continue
			}
			key := strings.TrimSpace(kvParts[0])
			value := strings.TrimSpace(kvParts[1])
			// Remove quotes from value if present
			value = strings.Trim(value, "\"")

			switch key {
			case "start":
				if start, err := strconv.ParseInt(value, 10, 64); err == nil {
					p.StartSector = &start
				}
			case "size":
				if size, err := strconv.ParseInt(value, 10, 64); err == nil {
					p.SizeInSectors = &size
				}
			case "type":
				p.TypeGUID = &value
			case "uuid":
				p.GUID = &value
			case "name":
				if value != "" {
					p.Label = &value
				}
			}
		}

		partitions[partNum] = p
	}

	return partitions, nil
}

// buildCompleteScript builds a complete partition table script from a partition map
func (op *Operation) buildCompleteScript(partitions map[int]partitioners.Partition) string {
	var script bytes.Buffer

	// sfdisk script mode requires a header
	script.WriteString("label: gpt\n")

	// Sort partition numbers for consistent output
	// Separate numbered partitions from auto-numbered (partition 0)
	var numberedParts []int
	var autoParts []partitioners.Partition

	for num, p := range partitions {
		if num == 0 || p.Number == 0 {
			// Partition with number 0 means auto-assign
			autoParts = append(autoParts, p)
		} else {
			numberedParts = append(numberedParts, num)
		}
	}
	sort.Ints(numberedParts)

	// Write numbered partitions first
	for _, num := range numberedParts {
		p := partitions[num]
		op.writePartitionLine(&script, p)
	}

	// Write auto-numbered partitions last
	for _, p := range autoParts {
		op.writePartitionLine(&script, p)
	}

	return script.String()
}

// writePartitionLine writes a single partition line to the script
func (op *Operation) writePartitionLine(script *bytes.Buffer, p partitioners.Partition) {
	if p.Number != 0 {
		fmt.Fprintf(script, "%d : ", p.Number)
	} else {
		// For partition number 0, let sfdisk auto-assign the next available number
		script.WriteString(": ")
	}

	if p.StartSector != nil && *p.StartSector != 0 {
		fmt.Fprintf(script, "start=%d ", *p.StartSector)
	}

	if p.SizeInSectors != nil && *p.SizeInSectors != 0 {
		fmt.Fprintf(script, "size=%d ", *p.SizeInSectors)
	} else if p.SizeInSectors != nil && *p.SizeInSectors == 0 {
		// Use size=+ to fill remaining space (like sgdisk "+0")
		script.WriteString("size=+ ")
	}

	if util.NotEmpty(p.TypeGUID) {
		fmt.Fprintf(script, "type=%s ", *p.TypeGUID)
	}

	if util.NotEmpty(p.GUID) {
		fmt.Fprintf(script, "uuid=%s ", *p.GUID)
	}

	if p.Label != nil {
		fmt.Fprintf(script, "name=\"%s\" ", *p.Label)
	}

	script.WriteString("\n")
}

// Pretend is like Commit() but uses the --no-act flag and returns the output
func (op *Operation) Pretend() (string, error) {
	if err := op.handleInfo(); err != nil {
		return "", err
	}

	// Read existing partitions
	existingParts := make(map[int]partitioners.Partition)
	if !op.wipe {
		var err error
		existingParts, err = op.readExistingPartitions()
		if err != nil {
			return "", fmt.Errorf("failed to read existing partitions: %v", err)
		}
	}

	// Apply deletions
	for _, delNum := range op.deletions {
		delete(existingParts, delNum)
	}

	// Apply creations/modifications
	for _, p := range op.parts {
		existingParts[p.Number] = p
	}

	// Build complete partition table script
	script := op.buildCompleteScript(existingParts)

	cmd := exec.Command(distro.SfdiskCmd(), "--no-act", "-X", "gpt", op.dev)
	cmd.Stdin = strings.NewReader(script)
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}
	output, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	errors, err := io.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to pretend to create partitions. Err: %v. Stderr: %v", err, string(errors))
	}

	return string(output), nil
}

// Commit commits an partitioning operation using read-modify-write approach.
func (op *Operation) Commit() error {
	// If wipe we need to reset the partition table
	if op.wipe {
		// Create a fresh GPT disk label and wipe signatures using sfdisk directly
		// Note: sfdisk returns exit code 1 when the partition table is changed, which is expected
		cmd := exec.Command(distro.SfdiskCmd(), "--wipe", "always", "--label", "gpt", op.dev)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Check if sfdisk succeeded despite exit code 1 by looking for "Done" in output
			if !strings.Contains(string(output), "Done") {
				return fmt.Errorf("wipe partition table failed: %v, output: %s", err, string(output))
			}
			// Exit code 1 with "Done" in output means success for sfdisk
			op.logger.Info("wiping partition table on %q", op.dev)
		} else {
			op.logger.Info("wiping partition table on %q", op.dev)
		}
	}

	// Read existing partitions (unless we just wiped)
	existingParts := make(map[int]partitioners.Partition)
	if !op.wipe {
		var err error
		existingParts, err = op.readExistingPartitions()
		if err != nil {
			return fmt.Errorf("failed to read existing partitions: %v", err)
		}
	}

	// Apply deletions
	for _, delNum := range op.deletions {
		delete(existingParts, delNum)
	}

	// Apply creations/modifications
	for _, p := range op.parts {
		existingParts[p.Number] = p
	}

	// If no partitions remain, we're done
	if len(existingParts) == 0 && len(op.parts) == 0 && len(op.deletions) == 0 {
		return nil
	}

	// Build complete partition table script
	script := op.buildCompleteScript(existingParts)

	// Write complete table
	if err := op.runSfdisk(script, true); err != nil {
		return fmt.Errorf("sfdisk commit failed: %v", err)
	}

	// Give the kernel time to process attribute changes and flush block device cache
	time.Sleep(100 * time.Millisecond)

	// Force block device cache flush so blkid sees updated metadata
	cmd := exec.Command("blockdev", "--flushbufs", op.dev)
	if _, err := op.logger.LogCmd(cmd, "flushing block device cache for %q", op.dev); err != nil {
		op.logger.Warning("failed to flush block device cache for %q: %v", op.dev, err)
	}

	return nil
}

func (op *Operation) runSfdisk(script string, shouldWrite bool) error {
	var opts []string
	if !shouldWrite {
		opts = append(opts, "--no-act")
	}
	// Always target GPT and wipe conflicting signatures to match sgdisk behavior
	opts = append(opts, "--wipe", "always", "-X", "gpt", op.dev)

	cmd := exec.Command(distro.SfdiskCmd(), opts...)
	cmd.Stdin = strings.NewReader(script)
	if _, err := op.logger.LogCmd(cmd, "deleting %d partitions and creating %d partitions on %q", len(op.deletions), len(op.parts), op.dev); err != nil {
		return fmt.Errorf("create partitions failed: %v", err)
	}

	return nil
}

// ParseOutput takes the output from sfdisk. Similarly to sgdisk
// it then uses regex to parse the output into understood values like 'start' 'size' and attempts
// to catch any failures and wrap them to return to the caller.
func (op *Operation) ParseOutput(sfdiskOutput string, partitionNumbers []int) (map[int]partitioners.Output, error) {
	if len(partitionNumbers) == 0 {
		return nil, nil
	}

	// Look for new lines starting with /dev/ and the following string it
	// Additionally Group on Start sector, and End sector
	// Example output match would be "/dev/vda1        2048  2057      10   5K 83 Linux"
	partitionRegex := regexp.MustCompile(`^/dev/\S+\s+\S*\s+(\d+)\s+(\d+)\s+\d+\s+\S+\s+\S+\s+\S+.*$`)
	output := map[int]partitioners.Output{}
	current := partitioners.Output{}
	i := 0
	lines := strings.Split(sfdiskOutput, "\n")
	for _, line := range lines {
		matches := partitionRegex.FindStringSubmatch(line)

		// Sanity check number of partition entries
		if i > len(partitionNumbers) {
			return nil, sharedErrors.ErrBadSfdiskPretend
		}

		// Verify that we are not reading a 'failed' or 'error'
		errorRegex := regexp.MustCompile(`(?i)(failed|error)`)
		if errorRegex.MatchString(line) {
			return nil, fmt.Errorf("%w: sfdisk returned :%v", sharedErrors.ErrBadSfdiskPretend, line)
		}

		// When we get a match it should be
		// Whole line at [0]
		// Start at [1]
		// End at [2]
		if len(matches) > 2 {
			start, err := strconv.Atoi(matches[1])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(matches[2])
			if err != nil {
				return nil, err
			}

			current.Start = int64(start)
			// Add one due to overlap
			current.Size = int64(end - start + 1)
			output[partitionNumbers[i]] = current
			i++
		}
	}

	return output, nil
}

func (op *Operation) handleInfo() error {
	for _, num := range op.infos {
		cmd := exec.Command(distro.SfdiskCmd(), "--list", op.dev)
		op.logger.Info("retrieving information for partition %d on %q", num, op.dev)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to retrieve partition info for %d: %v, output: %s", num, err, output)
		}
		op.logger.Info("partition info: %s", output)
	}
	return nil
}
