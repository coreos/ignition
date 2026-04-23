#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo "Usage: $0 VERSION [DATE]"
    echo ""
    echo "Move unreleased items in docs/release-notes.md to a new version section."
    echo ""
    echo "Arguments:"
    echo "  VERSION   Release version (e.g. 2.26.0)"
    echo "  DATE      Release date in YYYY-MM-DD format (default: today)"
    exit 1
}

if [[ $# -lt 1 ]]; then
    usage
fi

VERSION="$1"
DATE="${2:-$(date +%Y-%m-%d)}"

if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: VERSION must be in X.Y.Z format, got: $VERSION"
    exit 1
fi

if ! [[ "$DATE" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
    echo "Error: DATE must be in YYYY-MM-DD format, got: $DATE"
    exit 1
fi

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$REPO_ROOT"

RELNOTES="docs/release-notes.md"

if [[ ! -f "$RELNOTES" ]]; then
    echo "Error: $RELNOTES not found"
    exit 1
fi

# Find the unreleased section
UNRELEASED_LINE=$(grep -n '(unreleased)' "$RELNOTES" | head -1 | cut -d: -f1)
if [[ -z "$UNRELEASED_LINE" ]]; then
    echo "Error: Could not find an '(unreleased)' section in $RELNOTES"
    exit 1
fi

# Find the next release section (next "## " header after unreleased)
NEXT_RELEASE_LINE=$(awk "NR>$UNRELEASED_LINE && /^## /{print NR; exit}" "$RELNOTES")
if [[ -z "$NEXT_RELEASE_LINE" ]]; then
    NEXT_RELEASE_LINE=$(wc -l < "$RELNOTES")
fi

# Extract the unreleased content between the two headers
UNRELEASED_CONTENT=$(sed -n "$((UNRELEASED_LINE + 1)),$((NEXT_RELEASE_LINE - 1))p" "$RELNOTES")

# Parse sections and their items
declare -A SECTION_ITEMS
CURRENT_SECTION=""
SECTIONS_ORDER=()

while IFS= read -r line; do
    if [[ "$line" =~ ^###[[:space:]]+(.*) ]]; then
        CURRENT_SECTION="${BASH_REMATCH[1]}"
        SECTIONS_ORDER+=("$CURRENT_SECTION")
        SECTION_ITEMS["$CURRENT_SECTION"]=""
    elif [[ -n "$CURRENT_SECTION" && "$line" =~ ^-[[:space:]] ]]; then
        if [[ -n "${SECTION_ITEMS[$CURRENT_SECTION]}" ]]; then
            SECTION_ITEMS["$CURRENT_SECTION"]+=$'\n'
        fi
        SECTION_ITEMS["$CURRENT_SECTION"]+="$line"
    fi
done <<< "$UNRELEASED_CONTENT"

# Convert items to imperative mood
to_imperative() {
    local line="$1"
    # Strip leading "- " to get the first word
    local rest="${line#- }"
    local first_word="${rest%% *}"
    local remainder="${rest#* }"
    local lower_word=${first_word,,}

    local imperative=""
    case "$lower_word" in
        added|adding|adds)       imperative="Add" ;;
        fixed|fixing|fixes)      imperative="Fix" ;;
        updated|updating|updates) imperative="Update" ;;
        removed|removing|removes) imperative="Remove" ;;
        changed|changing|changes) imperative="Change" ;;
        created|creating|creates) imperative="Create" ;;
        supported|supporting|supports) imperative="Support" ;;
        included|including|includes) imperative="Include" ;;
        dropped|dropping|drops)  imperative="Drop" ;;
        marked|marking|marks)    imperative="Mark" ;;
        bumped|bumping|bumps)    imperative="Bump" ;;
        moved|moving|moves)      imperative="Move" ;;
        renamed|renaming|renames) imperative="Rename" ;;
        stabilized|stabilizing|stabilizes) imperative="Stabilize" ;;
        implemented|implementing|implements) imperative="Implement" ;;
        enabled|enabling|enables) imperative="Enable" ;;
        disabled|disabling|disables) imperative="Disable" ;;
        allowed|allowing|allows) imperative="Allow" ;;
        prevented|preventing|prevents) imperative="Prevent" ;;
        deprecated|deprecating|deprecates) imperative="Deprecate" ;;
        replaced|replacing|replaces) imperative="Replace" ;;
        migrated|migrating|migrates) imperative="Migrate" ;;
        resolved|resolving|resolves) imperative="Resolve" ;;
        improved|improving|improves) imperative="Improve" ;;
        refactored|refactoring|refactors) imperative="Refactor" ;;
        *)
            # Already imperative or unknown — leave as-is
            echo "$line"
            return
            ;;
    esac

    if [[ "$rest" == "$first_word" ]]; then
        echo "- $imperative"
    else
        echo "- $imperative $remainder"
    fi
}

# Calculate next unreleased version (bump minor)
MAJOR=$(echo "$VERSION" | cut -d. -f1)
MINOR=$(echo "$VERSION" | cut -d. -f2)
NEXT_MINOR=$((MINOR + 1))
NEXT_VERSION="${MAJOR}.${NEXT_MINOR}.0"

# Build the new file content
{
    # Everything before the unreleased header
    head -n "$((UNRELEASED_LINE - 1))" "$RELNOTES"

    # New unreleased section (empty)
    echo "## Upcoming Ignition $NEXT_VERSION (unreleased)"
    echo ""
    echo "### Breaking changes"
    echo ""
    echo "### Features"
    echo ""
    echo "### Changes"
    echo ""
    echo "### Bug fixes"
    echo ""
    echo ""

    # New release section
    echo "## Ignition $VERSION ($DATE)"

    # Only include sections that have items
    for section in "${SECTIONS_ORDER[@]}"; do
        items="${SECTION_ITEMS[$section]}"
        if [[ -n "$items" ]]; then
            echo ""
            echo "### $section"
            echo ""
            while IFS= read -r item; do
                to_imperative "$item"
            done <<< "$items"
        fi
    done
    echo ""

    # Everything from the next release onward
    tail -n +"$NEXT_RELEASE_LINE" "$RELNOTES"
} > "$RELNOTES.tmp"
mv "$RELNOTES.tmp" "$RELNOTES"

# Count items
TOTAL_ITEMS=0
echo "Release notes updated for $VERSION ($DATE)"
echo ""
for section in "${SECTIONS_ORDER[@]}"; do
    items="${SECTION_ITEMS[$section]}"
    if [[ -n "$items" ]]; then
        count=$(echo "$items" | wc -l)
        TOTAL_ITEMS=$((TOTAL_ITEMS + count))
        echo "  $count ${section,,}"
    fi
done
echo ""

if [[ $TOTAL_ITEMS -eq 0 ]]; then
    echo "  (no items in unreleased section)"
fi

echo "New unreleased section created for $NEXT_VERSION"
echo ""

# Commit
git add "$RELNOTES"
git commit -m "docs/release-notes: update for $VERSION"

echo "Committed: docs/release-notes: update for $VERSION"
echo ""
echo "Next steps:"
echo "  1. Review changes: git show"
echo "  2. Push to remote when ready"
echo "  3. Create a GitHub release/tag if needed"
