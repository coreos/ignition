#!/bin/bash

set -e

# OEM directory in the initramfs itself
src=/usr/share/oem

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

# if we have config.ign on boot, overwrite it
if [[ -e "/boot/config.ign" ]]; then
    cp "/boot/config.ign" "${dst}/user.ign"
fi
