// Copyright 2017 CoreOS, Inc.
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

package register

import (
	"fmt"
	"github.com/coreos/ignition/tests/types"
	"strconv"
	"strings"

	json "github.com/ajeddeloh/go-json"
	"github.com/coreos/go-semver/semver"
	maxTypes "github.com/coreos/ignition/config/v2_3_experimental/types" //todr: document and clean up code
	typesInternal "github.com/coreos/ignition/internal/config/types"
)

type TestType int

const (
	NegativeTest TestType = iota
	PositiveTest
)

var Tests map[TestType][]types.Test

func init() {
	Tests = make(map[TestType][]types.Test)
}

var count = 0

func register(tType TestType, t types.Test) {
	Tests[tType] = append(Tests[tType], t)
	fmt.Printf("test count: %d\n", count)
	count++
}

// Finds version, creates Test structs for all compatible versions and registers them
func Register(tType TestType, t types.Test) {
	// todo: how to handle errors? it errors on Empty Userdata, Preemption with default config, Preemption with base, default config, Preemption with no config but these test still need to run
	var config typesInternal.Config
	err := json.Unmarshal([]byte(t.Config), &config)
	version, semverErr := semver.NewVersion(config.Ignition.Version)

	register(tType, t)
	if err == nil && semverErr == nil && version.LessThan(maxTypes.MaxVersion) { // todo: how to identify max of different major versions?
		versionStr := strconv.FormatInt(version.Major, 10) + "." + strconv.FormatInt(version.Minor, 10) + "." + strconv.FormatInt(version.Patch, 10)

		for *version != maxTypes.MaxVersion {
			test := deepCopy(t)

			version.Minor++
			updatedVersionStr := strconv.FormatInt(version.Major, 10) + "." + strconv.FormatInt(version.Minor, 10) + "." + strconv.FormatInt(version.Patch, 10)
			if version.Minor == 3 {
				*version = maxTypes.MaxVersion
				updatedVersionStr += "-" + string(version.PreRelease) + version.Metadata
			}

			test.Config = strings.Replace(test.Config, versionStr, updatedVersionStr, 1)
			test.Name += " " + updatedVersionStr
			register(tType, test)
		}
	}
}

// Deep copy Test struct fields In, Out, MntDevices, OEMLookasideFiles, SystemDirFiles
// so each BB test with identical Test structs have their own independent Test copies
func deepCopy(t types.Test) types.Test {
	In_diskArr := make([]types.Disk, len(t.In))
	copy(In_diskArr, t.In)
	t.In = deepCopyPartitions(In_diskArr)

	Out_diskArr := make([]types.Disk, len(t.Out))
	copy(Out_diskArr, t.Out)
	t.Out = deepCopyPartitions(Out_diskArr)

	mntdevice := make([]types.MntDevice, len(t.MntDevices))
	copy(mntdevice, t.MntDevices)
	t.MntDevices = mntdevice

	OEMLookasideFiles := make([]types.File, len(t.OEMLookasideFiles))
	copy(OEMLookasideFiles, t.OEMLookasideFiles)
	t.OEMLookasideFiles = OEMLookasideFiles

	SystemDirFiles := make([]types.File, len(t.SystemDirFiles))
	copy(SystemDirFiles, t.SystemDirFiles)
	t.SystemDirFiles = SystemDirFiles

	return t
}

// Deep copy each partition in []*Partitions
func deepCopyPartitions(diskArr []types.Disk) []types.Disk {
	disk_count := 0
	for _, disk := range diskArr {
		partitionArr := make([]*types.Partition, len(disk.Partitions))
		copy(partitionArr, disk.Partitions)

		partition_count := 0
		for _, partition := range disk.Partitions {
			partition_tmp := *partition
			partitionArr[partition_count] = &partition_tmp

			// deep copy all slices in partition struct
			Files := make([]types.File, len(partition.Files))
			copy(Files, partition.Files)
			partitionArr[partition_count].Files = Files

			Directories := make([]types.Directory, len(partition.Directories))
			copy(Directories, partition.Directories)
			partitionArr[partition_count].Directories = Directories

			Links := make([]types.Link, len(partition.Links))
			copy(Links, partition.Links)
			partitionArr[partition_count].Links = Links

			RemovedNodes := make([]types.Node, len(partition.RemovedNodes))
			copy(RemovedNodes, partition.RemovedNodes)
			partitionArr[partition_count].RemovedNodes = RemovedNodes

			partition_count++
		}
		diskArr[disk_count].Partitions = partitionArr
		disk_count++
	}
	return diskArr
}
