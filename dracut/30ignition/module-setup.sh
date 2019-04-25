#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo qemu systemd url-lib network
}

install_ignition_unit() {
    unit=$1; shift
    inst_simple "$moddir/$unit" "$systemdsystemunitdir/$unit"
    ln_r "../$unit" "$systemdsystemunitdir/ignition-complete.target.requires/$unit"
}

install() {
    inst_multiple \
        chroot \
        groupadd \
        id \
        mkfs.ext4 \
        mkfs.vfat \
        mkfs.xfs \
        mkswap \
        mountpoint \
        sgdisk \
        systemd-detect-virt \
        useradd \
        usermod \
        realpath \
        touch

    # This one is optional; https://src.fedoraproject.org/rpms/ignition/pull-request/9
    inst_multiple -o mkfs.btrfs

    inst_script "$moddir/ignition-setup.sh" \
        "/usr/sbin/ignition-setup"

    # Distro packaging is expected to install the ignition binary into the
    # module directory.
    inst_simple "$moddir/ignition" \
        "/usr/bin/ignition"

    inst_simple "$moddir/ignition-generator" \
        "$systemdutildir/system-generators/ignition-generator"

    inst_simple "$moddir/ignition-complete.target" \
        "$systemdsystemunitdir/ignition-complete.target"

    mkdir -p "$initdir/$systemdsystemunitdir/ignition-complete.target.requires"

    install_ignition_unit ignition-setup.service
    install_ignition_unit ignition-disks.service
    install_ignition_unit ignition-mount.service
    install_ignition_unit ignition-files.service
    install_ignition_unit ignition-remount-sysroot.service

    # needed for openstack config drive support
    inst_rules 60-cdrom_id.rules
}

has_fw_cfg_module() {
    # this is like check_kernel_config() but it specifically checks for `m` and
    # also checks the OSTree-specific kernel location
    for path in /boot/config-$kernel \
                /usr/lib/modules/$kernel/config \
                /usr/lib/ostree-boot/config-$kernel; do
        if test -f $path; then
            rc=0
            grep -q CONFIG_FW_CFG_SYSFS=m $path || rc=$?
            return $rc
        fi
    done
    return 1
}

installkernel() {
    # We definitely need this one in the initrd to support Ignition cfgs on qemu
    # if available
    if has_fw_cfg_module; then
        instmods -c qemu_fw_cfg
    fi
}
