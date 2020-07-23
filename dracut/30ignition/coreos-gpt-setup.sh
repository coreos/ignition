#!/bin/bash
# randomizes the disk guid on the disk containing the partition specified by $1
# and moves the secondary gpt header/partition table to the end of the disk where it
# should be. If the disk guid is already randomized, it does nothing.
set -euo pipefail

UNINITIALIZED_GUID='00000000-0000-4000-a000-000000000001'

# On RHEL 8 the version of lsblk doesn't have PTUUID. Let's detect
# if lsblk supports it. In the future we can remove the 'if' and
# just use the 'else'.
if ! lsblk --help | grep -q PTUUID; then
    # Get the PKNAME
    eval $(lsblk --output PKNAME --pairs --paths --nodeps "$1")
    # Get the PTUUID
    eval $(blkid -o export $PKNAME)
else
    # PTUUID is the disk guid, PKNAME is the parent kernel name
    eval $(lsblk --output PTUUID,PKNAME --pairs --paths --nodeps "$1")
fi

[ "$PTUUID" != "$UNINITIALIZED_GUID" ] && exit 0

sgdisk --disk-guid=R --move-second-header "$PKNAME"
udevadm settle
