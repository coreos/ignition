#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

install() {
    # Run systemd-sysusers during the build so things like systemd-tmpfiles
    # will always be able to find users referenced by the baselayout files.
    cp -ar "/usr/lib/sysusers.d" \
        "${initdir}/usr/lib/"

    systemd-sysusers --root="${initdir}"
}
