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
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/partitioners"
)

var (
	ErrBadSgdiskOutput = errors.New("sgdisk had unexpected output")
)

type Operation struct {
	logger    *log.Logger
	dev       string
	wipe      bool
	parts     []partitioners.Partition
	deletions []int
	infos     []int
}

// Begin begins an sgdisk operation
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

// WipeTable toggles if the table is to be wiped first when commiting this operation.
func (op *Operation) WipeTable(wipe bool) {
	op.wipe = wipe
}

// Pretend is like Commit() but uses the --pretend flag and returns the output
// on stdout for parsing.
//
// Note: because sgdisk does not do any escaping on its output, callers should ensure
// the partitions' labels do not have any nasty characters that will interfere
// with parsing (e.g. \n)
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

// ParseOutput parses the output of running sgdisk pretend with --info specified for each partition
// number specified in partitionNumbers. E.g. if paritionNumbers is [1,4,5], it is expected that the sgdisk
// output was from running `sgdisk --pretend <commands> --info=1 --info=4 --info=5`. It assumes the the
// partition labels are well behaved (i.e. contain no control characters). It returns a list of partitions
// matching the partition numbers specified, but with the start and size information as determined by sgdisk.
// The partition numbers need to passed in because sgdisk includes them in its output.
func (op *Operation) ParseOutput(sgdiskOutput string, partitionNumbers []int) (map[int]partitioners.Output, error) {
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

	output := map[int]partitioners.Output{}
	state := START
	current := partitioners.Output{}
	i := 0

	lines := strings.Split(sgdiskOutput, "\n")
	for _, line := range lines {
		switch state {
		case START:
			start, err := parseLine(startRegex, line)
			if err != nil {
				return nil, err
			}
			if start != -1 {
				current.Start = start
				state = END
			}
		case END:
			end, err := parseLine(endRegex, line)
			if err != nil {
				return nil, err
			}
			if end != -1 {
				current.Size = 1 + end - current.Start
				output[partitionNumbers[i]] = current
				i++
				if i == len(partitionNumbers) {
					state = FAIL_ON_START_END
				} else {
					current = partitioners.Output{}
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

func partitionGetStart(p partitioners.Partition) string {
	if p.StartSector != nil {
		return fmt.Sprintf("%d", *p.StartSector)
	}
	return "0"
}

func partitionGetSize(p partitioners.Partition) string {
	if p.SizeInSectors != nil {
		return fmt.Sprintf("%d", *p.SizeInSectors)
	}
	return "0"
}
