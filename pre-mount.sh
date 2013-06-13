find_root() {
	root="block:/dev/sda4"
}

if [ -n "$root" -a -z "${root%%gptprio:*}" ]; then
    find_root
fi
