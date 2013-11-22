#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# Flexible mount directory for testing
[ -z ${BOOTENGINE_ROOT_DIR} ] && BOOTENGINE_ROOT_DIR=/tmp/boot
BOOTENGINE_KERNEL_PATH=${BOOTENGINE_ROOT_DIR}/boot/vmlinuz

# The current filesystem we are trying to kexec to, set in try_next()
BOOTENGINE_ROOT=
BOOTENGINE_ROOT_CMDLINE=

# Record the highest priority filesystem that appears usable.
# Will be used directly if kexec fails.
BOOTENGINE_ROOT_FALLBACK=

# A regex of modules to unload before running kexec
BOOTENGINE_MOD_BLACKLIST="virtio"

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
    unload=
    if [ -n "$BOOTENGINE_MOD_BLACKLIST" ]; then
        unload=$(awk \
            "\$1 ~ /$BOOTENGINE_MOD_BLACKLIST/ {print \$1}" \
            </proc/modules)
    fi
    if [ -n "$unload" ]; then
        bootengine_cmd modprobe -r $unload || \
            warn "bootengine: failed to remove blacklisted modules"
    fi

    info "bootengine: attempting to exec new kernel!"
    bootengine_cmd kexec --exec || warn "bootengine: :'("

    if [ -n "$unload" ]; then
        bootengine_cmd modprobe $unload || \
            warn "bootengine: failed to re-insert blacklisted modules"
    fi
}

# Note: This function always returns 0, exiting at all is the failure.
try_next() {
    root_partuuid=$(cgpt next)
    info "bootengine: selected PARTUUID $root_partuuid"

    BOOTENGINE_ROOT="/dev/disk/by-partuuid/${root_partuuid}"
    BOOTENGINE_ROOT_CMDLINE="PARTUUID=${root_partuuid}"

    mount_root || return 0

    if ! usable_root ${BOOTENGINE_ROOT_DIR}; then
        warn "bootengine: filesystem appears to be invalid."
        return 0
    fi

    # This filesystem can be used directly if kexec fails.
    if [ -z "$BOOTENGINE_ROOT_FALLBACK" ]; then
        BOOTENGINE_ROOT_FALLBACK=${BOOTENGINE_ROOT}
    fi

    load_kernel || return 0
    bootengine_cmd umount ${BOOTENGINE_ROOT_DIR} || return 0
    kexec_kernel || return 0
}

do_exec_or_find_root() {
    # Try booting the highest priority root filesystem
    try_next

    # Failed, clean up and try again...
    if ismounted ${BOOTENGINE_ROOT_DIR}; then
        bootengine_cmd umount ${BOOTENGINE_ROOT_DIR}
    fi
    try_next

    # Nope, still here. Hopefully there is a way out.
    if ismounted ${BOOTENGINE_ROOT_DIR}; then
        bootengine_cmd umount ${BOOTENGINE_ROOT_DIR}
    fi
    if [ -n "$BOOTENGINE_ROOT_FALLBACK" ]; then
        root="block:${BOOTENGINE_ROOT_FALLBACK}"
        warn "bootengine: giving up on kexec, proceeding with root=${root}"
    else
        # Well this is embarrassing...
        die "bootengine: failed to find a usable root filesystem!"
    fi
}

if [ -n "$root" -a -z "${root%%gptprio:}" ]; then
    do_exec_or_find_root
fi
