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
}

mount() {
	echo "mount $@"
}

umount() {
	echo "umount $@"
}

kexec() {
	echo "kexec $@"
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
