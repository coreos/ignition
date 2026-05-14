// Copyright 2024 Red Hat, Inc.
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

	sharedErrors "github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
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

// CreatePartition adds the supplied partition to the list of partitions to be created as part of an operation.
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

func (op *Operation) NeedsPartx() bool {
	return false
}

func (op *Operation) WritesCompleteTable() bool {
	return true
}

// getLastLBA reads the last usable LBA from the device's GPT header using
// sfdisk --dump. Returns -1 if unavailable.
func (op *Operation) getLastLBA() int64 {
	cmd := exec.Command(distro.SfdiskCmd(), "--dump", op.dev)
	stdout, err := cmd.Output()
	if err != nil {
		return -1
	}
	re := regexp.MustCompile(`(?m)^last-lba:\s*(\d+)`)
	match := re.FindSubmatch(stdout)
	if len(match) >= 2 {
		v, err := strconv.ParseInt(string(match[1]), 10, 64)
		if err == nil {
			return v
		}
	}
	return -1
}

// readExistingPartitions reads the current partition table from the device
// using sfdisk --dump. Returns a map of partition number to Partition.
func (op *Operation) readExistingPartitions() (map[int]partitioners.Partition, error) {
	cmd := exec.Command(distro.SfdiskCmd(), "--dump", op.dev)
	stdout, err := cmd.Output()
	if err != nil {
		// If sfdisk fails (e.g., no partition table), return empty map
		return map[int]partitioners.Partition{}, nil
	}

	partitions := make(map[int]partitioners.Partition)
	// Match lines like: /dev/sda1 : start=     2048, size=    65536, type=..., uuid=..., name="..."
	lineRegex := regexp.MustCompile(`^(/dev/\S+)\s*:\s*(.*)$`)
	// Extract partition number from device name: /dev/sda1 -> 1, /dev/nvme0n1p2 -> 2
	numRegex := regexp.MustCompile(`(\d+)$`)

	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(line)
		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		devName := matches[1]
		attrs := matches[2]

		numMatch := numRegex.FindStringSubmatch(devName)
		if numMatch == nil {
			continue
		}
		partNum, err := strconv.Atoi(numMatch[1])
		if err != nil {
			continue
		}

		p := partitioners.Partition{}
		p.Number = partNum

		// Parse key=value pairs from the attributes
		for _, field := range strings.Split(attrs, ",") {
			field = strings.TrimSpace(field)
			kv := strings.SplitN(field, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "start":
				v, err := strconv.ParseInt(value, 10, 64)
				if err == nil {
					p.StartSector = &v
				}
			case "size":
				v, err := strconv.ParseInt(value, 10, 64)
				if err == nil {
					p.SizeInSectors = &v
				}
			case "type":
				p.TypeGUID = util.StrToPtr(value)
			case "uuid":
				p.GUID = util.StrToPtr(value)
			case "name":
				// Remove surrounding quotes
				value = strings.Trim(value, "\"")
				p.Label = util.StrToPtr(value)
			}
		}

		partitions[partNum] = p
	}

	return partitions, nil
}

// writePartitionLine writes a single partition line to the script buffer.
// lastLBA is the last usable sector; if >= 0, fill-remaining partitions get an
// explicit size matching sgdisk's inclusive interpretation of last-lba.
func writePartitionLine(script *bytes.Buffer, p partitioners.Partition, lastLBA int64) {
	var line bytes.Buffer

	if p.Number > 0 {
		fmt.Fprintf(&line, "%d :", p.Number)
	} else {
		line.WriteString(":")
	}

	if p.StartSector != nil && *p.StartSector != 0 {
		fmt.Fprintf(&line, " start=%d,", *p.StartSector)
	}

	if p.SizeInSectors != nil && *p.SizeInSectors != 0 {
		fmt.Fprintf(&line, " size=%d,", *p.SizeInSectors)
	} else if lastLBA >= 0 && p.StartSector != nil && *p.StartSector > 0 {
		// sfdisk's "fill remaining" leaves 1 sector unused compared to
		// sgdisk.  Compute the exact size to match sgdisk behaviour.
		fmt.Fprintf(&line, " size=%d,", lastLBA-*p.StartSector+1)
	} else {
		line.WriteString(" size=+,")
	}

	if util.NotEmpty(p.TypeGUID) {
		fmt.Fprintf(&line, " type=%s,", *p.TypeGUID)
	}

	if util.NotEmpty(p.GUID) {
		fmt.Fprintf(&line, " uuid=%s,", *p.GUID)
	}

	if p.Label != nil {
		fmt.Fprintf(&line, " name=\"%s\",", *p.Label)
	}

	script.WriteString(strings.TrimSuffix(line.String(), ","))
	script.WriteString("\n")
}

// buildCompleteScript constructs an sfdisk script from the given partition map.
// lastLBA is forwarded to writePartitionLine for fill-remaining size
// calculation; pass -1 if unknown.
func buildCompleteScript(partitions map[int]partitioners.Partition, lastLBA int64) string {
	script := &bytes.Buffer{}
	script.WriteString("label: gpt\n")
	script.WriteString("grain: 512\n\n")

	// Sort partition numbers: numbered partitions first (ascending), then auto-numbered (0)
	numbers := make([]int, 0, len(partitions))
	for num := range partitions {
		numbers = append(numbers, num)
	}
	sort.Slice(numbers, func(i, j int) bool {
		if numbers[i] == 0 {
			return false
		}
		if numbers[j] == 0 {
			return true
		}
		return numbers[i] < numbers[j]
	})

	for _, num := range numbers {
		p := partitions[num]
		writePartitionLine(script, p, lastLBA)
	}

	return script.String()
}

// Pretend is like Commit() but uses the --no-act flag and returns the output
// on stdout for parsing.
func (op *Operation) Pretend() (string, error) {
	var existing map[int]partitioners.Partition
	var err error

	if op.wipe {
		existing = make(map[int]partitioners.Partition)
	} else {
		existing, err = op.readExistingPartitions()
		if err != nil {
			return "", err
		}
	}

	// Apply deletions
	for _, num := range op.deletions {
		delete(existing, num)
	}

	// Apply creations
	for _, p := range op.parts {
		existing[p.Number] = p
	}

	lastLBA := op.getLastLBA()
	scriptContent := buildCompleteScript(existing, lastLBA)

	if op.logger != nil {
		op.logger.Info("running sfdisk --no-act with script:\n%s", scriptContent)
	}

	cmd := exec.Command(distro.SfdiskCmd(), "--no-act", "-X", "gpt", op.dev)
	cmd.Stdin = strings.NewReader(scriptContent)

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

// Commit commits a partitioning operation.
func (op *Operation) Commit() error {
	var existing map[int]partitioners.Partition

	if op.wipe {
		// Create empty GPT table first
		if op.logger != nil {
			op.logger.Info("wiping partition table on %q", op.dev)
		}
		cmd := exec.Command(distro.SfdiskCmd(), "--wipe", "always", "--label", "gpt", op.dev)
		cmd.Stdin = strings.NewReader("label: gpt\n")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to wipe partition table on %q: %v: %s", op.dev, err, string(output))
		}
		existing = make(map[int]partitioners.Partition)
	} else {
		// Read the current table so that partitions present on disk but
		// not referenced in the ignition config are preserved.  The
		// caller already passes matching/modified partitions via
		// CreatePartition, but unreferenced ones would be lost without
		// this read-modify-write.
		var err error
		existing, err = op.readExistingPartitions()
		if err != nil {
			return err
		}
	}

	// Apply deletions
	for _, num := range op.deletions {
		delete(existing, num)
	}

	// Apply creations
	for _, p := range op.parts {
		existing[p.Number] = p
	}

	if len(existing) == 0 && !op.wipe {
		return nil
	}

	// Handle info requests
	if err := op.handleInfo(); err != nil {
		return err
	}

	if len(existing) == 0 {
		return nil
	}

	lastLBA := op.getLastLBA()
	scriptContent := buildCompleteScript(existing, lastLBA)

	if op.logger != nil {
		op.logger.Info("running sfdisk with script:\n%s", scriptContent)
	}

	cmd := exec.Command(distro.SfdiskCmd(), "--wipe", "auto", "-X", "gpt", op.dev)
	cmd.Stdin = strings.NewReader(scriptContent)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sfdisk failed on %q: %v: %s", op.dev, err, string(output))
	}

	return nil
}

// ParseOutput parses the table-format output from sfdisk (the "New situation:"
// section after --no-act) and returns a map of partition number to Output.
func (op *Operation) ParseOutput(sfdiskOutput string, partitionNumbers []int) (map[int]partitioners.Output, error) {
	// Check for error/failure keywords only in the preamble before the
	// partition table.  The table itself can contain type names like
	// "Linux filesystem" that are harmless, and user-supplied partition
	// labels could also contain "error" or "failed".
	preamble := sfdiskOutput
	if idx := strings.Index(sfdiskOutput, "Device"); idx >= 0 {
		preamble = sfdiskOutput[:idx]
	}
	lower := strings.ToLower(preamble)
	if strings.Contains(lower, "failed") || strings.Contains(lower, "error") {
		return nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadSfdiskPretend, sfdiskOutput)
	}

	result := make(map[int]partitioners.Output)

	// Match lines like: /dev/vda1   2048  67583   65536  32M Linux filesystem
	// Device, optional boot flag (*), start, end, sectors, size, type
	partitionRegex := regexp.MustCompile(`^/dev/\S+\s+\*?\s*(\d+)\s+(\d+)\s+(\d+)\s+`)

	// Extract partition number from device name
	numRegex := regexp.MustCompile(`^(/dev/\S+)`)
	devNumRegex := regexp.MustCompile(`(\d+)$`)

	for _, line := range strings.Split(sfdiskOutput, "\n") {
		line = strings.TrimSpace(line)
		matches := partitionRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		// Get device name to extract partition number
		devMatch := numRegex.FindStringSubmatch(line)
		if devMatch == nil {
			continue
		}
		devName := devMatch[1]
		partNumMatch := devNumRegex.FindStringSubmatch(devName)
		if partNumMatch == nil {
			continue
		}
		partNum, err := strconv.Atoi(partNumMatch[1])
		if err != nil {
			continue
		}

		start, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			continue
		}

		end, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			continue
		}

		size := end - start + 1

		result[partNum] = partitioners.Output{
			Start: start,
			Size:  size,
		}
	}

	return result, nil
}

// handleInfo logs partition information for each requested partition number.
func (op *Operation) handleInfo() error {
	if len(op.infos) == 0 {
		return nil
	}

	cmd := exec.Command(distro.SfdiskCmd(), "--list", op.dev)
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("sfdisk --list failed on %q: %v", op.dev, err)
	}

	if op.logger != nil {
		op.logger.Info("partition info for %q (requested partitions %v):\n%s", op.dev, op.infos, string(stdout))
	}

	return nil
}
