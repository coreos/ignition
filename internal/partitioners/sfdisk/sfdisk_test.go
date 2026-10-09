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

	script := buildScript([]partitioners.Partition{p3, p1, p5}, diskHeader{lastLBA: 657407})

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

func TestBuildScriptPinnedBeforeAutoPositioned(t *testing.T) {
	// Regression: kola coreos.ignition.mount.partitions class. An
	// auto-positioned (no start) fixed-size create must be emitted
	// AFTER position-pinned entries (preserved partitions re-listed
	// by the whole-table backend), or sfdisk parks the auto entry in
	// a block a later pinned line claims and the rewrite fails with
	// ERANGE. Fills stay last.
	pinned1 := partitioners.Partition{StartSector: int64Ptr(2048), SizeInSectors: int64Ptr(2048)}
	pinned1.Number = 1
	pinned2 := partitioners.Partition{StartSector: int64Ptr(4096), SizeInSectors: int64Ptr(260096)}
	pinned2.Number = 2
	autoFixed := partitioners.Partition{SizeInSectors: int64Ptr(2097152)}
	autoFixed.Number = 0
	fill := partitioners.Partition{SizeInSectors: int64Ptr(0)}
	fill.Number = 0

	script := buildScript([]partitioners.Partition{autoFixed, pinned2, fill, pinned1}, diskHeader{firstLBA: 2048, lastLBA: 20969472})
	lines := strings.Split(script, "\n")
	pos := map[string]int{}
	for i, line := range lines {
		t2 := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t2, "1 :"):
			pos["pinned1"] = i
		case strings.HasPrefix(t2, "2 :"):
			pos["pinned2"] = i
		case strings.HasPrefix(t2, ": size=2097152"):
			pos["autoFixed"] = i
		case strings.HasPrefix(t2, ": size=+"):
			pos["fill"] = i
		}
	}
	if len(pos) != 4 {
		t.Fatalf("missing lines in script:\n%s", script)
	}
	if pos["pinned1"] >= pos["autoFixed"] || pos["pinned2"] >= pos["autoFixed"] || pos["autoFixed"] >= pos["fill"] {
		t.Errorf("want pinned1,pinned2 < autoFixed < fill, got %v in:\n%s", pos, script)
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

	script := buildScript([]partitioners.Partition{p1, p2, p3}, diskHeader{lastLBA: -1})

	for _, name := range []string{"uno", "dos", "tres"} {
		if !strings.Contains(script, fmt.Sprintf(`name="%s"`, name)) {
			t.Errorf("missing partition %q in script:\n%s", name, script)
		}
	}
}

// TestMergePartitionsAssignsAutoNumbersInOrder locks the fix for the
// number-0 ordering bug: auto-numbered (Number == 0) entries must be
// assigned real numbers in op.parts (config) order using the first-free
// rule, independently of the tier order buildScript later sorts them into.
// Otherwise sfdisk numbers the bare script lines by script order and
// ParseOutput maps each partition's geometry onto the wrong one.
// wipe is set so mergePartitions skips the on-disk read and the test needs
// no real device.
func TestMergePartitionsAssignsAutoNumbersInOrder(t *testing.T) {
	op := &Operation{wipe: true}
	// Auto-numbered, no start (sorts last); explicit #2; auto-numbered
	// with an explicit start (sorts first). Despite the sort, the auto
	// entries must take the first free slots in op.parts order: 1 then 3.
	op.CreatePartition(partitioners.Partition{SizeInSectors: int64Ptr(100)})
	explicit := partitioners.Partition{StartSector: int64Ptr(5000), SizeInSectors: int64Ptr(100)}
	explicit.Number = 2
	op.CreatePartition(explicit)
	op.CreatePartition(partitioners.Partition{StartSector: int64Ptr(200), SizeInSectors: int64Ptr(100)})

	merged, err := op.mergePartitions()
	if err != nil {
		t.Fatalf("mergePartitions: %v", err)
	}
	got := []int{}
	for _, p := range merged {
		got = append(got, p.Number)
	}
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("expected %d partitions, got %v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("partition %d: expected number %d, got %d (all: %v)", i, want[i], got[i], got)
		}
	}
}

func TestBuildScriptLastLBAHeader(t *testing.T) {
	p := partitioners.Partition{SizeInSectors: int64Ptr(0)}
	p.Number = 1

	script := buildScript([]partitioners.Partition{p}, diskHeader{lastLBA: 67583, labelID: "12345678-1234-1234-1234-123456789abc"})
	if !strings.Contains(script, "last-lba: 67583") {
		t.Errorf("expected last-lba header, got:\n%s", script)
	}
	if !strings.Contains(script, "label-id: 12345678-1234-1234-1234-123456789abc") {
		t.Errorf("expected label-id header, got:\n%s", script)
	}

	script = buildScript([]partitioners.Partition{p}, diskHeader{lastLBA: -1})
	if strings.Contains(script, "last-lba:") {
		t.Errorf("should not contain last-lba when unknown, got:\n%s", script)
	}
	if strings.Contains(script, "label-id:") {
		t.Errorf("should not contain label-id when unknown, got:\n%s", script)
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

func TestWritePartitionLineAttrs(t *testing.T) {
	p := partitioners.Partition{Attrs: []string{"LegacyBIOSBootable", "NoAutomount"}}
	p.Number = 1

	var buf bytes.Buffer
	writePartitionLine(&buf, p)

	if !strings.Contains(buf.String(), `attrs="LegacyBIOSBootable,NoAutomount"`) {
		t.Errorf("expected attrs re-emitted, got %q", buf.String())
	}
}

func TestWritePartitionLineLabelQuoting(t *testing.T) {
	// Labels are carried raw and encoded to sfdisk's quoted/\xHH form,
	// mirroring `sfdisk --dump` output and the loader's
	// unhexmangle_string(), so exotic names round-trip byte-exactly.
	cases := []struct {
		label string
		want  string
	}{
		{`simple`, "\"simple\""},
		{`a:b`, "\"a:b\"" /* verbatim check below */},
		{`with space`, "\"with space\"" /* verbatim check below */},
		{`q"b`, "\"q\\x22b\"" /* verbatim check below */},
		{`back\slash`, "\"back\\x5cslash\"" /* verbatim check below */},
		{"ctl\x01byte", "\"ctl\\x01byte\"" /* verbatim check below */},
	}
	for _, c := range cases {
		p := partitioners.Partition{}
		p.Label = strPtr(c.label)
		p.Number = 1
		var buf bytes.Buffer
		writePartitionLine(&buf, p)
		if !strings.Contains(buf.String(), "name="+c.want) {
			t.Errorf("label %q: want name=%s in %q", c.label, c.want, buf.String())
		}
	}
}

func TestSfdiskNameCodec(t *testing.T) {
	// encode -> decode is the identity for every single byte value
	for i := 0; i < 256; i++ {
		raw := string([]byte{byte(i)}) + "x"
		enc := encodeSfdiskName(raw)
		dec, err := decodeSfdiskName(enc)
		if err != nil {
			t.Fatalf("byte %d: decode error %v", i, err)
		}
		if dec != raw {
			t.Errorf("byte %d: round-trip failed: enc=%q dec=%q", i, enc, dec)
		}
	}
}

func TestParseOutputLastLBACorrectionScopedToInspected(t *testing.T) {
	// A partition NOT queued for inspection (explicit geometry) that
	// happens to end at lastLBA-1 must keep its exact size; only fill
	// (inspected) partitions get the +1 correction.
	op := &Operation{lastLBA: 67583}

	sfdiskOutput := `New situation:
Disklabel type: gpt

Device     Start   End Sectors Size Type
/dev/vda1   2048 67582   65535  32M Linux filesystem

The partition table is unchanged (--no-act).`

	// partition 1 present on disk but not inspected (no zero geometry)
	result, err := op.ParseOutput(sfdiskOutput, []int{2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutput(t, result, 1, partitioners.Output{Start: 2048, Size: 65535})
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
