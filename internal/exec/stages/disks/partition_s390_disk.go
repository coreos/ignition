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

// s390x variant of partitionDisk, checks cmdline for rd.dasd and
// if set calls the DASD partitioning function else fall back to
// partitionGPTDisk

package disks

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/coreos/ignition/v2/config/v3_2_experimental/types"
)

// isDasdDisk checks whether the disk is a DASD disk by following the dev alias symlink to the real device name
func isDasdDisk(devAlias string) (bool, error) {
	diskPath, err := filepath.EvalSymlinks(devAlias)
	if err != nil {
		return false, fmt.Errorf("couldn't follow %s: %v", devAlias, err)
	}

	// the real device name should be /dev/dasdx for DASDs
	return strings.HasPrefix(diskPath, "/dev/dasd"), nil
}

// partitionDisk for s390x checks if the disk is DASD. If so, use the
// partitionDasdDisk call, if not fall back to the regular partitionGPTDisk which
// partitions using sgdisk
func (s stage) partitionDisk(dev types.Disk, devAlias string) error {
	if isDasd, err := isDasdDisk(devAlias); err == nil {
		if isDasd {
			return s.partitionDasdDisk(dev, devAlias)
		}
	} else {
		return err
	}
	return s.partitionGPTDisk(dev, devAlias)
}
