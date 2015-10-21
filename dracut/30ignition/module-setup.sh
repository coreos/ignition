#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd
}

install() {
    inst_multiple \
        ignition \
        useradd \
        usermod \
        groupadd \
        mkfs.btrfs \
        mkfs.ext4 \
        mkfs.xfs \
        "$systemdsystemunitdir/ignition.target" \
        "$systemdsystemunitdir/ignition-disks.service" \
        "$systemdsystemunitdir/ignition-files.service"

    inst_simple "$moddir/ignition-generator" \
        "$systemdutildir/system-generators/ignition-generator"
}
