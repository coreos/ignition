#!/bin/sh

. ./include.sh
. ./fixtures.sh

mount() {
	echo mount $@
	if [ $3 = "/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132" ]; then
		echo "ERROR: fake mount error" > /dev/stderr
		return 1
	fi
	_mounted=1
}

_kexec_exec() {
	assert [ $_mounted -eq 0 ]
	assert [ $root = "gptprio:" ]
	assert [ $BOOTENGINE_ROOT = "/dev/disk/by-partuuid/e03dd35c-7c2d-4a47-b3fe-27f15780a57c" ]

	cleanup_root
	exit 0
}

create_kernel_file
. ../parse-gptprio.sh
. ../mount-gptprio.sh
fail "didn't kexec"
cleanup_root
