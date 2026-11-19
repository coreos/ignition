#!/bin/bash
set -euo pipefail
# kola: { "timeoutMin": 120, "minMemory": 4096 }

export PATH=${KOLA_EXT_DATA}/bin:$PATH
exec ${KOLA_EXT_DATA}/bin/tests.test
