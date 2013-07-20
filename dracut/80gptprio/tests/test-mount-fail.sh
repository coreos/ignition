#!/bin/sh

. ./include.sh
. ./fixtures.sh

mount() {
	echo mount $@
	if [ $3 = "/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132" ]; then
		echo "ERROR: fake mount error" > /dev/stderr
		return 1
	fi
}

create_kernel_file
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
cleanup_root
