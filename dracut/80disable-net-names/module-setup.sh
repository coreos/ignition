#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd
}

install() {
    # Override the default link to disable the NamePolciy, that way
    # device names remain untouched until the real system boots.
    mkdir -p "$initdir/etc/systemd/network"
    {
        echo "[Link]"
        echo "NamePolicy="
        echo "MACAddressPolicy=persistent"
    } >> "$initdir/etc/systemd/network/98-disable-net-names.link"
}
