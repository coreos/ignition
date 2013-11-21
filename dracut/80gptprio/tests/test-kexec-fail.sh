#!/bin/sh

. ./include.sh
. ./fixtures.sh

_kexec_exec() {
	echo "ERROR: this is a fake kexec failure"
	return 1
}

create_kernel_file
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
assert [ $root = "block:/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132" ]
cleanup_root
