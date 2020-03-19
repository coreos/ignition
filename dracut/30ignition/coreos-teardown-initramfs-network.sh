#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

set -euo pipefail


# Propagate initramfs networking if desired. The policy here is:
#
#    - If a networking configuration was provided before this point
#      (most likely via Ignition) and exists in the real root then
#      we do nothing and don't propagate any initramfs networking.
#    - If a user did not provide any networking configuration
#      then we'll propagate the initramfs networking configuration
#      into the real root.
#
# See https://github.com/coreos/fedora-coreos-tracker/issues/394#issuecomment-599721173
propagate_initramfs_networking() {
    if [ -n "$(ls -A /sysroot/etc/NetworkManager/system-connections/)" ]; then
        echo "info: networking config is defined in the real root"
        echo "info: will not attempt to propagate initramfs networking"
    else
        echo "info: no networking config is defined in the real root"
        if [ -n "$(ls -A /run/NetworkManager/system-connections/)" ]; then
            echo "info: propagating initramfs networking config to the real root"
            cp /run/NetworkManager/system-connections/* /sysroot/etc/NetworkManager/system-connections/
        else
            echo "info: no initramfs networking information to propagate"
        fi
    fi
}

down_interface() {
    echo "info: taking down network device: $1"
    ip link set $1 down
    ip addr flush dev $1
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
    # Propagate initramfs networking if needed
    propagate_initramfs_networking
    # Now that the configuration has been propagated (or not)
    # clean it up so that no information from outside of the
    # real root is passed on to NetworkManager in the real root
    rm -rf /run/NetworkManager/
}

main
