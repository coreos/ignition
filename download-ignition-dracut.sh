#!/bin/bash
set -xeuo pipefail
rpmspec -P ignition.spec | grep 'Source1:' | tr -s ' ' | cut -d ' ' -f 2 | xargs wget
grep "ignition-dracut-[0-9a-z]*.tar.gz" sources | sha512sum -c -
