#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo fs-lib
}

install() {
    dracut_install grep ldconfig systemd-tmpfiles
    inst_hook pre-pivot 80 "$moddir/pre-pivot-setup-root.sh"
}
