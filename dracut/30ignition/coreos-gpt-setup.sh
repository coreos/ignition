#!/bin/bash
# randomizes the disk guid on the disking containing the partition specified by $1
# and moves the secondary gpt header/partition table to the end of the disk where it
# should be. If the disk guid is already randomized, it does nothing.
set -euo pipefail

UNINITIALIZED_GUID='00000000-0000-4000-a000-000000000001'

# PTUUID is the disk guid, PKANME is the parent kernel name
eval $(lsblk --output PTUUID,PKNAME --pairs --paths --nodeps "$1")
[ "$PTUUID" != "$UNINITIALIZED_GUID" ] && exit 0

sgdisk --disk-guid=R --move-second-header "$PKNAME"
udevadm settle
