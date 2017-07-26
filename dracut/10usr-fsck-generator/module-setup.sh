#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd
}

install() {
    dracut_install tr
    inst_simple "$moddir/usr-fsck-generator" \
        "$systemdutildir/system-generators/usr-fsck-generator"
}
