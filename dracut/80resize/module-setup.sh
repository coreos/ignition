#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

install() {
    dracut_install e2fsck resize2fs lsblk
    dracut_install /usr/bin/cgpt
    dracut_install /usr/bin/old_bins/cgpt
    inst_hook pre-mount 80 "$moddir/pre-mount-resize.sh"
}
