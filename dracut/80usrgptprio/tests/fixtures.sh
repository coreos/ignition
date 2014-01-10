# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

wait_for_dev() {
    echo "waited for $1"
}

vinfo() {
    while read line; do
        info $line
    done
}

info() {
    echo $@
}

warn() {
    echo $@
}

mkdir() {
    echo "mkdir $@"
    /bin/mkdir "$@"
}

_mounted=0
_mount_args=""
mount() {
    echo "mount $@"
    # no need to record remounts
    echo "$*" | grep -q remount && return 0
    if [ $_mounted -eq 0 ]; then
        _mounted=1
        _mount_args="$*"
    else
        fail "XXX already mounted $@"
        return 1
    fi
}

umount() {
    echo "umount $@"
    if [ $_mounted -eq 1 ]; then
        _mounted=0
        _mount_args=""
    else
        fail "XXX not mounted $@"
        return 1
    fi
}

kexec() {
    echo "kexec $@"
    case "$*" in
        *-e*|*--exec*) _kexec_exec "$@"; return $? ;;
        *-l*|*--load*) _kexec_load "$@"; return $? ;;
    esac
}

_kexec_exec() {
    fail "XXX tests should define _kexec_exec, false positives are too easy"
    return 1
}

_kexec_load() {
    return 0
}

usable_root() {
    echo "usable_root $@"
    local d
    [ $_mounted -eq 1 ] || return 1
    [ -d $1 ] || return 1
    for d in "$1/proc" "$1/sys" "$1/dev"; do
        [ -d "$d" ] || return 1
    done
    return 0
}

ismounted() {
    echo "ismounted $@"
    # always consider rootfs to be mounted to test /usr
    [ "$*" = "$BOOTENGINE_ROOT_DIR" ] && return 0
    [ $_mounted -eq 1 ] || return 1
}

modprobe() {
    echo "modprobe $@"
}

die() {
    fail "$@"
}

cgpt() {
    if [ "z$1" = "zfind" ]; then
        echo "/dev/sda3"
        echo "/dev/sda4"
        echo "/dev/sda5"
    fi

    # HACK: we have to save state to the filesystem because
    # this command executes in a subshell so we don't have
    # persistant variables.
    if [ ! -e ${BOOTENGINE_ROOT_DIR}/.cgpt_next ]; then
        CGPT_NEXT="7130c94a-213a-4e5a-8e26-6cce9662f132"
        touch ${BOOTENGINE_ROOT_DIR}/.cgpt_next
    else
        CGPT_NEXT="e03dd35c-7c2d-4a47-b3fe-27f15780a57c"
    fi
    echo ${CGPT_NEXT}
}
