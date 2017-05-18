#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd
}

install() {
    inst_multiple systemd-sysusers
    inst_simple "$moddir/ignition-sysusers.service" \
        "$systemdsystemunitdir/ignition-sysusers.service"
    systemctl --root "$initdir" enable ignition-sysusers.service
}
