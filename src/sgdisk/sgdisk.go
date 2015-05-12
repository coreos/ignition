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
	"os/exec"

	"github.com/coreos/ignition/src/log"
)

const sgdiskPath = "/sbin/sgdisk"

type Operation struct {
	logger *log.Logger
	dev    string
	wipe   bool
	parts  []Partition
}

type Partition struct {
	Number int
	Offset int64 // sectors
	Length int64 // sectors
	Label  string
}

func Begin(logger *log.Logger, dev string) *Operation {
	return &Operation{logger: logger, dev: dev}
}

// CreatePartition adds the supplied partition to the list of partitions to be created.
// If Partition overlaps with any already created partitions or has otherwise impossible values an error is returned.
func (op *Operation) CreatePartition(p Partition) error {
	// TODO(vc): sanity check p against op.parts
	// XXX(vc): I don't think sgdisk likes zero-based partition numbers, TODO: verify this! sgdisk _feels_ poorly made when using it, consider alternatives.
	op.parts = append(op.parts, p)
	return nil
}

// WipeTable toggles if the table is to be wiped first
func (op *Operation) WipeTable(wipe bool) {
	op.wipe = wipe
}

// Commit commits the operation prepared in op
func (op *Operation) Commit() error {
	if op.wipe {
		cmd := exec.Command(sgdiskPath, "--zap-all", op.dev)
		if err := op.logger.LogCmd(cmd, "wiping table on %q", op.dev); err != nil {
			return fmt.Errorf("wipe failed: %v")
		}
	}

	if len(op.parts) != 0 {
		opts := []string{}
		for _, p := range op.parts {
			opts = append(opts, fmt.Sprintf("--new=%d:%d:+%d", p.Number, p.Offset, p.Length))
			opts = append(opts, fmt.Sprintf("--change-name=%d:%s", p.Number, p.Label))
		}
		opts = append(opts, op.dev)
		cmd := exec.Command(sgdiskPath, opts...)
		if err := op.logger.LogCmd(cmd, "creating %d partitions on %q", len(op.parts), op.dev); err != nil {
			return fmt.Errorf("create partitions failed: %v", err)
		}
	}

	return nil
}
