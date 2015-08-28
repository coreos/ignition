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
        "$systemdsystemunitdir/mnt-oem.mount" \
        "$systemdsystemunitdir/ignition.target" \
        "$systemdsystemunitdir/ignition-disks.service" \
        "$systemdsystemunitdir/ignition-files.service"

    inst_rules "90-ignition.rules"

    inst_simple "$moddir/ignition-generator" \
        "$systemdutildir/system-generators/ignition-generator"
}
