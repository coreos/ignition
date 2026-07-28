---
name: stabilize-spec
description: Stabilize experimental Butane config spec versions
---

# Stabilize Spec Version

## What it does

Automates the complete stabilization workflow for Butane spec versions:

**Phase 1: Stabilize Experimental → Stable**
1. Validating the working directory and checking prerequisites
2. Renaming the experimental directory to stable (e.g., `v1_7_exp` → `v1_7`)
3. Updating all package statements in the renamed directory
4. Updating imports in the stabilized spec (base dependencies, Ignition versions)
5. Updating `config/config.go` registration (for distro specs only)

**Phase 2: Create Next Experimental Version**
6. Copying the newly stabilized version to create the next experimental version (e.g., `v1_7` → `v1_8_exp`)
7. Updating package statements in the new experimental directory
8. Bumping base and Ignition dependencies to experimental versions
9. Registering the new experimental version in `config/config.go`

**Phase 3: Validation & Documentation**
10. Running tests to validate all changes
11. Running `./generate` to update documentation
12. Reporting all changes and suggesting next steps

## Prerequisites

- Clean git working directory (or only expected changes)
- Go toolchain installed
- Experimental spec version exists
- Target stable version doesn't already exist

## Usage

```bash
# Stabilize a base version (creates v0_7 stable + v0_8_exp experimental)
/stabilize-spec --type base --version v0_7_exp

# Stabilize a distro version (creates v1_7 stable + v1_8_exp experimental)
/stabilize-spec --type fcos --version v1_7_exp

# Stabilize a distro version with Ignition downgrade
/stabilize-spec --type openshift --version v4_21_exp --base-version v1_6 --ignition-version v3_5

# Skip creating the next experimental version (not recommended)
/stabilize-spec --type fcos --version v1_7_exp --skip-next-exp
```

## Workflow

### Step 1: Gather Requirements

If not provided via arguments, ask the user:

1. **Spec type**: base, fcos, openshift, flatcar, r4e, or fiot?
2. **Version to stabilize**: Which experimental version? (e.g., `v1_7_exp`, `v4_21_exp`)
3. **For distro specs only**:
   - Which stable base version to depend on? (e.g., `v1_6`, `v0_6`)
   - Does the Ignition version change? If yes, what's the target? (e.g., `v3_5`)
4. **Create next experimental version?** (default: yes, per stabilize-checklist.md)
   - If yes, calculate the next version number (e.g., v1_7 → v1_8_exp)
   - Ask which experimental base to use (e.g., v0_8_exp)
   - Ask which experimental Ignition version to use (e.g., v3_7_experimental)

### Step 2: Pre-flight Validation

Run these checks in parallel:

```bash
# Check git status
git status --porcelain

# Verify experimental directory exists
ls -la base/{version}/ || ls -la config/{distro}/{version}/

# Verify stable directory doesn't exist
ls -la base/{stable_version}/ || ls -la config/{distro}/{stable_version}/

# For distro specs: verify base version exists
ls -la base/{base_version}/ || ls -la config/fcos/{base_version}/
```

**Validation criteria**:
- Git working directory should be clean or only contain expected changes
- Source experimental directory must exist
- Target stable directory must NOT exist
- If distro spec: base dependency must exist

If any check fails, report the error and stop.

### Step 3: Rename Directory

Use `git mv` to rename the experimental directory:

```bash
# For base specs:
git mv base/{version} base/{stable_version}

# For distro specs:
git mv config/{distro}/{version} config/{distro}/{stable_version}
```

**Example**:
```bash
git mv config/fcos/v1_7_exp config/fcos/v1_7
```

### Step 4: Update Package Statements

Find all `.go` files in the renamed directory and update package statements:

```bash
# Find all .go files
find {renamed_directory} -name "*.go"

# For each file, update the package statement
# OLD: package v1_7_exp
# NEW: package v1_7
```

Use the Edit tool to replace:
```
oldString: "package {version}"
newString: "package {stable_version}"
```

**Files typically affected**:
- Base specs: schema.go, translate.go, translate_test.go, util.go, validate.go, validate_test.go (6 files)
- Distro specs: schema.go, translate.go, translate_test.go, validate.go, validate_test.go (5-7 files)

### Step 5: Update Imports (Distro Specs Only)

For distro specs, update base dependency imports:

1. **Identify files that import base**:
   - schema.go
   - translate_test.go
   - validate.go
   - validate_test.go

2. **Update base import**:
```go
// OLD:
import (
    base "github.com/coreos/butane/base/v0_7_exp"
)

// NEW:
import (
    base "github.com/coreos/butane/base/v0_6"
)
```

3. **For OpenShift specs, also update fcos import in schema.go**:
```go
// OLD:
import (
    fcos "github.com/coreos/butane/config/fcos/v1_7_exp"
)

// NEW:
import (
    fcos "github.com/coreos/butane/config/fcos/v1_6"
)
```

### Step 6: Update Ignition Imports (If Version Changes)

If the Ignition version is changing (common for OpenShift stabilizations):

1. **Find files that import Ignition types**:
   - result/schema.go (OpenShift only)
   - translate.go
   - translate_test.go

2. **Update Ignition imports**:
```go
// OLD:
import (
    "github.com/coreos/ignition/v2/config/v3_6_experimental/types"
)

// NEW:
import (
    "github.com/coreos/ignition/v2/config/v3_5/types"
)
```

3. **Rename translation functions in translate.go**:
   - `ToIgn3_6Unvalidated` → `ToIgn3_5Unvalidated`
   - `ToIgn3_6` → `ToIgn3_5`
   - Update function comments
   - Update `cutil.Translate` and `cutil.TranslateBytes` calls

4. **Update test version strings in translate_test.go**:
```go
// OLD:
Version: "3.6.0-experimental",

// NEW:
Version: "3.5.0",
```

### Step 7: Update config/config.go (Distro Specs Only)

For distro specs, update the registration in `config/config.go`:

1. **Update import statement**:
```go
// OLD:
import (
    fcos1_7_exp "github.com/coreos/butane/config/fcos/v1_7_exp"
)

// NEW:
import (
    fcos1_7 "github.com/coreos/butane/config/fcos/v1_7"
)
```

2. **Update RegisterTranslator call in init()**:
```go
// OLD:
RegisterTranslator("fcos", "1.7.0-experimental", fcos1_7_exp.ToIgn3_6Bytes)

// NEW:
RegisterTranslator("fcos", "1.7.0", fcos1_7.ToIgn3_6Bytes)
```

**Pattern**: Remove `-experimental` suffix from version string, update import alias

### Step 8: Create Next Experimental Version

**Note**: This step implements lines 20-21 (base) and 28-30 (distro) from `.github/ISSUE_TEMPLATE/stabilize-checklist.md`.

If `--skip-next-exp` was NOT specified (default behavior):

#### 8a. Calculate Next Version

Determine the next experimental version:
- For base: `v0_7` → `v0_8_exp`
- For fcos: `v1_7` → `v1_8_exp`  
- For openshift: `v4_21` → `v4_22_exp`

Parse the stable version number and increment it.

#### 8b. Copy Stable to New Experimental

Use `cp -r` or recursive copy to duplicate the newly stabilized directory:

```bash
# For base specs:
cp -r base/{stable_version} base/{next_exp_version}

# For distro specs:
cp -r config/{distro}/{stable_version} config/{distro}/{next_exp_version}

# Then add to git:
git add base/{next_exp_version} || git add config/{distro}/{next_exp_version}
```

**Example**:
```bash
cp -r config/fcos/v1_7 config/fcos/v1_8_exp
git add config/fcos/v1_8_exp
```

#### 8c. Update Package Statements in New Experimental

Find all `.go` files in the new experimental directory and update package statements:

```bash
# For each .go file in the new experimental directory:
# OLD: package v1_7
# NEW: package v1_8_exp
```

Use the Edit tool to replace in each file.

#### 8d. Update Base Dependency (Distro Specs Only)

For distro specs, update the base import to use the new experimental base:

```go
// In schema.go, translate_test.go, validate.go, validate_test.go:
// OLD:
import (
    base "github.com/coreos/butane/base/v0_7"
)

// NEW:
import (
    base "github.com/coreos/butane/base/v0_8_exp"
)
```

**For OpenShift**, also update fcos import in schema.go if fcos has a new experimental version.

#### 8e. Update Ignition Imports (If Version Increases)

If the Ignition version is increasing (e.g., v3_6 → v3_7_experimental):

1. **Update Ignition imports in**:
   - result/schema.go (OpenShift only)
   - translate.go
   - translate_test.go

```go
// OLD:
import (
    "github.com/coreos/ignition/v2/config/v3_6/types"
)

// NEW:
import (
    "github.com/coreos/ignition/v2/config/v3_7_experimental/types"
)
```

2. **Rename translation functions in translate.go**:
   - `ToIgn3_6Unvalidated` → `ToIgn3_7Unvalidated`
   - `ToIgn3_6` → `ToIgn3_7`
   - Update function comments
   - Update `cutil.Translate` and `cutil.TranslateBytes` calls

3. **Update test version strings in translate_test.go**:
```go
// OLD:
Version: "3.6.0",

// NEW:
Version: "3.7.0-experimental",
```

#### 8f. Add New Experimental to config/config.go (Distro Specs Only)

For distro specs, register the new experimental version in `config/config.go`:

1. **Add import statement**:
```go
// After the just-stabilized import:
import (
    fcos1_7 "github.com/coreos/butane/config/fcos/v1_7"
    fcos1_8_exp "github.com/coreos/butane/config/fcos/v1_8_exp"  // ADD THIS
)
```

2. **Add RegisterTranslator call in init()**:
```go
// After the just-stabilized registration:
RegisterTranslator("fcos", "1.7.0", fcos1_7.ToIgn3_6Bytes)
RegisterTranslator("fcos", "1.8.0-experimental", fcos1_8_exp.ToIgn3_7Bytes)  // ADD THIS
```

**Pattern**: Add `-experimental` suffix, use experimental Ignition version

### Step 9: Run Tests

Execute the test suite to validate all changes:

```bash
./test
```

**Expected outcome**: All tests should pass.

If tests fail:
- Review the error messages
- Check for missed imports or package statements
- Verify function renames are complete
- Report the failure to the user and suggest manual review

### Step 10: Regenerate Documentation

Run the documentation generator:

```bash
./generate
```

**Expected outcome**: Documentation files in `docs/` are updated.

Check for uncommitted changes:
```bash
git status docs/
```

If `./generate` fails or produces unexpected changes, report to the user.

### Step 11: Report Results

Provide a comprehensive summary:

```
✅ Spec stabilization complete!

## Phase 1: Stabilization
### Directory Renamed:
- {old_path} → {new_path}

### Files Modified:
- {count} files with package statement updates
- {count} files with import updates
- config/config.go updated (distro specs only)

## Phase 2: Next Experimental Version Created
### Directory Created:
- {new_exp_path}

### Files Modified:
- {count} files with package statement updates
- {count} files with import updates (bumped to experimental versions)
- config/config.go updated with new experimental registration

## Validation:
✅ Tests passed (./test)
✅ Documentation regenerated (./generate)

## Git Status:
{output of git status}

## Next Steps (from stabilize-checklist.md):

1. Review the changes with `git diff`
2. Consider creating TWO commits:
   - Commit 1: Stabilization (e.g., "fcos/v1_7_exp: stabilize to v1_7")
   - Commit 2: New experimental (e.g., "fcos: add v1_8_exp")
3. Update docs/upgrading-*.md (requires manual content creation)
4. Note the stabilization in docs/release-notes.md
5. If this is a base stabilization, stabilize the distro versions that depend on it

## Suggested commit messages:

### Commit 1: Stabilization
{distro}/v{X}_exp: stabilize to v{X}

- Rename {distro}/v{X}_exp to {distro}/v{X}
- Update package statements and imports
- Drop -experimental from config registration
{additional details based on what changed}

### Commit 2: New Experimental
{distro}: add v{X+1}_exp

- Copy {distro}/v{X} to {distro}/v{X+1}_exp
- Update package statements to v{X+1}_exp
- Bump base dependency to {base}_exp
- Bump Ignition version to v{Y}_experimental
- Add experimental config registration
```

## Checklist Coverage

This skill automates the following items from `.github/ISSUE_TEMPLATE/stabilize-checklist.md`:

### For Base Stabilization (lines 17-21):
- ✅ Rename `base/vB_exp` to `base/vB` and update `package` statements
- ✅ Update imports
- ✅ **Copy `base/vB` to `base/vB+1_exp`**
- ✅ **Update `package` statements in `base/vB+1_exp`**

### For Distro Stabilization (lines 23-30):
- ✅ Rename `config/distro/vD_exp` to `config/distro/vD` and update `package` statements
- ✅ Update imports
- ✅ Drop `-experimental` from `init()` in `config/config.go`
- ✅ **Copy `config/distro/vD` to `config/distro/vD+1_exp`**
- ✅ **Update `package` statements in `config/distro/vD+1_exp`**
- ✅ **Bump base dependency to `base/vB+1_exp`**
- ✅ **Import `config/vD+1_exp` in `config/config.go` and add experimental registration**

### For Ignition Spec Version Bumps (lines 32-35):
- ✅ Bump Ignition types imports in new experimental version
- ✅ Rename `ToIgnI` functions in new experimental version
- ✅ Bump Ignition spec versions in translate_test.go

### For Documentation (lines 37-40):
- ✅ Run `./generate` to regenerate spec docs

## What's NOT Covered

This skill does NOT automate:

- ❌ **Bumping go.mod for Ignition releases** - done before stabilization (line 14)
- ❌ **Updating vendor directory** - done before stabilization (line 14)
- ❌ **Dropping -experimental from examples in docs/** - requires content analysis (line 27)
- ❌ **Updating `internal/doc/main.go`** - requires manual editing (line 39)
- ❌ **Updating docs/specs.md** - requires manual editing (line 41)
- ❌ **Updating docs/upgrading-*.md** - requires content creation (line 42)
- ❌ **Writing release notes** - requires human judgment (line 43)
- ❌ **Creating git commits** - user should review changes first

These steps require human judgment and should be done manually following the stabilization.

## Example Output

```
/stabilize-spec --type fcos --version v1_7_exp

═══════════════════════════════════════════════
 PHASE 1: Stabilize v1_7_exp → v1_7
═══════════════════════════════════════════════

Validating prerequisites...
✅ Git working directory is clean
✅ Experimental version exists: config/fcos/v1_7_exp
✅ Stable version doesn't exist: config/fcos/v1_7
✅ Base dependency exists: base/v0_7

Renaming directory...
✅ Renamed: config/fcos/v1_7_exp → config/fcos/v1_7

Updating package statements (5 files)...
✅ config/fcos/v1_7/schema.go
✅ config/fcos/v1_7/translate.go
✅ config/fcos/v1_7/translate_test.go
✅ config/fcos/v1_7/validate.go
✅ config/fcos/v1_7/validate_test.go

Updating imports...
✅ Updated base import in 4 files

Updating config/config.go...
✅ Import statement updated: fcos1_7_exp → fcos1_7
✅ Registration updated: removed -experimental suffix

═══════════════════════════════════════════════
 PHASE 2: Create Next Experimental v1_8_exp
═══════════════════════════════════════════════

Calculating next version...
✅ Next version: v1_8_exp
✅ Next experimental base: v0_8_exp
✅ Next Ignition version: v3_7_experimental

Copying stable to experimental...
✅ Copied: config/fcos/v1_7 → config/fcos/v1_8_exp

Updating package statements (5 files)...
✅ config/fcos/v1_8_exp/schema.go
✅ config/fcos/v1_8_exp/translate.go
✅ config/fcos/v1_8_exp/translate_test.go
✅ config/fcos/v1_8_exp/validate.go
✅ config/fcos/v1_8_exp/validate_test.go

Updating imports to experimental versions...
✅ Updated base import: v0_7 → v0_8_exp (4 files)
✅ Updated Ignition import: v3_6 → v3_7_experimental (2 files)

Updating translation functions...
✅ Renamed: ToIgn3_6Unvalidated → ToIgn3_7Unvalidated
✅ Renamed: ToIgn3_6 → ToIgn3_7
✅ Updated: ToIgn3_6Bytes → ToIgn3_7Bytes

Updating config/config.go...
✅ Added import: fcos1_8_exp
✅ Added registration: 1.8.0-experimental

═══════════════════════════════════════════════
 PHASE 3: Validation
═══════════════════════════════════════════════

Running tests...
✅ All tests passed (./test)

Regenerating documentation...
✅ Documentation updated (./generate)

═══════════════════════════════════════════════
 SUMMARY
═══════════════════════════════════════════════

📊 Phase 1 (Stabilization):
  - 1 directory renamed
  - 5 files updated in config/fcos/v1_7/
  - config/config.go updated

📊 Phase 2 (New Experimental):
  - 1 directory created (2874 lines)
  - 5 files updated in config/fcos/v1_8_exp/
  - config/config.go updated

✅ Tests: PASSED
✅ Docs: REGENERATED

🎯 Ready for review and commit!
```

## References

- Design document: `.opencode/skills/stabilize-spec/DESIGN.md`
- Examples: `.opencode/skills/stabilize-spec/examples/`
- Issue template: `.github/ISSUE_TEMPLATE/stabilize-checklist.md`
- Development docs: `docs/development.md`
