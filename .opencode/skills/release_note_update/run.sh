#!/usr/bin/env bash
# Maintained in https://github.com/coreos/repo-templates
# Do not edit downstream.

set -euo pipefail

usage() {
    echo "Usage: $0 VERSION [DATE]" >&2
    exit 1
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
    usage
fi

VERSION="$1"
DATE="${2:-$(date +%Y-%m-%d)}"
REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

cd "$REPO_ROOT"
exec go run ./internal/util/tools/release-notes finalize "$VERSION" "$DATE"
