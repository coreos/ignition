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

package blackbox

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/tests/types"

	"golang.org/x/sys/unix"
)

func regexpSearch(itemName, pattern string, data []byte) (string, error) {
	re := regexp.MustCompile(pattern)
	match := re.FindSubmatch(data)
	if len(match) < 2 {
		return "", fmt.Errorf("couldn't find %s", itemName)
	}
	return string(match[1]), nil
}

func getPartitionSet(device string) (map[int]struct{}, error) {
	sgdiskOverview, err := exec.Command("sgdisk", "-p", device).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sgdisk -p %s failed: %v", device, err)
	}

	//What this regex means:       num      start    end    size,code,name
	re := regexp.MustCompile("\n\\W+(\\d+)\\W+\\d+\\W+\\d+\\W+\\d+.*")
	ret := map[int]struct{}{}
	for _, match := range re.FindAllStringSubmatch(string(sgdiskOverview), -1) {
		if len(match) == 0 {
			continue
		}
		if len(match) != 2 {
			return nil, fmt.Errorf("Invalid regex result from parsing sgdisk")
		}
		num, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, err
		}
		ret[num] = struct{}{}
	}
	return ret, nil
}

func validateDisk(t *testing.T, d types.Disk) error {
	partitionSet, err := getPartitionSet(d.Device)
	if err != nil {
		return err
	}

	for _, e := range d.Partitions {
		if e.TypeCode == "blank" {
			continue
		}

		if _, ok := partitionSet[e.Number]; !ok {
			t.Errorf("Partition %d is missing", e.Number)
		}
		delete(partitionSet, e.Number)

		sgdiskInfo, err := exec.Command(
			"sgdisk", "-i", strconv.Itoa(e.Number),
			d.Device).CombinedOutput()
		if err != nil {
			t.Error("sgdisk -i", strconv.Itoa(e.Number), err)
			return nil
		}

		actualGUID, err := regexpSearch("GUID", "Partition unique GUID: (?P<partition_guid>[\\d\\w-]+)", sgdiskInfo)
		if err != nil {
			return err
		}
		actualTypeGUID, err := regexpSearch("type GUID", "Partition GUID code: (?P<partition_code>[\\d\\w-]+)", sgdiskInfo)
		if err != nil {
			return err
		}
		actualSectors, err := regexpSearch("partition size", "Partition size: (?P<sectors>\\d+) sectors", sgdiskInfo)
		if err != nil {
			return err
		}
		actualLabel, err := regexpSearch("partition name", "Partition name: '(?P<name>[\\d\\w-_]+)'", sgdiskInfo)
		if err != nil {
			return err
		}

		// have to align the size to the nearest sector alignment boundary first
		expectedSectors := types.Align(e.Length, d.Alignment)

		if e.TypeGUID != "" && formatUUID(e.TypeGUID) != formatUUID(actualTypeGUID) {
			t.Error("TypeGUID does not match!", e.TypeGUID, actualTypeGUID)
		}
		if e.GUID != "" && formatUUID(e.GUID) != formatUUID(actualGUID) {
			t.Error("GUID does not match!", e.GUID, actualGUID)
		}
		if e.Label != actualLabel {
			t.Error("Label does not match!", e.Label, actualLabel)
		}
		if strconv.Itoa(expectedSectors) != actualSectors {
			t.Error(
				"Sectors does not match!", expectedSectors, actualSectors)
		}
	}

	if len(partitionSet) != 0 {
		t.Error("Disk had extra partitions", partitionSet)
	}

	// TODO: inspect the disk without triggering partition rescans so we don't need to settle here
	if _, err := runWithoutContext("udevadm", "settle"); err != nil {
		t.Log(err)
	}
	return nil
}

func formatUUID(s string) string {
	return strings.ToUpper(strings.Replace(s, "-", "", -1))
}

func validateFilesystems(t *testing.T, expected []*types.Partition) error {
	for _, e := range expected {
		if e.FilesystemType == "" &&
			e.FilesystemUUID == "" &&
			e.FilesystemLabel == "" {
			continue
		}
		info, err := util.GetFilesystemInfo(e.Device, e.Ambivalent)
		if err != nil {
			return fmt.Errorf("couldn't get filesystem info: %v", err)
		}
		if e.FilesystemType != "" {
			if info.Type != e.FilesystemType {
				t.Errorf("FilesystemType does not match, expected:%q actual:%q",
					e.FilesystemType, info.Type)
			}
		}
		if e.FilesystemUUID != "" {
			if formatUUID(info.UUID) != formatUUID(e.FilesystemUUID) {
				t.Errorf("FilesystemUUID does not match, expected:%q actual:%q",
					e.FilesystemUUID, info.UUID)
			}
		}
		if e.FilesystemLabel != "" {
			if info.Label != e.FilesystemLabel {
				t.Errorf("FilesystemLabel does not match, expected:%q actual:%q",
					e.FilesystemLabel, info.Label)
			}
		}
	}
	return nil
}

func validatePartitionNodes(t *testing.T, ctx context.Context, partition *types.Partition) {
	if len(partition.Files) == 0 &&
		len(partition.Directories) == 0 &&
		len(partition.Links) == 0 &&
		len(partition.RemovedNodes) == 0 {
		return
	}
	if err := mountPartition(ctx, partition); err != nil {
		t.Errorf("failed to mount %s: %v", partition.Device, err)
	}
	defer func() {
		if err := umountPartition(partition); err != nil {
			// failing to unmount is not a validation failure
			t.Fatalf("Failed to unmount %s: %v", partition.MountPath, err)
		}
	}()
	for _, file := range partition.Files {
		validateFile(t, partition, file)
	}
	for _, dir := range partition.Directories {
		validateDirectory(t, partition, dir)
	}
	for _, link := range partition.Links {
		validateLink(t, partition, link)
	}
	for _, node := range partition.RemovedNodes {
		path := filepath.Join(partition.MountPath, node.Directory, node.Name)
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Error("Node was expected to be removed and is present!", path)
		}
	}
}

func validateFilesDirectoriesAndLinks(t *testing.T, ctx context.Context, expected []*types.Partition) {
	for _, partition := range expected {
		if partition.TypeCode == "blank" || partition.Length == 0 || partition.FilesystemType == "" || partition.FilesystemType == "swap" {
			continue
		}
		validatePartitionNodes(t, ctx, partition)
	}
}

func validateFile(t *testing.T, partition *types.Partition, file types.File) {
	path := filepath.Join(partition.MountPath, file.Node.Directory, file.Node.Name)
	fileInfo := unix.Stat_t{}
	if err := unix.Lstat(path, &fileInfo); err != nil {
		t.Errorf("Error stat'ing file %s: %v", path, err)
		return
	}
	if file.Contents != "" {
		dat, err := ioutil.ReadFile(path)
		if err != nil {
			t.Error("Error when reading file", path)
			return
		}

		actualContents := string(dat)
		if file.Contents != actualContents {
			t.Error("Contents of file", path, "do not match!",
				file.Contents, actualContents)
		}
	}

	validateMode(t, path, file.Mode)
	validateNode(t, fileInfo, file.Node)
}

func validateDirectory(t *testing.T, partition *types.Partition, dir types.Directory) {
	path := filepath.Join(partition.MountPath, dir.Node.Directory, dir.Node.Name)
	dirInfo := unix.Stat_t{}
	if err := unix.Lstat(path, &dirInfo); err != nil {
		t.Errorf("Error stat'ing directory %s: %v", path, err)
		return
	}
	if dirInfo.Mode&unix.S_IFDIR == 0 {
		t.Errorf("Node at %s is not a directory!", path)
	}
	validateMode(t, path, dir.Mode)
	validateNode(t, dirInfo, dir.Node)
}

func validateLink(t *testing.T, partition *types.Partition, link types.Link) {
	linkPath := filepath.Join(partition.MountPath, link.Node.Directory, link.Node.Name)
	linkInfo := unix.Stat_t{}
	if err := unix.Lstat(linkPath, &linkInfo); err != nil {
		t.Error("Error stat'ing link \"" + linkPath + "\": " + err.Error())
		return
	}
	if link.Hard {
		targetPath := filepath.Join(partition.MountPath, link.Target)
		targetInfo := unix.Stat_t{}
		if err := unix.Lstat(targetPath, &targetInfo); err != nil {
			t.Error("Error stat'ing target \"" + targetPath + "\": " + err.Error())
			return
		}
		if linkInfo.Ino != targetInfo.Ino {
			t.Error("Hard link and target don't have same inode value: " + linkPath + " " + targetPath)
			return
		}
	} else {
		if linkInfo.Mode&unix.S_IFLNK == 0 {
			t.Errorf("Node at symlink path is not a symlink (it's a %s): %s", os.FileMode(linkInfo.Mode).String(), linkPath)
			return
		}
		targetPath, err := os.Readlink(linkPath)
		if err != nil {
			t.Error("Error reading symbolic link: " + err.Error())
			return
		}
		if targetPath != link.Target {
			t.Errorf("Actual and expected symbolic link targets don't match. Expected %q, got %q", link.Target, targetPath)
			return
		}
	}
	validateNode(t, linkInfo, link.Node)
}

func validateMode(t *testing.T, path string, mode int) {
	if mode != 0 {
		fileInfo, err := os.Lstat(path)
		if err != nil {
			t.Error("Error running stat on node", path, err)
			return
		}

		if fileInfo.Mode() != os.FileMode(mode) {
			t.Error("Node Mode does not match", path, os.FileMode(mode), fileInfo.Mode())
		}
	}
}

func validateNode(t *testing.T, nodeInfo unix.Stat_t, node types.Node) {
	if nodeInfo.Uid != uint32(node.User) {
		t.Error("Node has the wrong owner", node.User, nodeInfo.Uid)
	}

	if nodeInfo.Gid != uint32(node.Group) {
		t.Error("Node has the wrong group owner", node.Group, nodeInfo.Gid)
	}
}
