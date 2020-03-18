#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

set -euo pipefail

down_interface() {
    echo "info: taking down network device: $1"
    ip link set $1 down
    ip addr flush dev $1
    rm -f -- /tmp/net.$1.did-setup
}

down_bonds() {
    if [ -f "/sys/class/net/bonding_masters" ]; then
        bonds="$(cat /sys/class/net/bonding_masters)"
        for b in ${bonds[@]}; do
            down_interface ${b}
            echo -"${b}" > /sys/class/net/bonding_masters
         done
    fi
}

# This mimics the behaviour of dracut's ifdown() in net-lib.sh
# Note that in the futre we would like to possibly use `nmcli` networking off`
# for this. See the following two comments for details:
# https://github.com/coreos/fedora-coreos-tracker/issues/394#issuecomment-599721763
# https://github.com/coreos/fedora-coreos-tracker/issues/394#issuecomment-599746049
down_interfaces() {
    if ! [ -z "$(ls /sys/class/net)" ]; then
        for f in /sys/class/net/*; do
            interface=$(basename "$f")
            # The `bonding_masters` entry is not a true interface and thus
            # cannot be taken down.  If they existed, the bonded interfaces
            # were taken down earlier in this script.
            if [ "$interface" == "bonding_masters" ]; then continue; fi
            down_interface $interface
        done
    fi
}

main() {
    # We want to take down the bonded interfaces first
    down_bonds
    # Clean up the interfaces set up in the initramfs
    down_interfaces
}

main
