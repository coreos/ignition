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
	lastLBA   int64
}

func Begin(logger *log.Logger, dev string) *Operation {
	return &Operation{logger: logger, dev: dev}
}

func (op *Operation) CreatePartition(p partitioners.Partition) {
	op.parts = append(op.parts, p)
}

func (op *Operation) DeletePartition(num int) {
	op.deletions = append(op.deletions, num)
}

func (op *Operation) Info(num int) {
	op.infos = append(op.infos, num)
}

func (op *Operation) WipeTable(wipe bool) {
	op.wipe = wipe
}

func (op *Operation) NeedsPartx() bool {
	return false
}

func (op *Operation) WritesCompleteTable() bool {
	return true
}

// ensureGPTLabel creates an empty GPT label on the device if one does
// not already exist. This is needed so that getLastLBA can read
// last-lba from the GPT header and so that sfdisk --no-act works.
func (op *Operation) ensureGPTLabel() {
	cmd := exec.Command(distro.SfdiskCmd(), "--dump", op.dev)
	if out, err := cmd.Output(); err == nil && strings.Contains(string(out), "label:") {
		return
	}
	if op.logger != nil {
		op.logger.Info("creating empty GPT label on %q", op.dev)
	}
	labelCmd := exec.Command(distro.SfdiskCmd(), "-X", "gpt", op.dev)
	labelCmd.Stdin = strings.NewReader("label: gpt\n")
	if output, err := labelCmd.CombinedOutput(); err != nil {
		if op.logger != nil {
			op.logger.Warning("failed to create GPT label on %q: %v: %s", op.dev, err, string(output))
		}
	}
}

// getLastLBA reads the last usable LBA from the device's GPT header.
func (op *Operation) getLastLBA() int64 {
	op.ensureGPTLabel()
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

// readExistingPartitions reads the current partition table using
// sfdisk --dump.
func (op *Operation) readExistingPartitions() ([]partitioners.Partition, error) {
	cmd := exec.Command(distro.SfdiskCmd(), "--dump", op.dev)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var partitions []partitioners.Partition
	lineRegex := regexp.MustCompile(`^(/\S+)\s*:\s*(.*)$`)
	numRegex := regexp.MustCompile(`(\d+)$`)

	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(line)
		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		numMatch := numRegex.FindStringSubmatch(matches[1])
		if numMatch == nil {
			continue
		}
		partNum, err := strconv.Atoi(numMatch[1])
		if err != nil {
			continue
		}

		p := partitioners.Partition{}
		p.Number = partNum

		for _, field := range strings.Split(matches[2], ",") {
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
				p.Label = util.StrToPtr(strings.Trim(value, "\""))
			}
		}

		partitions = append(partitions, p)
	}

	return partitions, nil
}

func writePartitionLine(script *bytes.Buffer, p partitioners.Partition) {
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

// buildScript constructs an sfdisk script. Fixed-size partitions are
// written first (sorted by start sector) so that fill-remaining
// partitions (size=+) see the correct free blocks.
func buildScript(partitions []partitioners.Partition, lastLBA int64) string {
	script := &bytes.Buffer{}
	script.WriteString("label: gpt\n")
	script.WriteString("grain: 512\n")
	if lastLBA >= 0 {
		fmt.Fprintf(script, "last-lba: %d\n", lastLBA)
	}
	script.WriteString("\n")

	sorted := make([]partitioners.Partition, len(partitions))
	copy(sorted, partitions)
	sort.SliceStable(sorted, func(i, j int) bool {
		iFixed := sorted[i].SizeInSectors != nil && *sorted[i].SizeInSectors != 0
		jFixed := sorted[j].SizeInSectors != nil && *sorted[j].SizeInSectors != 0
		if iFixed != jFixed {
			return iFixed
		}
		if iFixed && jFixed {
			iStart := int64(0)
			jStart := int64(0)
			if sorted[i].StartSector != nil {
				iStart = *sorted[i].StartSector
			}
			if sorted[j].StartSector != nil {
				jStart = *sorted[j].StartSector
			}
			return iStart < jStart
		}
		return false
	})

	for _, p := range sorted {
		writePartitionLine(script, p)
	}

	return script.String()
}

// mergePartitions builds the final partition list from existing disk
// state plus queued operations.
func (op *Operation) mergePartitions() ([]partitioners.Partition, error) {
	var merged []partitioners.Partition

	if !op.wipe {
		existing, err := op.readExistingPartitions()
		if err != nil {
			return nil, err
		}
		merged = existing
	}

	for _, delNum := range op.deletions {
		var filtered []partitioners.Partition
		for _, p := range merged {
			if p.Number != delNum {
				filtered = append(filtered, p)
			}
		}
		merged = filtered
	}

	for _, p := range op.parts {
		if p.Number == 0 {
			merged = append(merged, p)
		} else {
			replaced := false
			for i := range merged {
				if merged[i].Number == p.Number {
					merged[i] = p
					replaced = true
					break
				}
			}
			if !replaced {
				merged = append(merged, p)
			}
		}
	}

	return merged, nil
}

func (op *Operation) Pretend() (string, error) {
	merged, err := op.mergePartitions()
	if err != nil {
		return "", err
	}

	op.lastLBA = op.getLastLBA()
	scriptContent := buildScript(merged, op.lastLBA)

	if op.logger != nil {
		op.logger.Info("running sfdisk --no-act with script:\n%s", scriptContent)
	}

	cmd := exec.Command(distro.SfdiskCmd(), "--no-act", "--wipe-partitions", "always", "-X", "gpt", op.dev)
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

func (op *Operation) Commit() error {
	if op.wipe {
		if op.logger != nil {
			op.logger.Info("wiping partition table on %q", op.dev)
		}
		cmd := exec.Command(distro.SfdiskCmd(), "--wipe", "always", "--label", "gpt", op.dev)
		cmd.Stdin = strings.NewReader("label: gpt\n")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to wipe partition table on %q: %v: %s", op.dev, err, string(output))
		}
	}

	merged, err := op.mergePartitions()
	if err != nil {
		return err
	}

	if len(merged) == 0 {
		return nil
	}

	if err := op.handleInfo(); err != nil {
		return err
	}

	lastLBA := op.getLastLBA()
	scriptContent := buildScript(merged, lastLBA)

	if op.logger != nil {
		op.logger.Info("running sfdisk with script:\n%s", scriptContent)
	}

	cmd := exec.Command(distro.SfdiskCmd(), "--wipe", "auto", "--wipe-partitions", "always", "-X", "gpt", op.dev)
	cmd.Stdin = strings.NewReader(scriptContent)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sfdisk failed on %q: %v: %s", op.dev, err, string(output))
	}

	return nil
}

func (op *Operation) ParseOutput(sfdiskOutput string, partitionNumbers []int) (map[int]partitioners.Output, error) {
	preamble := sfdiskOutput
	if idx := strings.Index(sfdiskOutput, "Device"); idx >= 0 {
		preamble = sfdiskOutput[:idx]
	}
	lower := strings.ToLower(preamble)
	if strings.Contains(lower, "failed") || strings.Contains(lower, "error") {
		return nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadSfdiskPretend, sfdiskOutput)
	}

	result := make(map[int]partitioners.Output)

	partitionRegex := regexp.MustCompile(`^/\S+\s+\*?\s*(\d+)\s+(\d+)\s+(\d+)\s+`)
	numRegex := regexp.MustCompile(`^(/\S+)`)
	devNumRegex := regexp.MustCompile(`(\d+)$`)

	for _, line := range strings.Split(sfdiskOutput, "\n") {
		line = strings.TrimSpace(line)
		matches := partitionRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		devMatch := numRegex.FindStringSubmatch(line)
		if devMatch == nil {
			continue
		}
		partNumMatch := devNumRegex.FindStringSubmatch(devMatch[1])
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

		// sfdisk's fill-remaining (size=+) ends 1 sector before lastLBA.
		// Correct this to match sgdisk which fills to the exact lastLBA.
		if op.lastLBA > 0 && end == op.lastLBA-1 {
			size++
		}

		result[partNum] = partitioners.Output{
			Start: start,
			Size:  size,
		}
	}

	return result, nil
}

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
