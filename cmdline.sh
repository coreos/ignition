#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

case "$root" in
    GPTPRIO=*)
        root="${root#gptprio:}"
        root="$(echo $root | sed 's,/,\\x2f,g')"
	root="gptprio:${root}"
        rootok=1 ;;
esac

[ "${root%%:*}" = "gptprio" ] && wait_for_dev "${root#*GPTPRIO=}"
