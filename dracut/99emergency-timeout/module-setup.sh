#!/bin/bash
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

install() {
    inst_multiple \
        cut \
        date

    inst_hook emergency 99 "${moddir}/timeout.sh"
}
