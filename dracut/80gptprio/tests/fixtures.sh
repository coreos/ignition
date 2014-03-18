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

getarg() {
    return 0
}

_mounted=0
_mount_args=""
mount() {
	echo "mount $@"
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
}

ismounted() {
	echo "ismounted $@"
	[ $_mounted -eq 1 ] || return 1
}

die() {
	fail "$@"
}

systemctl() {
    fail "$@"
}

cgpt() {
    if [ "$1" != "next" ]; then
        fail "unexpected cgpt call"
    fi
	echo "7130c94a-213a-4e5a-8e26-6cce9662f132"
}
