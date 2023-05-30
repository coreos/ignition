#!/usr/bin/env bash
# Maintained in https://github.com/coreos/repo-templates
# Do not edit downstream.

set -e

[ $# == 2 ] || { echo "usage: $0 <version> <commit>" && exit 1; }

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

if [ -f Makefile ]; then
	make
else
	./build
fi

git tag --sign --message "Ignition ${VER}" "${VER}" "${COMMIT}"
git verify-tag --verbose "${VER}"
