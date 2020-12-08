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

package types

import (
	"fmt"
	"regexp"
	"strings"

	// UUID generation tool
	"github.com/google/uuid"
)

const (
	sectorSize             = 512 // in bytes
	gptHeaderSize          = 34  // in sectors
	gptHybridMBRHeaderSize = 63  // in sectors

	IgnitionAlignment = 2048 // 1MB in sectors
	DefaultAlignment  = 4096 // 2MB in sectors
)

type File struct {
	Node
	Contents string
	Mode     int
}

type Directory struct {
	Node
	Mode int
}

type Link struct {
	Node
	Target string
	Hard   bool
}

type Node struct {
	Name      string
	Directory string
	User      int
	Group     int
}

type Disk struct {
	ImageFile  string
	Device     string
	Alignment  int
	Partitions Partitions
}

type Partitions []*Partition

type Partition struct {
	Number          int
	Label           string
	TypeCode        string
	TypeGUID        string
	GUID            string
	Device          string
	Offset          int
	Length          int
	FilesystemType  string
	FilesystemLabel string
	FilesystemUUID  string
	FilesystemImage string // base64-encoded bzip2
	MountPath       string
	Hybrid          bool
	Ambivalent      bool // allow multiple FS types on validation
	Files           []File
	Directories     []Directory
	Links           []Link
	RemovedNodes    []Node
}

type MntDevice struct {
	Label        string
	Substitution string
}

type Test struct {
	Name              string
	In                []Disk // Disk state before running Ignition
	Out               []Disk // Expected disk state after running Ignition
	MntDevices        []MntDevice
	SystemDirFiles    []File
	Env               []string // Environment variables for Ignition
	Config            string
	ConfigMinVersion  string
	ConfigVersion     string
	ConfigShouldBeBad bool // Set to true to skip config validation step
}

func (ps Partitions) GetPartition(label string) *Partition {
	for _, p := range ps {
		if p.Label == label {
			return p
		}
	}
	panic(fmt.Sprintf("couldn't find partition with label %q", label))
}

func (ps Partitions) AddFiles(label string, fs []File) {
	p := ps.GetPartition(label)
	p.Files = append(p.Files, fs...)
}

func (ps Partitions) AddDirectories(label string, ds []Directory) {
	p := ps.GetPartition(label)
	p.Directories = append(p.Directories, ds...)
}

func (ps Partitions) AddLinks(label string, ls []Link) {
	p := ps.GetPartition(label)
	p.Links = append(p.Links, ls...)
}

func (ps Partitions) AddRemovedNodes(label string, ns []Node) {
	p := ps.GetPartition(label)
	p.RemovedNodes = append(p.RemovedNodes, ns...)
}

// SetOffsets sets the starting offsets for all of the partitions on the disk,
// according to its alignment.
func (d Disk) SetOffsets() {
	offset := gptHeaderSize
	for _, p := range d.Partitions {
		if p.Length == 0 {
			continue
		}
		offset = Align(offset, d.Alignment)
		p.Offset = offset
		offset += p.Length
	}
}

// CalculateImageSize determines the size of the disk, assuming the partitions are all aligned and completely
// fill the disk.
func (d Disk) CalculateImageSize() int64 {
	size := int64(Align(gptHybridMBRHeaderSize, d.Alignment))
	for _, p := range d.Partitions {
		size += int64(Align(p.Length, d.Alignment))
	}
	// convert to sectors and add secondary GPT header
	// subtract one because LBA0 (protective MBR) is not included in the secondary GPT header
	return sectorSize * (size + gptHeaderSize - 1)
}

// Align returns count aligned to the next multiple of alignment, or count itself if it is already aligned.
func Align(count int, alignment int) int {
	offset := count % alignment
	if offset != 0 {
		count += alignment - offset
	}
	return count
}

func GetBaseDisk() []Disk {
	return []Disk{
		{
			Alignment: DefaultAlignment,
			Partitions: Partitions{
				{
					Number:         1,
					Label:          "EFI-SYSTEM",
					TypeCode:       "efi",
					Length:         262144,
					FilesystemType: "vfat",
					Hybrid:         true,
				}, {
					Number:         6,
					Label:          "OEM",
					TypeCode:       "data",
					Length:         262144,
					FilesystemType: "ext4",
				}, {
					Number:         9,
					Label:          "ROOT",
					TypeCode:       "coreos-resize",
					Length:         262144,
					FilesystemType: "ext4",
				},
			},
		},
	}
}

// ReplaceAllUUIDVars replaces all UUID variables (format $uuid<num>) in configs and partitions with an UUID
func (test *Test) ReplaceAllUUIDVars() error {
	var err error
	UUIDmap := make(map[string]string)

	test.Config, err = replaceUUIDVars(test.Config, UUIDmap)
	if err != nil {
		return err
	}
	for _, disk := range test.In {
		if err := disk.replaceAllUUIDVarsInPartitions(UUIDmap); err != nil {
			return err
		}
	}
	for _, disk := range test.Out {
		if err := disk.replaceAllUUIDVarsInPartitions(UUIDmap); err != nil {
			return err
		}
	}
	return nil
}

// Replace all UUID variables (format $uuid<num>) in partitions with an UUID
func (disk *Disk) replaceAllUUIDVarsInPartitions(UUIDmap map[string]string) error {
	var err error

	for _, partition := range disk.Partitions {
		partition.TypeGUID, err = replaceUUIDVars(partition.TypeGUID, UUIDmap)
		if err != nil {
			return err
		}
		partition.GUID, err = replaceUUIDVars(partition.GUID, UUIDmap)
		if err != nil {
			return err
		}
		partition.FilesystemUUID, err = replaceUUIDVars(partition.FilesystemUUID, UUIDmap)
		if err != nil {
			return err
		}
	}
	return nil
}

// Identify and replace $uuid<num> with correct UUID
// Variables with matching <num> should have identical UUIDs
func replaceUUIDVars(str string, UUIDmap map[string]string) (string, error) {
	finalStr := str

	pattern := regexp.MustCompile(`\$uuid([0-9]+)`)
	for _, match := range pattern.FindAllStringSubmatch(str, -1) {
		if len(match) != 2 {
			return str, fmt.Errorf("find all string submatch error: want length of 2, got length of %d", len(match))
		}
		finalStr = strings.Replace(finalStr, match[0], getUUID(match[0], UUIDmap), 1)
	}
	return finalStr, nil
}

// Format: $uuid<num> where the uuid variable (uuid<num>) is the key
// value is the UUID for this uuid variable
func getUUID(key string, UUIDmap map[string]string) string {
	if _, ok := UUIDmap[key]; !ok {
		UUIDmap[key] = uuid.New().String()
	}
	return UUIDmap[key]
}

// ReplaceAllVersionVars replaces Version variable (format $version) in configs with ConfigMinVersion
// Updates the old config version (oldVersion) with a new one (newVersion)
func (t *Test) ReplaceAllVersionVars(version string) {
	pattern := regexp.MustCompile(`\$version`)
	t.Config = pattern.ReplaceAllString(t.Config, version)
	t.Name += " " + version
}

// Deep copy Test struct fields In, Out, MntDevices, SystemDirFiles
// so each BB test with identical Test structs have their own independent Test copies
func DeepCopy(t Test) Test {
	In_diskArr := make([]Disk, len(t.In))
	copy(In_diskArr, t.In)
	t.In = deepCopyPartitions(In_diskArr)

	Out_diskArr := make([]Disk, len(t.Out))
	copy(Out_diskArr, t.Out)
	t.Out = deepCopyPartitions(Out_diskArr)

	mntdevice := make([]MntDevice, len(t.MntDevices))
	copy(mntdevice, t.MntDevices)
	t.MntDevices = mntdevice

	SystemDirFiles := make([]File, len(t.SystemDirFiles))
	copy(SystemDirFiles, t.SystemDirFiles)
	t.SystemDirFiles = SystemDirFiles

	return t
}

// Deep copy each partition in []*Partitions
func deepCopyPartitions(diskArr []Disk) []Disk {
	disk_count := 0
	for _, disk := range diskArr {
		partitionArr := make([]*Partition, len(disk.Partitions))
		copy(partitionArr, disk.Partitions)

		partition_count := 0
		for _, partition := range disk.Partitions {
			partition_tmp := *partition
			partitionArr[partition_count] = &partition_tmp

			// deep copy all slices in partition struct
			Files := make([]File, len(partition.Files))
			copy(Files, partition.Files)
			partitionArr[partition_count].Files = Files

			Directories := make([]Directory, len(partition.Directories))
			copy(Directories, partition.Directories)
			partitionArr[partition_count].Directories = Directories

			Links := make([]Link, len(partition.Links))
			copy(Links, partition.Links)
			partitionArr[partition_count].Links = Links

			RemovedNodes := make([]Node, len(partition.RemovedNodes))
			copy(RemovedNodes, partition.RemovedNodes)
			partitionArr[partition_count].RemovedNodes = RemovedNodes

			partition_count++
		}
		diskArr[disk_count].Partitions = partitionArr
		disk_count++
	}
	return diskArr
}
