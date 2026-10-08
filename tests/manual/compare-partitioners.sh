#!/bin/bash
# Compare sgdisk and sfdisk partition output for identical Ignition configs.
# Must run as root on a system with loop device support.
# Usage: sudo ./tests/manual/compare-partitioners.sh [path-to-ignition-binary]
set -euo pipefail

IGNITION="${1:-./bin/amd64/ignition}"
IMG_SIZE=200M
PASS=0
FAIL=0

if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: must run as root" >&2
    exit 1
fi

if ! command -v sgdisk &>/dev/null || ! command -v sfdisk &>/dev/null; then
    echo "ERROR: both sgdisk and sfdisk must be installed" >&2
    exit 1
fi

tmpdir=$(mktemp -d /var/tmp/partitioner-compare.XXXXXX)
trap 'rm -rf "$tmpdir"' EXIT

dump_partition_table() {
    local dev="$1"
    # Normalize output: partition number, start, size, type GUID, partition GUID, label
    sfdisk --dump "$dev" 2>/dev/null | grep "^/dev/" | while read -r line; do
        local start size type uuid name
        start=$(echo "$line" | sed -n 's/.*start=\s*\([0-9]*\).*/\1/p')
        size=$(echo "$line" | sed -n 's/.*size=\s*\([0-9]*\).*/\1/p')
        type=$(echo "$line" | sed -n 's/.*type=\s*\([^ ,]*\).*/\1/p')
        uuid=$(echo "$line" | sed -n 's/.*uuid=\s*\([^ ,]*\).*/\1/p')
        name=$(echo "$line" | sed -n 's/.*name="\([^"]*\)".*/\1/p')
        echo "start=$start size=$size type=$type name=$name"
    done
}

run_test() {
    local test_name="$1"
    local config="$2"
    local setup_script="${3:-}"

    echo "=== TEST: $test_name ==="

    # --- sgdisk run ---
    local img_sg="$tmpdir/sgdisk.img"
    truncate -s "$IMG_SIZE" "$img_sg"
    local loop_sg
    loop_sg=$(losetup --find --show "$img_sg")
    trap "losetup -d $loop_sg 2>/dev/null; rm -f $img_sg" RETURN

    if [ -n "$setup_script" ]; then
        eval "$setup_script" "$loop_sg"
    fi

    local cfg_sg
    cfg_sg=$(echo "$config" | sed "s|\\\$disk|$loop_sg|g")
    local cwd_sg="$tmpdir/cwd-sg"
    mkdir -p "$cwd_sg"
    echo "$cfg_sg" > "$cwd_sg/ignition.json"
    touch "$cwd_sg/neednet" "$cwd_sg/state"

    IGNITION_PARTITIONER=sgdisk "$IGNITION" \
        -platform file -stage disks -root "$tmpdir/root-sg" \
        -log-to-stdout -config-cache "$cwd_sg/ignition.json" \
        -neednet "$cwd_sg/neednet" -state-file "$cwd_sg/state" \
        2>&1 | tail -5 || true

    local sg_result
    sg_result=$(dump_partition_table "$loop_sg")
    losetup -d "$loop_sg"

    # --- sfdisk run ---
    local img_sf="$tmpdir/sfdisk.img"
    truncate -s "$IMG_SIZE" "$img_sf"
    local loop_sf
    loop_sf=$(losetup --find --show "$img_sf")

    if [ -n "$setup_script" ]; then
        eval "$setup_script" "$loop_sf"
    fi

    local cfg_sf
    cfg_sf=$(echo "$config" | sed "s|\\\$disk|$loop_sf|g")
    local cwd_sf="$tmpdir/cwd-sf"
    mkdir -p "$cwd_sf"
    echo "$cfg_sf" > "$cwd_sf/ignition.json"
    touch "$cwd_sf/neednet" "$cwd_sf/state"

    IGNITION_PARTITIONER=sfdisk "$IGNITION" \
        -platform file -stage disks -root "$tmpdir/root-sf" \
        -log-to-stdout -config-cache "$cwd_sf/ignition.json" \
        -neednet "$cwd_sf/neednet" -state-file "$cwd_sf/state" \
        2>&1 | tail -5 || true

    local sf_result
    sf_result=$(dump_partition_table "$loop_sf")
    losetup -d "$loop_sf"

    # --- Compare ---
    if [ "$sg_result" = "$sf_result" ]; then
        echo "  PASS: partition tables match"
        echo "  Result: $sg_result"
        ((PASS++))
    else
        echo "  FAIL: partition tables differ"
        echo "  sgdisk: $sg_result"
        echo "  sfdisk: $sf_result"
        ((FAIL++))
    fi
    echo

    # Cleanup for this test
    rm -f "$img_sg" "$img_sf"
    rm -rf "$cwd_sg" "$cwd_sf" "$tmpdir/root-sg" "$tmpdir/root-sf"
}

# --- Test 1: Simple partition creation ---
run_test "create single partition" '{
    "ignition": {"version": "3.4.0"},
    "storage": {"disks": [{"device": "$disk", "partitions": [
        {"number": 1, "sizeMiB": 32, "label": "testpart",
         "typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"}
    ]}]}
}'

# --- Test 2: Multiple partitions ---
run_test "create two partitions" '{
    "ignition": {"version": "3.4.0"},
    "storage": {"disks": [{"device": "$disk", "partitions": [
        {"number": 1, "sizeMiB": 32, "label": "part1",
         "typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"},
        {"number": 2, "sizeMiB": 32, "label": "part2",
         "typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"}
    ]}]}
}'

# --- Test 3: Wipe table and create ---
run_test "wipe table then create" '{
    "ignition": {"version": "3.4.0"},
    "storage": {"disks": [{"device": "$disk", "wipeTable": true, "partitions": [
        {"number": 1, "sizeMiB": 32, "label": "fresh",
         "typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"}
    ]}]}
}' 'sgdisk --new=1:2048:+65536 --change-name=1:old'

# --- Test 4: Fill remaining space (sizeMiB 0) ---
run_test "fill remaining space" '{
    "ignition": {"version": "3.4.0"},
    "storage": {"disks": [{"device": "$disk", "partitions": [
        {"number": 1, "sizeMiB": 32, "label": "fixed",
         "typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"},
        {"number": 2, "sizeMiB": 0, "startMiB": 0, "label": "rest",
         "typeGuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"}
    ]}]}
}'

echo "=== SUMMARY: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ]
