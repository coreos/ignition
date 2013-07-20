#!/bin/sh

. ./include.sh
. ./fixtures.sh

kexec() {
	echo kexec $@
	if [ "z$1" = "z-e" ]; then
		exec `which true`
	fi
}

create_kernel_file
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
cleanup_root
