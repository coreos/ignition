#!/bin/bash
set -xeuo pipefail
rpmspec -P ignition.spec | grep 'Source0:' | tr -s ' ' | cut -d ' ' -f 2 | xargs wget
grep "ignition-[0-9a-z]*.tar.gz" sources | sha512sum -c -
