// Copyright 2020 Red Hat, Inc.
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
//
// +build s390x

package fdasd

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"

	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
)

type Partition struct {
	Number             int
	StartTrack         int64
	SizeInTracks       int64
	ShouldExist        bool
	WipePartitionEntry bool
}

type Operation struct {
	logger *log.Logger
	dev    string
	parts  []string
}

// Begin begins an fdasd operation
func Begin(logger *log.Logger, dev string) *Operation {
	return &Operation{logger: logger, dev: dev}
}

// CreatePartition adds the supplied partition start and size to a file which is used by fdasd to create
// partitions in non interactive mode. The non interactive options for fdasd are very limited and it uses
// a very simple input file to configure partitions. Specified here: https://www.ibm.com/support/knowledgecenter/linuxonibm/com.ibm.linux.z.lgdd/lgdd_r_fasdusingoptions.html
// a sample fdasd configuration file to configure partitions looks like this:
//    [2,500]
//    [501,1100]
//    [1101,45000]
// where the values are 'tracks' and 2 is the lowest track number to begin with. These partitions always have to be in
// ascending order of values and cannot overlap. Violation of these conditions will result in fdasd rejecting the config.
// The 'native' keyword used  below refers to the type of the partition and can be one of many like swap, raid etc..but
// it is just a guideline.
// One more thing to note is there is no way to preserve existing partitions. When a config is provided, fdasd destroys other partitions
// and creates the partitions anew.
func (op *Operation) CreatePartition(startTrack int64, sizeInTracks int64) error {
	if len(op.parts) == 3 {
		return fmt.Errorf("cannot create more than three partitions with CDL")
	}

	partconf := "[" + strconv.FormatInt(startTrack, 10) + "," + strconv.FormatInt(sizeInTracks+startTrack-1, 10) + ",native]\n"
	op.parts = append(op.parts, partconf)
	return nil
}

// GetDiskAndPartitionsInfo is used to get information about the disk and the partitions
// using the '-p' switch to fdasd. Since sgdisk does not work for DASD disks, this is used to get
// information like cylinders, blocks per track, bytes per block. These will be used to calculate the
// start and end of each partition. A sample output of this command:
//         Disk /dev/dasdb:
//           cylinders ............: 30051
//           tracks per cylinder ..: 15
//           blocks per track .....: 12
//           bytes per block ......: 4096
//           volume label .........: VOL1
//           volume serial ........: 0X0121
//           max partitions .......: 3
//
//          ------------------------------- tracks -------------------------------
//                        Device      start      end   length   Id  System
//                   /dev/dasdb1          2   172033   172032    1  Linux native
//                   /dev/dasdb2     172034   450764   278731    2  Linux native
//
func (op *Operation) GetDiskAndPartitionsInfo() (string, error) {
	opts := []string{"-p", op.dev}
	op.logger.Info("running fdasd with options: %v", opts)

	cmd := exec.Command(distro.FdasdCmd(), opts...)
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
		return "", fmt.Errorf("Failed to read disk and partition data. Err: %v. Stderr: %v", err, string(errors))
	}

	return string(output), nil
}

// Commit commits an partitioning operation.
// fdasd --silent --config <file>  <disk>
func (op *Operation) Commit() error {
	opts := []string{"--silent", "--config", "/dev/stdin", op.dev}
	op.logger.Info("running fdasd with options: %v", opts)
	cmd := exec.Command(distro.FdasdCmd(), opts...)
	cmd.Stdin = strings.NewReader(strings.Join(op.parts, "\n"))
	if _, err := op.logger.LogCmd(cmd, "creating partitions on %q", op.dev); err != nil {
		return fmt.Errorf("create partitions failed: %v", err)
	}

	return nil
}
