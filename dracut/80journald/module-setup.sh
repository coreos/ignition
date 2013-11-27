#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd
}

install() {
    # Journal needs to forward to dmesg so we see output on anything
    # defined by console= on the command line. Otherwise everything
    # logged before a kexec is completely lost.
    mkdir -p "$initdir/etc/systemd"
    {
        echo "[Journal]"
        echo "ForwardToKMsg=yes"
        echo "MaxLevelKMsg=debug"
    } >> "$initdir/etc/systemd/journald.conf"
}
