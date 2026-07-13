---
name: Stabilization checklist
about: Stabilization checklist template
title: New stabilization for Butane
labels: jira
---

# Bumping spec versions

This checklist describes bumping the Ignition spec version, `base` version, and distro versions. If your scenario is different, modify to taste.

## Stabilize Ignition spec version

- [ ] Bump `go.mod` for new Ignition release and update vendor.
- [ ] Update imports. Drop `-experimental` from Ignition spec versions in `*/translate_test.go`.

## Bump base version

- [ ] Rename `base/vB_exp` to `base/vB` and update `package` statements. Update imports.
- [ ] Copy `base/vB` to `base/vB+1_exp`.
- [ ] Update `package` statements in `base/vB+1_exp`.

## Bump distro version

- [ ] Rename `config/distro/vD_exp` to `config/distro/vD` and update `package` statements. Update imports.
- [ ] Drop `-experimental` from `init()` in `config/config.go`.
- [ ] Drop `-experimental` from examples in `docs/`.
- [ ] Copy `config/distro/vD` to `config/distro/vD+1_exp`.
- [ ] Update `package` statements in `config/distro/vD+1_exp`. Bump its base dependency to `base/vB+1_exp`.
- [ ] Import `config/vD+1_exp` in `config/config.go` and add `distro` `C+1-experimental` to `init()`.

## Bump Ignition spec version

- [ ] Bump Ignition types imports and rename `ToIgnI` and `TestToIgnI` functions in `base/vB+1_exp`. Bump Ignition spec versions in `base/vB+1_exp/translate_test.go`.
- [ ] Bump Ignition types imports in `config/distro/vD+1_exp`. Update `ToIgnI` function names, `util` calls, and header comments to `ToIgnI+1`.

## Update docs

- [ ] Update `internal/doc/main.go` to add the new stable spec and reference the new experimental spec in `generate()`.
- [ ] Run `generate` to regenerate spec docs.
- [ ] Update `docs/specs.md`.
- [ ] Update `docs/upgrading-*.md` for the new spec version. Copy the relevant section from Ignition's `doc/migrating-configs.md`, convert the configs to Butane configs, convert field names to snake case, and update wording as needed. Add subsections for any new Butane-specific features.
- [ ] Note the stabilization in `docs/release-notes.md`, following the format of previous stabilizations. Drop the `-exp` version suffix from any notes for the upcoming release.
