#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd
}

install() {
    inst_simple "$moddir/00-journal-log-forwarding.conf" \
        "/etc/systemd/journald.conf.d/00-journal-log-forwarding.conf"
}
