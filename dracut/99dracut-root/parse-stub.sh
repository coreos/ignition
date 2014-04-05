#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

# This is just a stub to get dracut to not die. If root is setup by a
# systemd generator dracut won't know but that case is perfectly fine.
[ -z "$root" ] && root=stub:
[ -z "$rootok" ] && rootok=1
