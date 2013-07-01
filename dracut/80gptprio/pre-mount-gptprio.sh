#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

find_root() {
    # TODO: use the cgpt next tool here
    root="block:/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132"
}

if [ -n "$root" -a -z "${root%%gptprio:}" ]; then
    find_root
fi
