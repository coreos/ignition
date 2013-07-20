#!/bin/sh -e

root="gptprio:"
BOOTENGINE_ROOT_DIR="./mnt"

create_kernel_file() {
	create_root
	echo "THIS IS A KERNEL HONEST" > $BOOTENGINE_ROOT_DIR/boot/vmlinuz
}

create_root() {
	/bin/mkdir -p $BOOTENGINE_ROOT_DIR/boot
}

cleanup_root() {
	/bin/rm -r ${BOOTENGINE_ROOT_DIR} > /dev/null 2>&1 || true
}
