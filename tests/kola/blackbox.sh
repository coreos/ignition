#!/bin/bash
set -euo pipefail

export PATH=${KOLA_EXT_DATA}/bin/amd64:$PATH
exec ${KOLA_EXT_DATA}/tests.test
