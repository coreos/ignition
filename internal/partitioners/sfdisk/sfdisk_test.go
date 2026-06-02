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
	"fmt"
	"strings"
	"testing"

	sharedErrors "github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/internal/partitioners"
)

func TestParseOutputSinglePartition(t *testing.T) {
	op := &Operation{lastLBA: 67583}

	sfdiskOutput := `Disk /dev/vda: 2 GiB, 2147483648 bytes, 4194304 sectors

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
	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
}

func TestParseOutputTwoPartitions(t *testing.T) {
	op := &Operation{}

	sfdiskOutput := `New situation:
Disklabel type: gpt

Device     Start    End Sectors Size Type
/dev/vda1   2048  67583   65536  32M Linux filesystem
/dev/vda2  67584 133119   65536  32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
	assertOutput(t, result, 2, partitioners.Output{Start: 67584, Size: 65536})
}

func TestParseOutputError(t *testing.T) {
	op := &Operation{}

	sfdiskOutput := `/dev/vda1: Start sector 0 out of range.
Failed to add #1 partition: Numerical result out of range`

	_, err := op.ParseOutput(sfdiskOutput, []int{1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sharedErrors.ErrBadSfdiskPretend) {
		t.Fatalf("expected ErrBadSfdiskPretend, got: %v", err)
	}
}

func TestParseOutputNoFalsePositiveOnTableContent(t *testing.T) {
	op := &Operation{}

	sfdiskOutput := `New situation:
Disklabel type: gpt

Device     Start   End Sectors Size Type
/dev/vda1   2048 67583   65536  32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
}

func TestParseOutputLastLBACorrection(t *testing.T) {
	op := &Operation{lastLBA: 67583}

	sfdiskOutput := `New situation:
Disklabel type: gpt

Device     Start   End Sectors Size Type
/dev/vda1   2048 67582   65535  32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
}

func TestParseOutputNvmeDevice(t *testing.T) {
	op := &Operation{}

	sfdiskOutput := `New situation:
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

func TestParseOutputDeviceAlias(t *testing.T) {
	op := &Operation{}

	sfdiskOutput := `New situation:
Disklabel type: gpt

Device                                        Start   End Sectors Size Type
/run/ignition/dev_aliases/dev/loop0p1          2048 67583   65536  32M Linux filesystem

The partition table is unchanged (--no-act).`

	result, err := op.ParseOutput(sfdiskOutput, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65536})
}

func TestParseOutputWithBootFlag(t *testing.T) {
	op := &Operation{}

	sfdiskOutput := `New situation:
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

func TestBuildScriptFixedSizeBeforeFillRemaining(t *testing.T) {
	p1 := partitioners.Partition{StartSector: int64Ptr(2048), SizeInSectors: int64Ptr(65536)}
	p1.Number = 1
	p5 := partitioners.Partition{StartSector: int64Ptr(460800), SizeInSectors: int64Ptr(65536)}
	p5.Number = 5
	p3 := partitioners.Partition{SizeInSectors: int64Ptr(0)}
	p3.Number = 3

	script := buildScript([]partitioners.Partition{p3, p1, p5}, 657407)

	lines := strings.Split(script, "\n")
	var idx1, idx5, idx3 = -1, -1, -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "1 :") {
			idx1 = i
		}
		if strings.HasPrefix(trimmed, "5 :") {
			idx5 = i
		}
		if strings.HasPrefix(trimmed, "3 :") {
			idx3 = i
		}
	}

	if idx1 < 0 || idx5 < 0 || idx3 < 0 {
		t.Fatalf("missing partition lines in script:\n%s", script)
	}
	if idx1 >= idx3 || idx5 >= idx3 {
		t.Errorf("expected fixed-size before fill-remaining; 1=%d, 5=%d, 3=%d in:\n%s", idx1, idx5, idx3, script)
	}
}

func TestBuildScriptMultipleAutoNumbered(t *testing.T) {
	p1 := partitioners.Partition{SizeInSectors: int64Ptr(65536)}
	p1.Number = 0
	p1.Label = strPtr("uno")
	p2 := partitioners.Partition{SizeInSectors: int64Ptr(65536)}
	p2.Number = 0
	p2.Label = strPtr("dos")
	p3 := partitioners.Partition{SizeInSectors: int64Ptr(65536)}
	p3.Number = 0
	p3.Label = strPtr("tres")

	script := buildScript([]partitioners.Partition{p1, p2, p3}, -1)

	for _, name := range []string{"uno", "dos", "tres"} {
		if !strings.Contains(script, fmt.Sprintf(`name="%s"`, name)) {
			t.Errorf("missing partition %q in script:\n%s", name, script)
		}
	}
}

func TestBuildScriptLastLBAHeader(t *testing.T) {
	p := partitioners.Partition{SizeInSectors: int64Ptr(0)}
	p.Number = 1

	script := buildScript([]partitioners.Partition{p}, 67583)
	if !strings.Contains(script, "last-lba: 67583") {
		t.Errorf("expected last-lba header, got:\n%s", script)
	}

	script = buildScript([]partitioners.Partition{p}, -1)
	if strings.Contains(script, "last-lba:") {
		t.Errorf("should not contain last-lba when unknown, got:\n%s", script)
	}
}

func TestWritePartitionLineMinimal(t *testing.T) {
	p := partitioners.Partition{}
	p.Number = 1

	var buf bytes.Buffer
	writePartitionLine(&buf, p)
	line := buf.String()

	if !strings.HasPrefix(line, "1 :") {
		t.Errorf("expected '1 :', got %q", line)
	}
	if !strings.Contains(line, "size=+") {
		t.Errorf("expected size=+, got %q", line)
	}
}

func int64Ptr(v int64) *int64 { return &v }
func strPtr(v string) *string { return &v }

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
