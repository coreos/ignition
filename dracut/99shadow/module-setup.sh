#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

install() {
    # Simply pull in all the shadow db files so things like systemd-tmpfiles
    # will always be able to find users referenced by the baselayout files.
    cp -af "/usr/share/baselayout/passwd" \
        "${initdir}/etc/passwd"

    cp -af "/usr/share/baselayout/shadow" \
        "${initdir}/etc/shadow"

    cp -af "/usr/share/baselayout/group" \
        "${initdir}/etc/group"

    cp -af "/usr/share/baselayout/gshadow" \
        "${initdir}/etc/gshadow"
}
