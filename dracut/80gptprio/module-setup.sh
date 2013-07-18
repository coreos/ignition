#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo fs-lib
}

install() {
    inst /usr/bin/cgpt
    inst /usr/sbin/kexec
    inst /usr/bin/old_bins/cgpt
    inst /usr/bin/tr
    inst_hook cmdline 80 "$moddir/parse-gptprio.sh"
    inst_hook pre-mount 80 "$moddir/pre-mount-gptprio.sh"
}
