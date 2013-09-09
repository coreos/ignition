#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo fs-lib
}

installkernel() {
    instmods squashfs
}

install() {
    inst_hook cmdline 80 "$moddir/parse-squashfs.sh"
}
