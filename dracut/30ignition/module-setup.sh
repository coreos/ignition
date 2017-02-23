#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd
}

install() {
    inst_multiple \
        ignition \
        coreos-metadata \
        useradd \
        usermod \
        groupadd \
        systemd-detect-virt \
        mkfs.btrfs \
        mkfs.ext4 \
        mkfs.xfs

    inst_simple "$moddir/ignition-generator" \
        "$systemdutildir/system-generators/ignition-generator"

    inst_simple "$moddir/ignition-disks.service" \
        "$systemdsystemunitdir/ignition-disks.service"

    inst_simple "$moddir/ignition-files.service" \
        "$systemdsystemunitdir/ignition-files.service"

    inst_simple "$moddir/ignition.target" \
        "$systemdsystemunitdir/ignition.target"

    inst_simple "$moddir/coreos-digitalocean-network.service" \
        "$systemdsystemunitdir/coreos-digitalocean-network.service"

    systemctl --root "$initdir" enable coreos-digitalocean-network.service

    inst_rules \
        60-cdrom_id.rules
}

installkernel() {
    instmods qemu_fw_cfg
}
