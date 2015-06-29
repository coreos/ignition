#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd dm
}

install() {
    inst_multiple veritysetup e2size systemd-escape tr
    inst_simple "$moddir/verity-generator" \
        "$systemdutildir/system-generators/verity-generator"
}
