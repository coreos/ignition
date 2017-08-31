#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

depends() {
    echo coreos-network
}

installkernel() {
    instmods iscsi_tcp crc32c
}

install() {
    inst_multiple iscsistart

    inst_simple "$moddir/oracle-oci-root.service" \
        "$systemdutildir/system/oracle-oci-root.service"

    inst_simple "$moddir/iscsi-root-generator" \
        "$systemdutildir/system-generators/iscsi-root-generator"
}
