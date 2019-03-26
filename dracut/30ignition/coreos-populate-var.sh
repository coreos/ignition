#!/bin/bash
set -euo pipefail

fatal() {
    echo "$@" >&2
    exit 1
}

if [ $# -ne 0 ]; then
    fatal "Usage: $0"
fi

# See the similar code block in Anaconda, which handles this today for Atomic
# Host and Silverblue:
# https://github.com/rhinstaller/anaconda/blob/b9ea8ce4e68196b30a524c1cc5680dcdc4b89371/pyanaconda/payload/rpmostreepayload.py#L332

# Simply manually mkdir /var/lib; the tmpfiles.d entries otherwise reference
# users/groups which we don't have access to from here (though... we *could*
# import them from the sysroot, and have nss-altfiles in the initrd, but meh...
# let's just wait for systemd-sysusers which will make this way easier:
# https://github.com/coreos/fedora-coreos-config/pull/56/files#r262592361).
mkdir -p /sysroot/var/lib

systemd-tmpfiles --create --boot --root=/sysroot \
    --prefix=/var/home \
    --prefix=/var/roothome \
    --prefix=/var/opt \
    --prefix=/var/srv \
    --prefix=/var/usrlocal \
    --prefix=/var/mnt \
    --prefix=/var/media

# Ask for /var to be relabeled.
# See also: https://github.com/coreos/ignition/issues/635.
mkdir -p /run/tmpfiles.d
echo "Z /var - - -" > /run/tmpfiles.d/var-relabel.conf

# XXX: https://github.com/systemd/systemd/pull/11903
for unit in systemd-{journal-catalog-update,random-seed}.service; do
    mkdir -p /run/systemd/system/${unit}.d
    cat > /run/systemd/system/${unit}.d/after-tmpfiles.conf <<EOF
[Unit]
After=systemd-tmpfiles-setup.service
EOF
done
