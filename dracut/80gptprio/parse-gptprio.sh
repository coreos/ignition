#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

if [ "${root%%:*}" = "gptprio" ]; then
    rootok=1

    # Wait for both root partitions to show up before we move on
    wait_for_dev "/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132"
    wait_for_dev "/dev/disk/by-partuuid/e03dd35c-7c2d-4a47-b3fe-27f15780a57c"
fi
