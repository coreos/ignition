#!/bin/sh

. ./include.sh
. ./fixtures.sh

die() {
    echo "die $@"
    assert [ "$*" = "bootengine: failed to find a usable root filesystem!" ]
    cleanup_root
    exit 0
}

create_empty_root
. ../parse-gptprio.sh
. ../mount-gptprio.sh
fail "failed to die!"
cleanup_root
