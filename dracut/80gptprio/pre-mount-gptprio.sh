#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# Flexible mount directory for testing
[ -z ${BOOTENGINE_MNT_DIR} ] && BOOTENGINE_MNT_DIR=/run/gptprio
BOOTENGINE_KERNEL_PATH=${BOOTENGINE_MNT_DIR}/boot/vmlinuz
EFI_SYSTAB_PATH=/sys/firmware/efi/systab

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

# Try to load a kernel from the given partition
load_kernel() {
    local partname="$1"
    local partuuid="$2"

    info "bootengine: waiting for devices to settle"
    bootengine_cmd udevadm settle

    info "bootengine: mounting PARTUUID=${partuuid} to ${BOOTENGINE_MNT_DIR}"
    bootengine_cmd mkdir -p "${BOOTENGINE_MNT_DIR}" || return $?
    if ! bootengine_cmd mount -o ro \
        "/dev/disk/by-partuuid/${partuuid}" "${BOOTENGINE_MNT_DIR}"
    then
        warn "bootengine: udev may have detected the partition table update"
        warn "bootengine: and deleted /dev/disk/by-partuuid/${partuuid}"
        warn "bootengine: waiting for udev to settle and trying again..."
        bootengine_cmd udevadm settle
        bootengine_cmd mount -o ro "/dev/disk/by-partuuid/${partuuid}" \
            "${BOOTENGINE_MNT_DIR}" || return $?
    fi

    if [ ! -s "${BOOTENGINE_KERNEL_PATH}" ]; then
      warn "bootengine: no kernel at ${BOOTENGINE_KERNEL_PATH}"
      return 1
    fi

    # for efi we need to grab the acpi rsdp address while in the efi-booted
    # environment and pass it on to the next kernel
    local acpi_rsdp
    if [ -e "${EFI_SYSTAB_PATH}" ]; then
        acpi_rsdp=$(grep -m1 '^ACPI' "${EFI_SYSTAB_PATH}")
        acpi_rsdp=${acpi_rsdp##*=}
        acpi_rsdp=${acpi_rsdp:+acpi_rsdp=${acpi_rsdp}}
    fi

    info "bootengine: loading kernel from ${BOOTENGINE_KERNEL_PATH}..."
    bootengine_cmd kexec --reuse-cmdline \
        --append="${partname}=PARTUUID=${partuuid} ${acpi_rsdp}" \
        --load "${BOOTENGINE_KERNEL_PATH}" || return $?

    bootengine_cmd umount "${BOOTENGINE_MNT_DIR}" || return $?
}

if [ "${root%%:*}" = "gptprio" ] || [ "${usr%%:*}" = "gptprio" ]; then
    partname=usr
    if [ "${root%%:*}" = "gptprio" ]; then
        partname=root
    fi

    partuuid=$(cgpt next)
    if [ $? -ne 0 -o -z "$partuuid" ]; then
        # there is no where to go if cgpt failed
        die "bootengine: cgpt failed to find a partition!"
    fi

    if load_kernel "${partname}" "${partuuid}"; then
        bootengine_cmd systemctl --no-block kexec
        info "bootengine: loaded kernel, preparing to kexec"
    else
        bootengine_cmd systemctl --no-block reboot
        warn "bootengine: aborting, attempting to fall back via reboot"
    fi
fi
