#!/bin/bash
set -euo pipefail

copy_file_if_exists() {
    src="${1}"; dst="${2}"
    if [ -f "${src}" ]; then
        echo "Copying ${src} to ${dst}"
        cp "${src}" "${dst}"
    else
        echo "File ${src} does not exist.. Skipping copy"
    fi
}

destination=/usr/lib/ignition
mkdir -p $destination

if command -v is-live-image >/dev/null && is-live-image; then
    # Live image. If the user has supplied a config.ign via an appended
    # initrd, put it in the right place.
    copy_file_if_exists "/config.ign" "${destination}/user.ign"
else
    # We will support a user embedded config in the boot partition
    # under $bootmnt/ignition/config.ign. Note that we mount /boot
    # but we don't unmount boot because we are run in a systemd unit
    # with MountFlags=slave so it is unmounted for us.
    bootmnt=/mnt/boot_partition
    mkdir -p $bootmnt
    # mount as read-only since we don't strictly need write access and we may be
    # running alongside other code that also has it mounted ro
    mount -o ro /dev/disk/by-label/boot $bootmnt
    copy_file_if_exists "${bootmnt}/ignition/config.ign" "${destination}/user.ign"
fi
