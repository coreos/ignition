---
name: stabilize-spec
description: Automate the Ignition config spec stabilization process
---

# Ignition Spec Stabilization

This skill automates the complete Ignition config spec stabilization process following the exact 8-commit structure from [PR #2202](https://github.com/coreos/ignition/pull/2202).

## What it does

Performs a complete spec stabilization by creating 8 atomic commits that:

1. Rename the experimental package to stable (e.g., `v3_6_experimental` → `v3_6`)
2. Stabilize the spec (remove PreRelease, update tests, update Accept header)
3. Copy the stable spec to a new experimental version (e.g., `v3_6` → `v3_7_experimental`)
4. Adapt the new experimental spec
5. Update all imports across the codebase (72+ files)
6. Update blackbox tests
7. Update documentation generation
8. Update docs and regenerate schemas

## Prerequisites

Before running this skill, ensure:

```bash
# Install schematyper (required for regenerating schemas)
cd /tmp
git clone https://github.com/idubinskiy/schematyper.git
cd schematyper
go mod init github.com/idubinskiy/schematyper
echo 'replace gopkg.in/alecthomas/kingpin.v2 => github.com/alecthomas/kingpin/v2 v2.4.0' >> go.mod
go mod tidy
go install .

# Install build dependencies (Fedora/RHEL)
sudo dnf install -y libblkid-devel
```

## Usage

When invoked, provide the version information:

```
Current experimental version: 3.6.0-experimental
Target stable version: 3.6.0
Next experimental version: 3.7.0-experimental
```

The skill will then:

1. ✅ Perform all code changes across the repository
2. ✅ Create 8 properly structured commits
3. ✅ Run `./generate` to regenerate schemas and docs
4. ✅ Run `./build` to verify compilation
5. ✅ Run `./test` to verify all tests pass
6. ✅ Provide a summary of what was done

## Checklist Coverage

This skill completes all items from the [stabilization checklist](https://github.com/coreos/ignition/blob/main/.github/ISSUE_TEMPLATE/stabilize-checklist.md):

### Making the experimental package stable ✅
- Rename `config/vX_Y_experimental` to `config/vX_Y`
- Drop `_experimental` from all imports
- Update `MaxVersion` to delete the `PreRelease` field
- Update `config.go` comment block
- Update `config_test.go` to test stable version valid, experimental invalid
- Update Accept header in `internal/resource/url.go`

### Creating the new experimental package ✅
- Copy `config/vX_Y` into `config/vX_(Y+1)_experimental`
- Update all imports to `config/vX_(Y+1)_experimental`
- Update `MaxVersion` with `PreRelease = "experimental"`
- Update `config.go` prev import to stable package
- Update `config_test.go` for new experimental version
- Simplify `translate.go` to only `translateIgnition` and `Translate`
- Update `translate_test.go` old import to stable version
- Update `config/config.go` imports to experimental version
- Update `config/config_test.go` to add new experimental version
- Update `generate` script

### Update all relevant places ✅
- Update all imports from `vX_Y_experimental` to `vX_(Y+1)_experimental`
- Add import `config/vX_Y/types` to `tests/register/register.go`
- Update import to `vX_(Y+1)_experimental` in `tests/register/register.go`
- Add `vX_Y/types` to `configVersions` in `Register()`

### Update the blackbox tests ✅
- Bump invalid `-experimental` version in `VersionOnlyConfig` test
- Change all `X.Y.0-experimental` to `X.Y.0` in tests
- Update Accept header checks in `tests/servers/servers.go`

### Update docs ✅
- Update `internal/doc/main.go`
- Run `generate` to regenerate schemas and docs
- Add section to `docs/migrating-configs.md`
- Update `docs/specs.md`
- Note stabilization in `docs/release-notes.md`

## What's NOT covered

The following sections from the checklist require external repos and are NOT automated:

- External tests (`.cci.jenkinsfile`, fedora-coreos-config)
- Other packages (ignition-config-rs, coreos-installer, ign-converter, Butane, FCOS docs, coreos-assembler, MCO)

These must be done manually after the PR is merged.

## Example Output

After running the skill, you'll see:

```
✨ Stabilization complete!

Created 8 commits:
  ceb03d33 docs: update for spec stabilization
  e5cac5c1 docs: shuffle for spec stabilization
  2aca7225 tests: update for new experimental spec
  3252aa50 *: update to v3_7_experimental spec
  0e5a3297 config/v3_7_experimental: adapt for new experimental spec
  21d51407 config: copy v3_6 to v3_7_experimental
  57d4d86e config/v3_6: stabilize
  8a071635 config: rename v3_6_experimental to v3_6

Next steps:
1. Review commits: git log --oneline -8
2. Create PR to main branch
3. Update issue checkboxes (see checklist above)
4. After merge, handle external tests and packages
```

## Reference

- [Issue #2200 - Stabilization Checklist](https://github.com/coreos/ignition/issues/2200)
- [PR #2202 - Example Stabilization (3.6.0)](https://github.com/coreos/ignition/pull/2202)
- [Stabilization Template](https://github.com/coreos/ignition/blob/main/.github/ISSUE_TEMPLATE/stabilize-checklist.md)
