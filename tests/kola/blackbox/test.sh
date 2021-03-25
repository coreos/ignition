#!/bin/bash
set -euo pipefail

export PATH=${KOLA_EXT_DATA}/bin:$PATH
exec ${KOLA_EXT_DATA}/bin/tests.test
