package disks

import (
	"errors"
	"reflect"
	"testing"

	internalErrors "github.com/coreos/ignition/v2/config/shared/errors"
)

func TestPartitionParse(t *testing.T) {
	// Define test cases
	tests := []struct {
		name             string
		sfdiskOut        string
		partitionNumbers []int
		expectedOutput   map[int]sfdiskOutput
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
			expectedOutput: map[int]sfdiskOutput{
				1: {start: 2048, size: 10},
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
			expectedOutput: map[int]sfdiskOutput{
				1: {start: 2048, size: 10},
				2: {start: 4096, size: 10},
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
			expectedOutput: map[int]sfdiskOutput{
				1: {start: 0, size: 0},
			},
			expectedError: internalErrors.ErrBadSfdiskPretend,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := parseSfdiskPretend(tt.sfdiskOut, tt.partitionNumbers)
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
