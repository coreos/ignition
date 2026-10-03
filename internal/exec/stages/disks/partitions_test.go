// Copyright 2025 CoreOS, Inc.
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

package disks

import (
	"testing"
)

func TestPartitionNumberPrefix(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"/dev/sda", ""},
		{"/dev/vda", ""},
		{"/dev/nvme0n1", "p"},
		{"/dev/mmcblk0", "p"},
		{"/dev/loop0", "p"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := partitionNumberPrefix(tt.in)
			if got != tt.out {
				t.Errorf("partitionNumberPrefix(%q) = %q, want %q", tt.in, got, tt.out)
			}
		})
	}
}

func TestPartitionDevPath(t *testing.T) {
	tests := []struct {
		blockDev string
		dmName   string
		prefix   string
		partNum  int
		out      string
	}{
		{"/dev/sda", "", "", 1, "/dev/sda1"},
		{"/dev/sda", "", "", 12, "/dev/sda12"},
		{"/dev/nvme0n1", "", "p", 1, "/dev/nvme0n1p1"},
		{"/dev/nvme0n1", "", "p", 3, "/dev/nvme0n1p3"},
		{"/dev/mmcblk0", "", "p", 2, "/dev/mmcblk0p2"},
		{"/dev/dm-0", "mpath0", "p", 1, "/dev/disk/by-id/dm-name-mpath0p1"},
		{"/dev/dm-0", "mpath0", "p", 5, "/dev/disk/by-id/dm-name-mpath0p5"},
	}
	for _, tt := range tests {
		t.Run(tt.out, func(t *testing.T) {
			got := partitionDevPath(tt.blockDev, tt.dmName, tt.prefix, tt.partNum)
			if got != tt.out {
				t.Errorf("partitionDevPath(%q, %q, %q, %d) = %q, want %q",
					tt.blockDev, tt.dmName, tt.prefix, tt.partNum, got, tt.out)
			}
		})
	}
}
