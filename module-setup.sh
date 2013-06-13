#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo fs-lib
}

install() {
    inst /usr/bin/cgpt
    inst_hook cmdline 89 "$moddir/cmdline.sh"
    inst_hook pre-mount 89 "$moddir/pre-mount.sh"
}
