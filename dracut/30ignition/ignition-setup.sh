#!/bin/bash
set -eux

# Grab our ignition configs from:
#  - A platform specific directory for this platform
#  - The boot partition (user/installer overrides)
sources=("/usr/share/platforms/${OEM_ID}/"
         "/boot/ignition/")

# files go into the /usr/lib/ignition directory
dst=/usr/lib/ignition
mkdir -p "${dst}"

for src in ${sources[*]}; do
    if [ -d "$src" ]; then
        for name in 'base.ign' 'user.ign'; do
            if [ -f "${src}/${name}" ]; then
                cp "${src}/${name}" "${dst}"
            fi
        done
    fi
done
