#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

if [ "${root%%:*}" = "squashfs" ]; then
    rootok=1
    mount -t squashfs newroot.squashfs $NEWROOT && { [ -e /dev/root ] || ln -s null /dev/root ; }

    # TODO: Separate this out into a separate step
    if [ -d /usr/share/oem ]; then
        mkdir -p $NEWROOT/usr/share/oem
        mount -t tmpfs -o size=0,mode=755,uid=0,gid=0 tmpfs $NEWROOT/usr/share/oem
        cp -Ra /usr/share/oem/. $NEWROOT/usr/share/oem
    fi
fi
