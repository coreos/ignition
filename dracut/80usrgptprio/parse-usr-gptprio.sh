#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

export usr="$(getarg usr=)"

if [ "${usr%%:*}" = "gptprio" -a "${root%%:*}" = "gptprio" ]; then
    die "Cannot use both root and usr gptprio"
fi

case "$usr" in
    block:LABEL=*|LABEL=*)
        usr="${usr#block:}"
        usr="$(echo $usr | sed 's,/,\\x2f,g')"
        usr="block:/dev/disk/by-label/${usr#LABEL=}"
        ;;
    block:UUID=*|UUID=*)
        usr="${usr#block:}"
        usr="${usr#UUID=}"
        usr="$(echo $usr | tr "[:upper:]" "[:lower:]")"
        usr="block:/dev/disk/by-uuid/${usr#UUID=}"
        ;;
    block:PARTUUID=*|PARTUUID=*)
        usr="${usr#block:}"
        usr="${usr#PARTUUID=}"
        usr="$(echo $usr | tr "[:upper:]" "[:lower:]")"
        usr="block:/dev/disk/by-partuuid/${usr}"
        ;;
    block:PARTLABEL=*|PARTLABEL=*)
        usr="${usr#block:}"
        usr="block:/dev/disk/by-partlabel/${usr#PARTLABEL=}"
        ;;
    /dev/*)
        usr="block:${usr}"
        ;;
esac

if [ "${usr%%:*}" = "gptprio" ]; then
    info "bootengine: waiting on CoreOS USR partitions"

    # Wait for both usr partitions to show up before we move on
    wait_for_dev "/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132"
    wait_for_dev "/dev/disk/by-partuuid/e03dd35c-7c2d-4a47-b3fe-27f15780a57c"

elif [ "${usr%%:*}" = "block" ]; then
    usrflags="$(getarg usrflags=)"
    usrfstype="$(getarg usrfstype=)"

    # Add basic block devices to the initrd's fstab
    add_mount_point "${usr#block:}" /sysroot/usr \
        "${usrfstype:-auto}" "${usrflags:-ro},x-initrd.mount"

    info "bootengine: waiting on /usr device ${usr#block:}"
    wait_for_dev "${usr#block:}"
fi
