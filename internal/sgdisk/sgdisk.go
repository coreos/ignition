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

	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/log"
)

const sgdiskPath = "/sbin/sgdisk"

type Operation struct {
	logger    *log.Logger
	dev       string
	wipe      bool
	parts     []types.Partition
	deletions []int
}

// Begin begins an sgdisk operation
func Begin(logger *log.Logger, dev string) *Operation {
	return &Operation{logger: logger, dev: dev}
}

// CreatePartition adds the supplied partition to the list of partitions to be created as part of an operation.
func (op *Operation) CreatePartition(p types.Partition) {
	op.parts = append(op.parts, p)
}

func (op *Operation) DeletePartition(num int) {
	op.deletions = append(op.deletions, num)
}

// WipeTable toggles if the table is to be wiped first when commiting this operation.
func (op *Operation) WipeTable(wipe bool) {
	op.wipe = wipe
}

// Pretend is like Commit() but uses the --pretend flag and returns the output
// on stdout for parsing. Unfortunately sgdisk does not have a machine readable
// output format, so parsing stdout is harder than it needs to be. It also adds
// the --print flag to dump the partition table.
//
// Note: because sgdisk does not do any escaping on its output, callers should ensure
//       the partitions' labels do not have any nasty characters that will interfere
//       with parsing (e.g. \n)
//
// TODO(ajeddeloh): add machine readable output flag if sgdisk adds one
func (op *Operation) Pretend() (string, error) {
	opts := []string{"--pretend"}
	opts = append(opts, op.buildOptions()...)

	cmd := exec.Command(sgdiskPath, opts...)
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
	cmd := exec.Command(sgdiskPath, op.buildOptions()...)
	if _, err := op.logger.LogCmd(cmd, "creating %d partitions on %q", len(op.parts), op.dev); err != nil {
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
		opts = append(opts, fmt.Sprintf("--new=%d:%d:+%d", p.Number, p.Start, p.Size))
		if p.Label != nil {
			opts = append(opts, fmt.Sprintf("--change-name=%d:%s", p.Number, *p.Label))
		}
		if p.TypeGUID != "" {
			opts = append(opts, fmt.Sprintf("--typecode=%d:%s", p.Number, p.TypeGUID))
		}
		if p.GUID != "" {
			opts = append(opts, fmt.Sprintf("--partition-guid=%d:%s", p.Number, p.GUID))
		}
	}

	if len(opts) == 0 {
		return nil
	}

	opts = append(opts, op.dev)
	return opts
}
