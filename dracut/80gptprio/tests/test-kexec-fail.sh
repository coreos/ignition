#!/bin/sh

. ./include.sh
. ./fixtures.sh

kexec() {
	echo kexec $@
	if [ "z$1" = "z-e" ]; then
		echo "ERROR: this is a fake kexec failure" > /dev/stderr
		return 1
	fi
}

create_kernel_file
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
[ $root = "block:/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132" ] || exit 1
cleanup_root
