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

# We will support grabbing a platform specific base.ign config
# from the initrd at /usr/lib/ignition/platform/${PLATFORM_ID}/base.ign
copy_file_if_exists "/usr/lib/ignition/platform/${PLATFORM_ID}/base.ign" "${destination}/base.ign"
