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
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	sharedErrors "github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/partitioners"
)

type Operation struct {
	logger    *log.Logger
	dev       string
	wipe      bool
	parts     []partitioners.Partition
	deletions []int
	infos     []int
	lastLBA   int64
}

func Begin(logger *log.Logger, dev string) *Operation {
	return &Operation{logger: logger, dev: dev}
}

func (op *Operation) CreatePartition(p partitioners.Partition) {
	op.parts = append(op.parts, p)
}

func (op *Operation) DeletePartition(num int) {
	op.deletions = append(op.deletions, num)
}

func (op *Operation) Info(num int) {
	op.infos = append(op.infos, num)
}

func (op *Operation) WipeTable(wipe bool) {
	op.wipe = wipe
}

// NeedsPartx returns true: the sfdisk CLI only issues a best-effort
// whole-disk BLKRRPART after writing (whose failure is ignored and which
// the kernel refuses while any partition of the disk is mounted). It
// does not use the per-partition BLKPG ioctls, so ignition must
// continue to sync the kernel view with partx for changed partitions.
func (op *Operation) NeedsPartx() bool {
	return true
}

func (op *Operation) WritesCompleteTable() bool {
	return true
}

// diskHeader holds the on-disk GPT header values we must preserve across
// a whole-table rewrite: the backup LBA bound and the disk identifier.
// Omitting label-id from a script makes sfdisk randomize the disk GUID
// on every load, which is an observable (and boot-breaking for
// anything keying on /dev/disk/by-partuuid... no: the disk UUID) change.
type diskHeader struct {
	firstLBA int64
	lastLBA  int64
	labelID  string
}

// readDiskHeader runs sfdisk --dump and extracts the table-level headers.
// Returns an empty header (lastLBA -1, no labelID) if the device has no
// readable partition table; callers must handle that (blank disk).
func (op *Operation) readDiskHeader() diskHeader {
	h := diskHeader{firstLBA: -1, lastLBA: -1}
	cmd := exec.Command(distro.SfdiskCmd(), "--dump", op.dev)
	stdout, err := cmd.Output()
	if err != nil {
		return h
	}
	lbaRe := regexp.MustCompile(`(?m)^(first|last)-lba:\s*(\d+)`)
	idRe := regexp.MustCompile(`(?m)^label-id:\s*(\S+)`)
	for _, m := range lbaRe.FindAllSubmatch(stdout, -1) {
		v, err := strconv.ParseInt(string(m[2]), 10, 64)
		if err != nil {
			continue
		}
		if string(m[1]) == "first" {
			h.firstLBA = v
		} else {
			h.lastLBA = v
		}
	}
	if m := idRe.FindSubmatch(stdout); len(m) >= 2 {
		h.labelID = string(m[1])
	}
	return h
}

var dumpLineRegex = regexp.MustCompile(`^(/\S+)\s*:\s*(.*)$`)
var dumpNumRegex = regexp.MustCompile(`(\d+)$`)

// diskSectorsRegex matches sfdisk's geometry banner ("Disk <dev>: 32
// MiB, 33554432 bytes, 65536 sectors") so the usable-end can be
// derived on devices that had no readable table.
var diskSectorsRegex = regexp.MustCompile(`Disk \S+: .*, (\d+) sectors`)

// splitFields splits an sfdisk script line on commas, but not inside
// double-quoted values (names can contain commas, colons and
// backslash-escapes; sfdisk --dump quotes them).
func splitFields(s string) []string {
	var fields []string
	inQuote := false
	current := &bytes.Buffer{}
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			current.WriteRune(r)
		case r == ',' && !inQuote:
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	fields = append(fields, current.String())
	return fields
}

// readExistingPartitions reads the current partition table using
// sfdisk --dump, preserving all identity fields (including GPT
// attribute bits) so a whole-table rewrite is lossless. Names are
// decoded from sfdisk's quoted/\xHH script form to raw bytes here and
// re-encoded on write (encodeSfdiskName), so exotic labels round-trip
// byte-exactly regardless of where the entry came from.
func (op *Operation) readExistingPartitions() ([]partitioners.Partition, error) {
	cmd := exec.Command(distro.SfdiskCmd(), "--dump", op.dev)
	stdout, err := cmd.Output()
	if err != nil {
		// No partition table (or unreadable): treat as empty. The
		// stage guarantees a GPT exists before calling us on the
		// main path, so this only happens on blank disks.
		return nil, nil
	}

	var partitions []partitioners.Partition
	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(line)
		matches := dumpLineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		numMatch := dumpNumRegex.FindStringSubmatch(matches[1])
		if numMatch == nil {
			continue
		}
		partNum, err := strconv.Atoi(numMatch[1])
		if err != nil {
			continue
		}

		p := partitioners.Partition{}
		p.Number = partNum

		for _, field := range splitFields(matches[2]) {
			field = strings.TrimSpace(field)
			kv := strings.SplitN(field, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "start":
				v, err := strconv.ParseInt(value, 10, 64)
				if err == nil {
					p.StartSector = &v
				}
			case "size":
				v, err := strconv.ParseInt(value, 10, 64)
				if err == nil {
					p.SizeInSectors = &v
				}
			case "type":
				p.TypeGUID = util.StrToPtr(value)
			case "uuid":
				p.GUID = util.StrToPtr(value)
			case "name":
				raw, err := decodeSfdiskName(value)
				if err == nil {
					p.Label = util.StrToPtr(raw)
				}
			case "attrs":
				p.Attrs = splitAttrs(value)
			}
		}

		partitions = append(partitions, p)
	}

	return partitions, nil
}

// splitAttrs parses an sfdisk attrs value: "A,B" or A,B (quoted list of
// bit names as emitted by --dump).
func splitAttrs(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"")
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

// sfdisk script name encoding mirrors what `sfdisk --dump` emits and
// what the loader's next_string()+unhexmangle_string() accepts: the
// value is wrapped in double quotes, and every byte that would break
// the quoted token (double quote, backslash) or cannot appear in it
// unescaped (control bytes, DEL) is hex-escaped as \xHH. Names from
// the config and from blkid are raw; names preserved from --dump are
// decoded to raw by readExistingPartitions; encodeSfdiskName is the
// inverse and is applied unconditionally, so the script round-trips
// byte-exactly for exotic labels (quotes, backslashes, control bytes).
var hexEscapeRegex = regexp.MustCompile(`\\x([0-9a-fA-F]{2})`)

func encodeSfdiskName(raw string) string {
	out := &bytes.Buffer{}
	out.WriteByte('"')
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c == '"' || c == '\\' || c < 0x20 || c == 0x7f {
			fmt.Fprintf(out, "\\x%02x", c)
		} else {
			out.WriteByte(c)
		}
	}
	out.WriteByte('"')
	return out.String()
}

func decodeSfdiskName(encoded string) (string, error) {
	if len(encoded) < 2 || !strings.HasPrefix(encoded, "\"") || !strings.HasSuffix(encoded, "\"") {
		// dump always quotes names; anything else is malformed input
		return encoded, nil
	}
	inner := encoded[1 : len(encoded)-1]
	// unhexmangle leaves backslash sequences that are not \xHH verbatim
	decoded := hexEscapeRegex.ReplaceAllStringFunc(inner, func(m string) string {
		b, err := strconv.ParseUint(m[2:], 16, 8)
		if err != nil {
			return m
		}
		return string([]byte{byte(b)})
	})
	return decoded, nil
}

func writePartitionLine(script *bytes.Buffer, p partitioners.Partition) {
	var line bytes.Buffer

	if p.Number > 0 {
		fmt.Fprintf(&line, "%d :", p.Number)
	} else {
		line.WriteString(":")
	}

	if p.StartSector != nil && *p.StartSector != 0 {
		fmt.Fprintf(&line, " start=%d,", *p.StartSector)
	}

	if p.SizeInSectors != nil && *p.SizeInSectors != 0 {
		fmt.Fprintf(&line, " size=%d,", *p.SizeInSectors)
	} else {
		line.WriteString(" size=+,")
	}

	if util.NotEmpty(p.TypeGUID) {
		fmt.Fprintf(&line, " type=%s,", *p.TypeGUID)
	}

	if util.NotEmpty(p.GUID) {
		fmt.Fprintf(&line, " uuid=%s,", *p.GUID)
	}

	if p.Label != nil {
		// Labels are carried in raw form (config, blkid, and decoded
		// --dump input); encode to sfdisk's quoted/\xHH form.
		fmt.Fprintf(&line, " name=%s,", encodeSfdiskName(*p.Label))
	}

	if len(p.Attrs) > 0 {
		fmt.Fprintf(&line, " attrs=\"%s\",", strings.Join(p.Attrs, ","))
	}

	script.WriteString(strings.TrimSuffix(line.String(), ","))
	script.WriteString("\n")
}

// buildScript constructs an sfdisk script. Fixed-size partitions are
// written first (sorted by start sector) so that fill-remaining
// partitions (size=+) see the correct free blocks.
//
// grain: 512 disables sfdisk's default end-alignment of fill
// partitions so the tail of the table reaches last-lba - 1, matching
// what sgdisk produces (see the lastLBA correction in ParseOutput); it
// also lets fill start at the raw head of the largest free block.
// last-lba and label-id are pinned to the on-disk header values so a
// rewrite neither truncates the usable area (first-lba/last-lba are
// the #1745 regression class) nor randomizes the disk GUID.
// scriptTier orders script entries the way sfdisk consumes them:
// position-pinned entries (explicit start, e.g. preserved partitions
// re-listed for a whole-table rewrite) first, so auto-positioned
// entries resolve against the space they do NOT consume; then
// auto-positioned fixed-size entries; fills last (they claim the
// remaining tail). Emitting an auto-positioned entry before a pinned
// one can park it in a block a later pinned entry claims, and the
// pinned line then fails with ERANGE.
func scriptTier(p partitioners.Partition) int {
	if p.StartSector != nil && *p.StartSector > 0 {
		return 0
	}
	if p.SizeInSectors != nil && *p.SizeInSectors != 0 {
		return 1
	}
	return 2
}

func scriptStart(p partitioners.Partition) int64 {
	if p.StartSector != nil {
		return *p.StartSector
	}
	return 0
}

func buildScript(partitions []partitioners.Partition, header diskHeader) string {
	script := &bytes.Buffer{}
	script.WriteString("label: gpt\n")
	script.WriteString("grain: 512\n")
	if header.labelID != "" {
		fmt.Fprintf(script, "label-id: %s\n", header.labelID)
	}
	// Pin both usable-area bounds: defaulting first-lba (sfdisk uses
	// its alignment default, 2048) would rewrite the header of tables
	// created with a tighter first-lba (e.g. sgdisk's 34) and could
	// contradict preserved partitions that start below the default.
	if header.firstLBA >= 0 {
		fmt.Fprintf(script, "first-lba: %d\n", header.firstLBA)
	}
	if header.lastLBA >= 0 {
		fmt.Fprintf(script, "last-lba: %d\n", header.lastLBA)
	}
	script.WriteString("\n")

	sorted := make([]partitioners.Partition, len(partitions))
	copy(sorted, partitions)
	sort.SliceStable(sorted, func(i, j int) bool {
		ti, tj := scriptTier(sorted[i]), scriptTier(sorted[j])
		if ti != tj {
			return ti < tj
		}
		if ti == 0 {
			return scriptStart(sorted[i]) < scriptStart(sorted[j])
		}
		return false
	})

	for _, p := range sorted {
		writePartitionLine(script, p)
	}

	return script.String()
}

// mergePartitions builds the final partition list from existing disk
// state plus queued operations.
func (op *Operation) mergePartitions() ([]partitioners.Partition, error) {
	var merged []partitioners.Partition

	if !op.wipe {
		existing, err := op.readExistingPartitions()
		if err != nil {
			return nil, err
		}
		merged = existing
	}

	for _, delNum := range op.deletions {
		var filtered []partitioners.Partition
		for _, p := range merged {
			if p.Number != delNum {
				filtered = append(filtered, p)
			}
		}
		merged = filtered
	}

	for _, p := range op.parts {
		if p.Number == 0 {
			merged = append(merged, p)
		} else {
			replaced := false
			for i := range merged {
				if merged[i].Number == p.Number {
					// GPT attribute bits are invisible to the
					// config (and to blkid-based matching), so a
					// queued partition never carries them; inherit
					// from the preserved entry so whole-table
					// rewrites don't silently clear them.
					if p.Attrs == nil {
						p.Attrs = merged[i].Attrs
					}
					merged[i] = p
					replaced = true
					break
				}
			}
			if !replaced {
				merged = append(merged, p)
			}
		}
	}

	return merged, nil
}

func (op *Operation) Pretend() (string, error) {
	merged, err := op.mergePartitions()
	if err != nil {
		return "", err
	}

	header := op.readDiskHeader()
	op.lastLBA = header.lastLBA
	scriptContent := buildScript(merged, header)

	if op.logger != nil {
		op.logger.Info("running sfdisk --no-act with script:\n%s", scriptContent)
	}

	// Do NOT pass --force here: --force makes sfdisk discard our
	// pinned last-lba header (see sfdisk.c "Ignoring last-lba script
	// header") and resolve fill/auto-positioned partitions against
	// the disk's default usable end instead. Commit runs without
	// --force and honors the pinned last-lba, so a geometry that
	// only Pretend agreed on overflows the real table at commit with
	// "Failed to add #N partition: Numerical result out of range".
	// --no-act already prevents any write, and -X gpt forces the
	// label type, so no-force is sufficient and side-effect free on
	// blank devices.
	cmd := exec.Command(distro.SfdiskCmd(), "--no-act", "-X", "gpt", op.dev)
	cmd.Stdin = strings.NewReader(scriptContent)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	output, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	errors, err := io.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to pretend to create partitions. Err: %v. Stderr: %v", err, string(errors))
	}

	// On a device that had no readable table, readDiskHeader could
	// not learn last-lba; the fill-partition size correction still
	// needs it. sfdisk prints the disk geometry ("..., N sectors")
	// and its GPT default reserves the last 33 sectors for the
	// backup table, making the usable end N-34. Derive it so a
	// size=+ fill resolves to the same final sector sgdisk would
	// have used (verified against real sfdisk output).
	if op.lastLBA < 0 {
		if m := diskSectorsRegex.FindStringSubmatch(string(output)); len(m) >= 2 {
			if n, err := strconv.ParseInt(m[1], 10, 64); err == nil && n > 34 {
				op.lastLBA = n - 34
			}
		}
	}

	return string(output), nil
}

// Commit writes the merged table. wipeTable is handled by the stage as a
// separate op carrying only WipeTable(true).
func (op *Operation) Commit() error {
	if op.wipe {
		if op.logger != nil {
			op.logger.Info("wiping partition table on %q", op.dev)
		}
		if err := op.zapTable(); err != nil {
			return fmt.Errorf("failed to wipe partition table on %q: %v", op.dev, err)
		}
		if len(op.parts) == 0 && len(op.deletions) == 0 {
			return nil
		}
	}

	merged, err := op.mergePartitions()
	if err != nil {
		return err
	}

	if len(merged) == 0 {
		// Nothing queued at all: no write. (Deletions alone still
		// have to hit the disk: the sgdisk backend committed each
		// --delete=N, leaving an empty table; skipping here would
		// leave the old table on disk after a delete-all config,
		// which blackbox's "extra partitions" validator catches.)
		if len(op.deletions) == 0 {
			return nil
		}
		header := op.readDiskHeader()
		scriptContent := buildScript(nil, header)
		if op.logger != nil {
			op.logger.Info("writing empty partition table to %q:\n%s", op.dev, scriptContent)
		}
		// --wipe never: deleting entries must not touch filesystem
		// data (or signatures) in the former partition areas; that
		// is the filesystems stage's job, as with sgdisk.
		cmd := exec.Command(distro.SfdiskCmd(), "--no-reread", "--wipe", "never", "--wipe-partitions", "never", "-X", "gpt", op.dev)
		cmd.Stdin = strings.NewReader(scriptContent)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to remove all partitions on %q: %v: %s", op.dev, err, string(output))
		}
		return nil
	}

	if err := op.handleInfo(); err != nil {
		return err
	}

	header := op.readDiskHeader()
	scriptContent := buildScript(merged, header)

	if op.logger != nil {
		op.logger.Info("running sfdisk with script:\n%s", scriptContent)
	}

	// --no-reread: skip sfdisk's whole-disk O_EXCL in-use check;
	// ignition performs its own per-partition usage checks and must
	// be allowed to modify a table while another partition of the
	// same disk is mounted (the case partx exists for).
	// --wipe auto / --wipe-partitions auto: never silently remove
	// filesystem signatures from preserved (re-listed) partitions;
	// non-interactive "auto" already declines wipes (same posture as
	// sgdisk), "always" would destroy the data of untouched ones.
	cmd := exec.Command(distro.SfdiskCmd(), "--no-reread", "--wipe", "auto", "--wipe-partitions", "auto", "-X", "gpt", op.dev)
	cmd.Stdin = strings.NewReader(scriptContent)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sfdisk failed on %q: %v: %s", op.dev, err, string(output))
	}

	return nil
}

// ParseOutput reads the "New situation" table printed by --no-act.
// partitionNumbers lists the partitions the stage queued for inspection
// (those with unspecified or zero geometry); only for those can
// end==last-lba-1 mean "fill" rather than an explicit size, so the
// last-lba correction is restricted to them.
func (op *Operation) ParseOutput(sfdiskOutput string, partitionNumbers []int) (map[int]partitioners.Output, error) {
	preamble := sfdiskOutput
	if idx := strings.Index(sfdiskOutput, "Device"); idx >= 0 {
		preamble = sfdiskOutput[:idx]
	}
	lower := strings.ToLower(preamble)
	if strings.Contains(lower, "failed") || strings.Contains(lower, "error") {
		return nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadSfdiskPretend, sfdiskOutput)
	}

	inspect := map[int]bool{}
	for _, n := range partitionNumbers {
		inspect[n] = true
	}

	result := make(map[int]partitioners.Output)

	partitionRegex := regexp.MustCompile(`^/\S+\s+\*?\s*(\d+)\s+(\d+)\s+(\d+)\s+`)
	numRegex := regexp.MustCompile(`^(/\S+)`)
	devNumRegex := regexp.MustCompile(`(\d+)$`)

	for _, line := range strings.Split(sfdiskOutput, "\n") {
		line = strings.TrimSpace(line)
		matches := partitionRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		devMatch := numRegex.FindStringSubmatch(line)
		if devMatch == nil {
			continue
		}
		partNumMatch := devNumRegex.FindStringSubmatch(devMatch[1])
		if partNumMatch == nil {
			continue
		}
		partNum, err := strconv.Atoi(partNumMatch[1])
		if err != nil {
			continue
		}

		start, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			continue
		}

		end, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			continue
		}

		size := end - start + 1

		// With grain: 512, a fill (size=+) partition ends at
		// lastLBA-1. sgdisk fills to lastLBA exactly, so report one
		// more sector to match. Restricted to inspected partitions:
		// an explicit size that happens to end at lastLBA-1 must NOT
		// be enlarged (that would flip-flop sizes across boots).
		if op.lastLBA > 0 && end == op.lastLBA-1 && inspect[partNum] {
			size++
		}

		result[partNum] = partitioners.Output{
			Start: start,
			Size:  size,
		}
	}

	return result, nil
}

func (op *Operation) handleInfo() error {
	if len(op.infos) == 0 {
		return nil
	}

	cmd := exec.Command(distro.SfdiskCmd(), "--list", op.dev)
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("sfdisk --list failed on %q: %v", op.dev, err)
	}

	if op.logger != nil {
		op.logger.Info("partition info for %q (requested partitions %v):\n%s", op.dev, op.infos, string(stdout))
	}

	return nil
}
