#!/usr/bin/env bash

set -e

[ $# == 2 ] || { echo "usage: $0 version commit" && exit 1; }

VER=$1
COMMIT=$2

[[ "${VER}" =~ ^v[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+(-.+)?$ ]] || {
	echo "malformed version: \"${VER}\""
	exit 2
}

[[ "${COMMIT}" =~ ^[[:xdigit:]]+$ ]] || {
	echo "malformed commit id: \"${COMMIT}\""
	exit 3
}

source ./build

git tag --sign --message "Ignition ${VER}" "${VER}" "${COMMIT}"

git verify-tag --verbose "${VER}"
