---
name: release_note_update
description: Finalize a release from release-note fragments without committing.
---

<!-- Maintained in https://github.com/coreos/repo-templates -->
<!-- Do not edit downstream. -->

# Release Note Update

This skill finalizes release notes from validated fragments by running the
repository's release-note tool.

## Usage

When invoked, gather the following from the user if not provided as arguments:

- **Version** (required): release version in X.Y.Z format, e.g. `2.27.0`
- **Date** (optional): release date in YYYY-MM-DD format, defaults to today

Then execute:

```bash
.opencode/skills/release_note_update/run.sh <version> [<date>]
```

The script validates and renders `docs/release-notes.d` fragments, creates the
versioned section, updates the Upcoming heading to the next minor version, and
deletes consumed fragments. It does not create a commit. Review and commit the
result separately.
