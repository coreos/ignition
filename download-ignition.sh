#!/bin/bash
set -xeuo pipefail
rpmspec -P ignition.spec | grep 'Source0:' | tr -s ' ' | cut -d ' ' -f 2 | xargs wget 
