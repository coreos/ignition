#!/bin/bash
# Unmount $1, retrying a few times.

set -e

mp="$1"
if [ -z "$mp" ] ; then
    echo "Usage: $0 <mountpoint>"
    exit 1
fi

if ! mountpoint -q "$mp"; then
    exit 0
fi

tries=5
while true; do
    umount "$mp" ||:
    # Apparently umount may return failure when it actually succeeded
    if ! mountpoint -q "$mp"; then
        exit 0
    fi

    tries=$(( $tries - 1 ))
    if [[ $tries = 0 ]]; then
        echo "Giving up." >&2
        exit 1
    fi

    echo "Couldn't unmount $mp; retrying..." >&2
    sleep 1
done
