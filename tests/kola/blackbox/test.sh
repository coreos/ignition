#!/bin/bash
set -euo pipefail
# kola: { "timeoutMin": 120, "minMemory": 4096 }

export PATH=${KOLA_EXT_DATA}/bin:$PATH

# CI may narrow the suite to the tests a PR can affect; the args file is
# written by ci/blackbox_select.py and stashed next to the test binary. Without
# it (branch builds, nightly runs) the full suite runs.
args=()
if [[ -f "${KOLA_EXT_DATA}/bin/selection.args" ]]; then
	mapfile -t args < "${KOLA_EXT_DATA}/bin/selection.args"
fi

exec "${KOLA_EXT_DATA}/bin/tests.test" "${args[@]}"
