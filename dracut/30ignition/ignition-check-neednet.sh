#!/bin/bash
set -euo pipefail

set +euo pipefail
. /usr/lib/dracut-lib.sh
set -euo pipefail

dracut_func() {
    # dracut is not friendly to set -eu
    set +euo pipefail
    "$@"; local rc=$?
    set -euo pipefail
    return $rc
}

# If we need networking and it hasn't been requested yet, request it.
if [ -f /run/ignition/neednet ] && ! dracut_func getargbool 0 'rd.neednet'; then
    echo "rd.neednet=1" > /etc/cmdline.d/40-ignition-neednet.conf

    # Hack: we need to rerun the NM cmdline hook because we run after
    # dracut-cmdline.service because we need udev. We should be able to move
    # away from this once we run NM as a systemd unit. See also:
    # https://github.com/coreos/fedora-coreos-config/pull/346#discussion_r409843428
    set +euo pipefail
    . /usr/lib/dracut/hooks/cmdline/99-nm-config.sh
    set -euo pipefail
fi
