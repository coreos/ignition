# Marking an experimental spec as stable

When an experimental version of the Ignition config spec (e.g.: `3.1.0-experimental`) is to be declared stable (e.g. `3.1.0`), there are a handful of changes that must be made to the code base. These changes should have the following effects:

- Any configs with a `version` field set to the previously experimental version will no longer pass validation. For example, if `3.1.0-experimental` is being marked as stable, any configs written for `3.1.0-experimental` should have their version fields changed to `3.1.0`, for Ignition will no longer accept them.
- A new experimental spec version will be created. For example, if `3.1.0-experimental` is being marked as stable, a new version of `3.2.0-experimental` (or `4.0.0-experimental` if backwards incompatible changes are being made) will now be accepted, and start to accumulate new changes to the spec.
- The new stable spec and the new experimental spec will be identical except for the accepted versions. The new experimental spec is a direct copy of the old experimental spec, and no new changes to the spec have been made yet, so initially the two specs will have the same fields and semantics.
- The HTTP `Accept` header that Ignition uses whenever fetching a config will be updated to advertise the new stable spec.
- New features will be documented in the [Upgrading Configs](migrating-configs.md) documentation.

The changes that are required to achieve these effects are typically the following:

## Making the experimental package stable

- [ ] Rename `config/vX_Y_experimental` to `config/vX_Y`, and update the golang `package` statements
- [ ] Drop `_experimental` from all imports in `config/vX_Y`
- [ ] Update `MaxVersion` in `config/vX_Y/types/config.go` to delete the `PreRelease` field
- [ ] Update `config/vX_Y/config.go` to update the comment block on `ParseCompatibleVersion`
- [ ] Update `config/vX_Y/config_test.go` to test that the new stable version is valid and the old experimental version is invalid
- [ ] Update the `Accept` header in `internal/resource/url.go` to specify the new spec version.

## Creating the new experimental package

- [ ] Copy `config/vX_Y` into `config/vX_(Y+1)_experimental`, and update the golang `package` statements
- [ ] Update all `config/vX_Y` imports in `config/vX_(Y+1)_experimental` to `config/vX_(Y+1)_experimental`
- [ ] Update `config/vX_(Y+1)_experimental/types/config.go` to set `MaxVersion` to the correct major/minor versions with `PreRelease` set to `"experimental"`
- [ ] Update `config/vX_(Y+1)_experimental/config.go` to point the `prev` import to the new stable `vX_Y` package and update the comment block on `ParseCompatibleVersion`
- [ ] Update `config/vX_(Y+1)_experimental/config_test.go` to test that the new stable version is invalid and the new experimental version is valid
- [ ] Update `config/vX_(Y+1)_experimental/translate/translate.go` to translate from the previous stable version.  Update the `old_types` import, delete all functions except `translateIgnition` and `Translate`, and ensure `translateIgnition` translates the entire `Ignition` struct.
- [ ] Update `config/vX_(Y+1)_experimental/translate/translate_test.go` to point the `old` import to the new stable `vX_Y/types` package
- [ ] Update `config/config.go` imports to point to the experimental version.
- [ ] Update `config/config_test.go` to add the new experimental version to `TestConfigStructure`.
- [ ] Update `generate` to generate the new stable and experimental versions.

## Update all relevant places to use the new experimental package

- [ ] All places that imported `config/vX_Y_experimental` should be updated to `config/vX_(Y+1)_experimental`.
- Update `tests/register/register.go` in the following ways:
  - [ ] Add import `config/vX_Y/types`
  - [ ] Update import `config/vX_Y_experimental/types` to `config/vX_(Y+1)_experimental/types`
  - [ ] Add `config/vX_Y/types`'s identifier to `configVersions` in `Register()`

## Update the blackbox tests

- [ ] Bump the invalid `-experimental` version in the relevant `VersionOnlyConfig` test in `tests/negative/general/config.go`.
- [ ] Find all tests using `X.Y.0-experimental` and alter them to use `X.Y.0`.
- [ ] Update the `Accept` header checks in `tests/servers/servers.go` to specify the new spec version.

## Update docs

- [ ] Update `internal/doc/main.go` to add the new stable spec and reference the new experimental spec in `generate()`.
- [ ] Run `generate` to regenerate Go schemas and spec docs.
- [ ] Add a section to `docs/migrating-configs.md`.
- [ ] In `docs/specs.md`, update the list of stable and experimental spec versions (listing the latest stable release first) and update the table listing the Ignition release where a spec has been marked as stable.
- [ ] Note the stabilization in `docs/release-notes.md`, following the format of previous stabilizations. Drop the `-exp` version suffix from any notes for the upcoming release.

## External tests

If there are any external kola tests that were using the now stabilized experimental spec that are not part of the Ignition repo (e.g. tests in the [fedora-coreos-config](https://github.com/coreos/fedora-coreos-config/tree/testing-devel/tests/kola) repo), CI will fail for the spec stabilization PR.

For tests using experimental Ignition configs:

- [ ] Uncomment the commented-out workaround for this in `.cci.jenkinsfile`.
- [ ] When bumping the Ignition package in fedora-coreos-config, you'll need to update the external test in that repo to make CI green.
- [ ] Comment out the workaround.

For tests using experimental Butane configs:

- [ ] Disable the affected tests in [`kola-denylist.yaml`](https://github.com/coreos/fedora-coreos-config/blob/testing-devel/kola-denylist.yaml) by making their test scripts non-executable.
- [ ] Stabilize the Butane spec and revendor into coreos-assembler.
- [ ] Re-enable the tests.

## Other packages

- [ ] Add a stable spec to [ignition-config-rs](https://github.com/coreos/ignition-config-rs) and [regenerate schema](https://github.com/coreos/ignition-config-rs/blob/main/docs/development.md#regenerating-schemars).
  - [ ] Put out a new release.
- [ ] Bump ignition-config-rs in coreos-installer to support the new spec in `iso customize` and `pxe customize`. Update release notes.
  - [ ] Put out a new coreos-installer release.
- [ ] Add a new downgrade translation to [ign-converter](https://github.com/coreos/ign-converter/).
- [ ] [Stabilize Butane specs](https://coreos.github.io/butane/development/#bumping-spec-versions).
  - [ ] Put out a new release.
- [ ] Drop `-experimental` from configs in [FCOS docs](https://github.com/coreos/fedora-coreos-docs/) and remove colocated experimental-config warnings
- [ ] Revendor Ignition and Butane into coreos-assembler and update `mantle/platform/conf/conf.go` and `conf_test.go`.
- [ ] Ask the [Machine Config Operator](https://github.com/openshift/machine-config-operator/) to support the new spec.
