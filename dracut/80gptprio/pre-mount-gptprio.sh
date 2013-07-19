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
    info "bootengine: cmd_line was $cmd_line"  > /dev/kmsg
    echo "bootengine: root_upper was $root_upper" > /dev/kmsg
    mkdir /tmp/boot
    echo "bootengine: preparing disk mount for /dev/disk/by-partuuid/${root_lower}" > /dev/kmsg
    mount -o ro /dev/disk/by-partuuid/${root_lower} /tmp/boot
    echo "bootengine: mount returned $?, setting up kexec on kernel $(ls -l /tmp/boot/boot/vmlinuz)" > /dev/kmsg
    kexec --command-line="${cmd_line} root=PARTUUID=${root_upper}" -l /tmp/boot/boot/vmlinuz 2>&1 > /dev/kmsg
    echo "bootengine: kexec returned $?" > /dev/kmsg
    kexec -e 2>&1 > /dev/kmsg
    echo "ERROR: bootengine: kexec -e shouldn't return!" > /dev/kmsg
    echo "cmd_line was $cmd_line" > /dev/kmsg
    echo "root_upper was $root_upper" > /dev/kmsg
    exit 1
}

if [ -n "$root" -a -z "${root%%gptprio:}" ]; then
    find_root
fi
