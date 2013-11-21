#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# Flexible mount directory for testing
[ -z ${BOOTENGINE_ROOT_DIR} ] && BOOTENGINE_ROOT_DIR=/tmp/boot
BOOTENGINE_KERNEL_PATH=${BOOTENGINE_ROOT_DIR}/boot/vmlinuz

# mount the BOOTENGINE_ROOT or return non-zero
mount_root() {
    info "bootengine: preparing disk mount for $BOOTENGINE_ROOT"
    mkdir ${BOOTENGINE_ROOT_DIR}
    mount -o ro ${BOOTENGINE_ROOT} ${BOOTENGINE_ROOT_DIR} > /tmp/bootengine.out 2>&1
    ret=$?
    info "bootengine: mount on ${BOOTENGINE_ROOT} returned $ret"
    return $ret
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

find_kernel() {
    if [ ! -s $BOOTENGINE_KERNEL_PATH ]; then
      warn "bootengine: No kernel at $BOOTENGINE_KERNEL_PATH"
      return 1
    fi
    return 0
}

kexec_kernel() {
    # Attempt to load up the kernel with kexec
    cmd_line=$(cat /proc/cmdline)
    kexec --command-line="${cmd_line} root=${BOOTENGINE_ROOT_CMDLINE}" \
      -l $BOOTENGINE_KERNEL_PATH > /tmp/bootengine.out 2>&1
    info "bootengine: kexec -l returned $?"
    vinfo < /tmp/bootengine.out

    kexec -e > /tmp/bootengine.out 2>&1
    info "bootengine: kexec -e returned $?"
    vinfo < /tmp/bootengine.out
    # If we reach here then kexec didn't work. We are the only
    info "ERROR: bootengine: kexec -e shouldn't return!"
    info "cmd_line was $cmd_line"
    info "root_upper was $root_upper"
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
    find_kernel && kexec_kernel || true

    # If there wasn't a kernel found or the kexec fails this kernel will have
    # to act as the runtime kernel. This is the common case on Xen for now.
    root=block:${BOOTENGINE_ROOT}
    info "bootengine: No kernel found or kexec failed, proceeding with root=$root"
    umount ${BOOTENGINE_ROOT_DIR}
}

if [ -n "$root" -a -z "${root%%gptprio:}" ]; then
    do_exec_or_find_root
fi
