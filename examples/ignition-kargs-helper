#!/bin/bash

# Requires sed to be present

set -euxo pipefail

# Mount /boot. Note that we mount /boot but we don't unmount it because we
# are run in a systemd unit with MountFlags=slave so it is unmounted for us.
bootmnt=/mnt/boot_partition
mkdir -p ${bootmnt}
bootdev=/dev/disk/by-label/boot
mount -o rw ${bootdev} ${bootmnt}
grubcfg="${bootmnt}/grub/grub.cfg"

orig_kernelopts="$(grep kernelopts= $grubcfg | sed s,^kernelopts=,,)"
# add leading and trailing whitespace to allow for easy sed replacements
kernelopts=" $orig_kernelopts "

while [[ $# -gt 0 ]]
do
    key="$1"

    case $key in
    --should-exist)
        arg="$2"
        # don't repeat the arg
        if [[ ! "${kernelopts[*]}" =~ " ${arg} " ]]; then
            kernelopts="$kernelopts$arg "
        fi
        shift 2
        ;;
    --should-not-exist)
        kernelopts="$(echo "$kernelopts" | sed "s| $2 | |g")"
        shift 2
        ;;
    *)
        echo "Unknown option"
        exit 1
        ;;
    esac
done

# trim the leading and trailing whitespace
kernelopts="$(echo "$kernelopts" | sed -e 's,^[[:space:]]*,,' -e 's,[[:space:]]*$,,')"

# only apply the changes & reboot if changes have been made
if [[ "$kernelopts" != "$orig_kernelopts" ]]; then
    sed -i "s|^\(kernelopts=\).*|\1$kernelopts|" $grubcfg

    systemctl reboot --force
fi
