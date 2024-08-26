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

	"github.com/coreos/ignition/v2/config/v3_5_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
)

type Operation struct {
	logger    *log.Logger
	dev       string
	wipe      bool
	parts     []Partition
	deletions []int
	infos     []int
}

// We ignore types.Partition.StartMiB/SizeMiB in favor of
// StartSector/SizeInSectors.  The caller is expected to do the conversion.
type Partition struct {
	types.Partition
	StartSector   *int64
	SizeInSectors *int64

	// shadow StartMiB/SizeMiB so they're not accidentally used
	StartMiB string
	SizeMiB  string
}

// Begin begins an sfdisk operation
func Begin(logger *log.Logger, dev string) *Operation {
	return &Operation{logger: logger, dev: dev}
}

// CreatePartition adds the supplied partition to the list of partitions to be created as part of an operation.
func (op *Operation) CreatePartition(p Partition) {
	op.parts = append(op.parts, p)
}

func (op *Operation) DeletePartition(num int) {
	op.deletions = append(op.deletions, num)
}

func (op *Operation) Info(num int) {
	op.infos = append(op.infos, num)
}

// WipeTable toggles if the table is to be wiped first when commiting this operation.
func (op *Operation) WipeTable(wipe bool) {
	op.wipe = wipe
}

// Pretend is like Commit() but uses the --no-act flag and returns the output
// on stdout for parsing.
func (op *Operation) Pretend() (string, error) {
	// Handle deletions first
	if err := op.handleDeletions(); err != nil {
		return "", err
	}

	// Handle info requests
	if err := op.handleInfo(); err != nil {
		return "", err
	}

	// Build the sfdisk script
	script := op.buildOptions()

	// Verify how to pass a script to sfdisk...
	cmd := exec.Command(distro.SfdiskCmd(), "--no-act", op.dev)
	cmd.Stdin = bytes.NewBufferString(script)
	println("sfdisk --no-act %w, with the script s", op.dev, script)

	// Capture the output and error streams
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
	// Build the partition script
	script := op.buildOptions()
	if len(script) == 0 {
		return nil // No operations to perform
	}

	op.logger.Info("running sfdisk with script: %v", script)
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' | sfdisk %s", script, op.dev))
	if _, err := op.logger.LogCmd(cmd, "deleting %d partitions and creating %d partitions on %q", len(op.deletions), len(op.parts), op.dev); err != nil {
		return fmt.Errorf("create partitions failed: %v", err)
	}

	return nil
}

func (op Operation) buildOptions() string {
	var script bytes.Buffer

	// If wipe we need to reset the partition table
	if op.wipe {
		op.logger.Err("WipeTable not implemented while using backing sfdisk")
	}
	// Always set the label to 'gpt', not sure if this is needed, currently causes an error without it
	script.WriteString("label: gpt\n")

	// Create partition entries in the script
	for _, p := range op.parts {
		// Default start sector and size
		start := "2048"
		size := "0"

		if p.StartSector != nil {
			if *p.StartSector < int64(2048) {
				op.logger.Info("StartSector is less than 2048, setting to 2048")
			}

			start = fmt.Sprintf("%d", *p.StartSector)
		}

		if p.SizeInSectors != nil {
			size = fmt.Sprintf("%d", *p.SizeInSectors)
		}

		// Build the partition entry
		script.WriteString(fmt.Sprintf("%d : start=%s, size=%s", p.Number, start, size))

		// Add type GUID if available
		if p.TypeGUID != nil {
			script.WriteString(fmt.Sprintf(", type=%s", *p.TypeGUID))
		}

		// Add partition GUID if available
		if p.GUID != nil {
			script.WriteString(fmt.Sprintf(", uuid=%s", *p.GUID))
		}

		script.WriteString("\n")
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

		// Extract and log specific partition info (example assumes you want to log it)
		// Parsing output would be required to get specific partition details
		op.logger.Info("partition info: %s", output)
	}
	return nil
}
