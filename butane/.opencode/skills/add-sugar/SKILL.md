---
name: add-sugar
description: Add syntactic sugar features to Butane experimental spec versions
---

# Add Sugar Feature

## What it does

Guides and scaffolds the addition of a new syntactic sugar feature to a Butane experimental spec version:

1. Gathers requirements from the user (spec type, field design, translation behavior)
2. Validates prerequisites (experimental spec exists, git status clean)
3. Adds struct definitions to `schema.go`
4. Implements translation (desugaring) logic in `translate.go`
5. Adds test cases in `translate_test.go`
6. Adds validation logic in `validate.go` and tests in `validate_test.go` (if needed)
7. Adds error constants to `config/common/errors.go`
8. Updates documentation descriptors in `internal/doc/butane.yaml`
9. Runs `./generate` to regenerate spec docs
10. Adds examples to `docs/examples.md`
11. Adds a release note to `docs/release-notes.md`
12. Runs `./test` to validate everything compiles and passes

## Prerequisites

- Go toolchain installed
- Target experimental spec version exists (directory ends with `_exp`)
- Understanding of what the sugar should do (what YAML the user writes, what Ignition config it generates)

## Usage

```bash
# Interactive mode - will ask for details
/add-sugar

# Target a specific spec
/add-sugar --spec fcos/v1_8_exp --field boot_device.luks.method

# Base spec sugar (distro-independent)
/add-sugar --spec base/v0_8_exp --field storage.files.parent
```

## Workflow

### Step 1: Gather Requirements

If not provided via arguments, ask the user:

1. **Target spec**: Where should the sugar live?
   - **Base spec** (`base/v0_8_exp`): distro-independent, will appear in all variants
   - **Distro spec** (e.g., `config/fcos/v1_8_exp`, `config/openshift/v4_22_exp`): distro-specific

2. **Feature description**: What does the sugar do?
   - What YAML fields does the user write?
   - What Ignition config does it expand to?
   - Any validation constraints?

3. **Schema design**: Ask the user to describe or confirm:
   - Field name(s) and types
   - Whether it's a new top-level field or nested within an existing struct
   - Any new struct types needed

4. **Translation approach**: Per `docs/development.md:62`:
   - **Config merging** (recommended): Generate a fresh Ignition config struct, then use `baseutil.MergeTranslatedConfigs()` to merge with the user's config. The desugared struct is the merge parent, user config is child.
   - **Direct modification**: Only if config merging is not expressive enough.

5. **Validation needs**: What input constraints exist?
   - Required fields
   - Valid value ranges
   - Mutually exclusive options

### Step 2: Pre-flight Validation

Run these checks:

```bash
# Verify experimental spec directory exists
ls -la base/{version}/ || ls -la config/{distro}/{version}/

# Check that the version ends with _exp
# CRITICAL: Sugar must ONLY be added to experimental specs

# Check git status
git status --porcelain
```

**Stop if**:
- Target spec does not exist
- Target spec is NOT experimental (name must end with `_exp`)
- Working directory has unexpected uncommitted changes

### Step 3: Update Schema

**File**: `{spec_dir}/schema.go`

Read the existing schema file first to understand the current struct layout.

#### 3a: Adding a new top-level field to Config

If the sugar is a new top-level section (like `boot_device` or `grub`), add a field to the `Config` struct:

```go
type Config struct {
    base.Config `yaml:",inline"`
    BootDevice  BootDevice `yaml:"boot_device"`
    Grub        Grub       `yaml:"grub"`
    NewSugar    NewSugar   `yaml:"new_sugar"`  // ADD THIS
}
```

Then add the new struct type(s):

```go
type NewSugar struct {
    FieldOne *string `yaml:"field_one"`
    FieldTwo *bool   `yaml:"field_two"`
}
```

#### 3b: Adding a field to an existing struct in base

If extending an existing base struct (like adding `parent` to `File`), modify the struct in `base/{version}/schema.go`:

```go
type File struct {
    // ... existing fields ...
    NewField NewFieldType `yaml:"new_field"`  // ADD THIS
}
```

**Conventions**:
- Use `*string`, `*bool`, `*int` for optional scalar fields
- Use `[]Type` for lists
- Use struct types for nested objects
- YAML tags use `snake_case`
- Add ` butane:"auto_skip"` tag for fields not in the Ignition spec that should be automatically filtered from the output (see `config/util/filter.go`)

### Step 4: Implement Translation

**File**: `{spec_dir}/translate.go`

Read the existing translate.go to understand the current translation pipeline.

#### 4a: Config Merging Pattern (Recommended)

This is the recommended approach per `docs/development.md:62`. The desugared config is the merge parent, user config is the child, so users can override sugar-generated values.

For **distro specs** (e.g., `config/fcos/v1_8_exp/translate.go`):

Add a new processing function and call it from `ToIgn3_7Unvalidated()`:

```go
func (c Config) ToIgn3_7Unvalidated(options common.TranslateOptions) (types.Config, translate.TranslationSet, report.Report) {
    ret, ts, r := c.Config.ToIgn3_7Unvalidated(options)
    if r.IsFatal() {
        return types.Config{}, translate.TranslationSet{}, r
    }
    // Existing sugar processing...
    r.Merge(c.processBootDevice(&ret, &ts, options))

    // ADD: Call new sugar processing
    retp, tsp, rp := c.processNewSugar(options)
    retConfig, ts := baseutil.MergeTranslatedConfigs(retp, tsp, ret, ts)
    ret = retConfig.(types.Config)
    r.Merge(rp)

    return ret, ts, r
}
```

Implement the processing function:

```go
func (c Config) processNewSugar(options common.TranslateOptions) (types.Config, translate.TranslationSet, report.Report) {
    rendered := types.Config{}
    ts := translate.NewTranslationSet("yaml", "json")
    var r report.Report

    // Early return if sugar is not being used
    if /* sugar not configured */ {
        return rendered, ts, r
    }

    yamlPath := path.New("yaml", "new_sugar")

    // Generate Ignition config elements
    // Example: creating a file
    file := types.File{
        Node: types.Node{
            Path: "/path/to/generated/file",
        },
        FileEmbedded1: types.FileEmbedded1{
            Contents: types.Resource{
                Source: util.StrToPtr("data:,generated-content"),
            },
        },
    }
    rendered.Storage.Files = append(rendered.Storage.Files, file)

    // Track translations for error reporting
    ts.AddFromCommonSource(yamlPath, path.New("json", "storage"), rendered.Storage)

    return rendered, ts, r
}
```

For **base specs** (e.g., `base/v0_8_exp/translate.go`):

The pattern is the same, but the processing function is called from the base `ToIgn3_7Unvalidated()` and operates on base types. When modifying translation at the base level, you may need to:
- Create or modify a custom translator function (e.g., `translateStorage()`)
- Register it with `tr.AddCustomTranslator()`

#### 4b: Direct Modification Pattern (Alternative)

Only use this when config merging isn't expressive enough:

```go
func (c Config) processNewSugar(config *types.Config, ts *translate.TranslationSet, options common.TranslateOptions) report.Report {
    var r report.Report

    if /* sugar not configured */ {
        return r
    }

    // Directly modify the Ignition config
    config.Storage.Files = append(config.Storage.Files, types.File{...})

    // Track translations
    yamlPath := path.New("yaml", "new_sugar")
    jsonPath := path.New("json", "storage", "files", len(config.Storage.Files)-1)
    ts.AddFromCommonSource(yamlPath, jsonPath, config.Storage.Files[len(config.Storage.Files)-1])

    return r
}
```

**Key imports** (add as needed):

```go
import (
    baseutil "github.com/coreos/butane/base/util"
    "github.com/coreos/butane/config/common"
    "github.com/coreos/butane/translate"

    "github.com/coreos/ignition/v2/config/util"
    "github.com/coreos/ignition/v2/config/v3_7_experimental/types"
    "github.com/coreos/vcontext/path"
    "github.com/coreos/vcontext/report"
)
```

**IMPORTANT**: The Ignition types import version must match the one already used in the file. Check the existing imports before adding new ones.

### Step 5: Write Tests

**File**: `{spec_dir}/translate_test.go`

Read the existing test file to understand the test patterns used.

Tests follow a table-driven pattern. Add a new test function:

```go
func TestTranslateNewSugar(t *testing.T) {
    tests := []struct {
        in  Config
        out types.Config
    }{
        // empty / no-op case
        {
            in:  Config{},
            out: types.Config{
                Ignition: types.Ignition{
                    Version: "3.7.0-experimental",
                },
            },
        },
        // basic sugar usage
        {
            in: Config{
                NewSugar: NewSugar{
                    FieldOne: util.StrToPtr("value"),
                },
            },
            out: types.Config{
                Ignition: types.Ignition{
                    Version: "3.7.0-experimental",
                },
                Storage: types.Storage{
                    Files: []types.File{
                        {
                            Node: types.Node{
                                Path: "/path/to/generated/file",
                            },
                            FileEmbedded1: types.FileEmbedded1{
                                Contents: types.Resource{
                                    Source: util.StrToPtr("data:,generated-content"),
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    for i, test := range tests {
        t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
            out, translations, r := test.in.ToIgn3_7Unvalidated(common.TranslateOptions{})
            r = confutil.TranslateReportPaths(r, translations)
            baseutil.VerifyReport(t, test.in, r)
            assert.Equal(t, test.out, out, "bad output")
            assert.Equal(t, report.Report{}, r, "expected empty report")
            assert.NoError(t, translations.DebugVerifyCoverage(out), "incomplete TranslationSet coverage")
        })
    }
}
```

**IMPORTANT**: The Ignition version in test expectations (e.g., `"3.7.0-experimental"`) must match the version used in the spec's translate.go. Check the existing tests for the correct value.

**Test categories to cover**:
- Empty/no-op: sugar not configured, should produce default output
- Basic usage: simplest valid configuration
- Complex usage: all options exercised
- User overrides: verify user can override sugar-generated values (for merge pattern)
- Edge cases: boundary conditions
- Error cases: invalid inputs (test separately in validate tests)

### Step 6: Add Validation (If Needed)

**File**: `{spec_dir}/validate.go`

Read the existing validate.go to understand validation patterns.

Add a `Validate` method on the new sugar type:

```go
func (s NewSugar) Validate(c path.ContextPath) (r report.Report) {
    if s.FieldOne != nil && *s.FieldOne == "" {
        r.AddOnError(c.Append("field_one"), common.ErrNewSugarFieldOneEmpty)
    }
    // ... more validations
    return
}
```

Or add validation to an existing `Validate` method on `Config`:

```go
func (conf Config) Validate(c path.ContextPath) (r report.Report) {
    // ... existing validations ...

    // New sugar validation
    if someCondition {
        r.AddOnError(c.Append("new_sugar", "field"), common.ErrSomething)
    }
    return
}
```

**Validation test file**: `{spec_dir}/validate_test.go`

```go
func TestValidateNewSugar(t *testing.T) {
    tests := []struct {
        in      NewSugar
        out     error
        errPath path.ContextPath
    }{
        // valid config
        {
            in:      NewSugar{FieldOne: util.StrToPtr("valid")},
            out:     nil,
            errPath: path.New("yaml"),
        },
        // invalid config
        {
            in:      NewSugar{FieldOne: util.StrToPtr("")},
            out:     common.ErrNewSugarFieldOneEmpty,
            errPath: path.New("yaml", "field_one"),
        },
    }

    for i, test := range tests {
        t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
            actual := test.in.Validate(path.New("yaml"))
            baseutil.VerifyReport(t, test.in, actual)
            expected := report.Report{}
            expected.AddOnError(test.errPath, test.out)
            assert.Equal(t, expected, actual, "bad validation report")
        })
    }
}
```

### Step 7: Add Error Constants

**File**: `config/common/errors.go`

Read the existing errors.go to understand the naming pattern.

Add new error variables in the appropriate section:

```go
var (
    // ... existing errors ...

    // New sugar
    ErrNewSugarFieldOneEmpty = errors.New("field_one must not be empty")
    ErrNewSugarInvalidCombo  = errors.New("field_one and field_two are mutually exclusive")
)
```

**Naming convention**: `Err` + CamelCase description. Error messages should be lowercase, concise, and actionable.

### Step 8: Update Documentation Descriptors

**File**: `internal/doc/butane.yaml`

Read the existing butane.yaml to understand the YAML structure for field documentation.

Add documentation descriptors for new fields. Place them in the correct location within the document hierarchy.

For a new top-level field (sibling of `boot_device`, `grub`):

```yaml
    - name: new_sugar
      after: $
      desc: describes the desired new sugar configuration.
      children:
        - name: field_one
          desc: the value for field one.
        - name: field_two
          desc: whether to enable feature two. If omitted, defaults to false.
```

For a field within an existing section (e.g., under `storage.files`):

```yaml
        - name: files
          children:
            # ... existing children ...
            - name: new_field
              after: $
              desc: description of the new field.
```

**Key patterns in butane.yaml**:
- `after: $` means "add at the end" (after all Ignition-defined fields)
- `after: ^` means "add at the beginning" (before all Ignition-defined fields)
- `transforms` can conditionally modify descriptions per variant/version
- `use: component_name` reuses a named component definition
- `required: true` marks a field as required
- Limit new fields to experimental spec versions using transforms if needed (add `"Unsupported"` replacement for older versions)

### Step 9: Regenerate Documentation

Run the documentation generator:

```bash
./generate
```

**Expected outcome**: Several `docs/config-*-exp.md` files are updated with the new field documentation.

Verify the docs were regenerated:

```bash
git diff docs/
```

If `./generate` fails, the schema or butane.yaml likely has an error. Fix and retry.

### Step 10: Add Usage Examples

**File**: `docs/examples.md`

Read the existing examples.md to understand the format.

Add a new example section:

```markdown
## New Sugar Feature Name

This example {describes what the example demonstrates}.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.8.0-experimental
new_sugar:
  field_one: value
  field_two: true
```
<!-- /butane-config -->

This {describes what gets generated/created}.
```

**Notes**:
- The `<!-- butane-config -->` comment markers are used for automated validation
- Use the experimental version string (e.g., `1.8.0-experimental`)
- Keep examples minimal but complete
- Show the simplest useful configuration first

### Step 11: Add Release Notes

**File**: `docs/release-notes.md`

Read the current release notes section.

Add a note under `## Upcoming Butane X.Y.Z (unreleased)` > `### Features`:

```markdown
### Features

- Add {sugar description} _(fcos 1.8.0-exp, openshift 4.22.0-exp, ...)_
```

**Notes**:
- List all affected variants/versions in the parenthetical
- For base sugar, list all experimental variants since they all inherit it
- Use the `-exp` suffix convention for experimental versions
- Keep the description concise (one line)

### Step 12: Run Tests

Execute the full test suite:

```bash
./test
```

**Expected outcome**: All tests pass.

If tests fail:
1. Read the error output carefully
2. Common issues:
   - Missing `TranslationSet` coverage: Add translation path tracking
   - Type mismatches: Check Ignition types vs Butane types
   - Import errors: Verify import paths match the experimental spec version
   - Validation failures: Check that test expectations match the validation logic
3. Fix issues and re-run

### Step 13: Report Results

Provide a comprehensive summary:

```
Sugar feature "{name}" added to {spec_type}/{version}

Files Modified:
  - {spec_dir}/schema.go (+N lines)
  - {spec_dir}/translate.go (+N lines)
  - {spec_dir}/translate_test.go (+N lines)
  - {spec_dir}/validate.go (+N lines) [if applicable]
  - {spec_dir}/validate_test.go (+N lines) [if applicable]
  - config/common/errors.go (+N lines) [if applicable]
  - internal/doc/butane.yaml (+N lines)
  - docs/examples.md (+N lines)
  - docs/release-notes.md (+N lines)
  - docs/config-*-exp.md (N files, regenerated)

Tests: PASSED
Docs: REGENERATED

Suggested commit message:

  {spec}/{version}: add {sugar_name} sugar

  {description of what the sugar does and why}

  resolves: #{issue_number}
```

## Checklist Coverage

This skill guides the following workflow:

- ✅ Schema definition in `schema.go`
- ✅ Translation logic in `translate.go` (config merging or direct modification)
- ✅ Comprehensive tests in `translate_test.go`
- ✅ Validation logic in `validate.go` (when needed)
- ✅ Validation tests in `validate_test.go` (when needed)
- ✅ Error constants in `config/common/errors.go`
- ✅ Documentation descriptors in `internal/doc/butane.yaml`
- ✅ Doc regeneration via `./generate`
- ✅ Usage examples in `docs/examples.md`
- ✅ Release notes in `docs/release-notes.md`
- ✅ Full test suite validation via `./test`

## What's NOT Covered

- ❌ **Designing the sugar semantics** - the user must know what Ignition config the sugar should produce
- ❌ **Complex translation logic** - the skill provides patterns, but domain-specific logic (e.g., partition layout calculations for boot_device) must be written by the developer
- ❌ **Integration tests** - only unit tests are scaffolded
- ❌ **Updating upgrading docs** - `docs/upgrading-*.md` must be updated manually when the sugar is stabilized
- ❌ **Creating git commits** - user should review changes first

## Example Output

```
/add-sugar --spec fcos/v1_8_exp

Analyzing config/fcos/v1_8_exp...

Current experimental spec:
  - Ignition version: 3.7.0-experimental
  - Base dependency: base/v0_8_exp
  - Existing sugar: boot_device, grub

What sugar would you like to add?
> Network configuration shortcut for static IPs

Gathering schema design...

Schema: New top-level field `network` with nested structs
Translation: Config merging pattern
Validation: Required fields, IP format validation

Phase 1: Schema
  schema.go updated (+15 lines)

Phase 2: Translation
  translate.go updated (+45 lines)

Phase 3: Tests
  translate_test.go updated (+120 lines)

Phase 4: Validation
  validate.go updated (+20 lines)
  validate_test.go updated (+40 lines)

Phase 5: Errors
  config/common/errors.go updated (+3 lines)

Phase 6: Documentation
  internal/doc/butane.yaml updated (+10 lines)
  ./generate completed
  docs/config-fcos-v1_8-exp.md regenerated
  docs/config-fiot-v1_1-exp.md regenerated
  docs/config-flatcar-v1_2-exp.md regenerated
  docs/config-openshift-v4_22-exp.md regenerated
  docs/config-r4e-v1_2-exp.md regenerated

Phase 7: Examples & Release Notes
  docs/examples.md updated (+12 lines)
  docs/release-notes.md updated (+1 line)

Phase 8: Validation
  ./test: All tests passed

Sugar feature "network" added to fcos/v1_8_exp

Suggested commit message:

  fcos/v1_8_exp: add network configuration sugar

  Add a `network` section that allows users to configure static
  IP addresses without manually creating NetworkManager keyfiles.

  resolves: #XXX
```

## References

- Design document: `.opencode/skills/add-sugar/DESIGN.md`
- Examples: `.opencode/skills/add-sugar/examples/`
- Development guide: `docs/development.md` (esp. lines 60-64 on sugar implementation)
- Stabilize checklist: `.github/ISSUE_TEMPLATE/stabilize-checklist.md`
- Current base experimental: `base/v0_8_exp/`
- Current FCOS experimental: `config/fcos/v1_8_exp/`
- Current OpenShift experimental: `config/openshift/v4_22_exp/`
