---
name: release_note_update:stage Notes
description: Automate release notes update and offer tagging.
---

# Ignition Release

This skill automates the Ignition release process, handling version bumps and release notes updates following the established conventions from [PR #2181](https://github.com/coreos/ignition/pull/2181).

## What it does

Performs a complete release preparation by:

1. Moving unreleased items from the "Unreleased" section to a new version section in `docs/release-notes.md`
2. Formatting the release notes with the proper date and version
3. Converting release note items to imperative mood (e.g., "Fixed" → "Fix")
4. Creating a properly formatted commit following the naming convention
5. Optionally creating a GitHub release and tag

## Release Types

The skill supports different types of releases:

- **Patch release** (e.g., 2.25.0 → 2.25.1): Bug fixes only
- **Minor release** (e.g., 2.25.0 → 2.26.0): New features and changes
- **Major release** (e.g., 2.25.0 → 3.0.0): Breaking changes

## Usage

When invoked, the skill will:

1. Analyze the current "Unreleased" section in `docs/release-notes.md`
2. Ask you to confirm the new version number (or you can provide it upfront)
3. Ask you to confirm the release date (defaults to today)
4. Move all unreleased items to a new version section
5. Format items in imperative mood
6. Create a commit with the format: `docs/release-notes: update for <version>`

### Example invocation

```
/release 2.26.0
```

or simply:

```
/release
```

The skill will detect the appropriate next version based on the unreleased items.

## Release Notes Format

The skill follows this structure:

```markdown
## Upcoming Ignition <NEXT_VERSION> (unreleased)

### Breaking changes

### Features

### Changes

### Bug fixes

## Ignition <VERSION> (<DATE>)

### Breaking changes
- Fix/Add/Update <description>

### Features
- Add <description>

### Changes
- Update <description>

### Bug fixes
- Fix <description>
```

### Formatting Rules

1. **Version header format**: `## Ignition <VERSION> (<YYYY-MM-DD>)`
2. **Date format**: ISO format (YYYY-MM-DD)
3. **Item format**: Imperative mood
   - Use "Fix" not "Fixed"
   - Use "Add" not "Added"
   - Use "Update" not "Updated"
4. **Sections**: Include only non-empty sections (Breaking changes, Features, Changes, Bug fixes)
5. **Empty sections**: The "Unreleased" section should have empty subsections after release

## Commit Format

The skill creates a commit with this exact format:

```
docs/release-notes: update for <version>
```

Example: `docs/release-notes: update for 2.26.0`

## What the skill does step-by-step

1. ✅ Read current `docs/release-notes.md`
2. ✅ Parse the "Unreleased" section
3. ✅ Determine the new version (provided or auto-detect)
4. ✅ Get the release date (provided or use today)
5. ✅ Convert all items to imperative mood
6. ✅ Create new version section with formatted date
7. ✅ Move all items from "Unreleased" to the new version section
8. ✅ Leave "Unreleased" section with empty subsections
9. ✅ Update `docs/release-notes.md`
10. ✅ Create commit with proper naming convention
11. ✅ Provide summary of changes

## Example Output

After running the skill, you'll see:

```
✨ Release preparation complete!

Version: 2.26.0
Date: 2026-02-17

Release notes updated:
  - 1 breaking change
  - 2 features
  - 0 changes
  - 1 bug fix

Created commit:
  docs/release-notes: update for 2.26.0

Next steps:
1. Review the changes: git show
2. Push the commit if everything looks good
3. Create a GitHub release/tag if needed
```

## Optional: Creating GitHub Release

The skill can optionally:

1. Create a git tag for the version
2. Create a GitHub release with the release notes

To enable this, confirm when prompted or pass the `--create-release` flag.

## Reference

- [PR #2181 - Example Release (2.25.1)](https://github.com/coreos/ignition/pull/2181)
- [Release Notes Documentation](https://github.com/coreos/ignition/blob/main/docs/release-notes.md)

## Checklist Coverage

This skill automates:

- ✅ Moving unreleased items to versioned section
- ✅ Formatting version header with date
- ✅ Converting items to imperative mood
- ✅ Creating properly named commit
- ✅ Optionally creating git tag and GitHub release

## What's NOT covered

The following tasks are NOT automated and must be done manually:

- Updating version numbers in code files (e.g., `version.go`, `package.json`)
- Running tests before release
- Building and publishing release artifacts
- Updating external documentation
- Announcing the release
