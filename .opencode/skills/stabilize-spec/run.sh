#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo "Usage: $0 MINOR_VERSION"
    echo ""
    echo "Stabilize an Ignition config spec version."
    echo "Creates 8 atomic commits following the established pattern."
    echo ""
    echo "Arguments:"
    echo "  MINOR_VERSION   Minor version to stabilize (e.g. 6 for v3_6_experimental -> v3_6)"
    echo ""
    echo "Example:"
    echo "  $0 7    # stabilizes v3_7_experimental -> v3_7, creates v3_8_experimental"
    exit 1
}

if [[ $# -lt 1 ]] || [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]]; then
    usage
fi

MAJOR=3
MINOR="$1"
NEXT_MINOR=$((MINOR + 1))
PREV_MINOR=$((MINOR - 1))

EXP_PKG="v${MAJOR}_${MINOR}_experimental"
STABLE_PKG="v${MAJOR}_${MINOR}"
NEXT_EXP_PKG="v${MAJOR}_${NEXT_MINOR}_experimental"
PREV_STABLE_PKG="v${MAJOR}_${PREV_MINOR}"

EXP_VERSION="${MAJOR}.${MINOR}.0-experimental"
STABLE_VERSION="${MAJOR}.${MINOR}.0"
NEXT_EXP_VERSION="${MAJOR}.${NEXT_MINOR}.0-experimental"

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$REPO_ROOT"

if [[ ! -d "config/$EXP_PKG" ]]; then
    echo "Error: config/$EXP_PKG does not exist"
    exit 1
fi

if [[ -d "config/$STABLE_PKG" ]]; then
    echo "Error: config/$STABLE_PKG already exists"
    exit 1
fi

echo "Stabilizing config spec:"
echo "  $EXP_VERSION -> $STABLE_VERSION"
echo "  New experimental: $NEXT_EXP_VERSION"
echo ""

# Helper: replace string in all Go files under a directory
replace_in_dir() {
    local dir="$1" old="$2" new="$3"
    find "$dir" -name '*.go' -type f -exec sed -i "s|${old}|${new}|g" {} +
}

# Helper: replace string in a single file
replace_in_file() {
    local file="$1" old="$2" new="$3"
    sed -i "s|${old}|${new}|g" "$file"
}

# ── Commit 1: Rename experimental package to stable ──
echo "Commit 1: Renaming $EXP_PKG -> $STABLE_PKG..."
git mv "config/$EXP_PKG" "config/$STABLE_PKG"
replace_in_dir "config/$STABLE_PKG" "$EXP_PKG" "$STABLE_PKG"

git add -A
git commit -m "config: rename $EXP_PKG to $STABLE_PKG"

# ── Commit 2: Stabilize the spec ──
echo "Commit 2: Stabilizing config/$STABLE_PKG..."

# Remove PreRelease field from MaxVersion in types/schema.go
if [[ -f "config/$STABLE_PKG/types/schema.go" ]]; then
    sed -i '/PreRelease:.*"experimental"/d' "config/$STABLE_PKG/types/schema.go"
fi

# Update version strings
replace_in_dir "config/$STABLE_PKG" "$EXP_VERSION" "$STABLE_VERSION"

# Update Accept header in internal/resource/url.go
if [[ -f "internal/resource/url.go" ]]; then
    replace_in_file "internal/resource/url.go" "$EXP_VERSION" "$STABLE_VERSION"
fi

git add -A
git commit -m "config/$STABLE_PKG: stabilize"

# ── Commit 3: Copy stable to new experimental ──
echo "Commit 3: Copying $STABLE_PKG -> $NEXT_EXP_PKG..."
cp -r "config/$STABLE_PKG" "config/$NEXT_EXP_PKG"
replace_in_dir "config/$NEXT_EXP_PKG" "$STABLE_PKG" "$NEXT_EXP_PKG"
replace_in_dir "config/$NEXT_EXP_PKG" "$STABLE_VERSION" "$NEXT_EXP_VERSION"

git add -A
git commit -m "config: copy $STABLE_PKG to $NEXT_EXP_PKG"

# ── Commit 4: Adapt new experimental spec ──
echo "Commit 4: Adapting $NEXT_EXP_PKG for experimental..."

# Add PreRelease field back to MaxVersion
if [[ -f "config/$NEXT_EXP_PKG/types/schema.go" ]]; then
    sed -i "/Minor:.*semver\.Minor/a\\\\t\\tPreRelease: \"experimental\"," \
        "config/$NEXT_EXP_PKG/types/schema.go"
fi

# Update prev import in config.go to point to stable
if [[ -f "config/$NEXT_EXP_PKG/config.go" ]]; then
    replace_in_file "config/$NEXT_EXP_PKG/config.go" \
        "config/$PREV_STABLE_PKG" "config/$STABLE_PKG"
fi

# Update translate.go old version import to stable
if [[ -f "config/$NEXT_EXP_PKG/translate.go" ]]; then
    replace_in_file "config/$NEXT_EXP_PKG/translate.go" \
        "config/$PREV_STABLE_PKG/types" "config/$STABLE_PKG/types"
fi

# Update translate_test.go old import
if [[ -f "config/$NEXT_EXP_PKG/translate_test.go" ]]; then
    replace_in_file "config/$NEXT_EXP_PKG/translate_test.go" \
        "config/$PREV_STABLE_PKG/types" "config/$STABLE_PKG/types"
fi

git add -A
git commit -m "config/$NEXT_EXP_PKG: adapt for new experimental spec"

# ── Commit 5: Update all imports across the codebase ──
echo "Commit 5: Updating imports across codebase..."

# Update all Go files that reference the old stable package to use new experimental
# Skip: config/$STABLE_PKG, config/$NEXT_EXP_PKG, tests/ (commit 6), internal/doc/ (commit 7)
UPDATED=0
while IFS= read -r file; do
    case "$file" in
        config/$STABLE_PKG/*|config/$NEXT_EXP_PKG/*|tests/*|internal/doc/*) continue ;;
    esac
    if grep -q "config/$STABLE_PKG" "$file" 2>/dev/null; then
        replace_in_file "$file" "config/$STABLE_PKG" "config/$NEXT_EXP_PKG"
        UPDATED=$((UPDATED + 1))
    fi
done < <(find . -name '*.go' -type f | sed 's|^\./||')

echo "  Updated $UPDATED files"

git add -A
git commit -m "*: update to $NEXT_EXP_PKG spec"

# ── Commit 6: Update blackbox tests ──
echo "Commit 6: Updating blackbox tests..."

if [[ -d "tests" ]]; then
    # Update imports from stable to new experimental
    find tests -name '*.go' -type f -exec \
        sed -i "s|config/$STABLE_PKG|config/$NEXT_EXP_PKG|g" {} +

    # Update version strings in test files
    find tests -name '*.go' -type f -exec \
        sed -i "s|$EXP_VERSION|$STABLE_VERSION|g" {} +

    # Update Accept header checks
    if [[ -f "tests/servers/servers.go" ]]; then
        replace_in_file "tests/servers/servers.go" "$EXP_VERSION" "$STABLE_VERSION"
    fi
fi

# Add stable types import to tests/register/register.go
if [[ -f "tests/register/register.go" ]]; then
    STABLE_TYPES_IMPORT="\\t\"github.com/coreos/ignition/v2/config/$STABLE_PKG/types\""
    if ! grep -q "config/$STABLE_PKG/types" "tests/register/register.go"; then
        sed -i "/config\/$NEXT_EXP_PKG\/types/i\\${STABLE_TYPES_IMPORT}" \
            "tests/register/register.go"
    fi
fi

git add -A
git diff --cached --quiet && echo "  (no changes)" || git commit -m "tests: update for new experimental spec"

# ── Commit 7: Update doc generation ──
echo "Commit 7: Updating doc generation..."

if [[ -f "internal/doc/main.go" ]]; then
    replace_in_file "internal/doc/main.go" "config/$STABLE_PKG" "config/$NEXT_EXP_PKG"
fi

if [[ -f "generate" ]]; then
    # The generate script lists version dirs like "v3_6 v3_7_experimental"
    # After stabilization, v3_7_experimental was renamed to v3_7 (commit 1),
    # so the script now has the stale name. We need to:
    # 1. Add the new stable version to the list
    # 2. Replace the old experimental name with the new one
    # Order matters: replace the full EXP_PKG first to avoid partial matches
    replace_in_file "generate" "$EXP_PKG" "$STABLE_PKG $NEXT_EXP_PKG"
fi

git add -A
git diff --cached --quiet && echo "  (no changes)" || git commit -m "docs: shuffle for spec stabilization"

# ── Commit 8: Update docs and regenerate ──
echo "Commit 8: Updating docs and regenerating..."

# Update specs.md — add new stable version to list
if [[ -f "docs/specs.md" ]]; then
    STABLE_ENTRY="- [v${STABLE_VERSION}](configuration-v${STABLE_PKG}.md)"
    # Insert before the first existing stable version entry
    FIRST_STABLE_LINE=$(grep -n '^\- \[v[0-9]' "docs/specs.md" | head -1 | cut -d: -f1)
    if [[ -n "$FIRST_STABLE_LINE" ]]; then
        sed -i "${FIRST_STABLE_LINE}i\\${STABLE_ENTRY}" "docs/specs.md"
    fi

    # Update experimental version references
    replace_in_file "docs/specs.md" "$STABLE_VERSION-experimental" "$NEXT_EXP_VERSION"
    replace_in_file "docs/specs.md" "configuration-v${STABLE_PKG}_experimental" \
        "configuration-v${NEXT_EXP_PKG}"
fi

# Update migrating-configs.md — add new migration section
if [[ -f "docs/migrating-configs.md" ]]; then
    # Insert after the TOC marker
    TOC_LINE=$(grep -n '{:toc}' "docs/migrating-configs.md" | head -1 | cut -d: -f1)
    if [[ -n "$TOC_LINE" ]]; then
        sed -i "${TOC_LINE}a\\\\n## From Version ${STABLE_VERSION} to ${NEXT_EXP_VERSION}\\n\\nThere are no breaking changes between versions ${STABLE_VERSION} and ${NEXT_EXP_VERSION}." \
            "docs/migrating-configs.md"
    fi
fi

# Update release-notes.md
if [[ -f "docs/release-notes.md" ]]; then
    FEATURES_LINE=$(grep -n '### Features' "docs/release-notes.md" | head -1 | cut -d: -f1)
    if [[ -n "$FEATURES_LINE" ]]; then
        sed -i "$((FEATURES_LINE))a\\\\n- Mark the ${STABLE_VERSION} config spec as stable\\n- No longer accept configs with version ${EXP_VERSION}\\n- Create new ${NEXT_EXP_VERSION} config spec from ${STABLE_VERSION}" \
            "docs/release-notes.md"
    fi
fi

# Run generate script
echo "  Running ./generate..."
if [[ -x "generate" ]]; then
    if ! ./generate; then
        echo "  WARNING: ./generate failed"
    fi
fi

git add -A
git commit -m "docs: update for spec stabilization"

echo ""
echo "Stabilization complete!"
echo ""
echo "Commits created:"
git log --oneline -8
echo ""

# Run build and test
echo "Running ./build..."
if ./build; then
    echo "  Build: passed"
else
    echo "  WARNING: Build failed"
fi

echo ""
echo "Running ./test..."
if ./test; then
    echo "  Tests: passed"
else
    echo "  WARNING: Tests failed"
fi

echo ""
echo "Next steps:"
echo "  1. Review commits: git log --oneline -8"
echo "  2. Create PR to main branch"
echo "  3. After merge, handle external tests and packages"
