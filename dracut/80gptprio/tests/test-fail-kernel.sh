#!/bin/sh

. ./include.sh
. ./fixtures.sh

create_root
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
cleanup_root
