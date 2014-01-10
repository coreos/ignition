#!/bin/sh -e
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

. ./include.sh

for i in test-*.sh; do
    echo -e "RUNNING $i\n"
    cleanup_root
    sh -e $i
    echo -e "\n"
done

echo SUCCESS
exit 0
