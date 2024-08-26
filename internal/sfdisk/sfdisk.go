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

	"github.com/coreos/ignition/v2/config/util"
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

	script := op.sfdiskBuildOptions()
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
func (op *Operation) SfdiskCommit() error {
	script := op.sfdiskBuildOptions()
	if len(script) == 0 {
		return nil
	}

	// If wipe we need to reset the partition table
	if op.wipe {
		// Erase the existing partition tables
		cmd := exec.Command("sudo", distro.WipefsCmd(), "-a", op.dev)
		if _, err := op.logger.LogCmd(cmd, "option wipe selected, and failed to execute on %q", op.dev); err != nil {
			return fmt.Errorf("wipe partition table failed: %v", err)
		}
	}

	op.logger.Info("running sfdisk with script: %v", script)
	exec.Command("sh", "-c", fmt.Sprintf("echo label: gpt | sudo %s %s", distro.SfdiskCmd(), op.dev))
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo \"%s\" | sudo %s %s", script, distro.SfdiskCmd(), op.dev))
	if _, err := op.logger.LogCmd(cmd, "deleting %d partitions and creating %d partitions on %q", len(op.deletions), len(op.parts), op.dev); err != nil {
		return fmt.Errorf("create partitions failed: %v", err)
	}

	return nil
}

func (op Operation) sfdiskBuildOptions() string {
	var script bytes.Buffer

	for _, p := range op.parts {
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
		script.WriteString("\\n ")

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

// Copy old functionality from sgdisk to switch between the two during testing.
// Will be removed.
func (op *Operation) SgdiskCommit() error {
	opts := op.sgdiskBuildOptions()
	if len(opts) == 0 {
		return nil
	}
	op.logger.Info("running sgdisk with options: %v", opts)
	cmd := exec.Command(distro.SgdiskCmd(), opts...)

	if _, err := op.logger.LogCmd(cmd, "deleting %d partitions and creating %d partitions on %q", len(op.deletions), len(op.parts), op.dev); err != nil {
		return fmt.Errorf("create partitions failed: %v", err)
	}

	return nil
}

// Copy old functionality from sgdisk to switch between the two during testing.
// Will be removed.
func (op Operation) sgdiskBuildOptions() []string {
	opts := []string{}

	if op.wipe {
		opts = append(opts, "--zap-all")
	}

	// Do all deletions before creations
	for _, partition := range op.deletions {
		opts = append(opts, fmt.Sprintf("--delete=%d", partition))
	}

	for _, p := range op.parts {
		opts = append(opts, fmt.Sprintf("--new=%d:%s:+%s", p.Number, partitionGetStart(p), partitionGetSize(p)))
		if p.Label != nil {
			opts = append(opts, fmt.Sprintf("--change-name=%d:%s", p.Number, *p.Label))
		}
		if util.NotEmpty(p.TypeGUID) {
			opts = append(opts, fmt.Sprintf("--typecode=%d:%s", p.Number, *p.TypeGUID))
		}
		if util.NotEmpty(p.GUID) {
			opts = append(opts, fmt.Sprintf("--partition-guid=%d:%s", p.Number, *p.GUID))
		}
	}

	for _, partition := range op.infos {
		opts = append(opts, fmt.Sprintf("--info=%d", partition))
	}

	if len(opts) == 0 {
		return nil
	}

	opts = append(opts, op.dev)
	return opts
}

// Copy old functionality from sgdisk to switch between the two during testing.
// Will be removed.
func partitionGetStart(p Partition) string {
	if p.StartSector != nil {
		return fmt.Sprintf("%d", *p.StartSector)
	}
	return "0"
}

func partitionGetSize(p Partition) string {
	if p.SizeInSectors != nil {
		return fmt.Sprintf("%d", *p.SizeInSectors)
	}
	return "0"
}
