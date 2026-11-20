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

# If networking hasn't been requested yet, request it.
if ! dracut_func getargbool 0 'rd.neednet'; then
    echo "rd.neednet=1" > /etc/cmdline.d/40-ignition-neednet.conf

    # Hack: we need to rerun the NM cmdline hook because we run after
    # dracut-cmdline.service because we need udev. We should be able to move
    # away from this once we run NM as a systemd unit. See also:
    # https://github.com/coreos/fedora-coreos-config/pull/346#discussion_r409843428
    #
    # See supported locations for hooks:
    # https://github.com/dracut-ng/dracut/commit/04d5e2944e1b1a85a945910995cb8df00d983a7c
    for hook_path in /var/lib/dracut/hooks /usr/lib/dracut/hooks; do
        hook_file="${hook_path}/cmdline/99-nm-config.sh"
        if [ -f "${hook_file}" ]; then
            set +euo pipefail
            . "${hook_file}"
            set -euo pipefail
        fi
    done
fi
