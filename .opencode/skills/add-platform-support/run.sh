#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo "Usage: $0 --id ID --name NAME --url METADATA_URL --docs DOCS_URL [--description DESC]"
    echo ""
    echo "Add support for a new cloud provider/platform to Ignition."
    echo ""
    echo "Required:"
    echo "  --id          Platform ID (lowercase, alphanumeric, hyphens allowed)"
    echo "  --name        Display name (e.g. 'Hetzner Cloud')"
    echo "  --url         Metadata URL to fetch config from"
    echo "  --docs        Documentation URL for the platform"
    echo ""
    echo "Optional:"
    echo "  --description Platform description for supported-platforms.md"
    echo "                (default: standard userdata description)"
    exit 1
}

PROVIDER_ID=""
PROVIDER_NAME=""
METADATA_URL=""
DOCS_URL=""
DESCRIPTION="Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately."

while [[ $# -gt 0 ]]; do
    case "$1" in
        --id)          PROVIDER_ID="$2"; shift 2 ;;
        --name)        PROVIDER_NAME="$2"; shift 2 ;;
        --url)         METADATA_URL="$2"; shift 2 ;;
        --docs)        DOCS_URL="$2"; shift 2 ;;
        --description) DESCRIPTION="$2"; shift 2 ;;
        -h|--help)     usage ;;
        *)             echo "Unknown option: $1"; usage ;;
    esac
done

if [[ -z "$PROVIDER_ID" || -z "$PROVIDER_NAME" || -z "$METADATA_URL" || -z "$DOCS_URL" ]]; then
    echo "Error: --id, --name, --url, and --docs are all required"
    usage
fi

# Find repo root (where .opencode/ lives)
REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$REPO_ROOT"

# Validate provider ID format
if ! [[ "$PROVIDER_ID" =~ ^[a-z][a-z0-9-]*$ ]]; then
    echo "Error: Provider ID must be lowercase alphanumeric (hyphens allowed), got: $PROVIDER_ID"
    exit 1
fi

# Check provider doesn't already exist
if [[ -d "internal/providers/$PROVIDER_ID" ]]; then
    echo "Error: Provider '$PROVIDER_ID' already exists at internal/providers/$PROVIDER_ID"
    exit 1
fi

# Parse metadata URL
if ! [[ "$METADATA_URL" =~ ^([a-z]+)://([^/]+)(/.*)?$ ]]; then
    echo "Error: Could not parse metadata URL: $METADATA_URL" >&2
    exit 1
fi
SCHEME="${BASH_REMATCH[1]}"
HOST="${BASH_REMATCH[2]}"
URL_PATH="${BASH_REMATCH[3]:-/}"

# Discover current experimental spec version
SPEC_DIR=$(ls -d config/v*_experimental/ 2>/dev/null | sort -V | tail -1 | sed 's|/$||')
if [[ -z "$SPEC_DIR" ]]; then
    echo "Error: Could not determine current experimental spec version"
    exit 1
fi
SPEC_PKG=$(basename "$SPEC_DIR")

YEAR=$(date +%Y)

echo "Adding platform: $PROVIDER_NAME ($PROVIDER_ID)"
echo "  Metadata URL: $METADATA_URL"
echo "  Spec version: $SPEC_PKG"
echo ""

# Step 1: Generate provider Go file
mkdir -p "internal/providers/$PROVIDER_ID"
cat > "internal/providers/$PROVIDER_ID/$PROVIDER_ID.go" <<EOF
// Copyright ${YEAR} CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The ${PROVIDER_ID} provider fetches a remote configuration from the ${PROVIDER_NAME}
// user-data metadata service URL.

package ${PROVIDER_ID}

import (
	"net/url"

	"github.com/coreos/ignition/v2/config/${SPEC_PKG}/types"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var (
	userdataURL = url.URL{
		Scheme: "${SCHEME}",
		Host:   "${HOST}",
		Path:   "${URL_PATH}",
	}
)

func init() {
	platform.Register(platform.Provider{
		Name:  "${PROVIDER_ID}",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	data, err := f.FetchToBuffer(userdataURL, resource.FetchOptions{})

	if err != nil && err != resource.ErrNotFound {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}
EOF
echo "  Created internal/providers/$PROVIDER_ID/$PROVIDER_ID.go"

# Step 2: Update provider registry (alphabetical insertion)
REGISTRY="internal/register/providers.go"
NEW_IMPORT="	_ \"github.com/coreos/ignition/v2/internal/providers/$PROVIDER_ID\""

if grep -q "providers/$PROVIDER_ID\"" "$REGISTRY"; then
    echo "Error: Provider '$PROVIDER_ID' is already registered in $REGISTRY"
    exit 1
fi

# Extract imports, add new one, sort, reconstruct
IMPORT_START=$(grep -n 'import (' "$REGISTRY" | head -1 | cut -d: -f1)
IMPORT_END=$(awk "NR>$IMPORT_START && /^\)/{print NR; exit}" "$REGISTRY")

{
    head -n "$IMPORT_START" "$REGISTRY"
    {
        sed -n "$((IMPORT_START + 1)),$((IMPORT_END - 1))p" "$REGISTRY"
        echo "$NEW_IMPORT"
    } | sort
    tail -n +"$IMPORT_END" "$REGISTRY"
} > "$REGISTRY.tmp"
mv "$REGISTRY.tmp" "$REGISTRY"
echo "  Updated $REGISTRY (+1 import)"

# Step 3: Update supported-platforms.md
PLATFORMS_DOC="docs/supported-platforms.md"
PLATFORM_ENTRY="* [$PROVIDER_NAME] (\`$PROVIDER_ID\`) - $DESCRIPTION"
REF_ENTRY="[$PROVIDER_NAME]: $DOCS_URL"

# Insert platform entry in alphabetical order among bullet lines
# Find the line after which to insert (last bullet whose name sorts before ours)
INSERT_AFTER=$(grep -n '^\* \[' "$PLATFORMS_DOC" | while IFS=: read -r num line; do
    name=$(echo "$line" | sed -E 's/^\* \[([^]]+)\].*/\1/')
    if [[ "$name" < "$PROVIDER_NAME" ]]; then
        echo "$num"
    fi
done | tail -1)

if [[ -n "$INSERT_AFTER" ]]; then
    sed -i "${INSERT_AFTER}a\\${PLATFORM_ENTRY}" "$PLATFORMS_DOC"
else
    # Insert before the first bullet
    FIRST_BULLET=$(grep -n '^\* \[' "$PLATFORMS_DOC" | head -1 | cut -d: -f1)
    sed -i "${FIRST_BULLET}i\\${PLATFORM_ENTRY}" "$PLATFORMS_DOC"
fi

# Insert URL reference in alphabetical order
# Only consider the first contiguous block of reference lines (stop at blank line)
FIRST_REF=$(grep -n '^\[.*\]: https\?://' "$PLATFORMS_DOC" | head -1 | cut -d: -f1)
LAST_REF=""
if [[ -n "$FIRST_REF" ]]; then
    LAST_REF=$(awk "NR>=$FIRST_REF && /^\[.*\]: https?:\/\//{last=NR} NR>$FIRST_REF && /^$/{print last; exit}" "$PLATFORMS_DOC")
    if [[ -z "$LAST_REF" ]]; then
        LAST_REF=$(grep -n '^\[.*\]: https\?://' "$PLATFORMS_DOC" | tail -1 | cut -d: -f1)
    fi
fi

REF_INSERT_AFTER=$(sed -n "${FIRST_REF},${LAST_REF}p" "$PLATFORMS_DOC" | grep -n '^\[' | while IFS=: read -r num line; do
    name=$(echo "$line" | sed -E 's/^\[([^]]+)\].*/\1/')
    if [[ "$name" < "$PROVIDER_NAME" ]]; then
        echo $((FIRST_REF + num - 1))
    fi
done | tail -1)

if [[ -n "$REF_INSERT_AFTER" ]]; then
    sed -i "${REF_INSERT_AFTER}a\\${REF_ENTRY}" "$PLATFORMS_DOC"
else
    sed -i "${FIRST_REF}i\\${REF_ENTRY}" "$PLATFORMS_DOC"
fi
echo "  Updated $PLATFORMS_DOC (+2 lines)"

# Step 4: Update release notes
RELNOTES="docs/release-notes.md"
FEATURES_LINE=$(grep -n '### Features' "$RELNOTES" | head -1 | cut -d: -f1)

if [[ -z "$FEATURES_LINE" ]]; then
    echo "Error: Could not find '### Features' in $RELNOTES"
    exit 1
fi

# Insert after ### Features and any blank lines
INSERT_LINE=$((FEATURES_LINE + 1))
while [[ "$(sed -n "${INSERT_LINE}p" "$RELNOTES")" =~ ^[[:space:]]*$ ]]; do
    INSERT_LINE=$((INSERT_LINE + 1))
done

sed -i "${INSERT_LINE}i\\- Support ${PROVIDER_NAME}" "$RELNOTES"
echo "  Updated $RELNOTES (+1 line)"

# Step 5: Run validation
echo ""
echo "Running ./test..."
if ./test; then
    echo "  Validation passed"
else
    echo "  WARNING: ./test failed"
    exit 1
fi

# Step 6: Stage and commit
git add "internal/providers/$PROVIDER_ID/$PROVIDER_ID.go" \
        "$REGISTRY" \
        "$PLATFORMS_DOC" \
        "$RELNOTES"

git commit -m "$(cat <<EOF
providers/$PROVIDER_ID: add support for $PROVIDER_NAME

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"

echo ""
echo "Platform support added successfully!"
echo ""
echo "  Platform: $PROVIDER_NAME ($PROVIDER_ID)"
echo "  Metadata URL: $METADATA_URL"
echo ""
echo "  Files created/modified:"
echo "    internal/providers/$PROVIDER_ID/$PROVIDER_ID.go (new)"
echo "    internal/register/providers.go (+1 import)"
echo "    docs/supported-platforms.md (+2 lines)"
echo "    docs/release-notes.md (+1 line)"
echo ""
echo "  Next steps:"
echo "    1. Review changes: git show"
echo "    2. Push to remote when ready"
echo "    3. Create a PR to main branch"
