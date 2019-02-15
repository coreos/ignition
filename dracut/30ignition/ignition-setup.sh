#!/bin/bash
set -eu

copy_file_if_exists() {
    src="${1}"; dst="${2}"
    if [ -f "${src}" ]; then
        echo "Copying ${src} to ${dst}"
        cp "${src}" "${dst}"
    else
        echo "File ${src} does not exist.. Skipping copy"
    fi
}

destination=/usr/lib/ignition/
mkdir -p $destination

# We will support grabbing a platform specific base.ign config
# from the initrd at /usr/lib/ignition/platform/${OEM_ID}/base.ign
copy_file_if_exists "/usr/lib/ignition/platform/${OEM_ID}/base.ign" "${destination}/base.ign"

# We will support a user embedded config in the boot partition
# under $bootmnt/ignition/config.ign. Note that we mount /boot
# but we don't unmount boot because we are run in a systemd unit
# with MountFlags=slave so it is unmounted for us.
bootmnt=/mnt/boot_partition
mkdir -p $bootmnt
mount /dev/disk/by-label/boot $bootmnt
copy_file_if_exists "${bootmnt}/ignition/config.ign" "${destination}/user.ign"
