#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo systemd dm
}

install() {
    inst_multiple veritysetup e2size systemd-escape tr
    inst_simple "$moddir/verity-generator" \
        "$systemdutildir/system-generators/verity-generator"

    # Device units default to a finite JobRunningTimeout, which would cause
    # dev-mapper-usr.device to fail if its dependencies took too long to
    # start. This was causing boot failures when Ignition was asked to
    # format multiple large filesystems.
    inst_simple "$moddir/no-job-timeout.conf" \
        "$systemdsystemunitdir/dev-mapper-usr.device.d/no-job-timeout.conf"
}
