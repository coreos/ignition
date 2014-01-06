# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

. ./include.sh
. ./fixtures.sh

die() {
    echo "die $@"
    assert [ "$*" = "bootengine: failed to find a usable usr filesystem!" ]
    cleanup_root
    exit 0
}

create_empty_root
. ../parse-usr-gptprio.sh
. ../pre-pivot-usr-gptprio.sh
fail "failed to die!"
cleanup_root
