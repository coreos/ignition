---
name: remove-feature
description: Remove unsupported sugar features from stabilized OpenShift spec versions
---

# Remove Feature from Stabilized Spec

## What it does

Automates the removal of sugar features from stabilized OpenShift spec versions:

1. Validates the target spec directory and feature existence
2. Removes the feature's translation function call from `translate.go`
3. Removes the translation function definition from `translate.go`
4. Removes related test cases from `translate_test.go`
5. Bumps the `max` version in `internal/doc/butane.yaml` for doc transforms
6. Runs `./generate` to regenerate spec documentation
7. Runs `./test` to validate all changes

## Prerequisites

- Go toolchain installed
- Target spec version is stabilized (directory does NOT end with `_exp`)
- Feature exists in the target spec's `translate.go`
- Feature has doc transform entries in `internal/doc/butane.yaml`

## Usage

```bash
# Interactive mode - will ask for details
/remove-feature

# Specify the version and feature
/remove-feature --distro openshift --version v4_22 --feature grub

# With tracking reference
/remove-feature --distro openshift --version v4_22 --feature grub --ref MCO-630
```

## Workflow

### Step 1: Gather Requirements

If not provided via arguments, ask the user:

1. **Distro**: Which distro variant? (typically `openshift`)
2. **Version**: Which stabilized version? (e.g., `v4_22`)
3. **Feature**: Which feature to remove? (e.g., `grub`)
4. **Reference**: Optional tracking issue (e.g., `MCO-630`, `#515`)

### Step 2: Pre-flight Validation

Run these checks:

```bash
# Verify target directory exists and is NOT experimental
ls -la config/{distro}/{version}/

# Verify version does NOT end with _exp
# CRITICAL: Only remove features from stabilized specs

# Check git status
git status --porcelain
```

**Stop if**:
- Target directory does not exist
- Version ends with `_exp` (experimental specs should be modified differently)
- Working directory has unexpected uncommitted changes

### Step 3: Identify Feature Code

Read the target files to locate the feature:

```bash
# Read translate.go to find feature function
cat config/{distro}/{version}/translate.go

# Read translate_test.go to find test cases
cat config/{distro}/{version}/translate_test.go

# Read butane.yaml to find doc transform entries
cat internal/doc/butane.yaml
```

**Identify**:
1. The function call in the main translation pipeline (e.g., `ts = translateUserGrubCfg(&cfg, &ts)`)
2. The function definition (e.g., `func translateUserGrubCfg(...)`)
3. The test case block (e.g., `// Test Grub config` test struct)
4. The butane.yaml transform entries with `replacement: "Unsupported"` and `max` version

**Stop if** the feature code is not found in `translate.go` - it may have already been removed or never existed in this version.

### Step 4: Remove Translation Function Call

**File**: `config/{distro}/{version}/translate.go`

Read the file and find the function call within the main `ToMachineConfig{Version}Unvalidated()` function.

Remove the line calling the feature's translation function. For example:

```go
// REMOVE THIS LINE:
ts = translateUserGrubCfg(&cfg, &ts)
```

Use the Edit tool:
```
oldString: "\tts = translateUserGrubCfg(&cfg, &ts)\n"
newString: ""
```

### Step 5: Remove Translation Function Definition

**File**: `config/{distro}/{version}/translate.go`

Remove the entire function definition. The function is typically at the end of the file.

For the GRUB removal pattern, remove the entire `translateUserGrubCfg` function:

```go
// REMOVE THIS ENTIRE BLOCK:

// fcos config generates a user.cfg file using append; however, OpenShift config
// does not support append (since MCO does not support it). Let change the file to use contents
func translateUserGrubCfg(config *types.Config, ts *translate.TranslationSet) translate.TranslationSet {
    // ... function body ...
}
```

Use the Edit tool to remove from the comment above the function through the closing brace.

### Step 6: Clean Up Dead Imports

After removing the function, check if any imports are now unused. Common imports that may become dead:

- For GRUB removal: no imports typically become dead (the remaining code uses the same packages)

If imports are dead, remove them. Run `./test` later to catch any remaining issues.

### Step 7: Remove Test Cases

**File**: `config/{distro}/{version}/translate_test.go`

Read the file and identify the test case block for the removed feature. Test cases are typically marked with a comment like `// Test Grub config` and consist of a struct literal in the test table.

Remove the entire test case struct. For GRUB removal, this includes:
- The comment marker (e.g., `// Test Grub config`)
- The Config input struct
- The expected result.MachineConfig output struct  
- The expected translate.Translation slice

Use the Edit tool to remove the entire block. Be careful to:
- Include the leading comment
- Include all three struct elements (input, expected output, expected translations)
- Preserve the closing of the test table (`}` and `for` loop)

### Step 8: Update Doc Descriptors

**File**: `internal/doc/butane.yaml`

Find the doc transform entries for the removed feature. These have the pattern:

```yaml
transforms:
  - regex: ".*"
    replacement: "Unsupported"
    if:
      - variant: openshift
        max: {PREVIOUS_VERSION}
```

**Determine the new max version**:
- From the directory version `v4_22`, derive `4.22.0`
- The current `max` should be the previous stabilized version (e.g., `4.21.0`)
- Bump `max` to the new version: `4.22.0`

**Update ALL occurrences** for the feature. For GRUB, there are 4 entries:
1. `grub` itself
2. `grub.users`
3. `grub.users.name`
4. `grub.users.password_hash`

Use the Edit tool for each occurrence:
```
oldString: "max: 4.21.0"
newString: "max: 4.22.0"
```

**IMPORTANT**: Only update `max` values within the feature's section. Verify the surrounding context (field names) to avoid changing unrelated transforms.

Alternatively, if all occurrences have the same old value, use `replaceAll` with enough context to scope the changes correctly.

### Step 9: Regenerate Documentation

Run the documentation generator:

```bash
./generate
```

**Expected outcome**: The `docs/config-openshift-{version}.md` file is updated, with the removed feature's fields now showing "Unsupported" instead of their original descriptions.

Verify the change:

```bash
git diff docs/config-openshift-{version}.md
```

The diff should show lines like:
```
-* **_grub_** (object): describes the desired GRUB bootloader configuration.
+* **_grub_** (object): Unsupported
```

If `./generate` fails, check the butane.yaml changes for syntax errors.

### Step 10: Run Tests

```bash
./test
```

**Expected outcome**: All tests pass.

If tests fail:
1. Check for compilation errors (dead imports, missing functions)
2. Check for test expectation mismatches
3. Fix and re-run

### Step 11: Report Results

Provide a comprehensive summary:

```
Feature "{feature}" removed from {distro}/{version}

Files Modified:
  - config/{distro}/{version}/translate.go (-N lines)
  - config/{distro}/{version}/translate_test.go (-N lines)
  - internal/doc/butane.yaml (N version bumps)
  - docs/config-{distro}-{version}.md (regenerated, N fields -> "Unsupported")

Tests: PASSED
Docs: REGENERATED

Suggested commit message:

  {distro}/{version}: Remove {feature_description}

  {reason for removal, e.g., "Support is still missing in the MCO."}

  See: {reference_link}
  See: #{github_issue}
```

## Checklist Coverage

This skill automates the following steps:

- ✅ Remove feature translation function call from `translate.go`
- ✅ Remove feature translation function definition from `translate.go`
- ✅ Remove related test cases from `translate_test.go`
- ✅ Bump `max` version in `internal/doc/butane.yaml` for all feature fields
- ✅ Regenerate documentation via `./generate`
- ✅ Validate changes via `./test`

## What's NOT Covered

- ❌ **Determining which features to remove** - requires knowledge of MCO support status
- ❌ **Removing features from experimental specs** - experimental specs should use different approaches
- ❌ **Removing schema definitions** - this skill only removes translation/test code; schemas are inherited from parent and remain (they just become dead code for that version)
- ❌ **Removing validation logic** - if the feature has validation in `validate.go`, that must be handled separately
- ❌ **Creating git commits** - user should review changes first
- ❌ **Updating release notes** - user should note the removal if appropriate

## Example Output

```
/remove-feature --distro openshift --version v4_22 --feature grub --ref MCO-630

Validating prerequisites...
✅ Target directory exists: config/openshift/v4_22
✅ Version is stabilized (not experimental)
✅ Git working directory is clean

Identifying feature code...
✅ Found function call: ts = translateUserGrubCfg(&cfg, &ts)
✅ Found function definition: translateUserGrubCfg (21 lines)
✅ Found test case: // Test Grub config (83 lines)
✅ Found 4 butane.yaml transform entries (max: 4.21.0)

Phase 1: Remove translation code
  ✅ Removed function call from translate.go
  ✅ Removed function definition from translate.go (-22 lines)

Phase 2: Remove test cases
  ✅ Removed test case from translate_test.go (-83 lines)

Phase 3: Update doc descriptors
  ✅ Bumped max: 4.21.0 → 4.22.0 for grub (4 entries)

Phase 4: Regenerate docs
  ✅ ./generate completed
  ✅ docs/config-openshift-v4_22.md updated (4 fields → "Unsupported")

Phase 5: Validate
  ✅ ./test passed

Feature "grub" removed from openshift/v4_22

Suggested commit message:

  openshift/v4_22: Remove GRUB config support

  See: https://issues.redhat.com/browse/MCO-630
  See: #515
```

## References

- Design document: `.opencode/skills/remove-feature/DESIGN.md`
- Examples: `.opencode/skills/remove-feature/examples/`
- Example commits: `4a2be91`, `2d9a25e`, `aa6ad0b`, `9821f9b`, `cd75f80`, `1f65fb6`
- Current experimental spec with GRUB code: `config/openshift/v4_22_exp/translate.go:264-285`
- Doc descriptors: `internal/doc/butane.yaml:402-438`
