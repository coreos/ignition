#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# stub just to make dracut happy if fstab is in use
if [ -z "$root" -o -z "$rootok" ] && grep -qs x-initrd.mount /etc/fstab; then
    [ -z "$root" ] && root=fstab:
    rootok=1
fi
