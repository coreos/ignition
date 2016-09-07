#!/bin/bash

# This module extends dracut's systemd-networkd module to include additional
# networking configuration for Ignition.

# called by dracut
depends() {
    echo systemd-networkd
}

# called by dracut
install() {
    inst_multiple -o \
        $systemdutildir/systemd-resolved \
        $systemdsystemunitdir/systemd-resolved.service \
        /etc/systemd/resolved.conf

    inst_simple "$moddir/10-down.conf" \
        "$systemdsystemunitdir/systemd-networkd.service.d/10-down.conf"

    inst_simple "$moddir/10-nodeps.conf" \
        "$systemdsystemunitdir/systemd-resolved.service.d/10-nodeps.conf"

    inst_simple "$moddir/yy-digitalocean.network" \
        "$systemdutildir/network/yy-digitalocean.network"

    inst_simple "$moddir/yy-pxe.network" \
        "$systemdutildir/network/yy-pxe.network"

    inst_simple "$moddir/zz-default.network" \
        "$systemdutildir/network/zz-default.network"

    # user/group required for systemd-resolved
    getent passwd systemd-resolve >> "$initdir/etc/passwd"
    getent group systemd-resolve >> "$initdir/etc/group"

    # point /etc/resolv.conf @ systemd-resolved's resolv.conf
    ln -s ../run/systemd/resolve/resolv.conf "$initdir/etc/resolv.conf"
}
