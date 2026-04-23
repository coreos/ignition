---
name: release_note_update
description: Automate release notes update and offer tagging.
---

# Release Note Update

This skill prepares release notes by running the codified script.

## Usage

When invoked, gather the following from the user if not provided as arguments:

- **Version** (required): release version in X.Y.Z format, e.g. `2.27.0`
- **Date** (optional): release date in YYYY-MM-DD format, defaults to today

Then execute:

```bash
.opencode/skills/release_note_update/run.sh <version> [<date>]
```

The script handles everything: parsing the unreleased section, converting items to imperative mood, creating the versioned section, updating the unreleased header for the next version, and creating the commit.

See `README.md` in this directory for detailed documentation.
