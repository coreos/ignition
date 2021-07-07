#!/bin/bash

set -euo pipefail

# Clean up the user embedded config read by ignition-setup-user.service, and
# also anything else that might have been placed under $bootmnt/ignition.
# Note that we mount /boot but we don't unmount it because we are run in a
# systemd unit with MountFlags=slave so it is unmounted for us.
bootmnt=/mnt/boot_partition
mkdir -p $bootmnt
mount /dev/disk/by-label/boot $bootmnt
rm -rf "${bootmnt}/ignition"
