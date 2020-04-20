#!/bin/bash
set -xeuo pipefail
rpmspec -P ignition.spec | grep 'Source1:' | tr -s ' ' | cut -d ' ' -f 2 | xargs wget 
