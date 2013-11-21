#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# Flexible mount directory for testing
[ -z ${BOOTENGINE_ROOT_DIR} ] && BOOTENGINE_ROOT_DIR=/tmp/boot
BOOTENGINE_KERNEL_PATH=${BOOTENGINE_ROOT_DIR}/boot/vmlinuz

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

# mount the BOOTENGINE_ROOT or return non-zero
mount_root() {
    info "bootengine: mounting ${BOOTENGINE_ROOT} to ${BOOTENGINE_ROOT_DIR}"
    bootengine_cmd mkdir -p ${BOOTENGINE_ROOT_DIR} || return $?
    bootengine_cmd mount -o ro ${BOOTENGINE_ROOT} ${BOOTENGINE_ROOT_DIR} || return $?
}

# If we call this we are in a bad position. The root filesystem that
# we expected to work out failed to mount. Try our best to get out of
# this problem by looping over all coreos-rootfs filesystems until we
# find one that mounts
root_emergency() {
    warn "bootengine: Mounting next root failed! Trying to recover."
    cgpt_next
    mount_root
}

load_kernel() {
    if [ ! -s $BOOTENGINE_KERNEL_PATH ]; then
      warn "bootengine: No kernel at $BOOTENGINE_KERNEL_PATH"
      return 1
    fi

    info "bootengine: loading kernel from ${BOOTENGINE_KERNEL_PATH}..."
    bootengine_cmd kexec --reuse-cmdline \
        --append="root=${BOOTENGINE_ROOT_CMDLINE}" \
        --load $BOOTENGINE_KERNEL_PATH || return $?
}

kexec_kernel() {
    info "bootengine: attempting to exec new kernel!"
    bootengine_cmd kexec --exec || return $?
}

cgpt_next() {
    root_partuuid=$(cgpt next)
    info "bootengine: selected PARTUUID $root_partuuid"

    BOOTENGINE_ROOT="/dev/disk/by-partuuid/${root_partuuid}"
    BOOTENGINE_ROOT_CMDLINE="PARTUUID=${root_partuuid}"
}

do_exec_or_find_root() {
    cgpt_next
    mount_root || root_emergency

    # Find a kernel and kexec it. Fall through on failure of either.
    load_kernel && kexec_kernel || true

    # If there wasn't a kernel found or the kexec fails this kernel will have
    # to act as the runtime kernel. This is the common case on Xen for now.
    root=block:${BOOTENGINE_ROOT}
    info "bootengine: No kernel found or kexec failed, proceeding with root=$root"
    umount ${BOOTENGINE_ROOT_DIR}
}

if [ -n "$root" -a -z "${root%%gptprio:}" ]; then
    do_exec_or_find_root
fi
