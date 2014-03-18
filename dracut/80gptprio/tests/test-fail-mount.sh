#!/bin/sh

. ./include.sh
. ./fixtures.sh

mount() {
    echo "mount $@"
	echo "fake mount error"
    return 1
}

_rebooted=0
systemctl() {
    echo "systemctl $@"
    assert [ "$*" = "--no-block reboot" ]
    _rebooted=1
}

create_empty_root
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
assert [ $_rebooted -eq 1 ]
cleanup_root
