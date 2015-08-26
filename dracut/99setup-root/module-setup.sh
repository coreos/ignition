#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo fs-lib
}

install() {
    dracut_install grep ldconfig mountpoint systemd-tmpfiles coreos-tmpfiles

    inst_script "${moddir}/initrd-setup-root" \
	        "/sbin/initrd-setup-root"

    inst_simple "${moddir}/initrd-setup-root.service" \
        "${systemdsystemunitdir}/initrd-setup-root.service"

    ln_r "${systemdsystemunitdir}/initrd-setup-root.service" \
        "${systemdsystemunitdir}/initrd.target.wants/initrd-setup-root.service"
}
