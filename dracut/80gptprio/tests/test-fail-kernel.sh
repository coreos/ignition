#!/bin/sh

. ./include.sh
. ./fixtures.sh

create_root
. ../parse-gptprio.sh
. ../pre-mount-gptprio.sh
assert [ $root = "block:/dev/disk/by-partuuid/7130c94a-213a-4e5a-8e26-6cce9662f132" ]
cleanup_root
