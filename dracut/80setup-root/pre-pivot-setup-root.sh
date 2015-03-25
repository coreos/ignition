#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# /etc/machine-id after a new image is created:
COREOS_BLANK_MACHINE_ID="42000000000000000000000000000042"
MACHINE_ID_FILE="/sysroot/etc/machine-id"

# Run and log a command
bootengine_cmd() {
    ret=0
    "$@" >/tmp/bootengine.out 2>&1 || ret=$?
    vinfo < /tmp/bootengine.out
    if [ $ret -ne 0 ]; then
        warn "bootengine: command failed: $*"
        warn "bootengine: command returned $ret"
    fi
    return $ret
}

do_setup_root() {
    # Initialize base filesystem
    bootengine_cmd systemd-tmpfiles --root=/sysroot --create \
        baselayout.conf baselayout-etc.conf baselayout-usr.conf

    # Not all images provide this file so check before using it.
    if [ -e "/sysroot/usr/lib/tmpfiles.d/baselayout-ldso.conf" ]; then
        bootengine_cmd systemd-tmpfiles --root=/sysroot --create \
            baselayout-ldso.conf
    fi

    # Remove our phony id. systemd will initialize this during boot.
    if grep -qs "${COREOS_BLANK_MACHINE_ID}" "${MACHINE_ID_FILE}"; then
        bootengine_cmd rm "${MACHINE_ID_FILE}"
    fi
}

# Skip if root and root/usr are not mount points
if ismounted /sysroot && ismounted /sysroot/usr; then
    do_setup_root
fi

# PXE initrds may provide OEM
if [ -d /usr/share/oem ] && ismounted /sysroot/usr/share/oem; then
    cp -Ra /usr/share/oem/. /sysroot/usr/share/oem
fi
