// Copyright 2015 CoreOS, Inc.
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

package sgdisk

import (
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
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

// Begin begins an sgdisk operation
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

// Pretend is like Commit() but uses the --pretend flag and returns the output
// on stdout for parsing.
//
// Note: because sgdisk does not do any escaping on its output, callers should ensure
//       the partitions' labels do not have any nasty characters that will interfere
//       with parsing (e.g. \n)
func (op *Operation) Pretend() (string, error) {
	opts := []string{"--pretend"}
	opts = append(opts, op.buildOptions()...)
	op.logger.Info("running sgdisk with options: %v", opts)

	cmd := exec.Command(distro.SgdiskCmd(), opts...)
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
	output, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	errors, err := ioutil.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("Failed to pretend to create partitions. Err: %v. Stderr: %v", err, string(errors))
	}

	return string(output), nil
}

// Commit commits an partitioning operation.
func (op *Operation) Commit() error {
	opts := op.buildOptions()
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

func (op Operation) buildOptions() []string {
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
		if p.TypeGUID != nil && *p.TypeGUID != "" {
			opts = append(opts, fmt.Sprintf("--typecode=%d:%s", p.Number, *p.TypeGUID))
		}
		if p.GUID != nil && *p.GUID != "" {
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
