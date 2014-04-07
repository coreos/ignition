#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd
    echo usr-generator
}

install() {
    dracut_install mkfs.btrfs truncate
    inst_simple "$moddir/diskless-btrfs" "$systemdutildir/diskless-btrfs"
    inst_simple "$moddir/diskless-generator" \
        "$systemdutildir/system-generators/diskless-generator"
}
