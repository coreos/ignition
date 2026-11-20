#!/bin/bash
# Generate the blackbox test-name manifest used for diff-driven selection.
# Usage: ci/blackbox_manifest.sh <output.tsv>
#
# One row per registered subtest: <dir> <P|N> <name> <min-version> <version>.
# Names are taken from register.Tests, not scraped from source, so tests that
# build their name with fmt.Sprintf and tests that fan out per config version
# or partitioner backend are all represented.
set -euo pipefail

out=${1:?usage: $0 <output.tsv>}
export GOFLAGS=-mod=vendor
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Capture package discovery with a checked assignment: a failure inside
# `for d in $(go list ...)` would otherwise skip the loop and leave an empty
# manifest behind a successful exit.
if ! pkgs=$(go list ./tests/... | grep -E '/tests/(positive|negative)/'); then
	echo "test package discovery failed" >&2
	exit 1
fi

gen=$(mktemp -d ./.bbgen.XXXXXX)
trap 'rm -rf "${gen}"' EXIT
gen_pkg=./$(basename "${gen}")

: > "${out}"
for d in ${pkgs}; do
	rel=${d#github.com/coreos/ignition/v2/}
	cat > "${gen}/main.go" <<EOF
package main

import (
	"fmt"

	"github.com/coreos/ignition/v2/tests/register"
	_ "${d}"
)

func main() {
	for _, t := range register.Tests[register.PositiveTest] {
		fmt.Printf("P\t%s\t%s\t%s\n", t.Name, t.ConfigMinVersion, t.ConfigVersion)
	}
	for _, t := range register.Tests[register.NegativeTest] {
		fmt.Printf("N\t%s\t%s\t%s\n", t.Name, t.ConfigMinVersion, t.ConfigVersion)
	}
}
EOF
	go run "${gen_pkg}" | sed "s#^#${rel}\t#" >> "${out}"
done
