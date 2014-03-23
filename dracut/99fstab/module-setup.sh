#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    return 0
}

install() {
    dracut_install grep
    inst_hook cmdline 99 "$moddir/parse-fstab.sh"
}
