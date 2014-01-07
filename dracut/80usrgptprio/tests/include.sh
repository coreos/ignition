# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

root="fake"
usr="gptprio:"
BOOTENGINE_ROOT_DIR="./mnt"

create_kernel_file() {
    create_root
    # TEMPORARY DELETE ME, /usr/boot has priority over /boot
    echo "THIS IS A KERNEL HONEST" > $BOOTENGINE_ROOT_DIR/boot/vmlinuz
    # TEMPORARY END
    echo "THIS IS A KERNEL HONEST" > $BOOTENGINE_ROOT_DIR/usr/boot/vmlinuz
}

create_empty_root() {
    /bin/rm -rf ${BOOTENGINE_ROOT_DIR}
    /bin/mkdir -p $BOOTENGINE_ROOT_DIR
}

create_root() {
    /bin/rm -rf ${BOOTENGINE_ROOT_DIR}
    /bin/mkdir -p $BOOTENGINE_ROOT_DIR/boot \
        $BOOTENGINE_ROOT_DIR/dev \
        $BOOTENGINE_ROOT_DIR/proc \
        $BOOTENGINE_ROOT_DIR/sys \
        $BOOTENGINE_ROOT_DIR/usr/boot \
        $BOOTENGINE_ROOT_DIR/usr/bin \
        $BOOTENGINE_ROOT_DIR/usr/sbin \
        $BOOTENGINE_ROOT_DIR/usr/lib64
    /bin/ln -s lib64 $BOOTENGINE_ROOT_DIR/usr/lib
}

cleanup_root() {
    if [ -e ${BOOTENGINE_ROOT_DIR}/.failed ]; then
        cat ${BOOTENGINE_ROOT_DIR}/.failed
        /bin/rm -rf ${BOOTENGINE_ROOT_DIR}
        echo FAILED
        exit 1
    fi
    /bin/rm -rf ${BOOTENGINE_ROOT_DIR}
}

fail() {
    echo "FAIL: $@"
    echo "FAIL: $@" >> ${BOOTENGINE_ROOT_DIR}/.failed
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
