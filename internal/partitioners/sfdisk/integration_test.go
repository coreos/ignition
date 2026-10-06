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
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/internal/partitioners"
)

// requireTools locates sfdisk (under test, invoked through PATH the
// same way distro.SfdiskCmd resolves it) and sgdisk (independent read
// oracle). RHEL/Fedora CI ships both.
func requireTools(t *testing.T) (sgdiskPath string) {
	t.Helper()
	if _, err := exec.LookPath("sfdisk"); err != nil {
		t.Skip("sfdisk not available")
	}
	var err error
	sgdiskPath, err = exec.LookPath("sgdisk")
	if err != nil {
		t.Skip("sgdisk not available (needed as independent read oracle)")
	}
	return sgdiskPath
}

func makeImage(t *testing.T, path string, mib int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(int64(mib) * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func runCmd(t *testing.T, input string, args ...string) string {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

// dumpFields parses `sfdisk --dump` output into script fields keyed by
// partition number, using the package's own parser (readExistingPartitions
// relies on it, so parsing here is consistent with the code under test).
func dumpFields(t *testing.T, img string) map[int]map[string]string {
	t.Helper()
	out := runCmd(t, "", "sfdisk", "--dump", img)
	fields := map[int]map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		m := dumpLineRegex.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		nm := dumpNumRegex.FindStringSubmatch(m[1])
		if nm == nil {
			continue
		}
		num, err := strconv.Atoi(nm[1])
		if err != nil {
			continue
		}
		pf := map[string]string{}
		for _, field := range splitFields(m[2]) {
			kv := strings.SplitN(strings.TrimSpace(field), "=", 2)
			if len(kv) == 2 {
				pf[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
			}
		}
		fields[num] = pf
	}
	return fields
}

func headerValue(t *testing.T, img, key string) string {
	t.Helper()
	out := runCmd(t, "", "sfdisk", "--dump", img)
	for _, line := range strings.Split(out, "\n") {
		kv := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == key {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

// diskGUIDFromSgdisk reads the disk GUID with sgdisk as independent
// oracle (not trusting the code under test to read its own writes).
func diskGUIDFromSgdisk(t *testing.T, sgdiskPath, img string) string {
	t.Helper()
	out := runCmd(t, "", sgdiskPath, "-p", img)
	return strings.ToUpper(strings.Fields(strings.SplitAfter(out, "Disk identifier (GUID): ")[1])[0])
}

// TestIntegrationLosslessWholeTableRewrite verifies the core property
// of the whole-table backend: read the on-disk table, merge a queued
// change, rewrite the whole table — every untouched partition must
// survive verbatim (start/size/type GUID/partition GUID/name incl.
// exotic characters/GPT attribute bits), the disk GUID must not churn,
// the queued change must apply, and filesystem data inside preserved
// partitions must not be damaged.
func TestIntegrationLosslessWholeTableRewrite(t *testing.T) {
	sgdiskPath := requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 256)

	// Seed with sgdisk: p1 BIOS boot type + LegacyBIOSBootable attr +
	// fixed partition GUID; p4 plain fixed size. Name with an embedded
	// colon is set via sfdisk afterwards because sgdisk's CLI splits
	// --change-name at the first colon.
	runCmd(t, "", sgdiskPath, "--zap-all", img)
	runCmd(t, "", sgdiskPath,
		"--new=1:2048:+20480",
		"--typecode=1:EF02",
		"--partition-guid=1:11111111-1111-1111-1111-111111111111",
		"--attributes=1:set:2",
		"--new=4:65536:+20480",
		"--typecode=4:8300",
		"--partition-guid=4:44444444-4444-4444-4444-444444444444",
		img)
	// FAT32-looking boot block inside p1 (signature that
	// --wipe-partitions always would erase)
	fat := make([]byte, 512)
	copy(fat[0:3], []byte{0xEB, 0x58, 0x90})
	copy(fat[3:11], []byte("MSWIN4.1"))
	fat[11], fat[12] = 0x00, 0x02
	fat[13] = 1
	fat[14], fat[15] = 0x20, 0x00
	fat[16] = 2
	fat[21] = 0xF8
	fat[24], fat[25] = 0x3F, 0x00
	fat[26], fat[27] = 0xFF, 0x00
	fat[36] = 1
	fat[44], fat[45] = 0x02, 0x00
	fat[46], fat[47] = 0x01, 0x00
	fat[48], fat[49] = 0x06, 0x00
	fat[510], fat[511] = 0x55, 0xAA
	fh, err := os.OpenFile(img, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fh.WriteAt(fat, 2048*512); err != nil {
		t.Fatal(err)
	}
	if err := fh.Close(); err != nil {
		t.Fatal(err)
	}

	// Give p1 an exotic name (embedded colon; forces quoting in dump
	// scripts) through a first whole-table rewrite. This rewrite
	// doubles as the "attrs survive a rewrite" check.
	p1seed := dumpFields(t, img)[1]
	p4seed := dumpFields(t, img)[4]
	rewrite := "label: gpt\ngrain: 512\n"
	for _, k := range []string{"label-id", "first-lba", "last-lba"} {
		if v := headerValue(t, img, k); v != "" {
			rewrite += k + ": " + v + "\n"
		}
	}
	rewrite += "\n" +
		"1 : start=" + p1seed["start"] + ", size=" + p1seed["size"] +
		", type=" + p1seed["type"] + ", uuid=" + p1seed["uuid"] +
		`, name="bi=os:name", attrs="LegacyBIOSBootable"` + "\n" +
		"4 : start=" + p4seed["start"] + ", size=" + p4seed["size"] +
		", type=" + p4seed["type"] + ", uuid=" + p4seed["uuid"] + "\n"
	runCmd(t, rewrite, "sfdisk", "--no-reread", "--wipe", "auto", "--wipe-partitions", "auto", "-X", "gpt", img)

	// attrs survived that first rewrite
	attrOut := runCmd(t, "", sgdiskPath, "-i", "1", img)
	if !strings.Contains(strings.ToLower(attrOut), "0000000000000004") {
		t.Fatalf("LegacyBIOSBootable lost on seed rewrite (test setup): %s", attrOut)
	}

	before := dumpFields(t, img)
	diskGUIDBefore := diskGUIDFromSgdisk(t, sgdiskPath, img)

	// Now the real test: the stage deletes+recreates p4 for a resize;
	// p1 must survive verbatim WITHOUT the config mentioning its attrs.
	op := Begin(nil, img)
	op.DeletePartition(4)
	p4 := partitioners.Partition{}
	p4.Number = 4
	p4.Label = util.StrToPtr("data")
	size := int64(40960)
	p4.SizeInSectors = &size
	start := int64(65536)
	p4.StartSector = &start
	typeGUID := "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
	p4.TypeGUID = &typeGUID
	op.CreatePartition(p4)

	if err := op.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	after := dumpFields(t, img)
	a1, ok := after[1]
	if !ok {
		t.Fatalf("p1 vanished after rewrite: %v", after)
	}
	b1 := before[1]
	for _, key := range []string{"start", "size", "type", "uuid", "name"} {
		if !strings.EqualFold(a1[key], b1[key]) {
			t.Errorf("untouched p1 %s churned: %q -> %q", key, b1[key], a1[key])
		}
	}
	if a1["attrs"] == "" {
		t.Errorf("attrs field missing for preserved p1 after rewrite: %v", a1)
	}
	// queued change applied
	a4, ok := after[4]
	if !ok {
		t.Fatalf("p4 vanished after rewrite")
	}
	if a4["start"] != "65536" || a4["size"] != "40960" {
		t.Errorf("p4 resize not applied: %v", a4)
	}
	if !strings.EqualFold(a4["type"], typeGUID) {
		t.Errorf("p4 type not applied: %v", a4["type"])
	}
	// disk GUID preserved (independent oracle)
	if got := diskGUIDFromSgdisk(t, sgdiskPath, img); got != diskGUIDBefore {
		t.Errorf("disk GUID churned: %s -> %s", diskGUIDBefore, got)
	}
	// attribute bit on p1 still there via oracle (attrs round-tripped
	// through readExistingPartitions -> script -> sfdisk)
	attrOut = runCmd(t, "", sgdiskPath, "-i", "1", img)
	if !strings.Contains(strings.ToLower(attrOut), "0000000000000004") {
		t.Errorf("LegacyBIOSBootable lost on untouched p1:\n%s", attrOut)
	}
	// FAT boot block inside untouched p1 still intact
	fh, err = os.Open(img)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := fh.Close(); err != nil {
			t.Error(err)
		}
	}()
	buf := make([]byte, 512)
	if _, err := fh.ReadAt(buf, 2048*512); err != nil {
		t.Fatal(err)
	}
	if buf[0] != 0xEB || buf[510] != 0x55 || buf[511] != 0xAA {
		t.Errorf("FAT boot block wiped on untouched p1")
	}
}

// TestIntegrationIdempotentRewrite: committing with no queued changes
// (what the stage does when everything already matches) must be a
// no-op for every field — including header values. Seeds a
// non-default first-lba (34, sgdisk's own default) to catch
// header-defaulting churn (sfdisk defaults first-lba to 2048).
func TestIntegrationIdempotentRewrite(t *testing.T) {
	sgdiskPath := requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 64)
	// Seed through sfdisk with a non-default first-lba (34) and a fill
	// partition: sfdisk's own GPT default first-lba is 2048, so a
	// rewrite that failed to pin the header would churn first-lba,
	// shift preserved geometry, and the sgdisk-built tables ignition
	// provisions start at 34-class offsets in the wild (butane's
	// default on some paths).
	seed := "label: gpt\ngrain: 512\nfirst-lba: 34\n\n1 : start=34, size=2048\n2 : size=+\n"
	runCmd(t, seed, "sfdisk", "--force", "-X", "gpt", img)

	before := dumpFields(t, img)
	beforeGUID := diskGUIDFromSgdisk(t, sgdiskPath, img)

	op := Begin(nil, img)
	if err := op.Commit(); err != nil {
		t.Fatalf("no-op commit failed: %v", err)
	}

	after := dumpFields(t, img)
	for num, bf := range before {
		af, ok := after[num]
		if !ok {
			t.Errorf("p%d vanished on no-op commit", num)
			continue
		}
		for _, key := range []string{"start", "size", "type", "uuid", "name", "attrs"} {
			if !strings.EqualFold(bf[key], af[key]) {
				t.Errorf("p%d %s churned on no-op commit: %q -> %q", num, key, bf[key], af[key])
			}
		}
	}
	if got := diskGUIDFromSgdisk(t, sgdiskPath, img); got != beforeGUID {
		t.Errorf("disk GUID churned on no-op commit: %s -> %s", beforeGUID, got)
	}
	if fl := headerValue(t, img, "first-lba"); fl != "34" {
		t.Errorf("first-lba churned on no-op commit: want 34, got %q", fl)
	}
}

// TestIntegrationAutoPositionedWithPreserved reproduces the kola
// coreos.ignition.mount.partitions failure class: a config partition
// with no start (auto-positioned) and no number, mixed with preserved
// (pinned) partitions re-listed by the whole-table backend. The
// auto-positioned entry must not be parked inside a pinned block
// (that produced "Failed to add #2 partition: Numerical result out of
// range" and an emergency shell).
func TestIntegrationAutoPositionedWithPreserved(t *testing.T) {
	requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 256)
	// FCOS-like layout: BIOS boot + ESP at the front, boot and root
	// further in, free space in between and after.
	seed := "label: gpt\ngrain: 512\n\n\n1 : start=2048, size=2048, type=21686148-6449-6E6F-744E-656564454649, name=\"BIOS-BOOT\"\n2 : start=4096, size=16256, type=C12A7328-F81F-11D2-BA4B-00A0C93EC93B, name=\"EFI-SYSTEM\"\n3 : start=204192, size=49152, type=BC13C2FF-59E6-4262-A352-B275FD6F7172, name=\"boot\"\n4 : start=264192, size=131072, type=4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709, name=\"root\"\n"
	runCmd(t, seed, "sfdisk", "--force", "-X", "gpt", img)

	// The stage queued an auto-positioned fixed-size partition (no
	// number, no start) on top of preserved p1-p4 which the whole-
	// table backend re-lists with pinned geometry.
	op := Begin(nil, img)
	contr := partitioners.Partition{}
	contr.SizeInSectors = int64Ptr(131072)
	contr.TypeGUID = strPtr("63194b49-e4b7-43f9-9a8b-df0fd8279bb7")
	op.CreatePartition(contr)

	if _, err := op.Pretend(); err != nil {
		t.Fatalf("pretend failed (ERANGE class): %v", err)
	}
	if err := op.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	after := dumpFields(t, img)
	for num, start := range map[int]string{1: "2048", 2: "4096", 3: "204192", 4: "264192"} {
		if after[num]["start"] != start {
			t.Errorf("p%d pinned geometry moved: want start %s, got %v", num, start, after[num])
		}
	}
	if _, ok := after[5]; !ok {
		t.Fatalf("auto-positioned partition not created: %v", after)
	}
	if !strings.EqualFold(after[5]["type"], "63194b49-e4b7-43f9-9a8b-df0fd8279bb7") {
		t.Errorf("p5 type wrong: %v", after[5])
	}
	for num := 1; num <= 4; num++ {
		if after[5]["start"] == after[num]["start"] {
			t.Errorf("p5 overlaps p%d", num)
		}
	}
}

// stageFlow mimics what the disks stage does between Pretend and
// Commit: run Pretend, parse resolved geometry for inspected
// partitions, write those values back into the queued partitions,
// then commit a fresh op with the resolved (explicit) geometry.
func stageFlow(t *testing.T, img string, queued []partitioners.Partition, inspected []int) map[int]partitioners.Output {
	t.Helper()
	op := Begin(nil, img)
	for _, p := range queued {
		op.CreatePartition(p)
	}
	out, err := op.Pretend()
	if err != nil {
		t.Fatalf("pretend: %v", err)
	}
	dims, err := op.ParseOutput(out, inspected)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	op2 := Begin(nil, img)
	for _, p := range queued {
		if d, ok := dims[p.Number]; ok {
			s, z := d.Start, d.Size
			p.StartSector = &s
			p.SizeInSectors = &z
		}
		op2.CreatePartition(p)
	}
	if err := op2.Commit(); err != nil {
		t.Fatalf("commit with resolved geometry: %v", err)
	}
	return dims
}

// TestIntegrationStageFlowBlankFill verifies the blackbox
// partition.create.startsize0 class end-to-end on a blank device: a
// size=0/number=1/start=0 (fill) partition must be committed and
// reported with identical geometry, with the fill reaching the last
// usable sector (sfdisk default usable end = totalSectors-34) exactly
// as sgdisk's would.
func TestIntegrationStageFlowBlankFill(t *testing.T) {
	requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 64) // 131072 sectors -> default usable end 131038

	p := partitioners.Partition{}
	p.Number = 1
	zero := int64(0)
	p.StartSector = &zero
	p.SizeInSectors = &zero
	p.Label = strPtr("fills-disk")

	dims := stageFlow(t, img, []partitioners.Partition{p}, []int{1})
	after := dumpFields(t, img)
	got := after[1]
	start, _ := strconv.ParseInt(got["start"], 10, 64)
	size, _ := strconv.ParseInt(got["size"], 10, 64)
	// reported == committed on disk (no boot-to-boot churn)
	if dims[1].Start != start || dims[1].Size != size {
		t.Errorf("reported %v != on-disk start=%d size=%d", dims[1], start, size)
	}
	// fill reaches last usable exactly: end 131038 == 131072-34
	if end := start + size - 1; end != 131072-34 {
		t.Errorf("fill end=%d, want %d (sgdisk parity)", end, 131072-34)
	}
}

// TestIntegrationStageFlowExistingFillOnPinnedTable covers the kola
// class: a fill queued against an EXISTING table whose last-lba is
// pinned from the header. Whatever Pretend resolves must be exactly
// what Commit puts on disk.
func TestIntegrationStageFlowExistingFillOnPinnedTable(t *testing.T) {
	requireTools(t)
	sgdisk := requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 128)
	runCmd(t, "", sgdisk, "--zap-all", img)
	runCmd(t, "", sgdisk, "--new=1:2048:+20480", "--new=2:45056:+20480", img)

	p := partitioners.Partition{}
	p.Number = 3
	p.SizeInSectors = int64Ptr(0) // fill
	p.Label = strPtr("log")
	dims := stageFlow(t, img, []partitioners.Partition{p}, []int{3})

	after := dumpFields(t, img)
	got := after[3]
	start, _ := strconv.ParseInt(got["start"], 10, 64)
	size, _ := strconv.ParseInt(got["size"], 10, 64)
	if dims[3].Start != start || dims[3].Size != size {
		t.Errorf("reported %v != on-disk start=%d size=%d", dims[3], start, size)
	}
	// preserved p1/p2 untouched
	p1 := dumpFields(t, img)[1]
	if p1["start"] != "2048" || p1["size"] != "20480" {
		t.Errorf("p1 churned: %v", p1)
	}
	// the fill must not overflow the header's last usable sector
	hdr := headerValue(t, img, "last-lba")
	lastUsable, _ := strconv.ParseInt(hdr, 10, 64)
	if end := start + size - 1; end > lastUsable {
		t.Errorf("committed fill end %d exceeds last-lba %d", end, lastUsable)
	}
}

// TestIntegrationDeleteAllWritesEmptyTable covers the blackbox
// partition.delete / partition.delete.all class: deleting every
// partition must actually rewrite an empty table (the sgdisk backend
// committed each delete), preserving the disk GUID and not touching
// data areas.
func TestIntegrationDeleteAllWritesEmptyTable(t *testing.T) {
	requireTools(t)
	sgdisk := requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 64)
	runCmd(t, "", sgdisk, "--zap-all", img)
	runCmd(t, "", sgdisk, "--new=1:2048:+20480", "--new=2:45056:+20480", img)
	guidBefore := diskGUIDFromSgdisk(t, sgdisk, img)

	op := Begin(nil, img)
	op.DeletePartition(1)
	op.DeletePartition(2)
	if err := op.Commit(); err != nil {
		t.Fatalf("delete-all commit: %v", err)
	}
	after := dumpFields(t, img)
	if len(after) != 0 {
		t.Errorf("delete-all left partitions on disk: %v", after)
	}
	if guidAfter := diskGUIDFromSgdisk(t, sgdisk, img); guidAfter != guidBefore {
		t.Errorf("disk GUID churned on delete-all: %s -> %s", guidBefore, guidAfter)
	}
}

// TestIntegrationPretendHonorsPinnedLastLBA locks in the --force
// removal: Pretend must resolve fills against the pinned table
// header, never sfdisk's default usable end (that divergence is what
// made commits fail with "out of range").
func TestIntegrationPretendHonorsPinnedLastLBA(t *testing.T) {
	requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 128) // 262144 sectors, default usable end 262110
	// build a table whose last-lba is deliberately small
	seed := "label: gpt\ngrain: 512\nfirst-lba: 34\nlast-lba: 130000\n\n1 : start=34, size=2048\n"
	runCmd(t, seed, "sfdisk", "-X", "gpt", img)

	op := Begin(nil, img)
	fill := partitioners.Partition{}
	fill.Number = 2
	fill.SizeInSectors = int64Ptr(0)
	op.CreatePartition(fill)
	out, err := op.Pretend()
	if err != nil {
		t.Fatalf("pretend: %v", err)
	}
	dims, err := op.ParseOutput(out, []int{2})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if op.lastLBA != 130000 {
		t.Fatalf("op.lastLBA=%d, want pinned 130000", op.lastLBA)
	}
	// fill end must respect the pinned usable end (correction makes
	// reported end == 130000), and stay far below the disk default
	env := dims[2]
	if end := env.Start + env.Size - 1; end > 130000 {
		t.Errorf("fill resolved to end %d > pinned last-lba 130000 (--force regression)", end)
	}
}

// TestIntegrationStageFlowWipeReprovision mirrors the kola
// root-reprovision.swap-before-root sequence: wipeTable on a fresh
// disk, then queue a fixed-size partition with auto start plus a fill
// root through Pretend -> ParseOutput -> Commit, asserting the
// geometry ignition reports is exactly what lands on disk and the
// fill ends on the header's last usable sector (sgdisk parity).
func TestIntegrationStageFlowWipeReprovision(t *testing.T) {
	requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 256)
	// pre-existing junk table to wipe
	runCmd(t, "label: gpt\ngrain: 512\n\n\n1 : start=2048, size=2048\n", "sfdisk", "--force", "-X", "gpt", img)

	// stage: wipeTable op
	wipeOp := Begin(nil, img)
	wipeOp.WipeTable(true)
	if err := wipeOp.Commit(); err != nil {
		t.Fatalf("wipe commit: %v", err)
	}

	// stage: swap (fixed size, auto start, number 1) + root fill (number 2)
	swap := partitioners.Partition{}
	swap.Number = 1
	swap.SizeInSectors = int64Ptr(20480)
	swap.Label = strPtr("swap")
	root := partitioners.Partition{}
	root.Number = 2
	root.SizeInSectors = int64Ptr(0) // fill
	root.Label = strPtr("root")

	dims := stageFlow(t, img, []partitioners.Partition{swap, root}, []int{1, 2})
	after := dumpFields(t, img)

	for num := 1; num <= 2; num++ {
		start, _ := strconv.ParseInt(after[num]["start"], 10, 64)
		size, _ := strconv.ParseInt(after[num]["size"], 10, 64)
		if dims[num].Start != start || dims[num].Size != size {
			t.Errorf("p%d reported %v != on-disk start=%d size=%d", num, dims[num], start, size)
		}
	}
	lastUsable, _ := strconv.ParseInt(headerValue(t, img, "last-lba"), 10, 64)
	if end := dims[2].Start + dims[2].Size - 1; end != lastUsable {
		t.Errorf("root fill end %d != last usable %d", end, lastUsable)
	}
	// re-running the same stage flow must be a no-op (stable-boot class
	// regression: match detection needs byte-stable geometry across boots).
	// On the second boot p1 MATCHES, so the stage re-lists it with the
	// on-disk identity (see the "exists && shouldExist && matches"
	// branch: StartSector/SizeInSectors/TypeGUID/GUID/Label from info);
	// p2 stays queued as a fill because size 0 means "don't care".
	op2 := Begin(nil, img)
	p1 := partitioners.Partition{}
	p1.Number = 1
	p1.Label = strPtr("swap")
	swapSize := int64(20480)
	p1.SizeInSectors = &swapSize
	p1.StartSector = int64Ptr(dims2start(after[1]))
	if g := after[1]["uuid"]; g != "" {
		p1.GUID = strPtr(strings.ToUpper(g))
	}
	op2.CreatePartition(p1)
	out, err := op2.Pretend()
	if err != nil {
		t.Fatalf("second pretend: %v", err)
	}
	dims2, err := op2.ParseOutput(out, []int{1})
	if err != nil {
		t.Fatalf("second parse: %v", err)
	}
	if err := op2.Commit(); err != nil {
		t.Fatalf("second commit: %v", err)
	}
	after2 := dumpFields(t, img)
	for num := 1; num <= 2; num++ {
		for _, k := range []string{"start", "size", "uuid", "name"} {
			if !strings.EqualFold(after[num][k], after2[num][k]) {
				t.Errorf("p%d %s churned on re-run: %q -> %q", num, k, after[num][k], after2[num][k])
			}
		}
	}
	if !strings.EqualFold(after[1]["start"], strconv.FormatInt(dims2[1].Start, 10)) {
		t.Errorf("swap geometry drifted across boots: on-disk %v vs reported %v", after[1], dims2[1])
	}
}

// dims2start parses the start sector out of a dumpFields entry.
func dims2start(fields map[string]string) int64 {
	v, _ := strconv.ParseInt(fields["start"], 10, 64)
	return v
}

// TestIntegrationWipeLeavesNoTable reproduces the kola
// coreos.misc.disk.varlibcontainers class: wipeTable on a disk with a
// GPT must leave NO partition table signature (systemd-mkfs and
// friends refuse to whole-disk-format a disk blkid still calls
// "gpt"), while leaving filesystem data inside former partitions
// intact (sgdisk --zap-all semantics).
func TestIntegrationWipeLeavesNoTable(t *testing.T) {
	requireTools(t)
	sgdisk := requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 64)
	runCmd(t, "", sgdisk, "--zap-all", img)
	runCmd(t, "", sgdisk, "--new=1:2048:+20480", img)
	// FAT-looking boot block inside p1: data that must survive
	fat := make([]byte, 512)
	fat[0], fat[1], fat[2] = 0xEB, 0x58, 0x90
	copy(fat[3:11], []byte("MSWIN4.1"))
	fat[510], fat[511] = 0x55, 0xAA
	fh, err := os.OpenFile(img, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fh.WriteAt(fat, 2048*512); err != nil {
		t.Fatal(err)
	}
	if err := fh.Close(); err != nil {
		t.Fatal(err)
	}

	op := Begin(nil, img)
	op.WipeTable(true)
	if err := op.Commit(); err != nil {
		t.Fatalf("wipe commit: %v", err)
	}

	// sfdisk must no longer see any table
	cmdOut, cmdErr := runCmdAllowFail(t, "sfdisk", "--dump", img)
	if cmdErr == nil || !strings.Contains(cmdOut, "does not contain a recognized partition table") {
		t.Errorf("wipe left a detectable table (out %q err %v)", cmdOut, cmdErr)
	}
	// data inside the former partition intact
	fh, err = os.Open(img)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fh.Close() }()
	buf := make([]byte, 512)
	if _, err := fh.ReadAt(buf, 2048*512); err != nil {
		t.Fatal(err)
	}
	if buf[0] != 0xEB || buf[510] != 0x55 || buf[511] != 0xAA {
		t.Errorf("wipe destroyed filesystem data in former p1 area")
	}
}

// runCmdAllowFail runs a command without failing the test on error.
func runCmdAllowFail(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestIntegrationWipeThenCreate mirrors the reprovision boot-1 flow:
// wipeTable on a disk with an existing layout, then queue explicit
// partitions; the create must succeed on the freshly-zapped disk.
func TestIntegrationWipeThenCreate(t *testing.T) {
	requireTools(t)
	sgdisk := requireTools(t)
	dir := t.TempDir()
	img := dir + "/disk.img"
	makeImage(t, img, 128)
	runCmd(t, "", sgdisk, "--zap-all", img)
	runCmd(t, "", sgdisk, "--new=1:2048:+20480", img)

	op := Begin(nil, img)
	op.WipeTable(true)
	if err := op.Commit(); err != nil {
		t.Fatalf("wipe commit: %v", err)
	}

	p := partitioners.Partition{}
	p.Number = 1
	p.StartSector = int64Ptr(2048)
	p.SizeInSectors = int64Ptr(65536)
	p.Label = strPtr("fresh")
	dims := stageFlow(t, img, []partitioners.Partition{p}, nil)
	after := dumpFields(t, img)
	if after[1]["size"] != "65536" || after[1]["name"] != `"fresh"` {
		t.Errorf("post-wipe create wrong: %v", after[1])
	}
	if dims[1].Size != 65536 {
		t.Errorf("post-wipe reported size %v", dims[1])
	}
}
