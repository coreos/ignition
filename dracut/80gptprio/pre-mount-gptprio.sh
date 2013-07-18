#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

find_root() {
    # Run cgpt to get the partition uuid that we should boot.
    # cgpt prints out to stdout a uppercase string, which is what the kernel
    # needs as the root partition, but udev creates by-partuuid symlinks in
    # lowercase, so we need both versions of the root partition uuid in order
    # to be able to handle everything.  Isn't it grand...
    root_upper=$(cgpt next)
    root_lower=$(echo "${root_upper}" | tr [:upper:] [:lower:])
    cmd_line=$(cat /proc/cmdline)
    mkdir /tmp/boot
    mount -o ro /dev/disk/by-partuuid/${root_lower} /tmp/boot
    kexec --command-line="${cmd_line} root=PARTUUID=${root_upper}" -l /tmp/boot/boot/vmlinuz
    kexec -e
}

if [ -n "$root" -a -z "${root%%gptprio:}" ]; then
    find_root
fi
