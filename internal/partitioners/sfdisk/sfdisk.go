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

// Pretend is like Commit() but uses the --no-act flag and returns the output
func (op *Operation) Pretend() (string, error) {
	// Handle deletions first
	if err := op.handleDeletions(); err != nil {
		return "", err
	}

	if err := op.handleInfo(); err != nil {
		return "", err
	}

	script := op.buildOptions()
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo -e \"%s\" | sudo %s --no-act %s", script, distro.SfdiskCmd(), op.dev))
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

// Commit commits an partitioning operation.
func (op *Operation) Commit() error {
	println("Commit: HERE")
	fmt.Println(op.parts)
	script := op.buildOptions()
	if len(script) == 0 {
		return nil
	}

	// If wipe we need to reset the partition table
	if op.wipe {
		// Erase the existing partition tables
		cmd := exec.Command(distro.WipefsCmd(), "-a", op.dev)
		if _, err := op.logger.LogCmd(cmd, "option wipe selected, and failed to execute on %q", op.dev); err != nil {
			return fmt.Errorf("wipe partition table failed: %v", err)
		}
	}

	if err := op.runSfdisk(true); err != nil {
		return fmt.Errorf("sfdisk commit failed with: %v", err)
	}

	return nil
}

func (op *Operation) runSfdisk(shouldWrite bool) error {
	var opts []string
	if !shouldWrite {
		opts = append(opts, "--no-act")
	}
	opts = append(opts, "-X", "gpt", op.dev)
	fmt.Printf("The options are %v", opts)
	cmd := exec.Command(distro.SfdiskCmd(), opts...)
	cmd.Stdin = strings.NewReader(op.buildOptions())
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

func (op Operation) buildOptions() string {
	var script bytes.Buffer

	for _, p := range op.parts {
		println("Starting Build Options Script Building")

		fmt.Println(p)
		println(script.String())

		if p.Number != 0 {
			script.WriteString(fmt.Sprintf("%d : ", p.Number))
		}

		if p.StartSector != nil {
			script.WriteString(fmt.Sprintf("start=%d ", *p.StartSector))

		}

		if p.SizeInSectors != nil {
			script.WriteString(fmt.Sprintf("size=%d ", *p.SizeInSectors))
		}

		if util.NotEmpty(p.TypeGUID) {
			script.WriteString(fmt.Sprintf("type=%s ", *p.TypeGUID))
		}

		if util.NotEmpty(p.GUID) {
			script.WriteString(fmt.Sprintf("uuid=%s ", *p.GUID))
		}

		if p.Label != nil {
			script.WriteString(fmt.Sprintf("name=%s ", *p.Label))
		}

		// Add escaped new line to allow for 1 or more partitions
		// i.e "1: size=50 \\n size=10" will result in part 1, and 2
		script.WriteString("\n")
		println("here!")
		println(script.String())

	}

	return script.String()
}

func (op *Operation) handleDeletions() error {
	for _, num := range op.deletions {
		cmd := exec.Command(distro.SfdiskCmd(), "--delete", op.dev, fmt.Sprintf("%d", num))
		op.logger.Info("running sfdisk to delete partition %d on %q", num, op.dev)

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to delete partition %d: %v, output: %s", num, err, output)
		}
	}
	return nil
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
