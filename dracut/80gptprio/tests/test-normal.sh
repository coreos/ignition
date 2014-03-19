#!/bin/sh

. ./include.sh
. ./fixtures.sh

_kexeced=0
systemctl() {
	assert [ $_mounted -eq 0 ]
    assert [ "$*" = "--no-block kexec" ]
    _kexeced=1
}

create_kernel_file
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
assert [ $_kexeced -eq 1 ]
cleanup_root
