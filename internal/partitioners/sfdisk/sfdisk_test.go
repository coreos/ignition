// Copyright 2024 Red Hat, Inc.
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

package sfdisk

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	sharedErrors "github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/internal/partitioners"
)

func TestParseOutputSinglePartition(t *testing.T) {
	op := Begin(nil, "")

	sfdiskOutput := `Disk /dev/vda: 2 GiB, 2147483648 bytes, 4194304 sectors
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes

>>> Created a new GPT disklabel.
/dev/vda1: Created a new partition 1 of type 'Linux filesystem' and of size 32 MiB.
/dev/vda2: Done.

New situation:
Disklabel type: gpt

Device     Start   End Sectors Size Type
/dev/vda1   2048 67583   65536  32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 partition, got %d", len(result))
	}

	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
}

func TestParseOutputTwoPartitions(t *testing.T) {
	op := Begin(nil, "")

	sfdiskOutput := `Disk /dev/vda: 2 GiB, 2147483648 bytes, 4194304 sectors

New situation:
Disklabel type: gpt

Device     Start    End Sectors Size Type
/dev/vda1   2048  67583   65536  32M Linux filesystem
/dev/vda2  67584 133119   65536  32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 partitions, got %d", len(result))
	}

	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
	assertOutput(t, result, 2, partitioners.Output{Start: 67584, Size: 65536})
}

func TestParseOutputError(t *testing.T) {
	op := Begin(nil, "")

	sfdiskOutput := `/dev/vda1: Start sector 0 out of range.
Failed to add #1 partition: Numerical result out of range`

	_, err := op.ParseOutput(sfdiskOutput, []int{1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, sharedErrors.ErrBadSfdiskPretend) {
		t.Fatalf("expected error wrapping ErrBadSfdiskPretend, got: %v", err)
	}
}

func TestParseOutputNoFalsePositiveOnTableContent(t *testing.T) {
	op := Begin(nil, "")

	sfdiskOutput := `Disk /dev/vda: 2 GiB, 2147483648 bytes, 4194304 sectors

New situation:
Disklabel type: gpt

Device     Start   End Sectors Size Type
/dev/vda1   2048 67583   65536  32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1})
	if err != nil {
		t.Fatalf("unexpected error (false positive on table content): %v", err)
	}

	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
}

func TestBuildCompleteScriptFillRemaining(t *testing.T) {
	start := int64(67584)
	size := int64(65536)
	partitions := map[int]partitioners.Partition{
		1: {StartSector: &start, SizeInSectors: &size},
		2: {StartSector: &start},
	}
	partitions[1] = partitioners.Partition{StartSector: func() *int64 { v := int64(2048); return &v }(), SizeInSectors: &size}
	partitions[2] = partitioners.Partition{StartSector: &start, SizeInSectors: func() *int64 { v := int64(0); return &v }()}

	script := buildCompleteScript(partitions, 204766)

	// Should contain explicit size for fill-remaining partition
	if expected := "size=137183"; !strings.Contains(script, expected) {
		t.Errorf("expected script to contain %q for fill-remaining partition, got:\n%s", expected, script)
	}
	if strings.Contains(script, "size=+") {
		t.Errorf("script should not contain size=+ when lastLBA is known, got:\n%s", script)
	}
}

func TestBuildCompleteScriptFillRemainingUnknownLBA(t *testing.T) {
	start := int64(67584)
	partitions := map[int]partitioners.Partition{
		1: {StartSector: &start, SizeInSectors: func() *int64 { v := int64(0); return &v }()},
	}

	script := buildCompleteScript(partitions, -1)

	if !strings.Contains(script, "size=+") {
		t.Errorf("expected script to contain size=+ when lastLBA is unknown, got:\n%s", script)
	}
}

func TestParseOutputNvmeDevice(t *testing.T) {
	op := Begin(nil, "")

	sfdiskOutput := `Disk /dev/nvme0n1: 100 GiB, 107374182400 bytes, 209715200 sectors

New situation:
Disklabel type: gpt

Device            Start      End  Sectors  Size Type
/dev/nvme0n1p1     2048    67583    65536   32M Linux filesystem
/dev/nvme0n1p2    67584   133119    65536   32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
	assertOutput(t, result, 2, partitioners.Output{Start: 67584, Size: 65536})
}

func TestParseOutputWithBootFlag(t *testing.T) {
	op := Begin(nil, "")

	sfdiskOutput := `Disk /dev/sda: 50 GiB

New situation:
Disklabel type: gpt

Device     Start    End Sectors  Size Type
/dev/sda1  *  2048  67583   65536   32M EFI System
/dev/sda2    67584 133119   65536   32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
	assertOutput(t, result, 2, partitioners.Output{Start: 67584, Size: 65536})
}

func TestParseOutputEmptyPartitions(t *testing.T) {
	op := Begin(nil, "")
	result, err := op.ParseOutput("anything", []int{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result for empty partitionNumbers, got %v", result)
	}
}

func TestBuildCompleteScriptMixedNumbering(t *testing.T) {
	start1 := int64(2048)
	size1 := int64(65536)
	start2 := int64(67584)
	size2 := int64(65536)

	p1 := partitioners.Partition{StartSector: &start1, SizeInSectors: &size1}
	p1.Number = 1
	p3 := partitioners.Partition{StartSector: &start2, SizeInSectors: &size2}
	p3.Number = 3
	p0 := partitioners.Partition{SizeInSectors: &size2}
	p0.Number = 0

	partitions := map[int]partitioners.Partition{
		1: p1,
		3: p3,
		0: p0,
	}

	script := buildCompleteScript(partitions, -1)

	lines := strings.Split(script, "\n")
	var num1Idx, num3Idx, autoIdx int
	num1Idx, num3Idx, autoIdx = -1, -1, -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "1 :") {
			num1Idx = i
		}
		if strings.HasPrefix(trimmed, "3 :") {
			num3Idx = i
		}
		if trimmed == ":" || strings.HasPrefix(trimmed, ": ") {
			autoIdx = i
		}
	}

	if num1Idx < 0 || num3Idx < 0 || autoIdx < 0 {
		t.Fatalf("missing partition lines in script:\n%s", script)
	}
	if num1Idx >= num3Idx || num3Idx >= autoIdx {
		t.Errorf("expected partition order: 1, 3, auto-numbered; got lines %d, %d, %d in script:\n%s",
			num1Idx, num3Idx, autoIdx, script)
	}
}

func TestWritePartitionLineMinimal(t *testing.T) {
	p := partitioners.Partition{}
	p.Number = 1

	var buf bytes.Buffer
	writePartitionLine(&buf, p, -1)
	line := buf.String()

	if !strings.HasPrefix(line, "1 :") {
		t.Errorf("expected line to start with '1 :', got %q", line)
	}
	if !strings.Contains(line, "size=+") {
		t.Errorf("expected size=+ for partition with no size, got %q", line)
	}
	if !strings.HasSuffix(line, "\n") {
		t.Errorf("expected line to end with newline, got %q", line)
	}
}

func assertOutput(t *testing.T, result map[int]partitioners.Output, num int, expected partitioners.Output) {
	t.Helper()
	out, ok := result[num]
	if !ok {
		t.Fatalf("partition %d not found in result", num)
	}
	if out.Start != expected.Start {
		t.Errorf("partition %d: expected start %d, got %d", num, expected.Start, out.Start)
	}
	if out.Size != expected.Size {
		t.Errorf("partition %d: expected size %d, got %d", num, expected.Size, out.Size)
	}
}
