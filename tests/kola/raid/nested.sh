#!/bin/bash
# kola: {"platforms": "qemu", "additionalDisks": ["1G", "1G", "1G", "1G"]}
set -euo pipefail

srcdev=$(findmnt -nvr /var -o SOURCE)
[[ ${srcdev} == /dev/md* ]]

devtype=$(lsblk --nodeps --noheadings "${srcdev}" -o TYPE)
[[ ${devtype} == raid0 ]]

for dev in /sys/block/"$(basename "${srcdev}")"/slaves/*; do
    dev=/dev/$(basename "${dev}")
    devtype=$(lsblk --nodeps --noheadings "${dev}" -o TYPE)
    [[ ${devtype} == raid1 ]]
done
