#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# Flexible mount directory for testing
[ -z ${BOOTENGINE_ROOT_DIR} ] && BOOTENGINE_ROOT_DIR=/sysroot
BOOTENGINE_USR_DIR=${BOOTENGINE_ROOT_DIR}/usr
BOOTENGINE_KERNEL_PATH=${BOOTENGINE_USR_DIR}/boot/vmlinuz
# TEMPORARY DELETE ME
BOOTENGINE_OLD_KERNEL_PATH=${BOOTENGINE_ROOT_DIR}/boot/vmlinuz
# TEMPORARY END

# The current filesystem we are trying to kexec to, set in try_next()
BOOTENGINE_USR=
BOOTENGINE_USR_CMDLINE=

# Record the highest priority filesystem that appears usable.
# Will be used directly if kexec fails.
BOOTENGINE_USR_FALLBACK=

# A regex of modules to unload before running kexec
BOOTENGINE_MOD_BLACKLIST="virtio"

# Directories in the root directory that need to be mapped
# to directories in /usr
BOOTENGINE_ROOT_DIRS="bin
sbin
lib
lib64"

# Similarly, /usr must contain the above directories
usable_usr() {
    for usrdir in $BOOTENGINE_ROOT_DIRS; do
        [ -d "$1/$usrdir" ] || return 1
    done
    return 0
}

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

# mount the usr directory of fail
mount_usr() {
    info "bootengine: mounting ${BOOTENGINE_USR} to ${BOOTENGINE_USR_DIR}"

    bootengine_cmd mount -o ro ${BOOTENGINE_USR} ${BOOTENGINE_USR_DIR} || return $?
}

load_kernel() {
    local kernel_path="$BOOTENGINE_KERNEL_PATH"
    if [ ! -s "$kernel_path" ]; then
        warn "bootengine: No kernel at $kernel_path"
        # TEMPORARY DELETE ME
        kernel_path="$BOOTENGINE_OLD_KERNEL_PATH"
        if [ -s "$kernel_path" ]; then
            warn "bootengine: using kernel from root partition instead"
        else
            return 1
        fi
        # TEMPORARY END
        #return 1
    fi

    info "bootengine: loading kernel from ${kernel_path}..."
    bootengine_cmd kexec --reuse-cmdline \
        --append="usr=${BOOTENGINE_USR_CMDLINE}" \
        --load "$kernel_path" || return $?
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

    BOOTENGINE_USR="/dev/disk/by-partuuid/${root_partuuid}"
    BOOTENGINE_USR_CMDLINE="PARTUUID=${root_partuuid}"

    mount_usr || return 0

    if ! usable_usr ${BOOTENGINE_USR_DIR}; then
        warn "bootengine: filesystem appears to be invalid."
        return 0
    fi

    # This filesystem can be used directly if kexec fails.
    if [ -z "$BOOTENGINE_USR_FALLBACK" ]; then
        BOOTENGINE_USR_FALLBACK=${BOOTENGINE_USR}
    fi

    load_kernel || return 0
    bootengine_cmd umount ${BOOTENGINE_USR_DIR} || return 0
    kexec_kernel || return 0
}

setup_usr_link() {
    path="${BOOTENGINE_ROOT_DIR}/$1"
    target="usr/$1"

    if [ -h "$path" -a "$(readlink "$path")" = "$target" ]; then
        return 0
    fi

    # Find an unused name and backup anything currently existing at $path
    if [ -e "$path" -o -h "$path" ]; then
        i=0
        backup="${path}.bak"
        while [ -e "$backup" ]; do
            i=$((i + 1))
            backup="${path}.bak${i}"
        done

        bootengine_cmd mv "$path" "$backup"
    fi

    bootengine_cmd ln -s "$target" "$path"
}

setup_root_symlinks() {
    # TODO: there is no reason to risk the remount if everything is in place, add
    # this logic later.
    bootengine_cmd mount -o remount,rw ${BOOTENGINE_ROOT_DIR} || die "Can't remount root rw"

    bootengine_cmd mkdir -p ${BOOTENGINE_USR_DIR} || die "Can't create /usr"

    # Cleanup all of the directories in root and symlink them to /usr/
    for link in ${BOOTENGINE_ROOT_DIRS}; do
        setup_usr_link "$link"
    done

    bootengine_cmd mount -o remount,ro ${BOOTENGINE_ROOT_DIR} || die "Can't remount root ro"
}

do_exec_or_find_usr() {
    if ! ismounted ${BOOTENGINE_ROOT_DIR}; then
      die "bootengine: root is not mounted at ${BOOTENGINE_ROOT_DIR}! Failing."
    fi

    setup_root_symlinks

    # Try booting the highest priority usr filesystem
    try_next

    # Failed, clean up and try again...
    if ismounted ${BOOTENGINE_USR_DIR}; then
        bootengine_cmd umount ${BOOTENGINE_USR_DIR}
    fi
    try_next

    # Nope, still here. Hopefully there is a way out.
    if ismounted ${BOOTENGINE_USR_DIR}; then
        bootengine_cmd umount ${BOOTENGINE_USR_DIR}
    fi

    if [ -n "$BOOTENGINE_USR_FALLBACK" ]; then
        warn "bootengine: giving up on kexec!!!"
        warn "bootengine: directly booting ${BOOTENGINE_USR_FALLBACK}"
        bootengine_cmd mount -o ro \
            ${BOOTENGINE_USR_FALLBACK} ${BOOTENGINE_USR_DIR} || \
            die "bootengine: failed to mount ${BOOTENGINE_USR_FALLBACK}"
    else
        # Well this is embarrassing...
        die "bootengine: failed to find a usable usr filesystem!"
    fi
}

if [ "${usr%%:*}" = "gptprio" ]; then
    do_exec_or_find_usr
fi
