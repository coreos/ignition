#!/bin/sh -e

root="gptprio:"
BOOTENGINE_MNT_DIR="./mnt"

create_kernel_file() {
	create_root
	echo "THIS IS A KERNEL HONEST" > $BOOTENGINE_MNT_DIR/boot/vmlinuz
}

create_empty_root() {
	/bin/rm -rf ${BOOTENGINE_MNT_DIR}
	/bin/mkdir -p $BOOTENGINE_MNT_DIR
}

create_root() {
	/bin/rm -rf ${BOOTENGINE_MNT_DIR}
	/bin/mkdir -p $BOOTENGINE_MNT_DIR/boot
}

cleanup_root() {
	if [ -e ${BOOTENGINE_MNT_DIR}/.failed ]; then
		cat ${BOOTENGINE_MNT_DIR}/.failed
		/bin/rm -rf ${BOOTENGINE_MNT_DIR}
		echo FAILED
		exit 1
	fi
	/bin/rm -rf ${BOOTENGINE_MNT_DIR}
}

fail() {
	echo "FAIL: $@"
	echo "FAIL: $@" >> ${BOOTENGINE_MNT_DIR}/.failed
}

fail_if() {
	if "$@"; then
		fail "$@"
	fi
}

assert() {
	if ! "$@"; then
		fail assert "$@"
	fi
}
