#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# Only attempt if root is the correct type of block device
# Also skip if usr is gptprio, resize should run after kexec.
if [ "${root%%:*}" = "block" -a "${usr%%:*}" != "gptprio" ]; then
    rpart=$(lsblk -n -o PARTTYPE "${root#*:}")
    if [ "${rpart}" = "3884dd41-8582-4404-b9a8-e9b84f2df50e" ]; then
        /usr/bin/cgpt resize 2>&1 | vinfo
    fi
fi
