#!/bin/sh

. ./include.sh
. ./fixtures.sh

_kexec_exec() {
	#assert [ $_mounted -eq 0 ]
	assert [ $root = "gptprio:" ]
	assert [ $BOOTENGINE_ROOT = "/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132" ]

	cleanup_root
	exit 0
}

create_kernel_file
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
fail "didn't kexec"
cleanup_root
