#!/bin/sh

. ./include.sh
. ./fixtures.sh

_kexec_exec() {
	echo "ERROR: this is a fake kexec failure"
	return 1
}

create_kernel_file
. ../parse-gptprio.sh
. ../mount-gptprio.sh
assert [ $_mounted -eq 1 ]
assert [ "$_mount_args" = "-o ro /dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132 ./mnt" ]
cleanup_root
