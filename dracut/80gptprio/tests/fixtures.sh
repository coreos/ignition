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
		CGPT_NEXT="7130C94A-213A-4E5A-8E26-6CCE9662F132"
		touch ${BOOTENGINE_ROOT_DIR}/.cgpt_next
	else
		CGPT_NEXT="E03DD35C-7C2D-4A47-B3FE-27F15780A57C"
	fi
	echo ${CGPT_NEXT}
}
