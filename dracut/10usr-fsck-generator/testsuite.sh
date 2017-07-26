#!/bin/bash

moddir=$(dirname "$0")
failed=0

run_test() {
    local cmdline="$1"
    local checkfunc="$2"
    local unitdir=$(mktemp -d)
    trap "rm -rf '${unitdir}'" EXIT

    echo "Starting test ${checkfunc}"
    export USR_FSCK_GENERATOR_CMDLINE="${cmdline}"
    if ! "${moddir}/usr-fsck-generator" "${unitdir}"; then
        echo "FAILED ${checkfunc}"
        failed=$((failed + 1))
    elif ! "${checkfunc}" "${unitdir}"; then
        echo "FAILED ${checkfunc}"
        failed=$((failed + 1))
    else
        echo "PASSED ${checkfunc}"
    fi

    rm -rf "${unitdir}"
    trap - EXIT
}

test_noop() {
    if [[ -e "$1"/* ]]; then
        echo "$1 not empty"
        return 1
    fi
}
run_test "nothing" test_noop
run_test "usr=nothing" test_noop
run_test "mount.usr=nothing" test_noop
run_test "usr=nothing mount.usr=nothing" test_noop


test_simple() {
    [[ -e $1/systemd-fsck@foo-bar-baz.service.d/disable.conf ]]
    return $?
}
run_test "usr=/foo/bar/baz" test_simple
run_test "mount.usr=/foo/bar/baz" test_simple
run_test "usr=bar mount.usr=/foo/bar/baz" test_simple
run_test "usr=foo mount.usr=bar usr=baz mount.usr=/foo/bar/baz usr=bleh" test_simple


test_label() {
    [[ -e $1/systemd-fsck@dev-disk-by\\x2dlabel-foo.service.d/disable.conf ]]
    return $?
}
run_test "mount.usr=LABEL=foo" test_label


test_uuid() {
    [[ -e $1/systemd-fsck@dev-disk-by\\x2duuid-a123.service.d/disable.conf ]]
    return $?
}
run_test "mount.usr=UUID=A123" test_uuid


test_partuuid() {
    [[ -e $1/systemd-fsck@dev-disk-by\\x2dpartuuid-a123.service.d/disable.conf ]]
    return $?
}
run_test "mount.usr=PARTUUID=A123" test_partuuid


test_partlabel() {
    [[ -e $1/systemd-fsck@dev-disk-by\\x2dpartlabel-foo.service.d/disable.conf ]]
    return $?
}
run_test "mount.usr=PARTLABEL=foo" test_partlabel


if [[ "${failed}" -ne 0 ]]; then
    echo "${failed} test(s) failed!"
    exit 1
fi
