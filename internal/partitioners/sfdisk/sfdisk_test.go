// Copyright 2024 Red Hat
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

package sfdisk_test

import (
	"errors"
	"reflect"
	"testing"

	internalErrors "github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/internal/partitioners"
	"github.com/coreos/ignition/v2/internal/partitioners/sfdisk"
)

func TestPartitionParse(t *testing.T) {
	// Define test cases
	tests := []struct {
		name             string
		sfdiskOut        string
		partitionNumbers []int
		expectedOutput   map[int]partitioners.Output
		expectedError    error
	}{
		{
			name: "valid input with single partition",
			sfdiskOut: `
Disk /dev/vda: 2 GiB, 2147483648 bytes, 4194304 sectors
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes

>>> Created a new DOS (MBR) disklabel with disk identifier 0x501fc254.
/dev/vda1: Created a new partition 1 of type 'Linux' and of size 5 KiB.
/dev/vda2: Done.

New situation:
Disklabel type: dos
Disk identifier: 0x501fc254

Device     Boot Start   End Sectors Size Id Type
/dev/vda1        2048  2057      10   5K 83 Linux
The partition table is unchanged (--no-act).`,
			partitionNumbers: []int{1},
			expectedOutput: map[int]partitioners.Output{
				1: {Start: 2048, Size: 10},
			},
			expectedError: nil,
		},
		{
			name: "valid input with two partitions",
			sfdiskOut: `
Disk /dev/vda: 2 GiB, 2147483648 bytes, 4194304 sectors
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes

>>> Created a new DOS (MBR) disklabel with disk identifier 0x8d8dd38c.
/dev/vda1: Created a new partition 1 of type 'Linux' and of size 5 KiB.
/dev/vda2: Created a new partition 2 of type 'Linux' and of size 5 KiB.
/dev/vda3: Done.

New situation:
Disklabel type: dos
Disk identifier: 0x8d8dd38c

Device     Boot Start   End Sectors Size Id Type
/dev/vda1        2048  2057      10   5K 83 Linux
/dev/vda2        4096  4105      10   5K 83 Linux
The partition table is unchanged (--no-act).`,
			partitionNumbers: []int{1, 2},
			expectedOutput: map[int]partitioners.Output{
				1: {Start: 2048, Size: 10},
				2: {Start: 4096, Size: 10},
			},
			expectedError: nil,
		},
		{
			name: "invalid input with 1 partition starting on sector 0",
			sfdiskOut: `
Disk /dev/vda: 2 GiB, 2147483648 bytes, 4194304 sectors
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes

>>> Created a new DOS (MBR) disklabel with disk identifier 0xdebbe997.
/dev/vda1: Start sector 0 out of range.
Failed to add #1 partition: Numerical result out of range
Leaving.
`,
			partitionNumbers: []int{1},
			expectedOutput: map[int]partitioners.Output{
				1: {Start: 0, Size: 0},
			},
			expectedError: internalErrors.ErrBadSfdiskPretend,
		},
	}
	op := sfdisk.Begin(nil, "")
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := op.ParseOutput(tt.sfdiskOut, tt.partitionNumbers)
			if tt.expectedError != nil {
				if !errors.Is(err, tt.expectedError) {
					t.Errorf("#%d: bad error: result = %v, expected = %v", i, err, tt.expectedError)
				}
			} else if !reflect.DeepEqual(output, tt.expectedOutput) {
				t.Errorf("#%d: result = %v, expected = %v", i, output, tt.expectedOutput)
			}
		})
	}
}
