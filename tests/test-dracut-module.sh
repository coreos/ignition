#!/bin/sh -x -e

function wait_for_dev {
	echo "waited for $1"
}

root="GPTPRIO"

. ../10gptprio/parse-gptprio.sh
. ../10gptprio/pre-mount-gptprio.sh

echo "\$root = $root"
