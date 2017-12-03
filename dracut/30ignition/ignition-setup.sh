#!/bin/bash
# Set up /usr/lib/ignition, copying contents from /usr/share/oem.

set -e

case "$1" in
normal)
    src=/mnt/oem
    mkdir -p "${src}"
    mount /dev/disk/by-label/OEM "${src}"
    # retry-umount may not be necessary, but be cautious
    trap 'retry-umount "${src}"' EXIT
    ;;
pxe)
    # OEM directory in the initramfs itself
    src=/usr/share/oem
    ;;
*)
    echo "Usage: $0 {normal|pxe}" >&2
    exit 1
esac

dst=/usr/lib/ignition
mkdir -p "${dst}"
for name in base.ign default.ign; do
    if [[ -e "${src}/base/${name}" ]]; then
        cp "${src}/base/${name}" "${dst}"
    fi
done
if [[ -e "${src}/config.ign" ]]; then
    cp "${src}/config.ign" "${dst}/user.ign"
fi
