#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo qemu systemd url-lib network
}

install() {
    inst_multiple \
        chroot \
        groupadd \
        id \
        ignition \
        mkfs.ext4 \
        mkfs.vfat \
        mkfs.xfs \
        mkswap \
        mountpoint \
        sgdisk \
        systemd-detect-virt \
        useradd \
        usermod \
        touch

    # This one is optional; https://src.fedoraproject.org/rpms/ignition/pull-request/9
    inst_multiple -o mkfs.btrfs

#   inst_script "$moddir/ignition-setup.sh" \
#       "/usr/sbin/ignition-setup"

#   inst_script "$moddir/retry-umount.sh" \
#       "/usr/sbin/retry-umount"

    inst_simple "$moddir/ignition-generator" \
        "$systemdutildir/system-generators/ignition-generator"

    inst_simple "$moddir/ignition-disks.service" \
        "$systemdsystemunitdir/ignition-disks.service"

    inst_simple "$moddir/ignition-files.service" \
        "$systemdsystemunitdir/ignition-files.service"

    inst_simple "$moddir/ignition-ask-var-mount.service" \
        "$systemdsystemunitdir/ignition-ask-var-mount.service"

    inst_simple "$moddir/ignition-remount-sysroot.service" \
        "$systemdutildir/system/ignition-remount-sysroot.service"

#   inst_simple "$moddir/sysroot-boot.service" \
#       "$systemdsystemunitdir/sysroot-boot.service"

#   inst_simple "$moddir/coreos-digitalocean-network.service" \
#       "$systemdsystemunitdir/coreos-digitalocean-network.service"

#   inst_simple "$moddir/coreos-static-network.service" \
#       "$systemdsystemunitdir/coreos-static-network.service"
}

has_builtin_fw_cfg() {
    # this is like check_kernel_config() but it specifically checks for `y` and
    # also checks the OSTree-specific kernel location
    for path in /boot/config-$kernel \
                /usr/lib/modules/$kernel/config \
                /usr/lib/ostree-boot/config-$kernel; do
        if test -f $path; then
            rc=0
            grep -q CONFIG_FW_CFG_SYSFS=y $path || rc=$?
            return $rc
        fi
    done
    return 1
}

installkernel() {
    # We definitely need this one in the initrd to support Ignition cfgs on qemu
    if ! has_builtin_fw_cfg; then
        instmods -c qemu_fw_cfg
    fi
}
