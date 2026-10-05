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
