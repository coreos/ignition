---
nav_order: 9
---

# Release notes

## Upcoming Butane 0.29.0 (unreleased)

### Breaking changes

### Features

### Bug fixes

### Misc. changes

### Docs changes

## Butane 0.28.0 (2026-05-19)

Starting with this release, Butane binaries are signed with the [Fedora 44
key](https://getfedora.org/security/).

### Features

- Stabilize OpenShift spec 4.22.0, targeting Ignition spec 3.6.0
- Add OpenShift spec 4.23.0-experimental, targeting Ignition spec
  3.7.0-experimental
- Add `systemd.quadlets` section for embedding Podman Quadlet files
  _(fcos 1.8.0-exp, fiot 1.1.0-exp, flatcar 1.2.0-exp, openshift
  4.23.0-exp, r4e 1.2.0-exp)_

### Bug fixes

- Don't warn about partitions being reused by label for boot_device.mirror disks

### Misc. changes

- Warn on root partition size is too small _(fcos 1.3.0-1.8.0-exp)_
- Warn on root partition constrained by another partition _(fcos 1.3.0-1.8.0-exp)_

## Butane 0.27.0 (2026-02-27)

### Features

- Stabilize Fcos spec 1.7.0, targeting Ignition spec 3.6.0
- Add Fcos spec 1.8.0-experimental, targeting Ignition spec 3.7.0-experimental
- Update Flatcar spec 1.2.0-experimental to target Ignition spec 3.7.0-experimental
- Update Fiot spec 1.1.0-experimental to target Ignition spec 3.7.0-experimental
- Update R4E spec 1.2.0-experimental to target Ignition spec 3.7.0-experimental
- Update OpenShift spec 4.22.0-experimental to target Ignition spec 3.7.0-experimental

### Docs changes

- Re-vendor latest ignition release; 3.6.0-experimental becomes 3.6.0

## Butane 0.26.0 (2026-01-16)

Starting with this release, Butane binaries are signed with the [Fedora 43
key](https://getfedora.org/security/).

### Breaking changes

- Require `boot_device.layout` when using `boot_device.mirror` _(fcos 1.7.0-exp)_

### Features

- Stabilize OpenShift spec 4.21.0, targeting Ignition spec 3.5.0
- Add OpenShift spec 4.22.0-experimental, targeting Ignition spec
  3.6.0-experimental
- Add support for mode and ownership settings for trees.

### Bug fixes

- Warn for `boot_device.layout` to be specified when using `boot_device.mirror` _(fcos 1.3.0-1.6.0)_

### Docs changes

- Update `boot_device.mirror` examples to specify `boot_device.layout`

## Butane 0.25.1 (2025-09-24)

### Docs changes

- Update docs around the use of setuid/gid from Ignition [bug](coreos/ignition#2042)'

### Misc. changes

- Update vendor'd Ignition dependency to point to latest v2.23.0

## Butane 0.25.0 (2025-09-08)

### Features

- Stabilize OpenShift spec 4.20.0, targeting Ignition spec 3.5.0
- Add OpenShift spec 4.21.0-experimental, targeting Ignition spec
  3.6.0-experimental

### Bug fixes

- Stop overriding default LUKS cipher algorithm in FIPS mode _(openshift 4.20.0)_

### Docs changes

- Add missing examples in upgrading-openshift _(openshift 4.14)_

## Butane 0.24.0 (2025-05-27)

### Features

- Validate merged/replaced Ignition configs if they are local/inline _(all base specifications)_
- Stabilize OpenShift spec 4.19.0, targeting Ignition spec 3.5.0
- Add OpenShift spec 4.20.0-experimental, targeting Ignition spec
  3.6.0-experimental
- Add TMT test support with initial smoke test

### Bug fixes

- Fail if LUKS method is not specified while `boot_device.luks.device` is set _(fcos 1.7.0-exp)_
- Validate kernel arguments when CEX support is enabled on s390x _(4.19.0 and 4.20.0)_

### Misc. changes

- Roll back to Ignition spec 3.5.0 _(openshift 4.19.0)_

## Butane 0.23.0 (2024-12-03)

Starting with this release, Butane binaries are signed with the [Fedora 41
key](https://getfedora.org/security/).

### Features

- Add OpenShift spec 4.19.0-experimental, targeting Ignition spec
  3.6.0-experimental
- Stabilize OpenShift spec 4.18.0, targeting Ignition spec 3.4.0
- Stabilize Fcos spec 1.6.0, targeting Ignition spec 3.5.0
- Add Fcos spec 1.7.0-experimental, targeting Ignition spec
  3.6.0-experimental
- Update Fiot spec 1.1.0-experimental to target Ignition spec
  3.6.0-experimental
- Update Flatcar spec 1.2.0-experimental to target Ignition spec
  3.6.0-experimental
- Update OpenShift spec 4.18.0-experimental, targeting Ignition spec
  3.6.0-experimental
- Update R4e spec 1.2.0-experimental to target Ignition spec
  3.6.0-experimental
- Support LUKS encryption using IBM CEX secure keys on s390x _(fcos 1.6)_ _(openshift 4.18.0-exp)_

### Docs changes

- Re-vendor latest ignition release; 3.5.0-experimental becomes 3.5.0

## Butane 0.22.0 (2024-09-20)

### Features

- Stabilize OpenShift spec 4.17.0, targeting Ignition spec 3.4.0
- Add OpenShift spec 4.18.0-experimental, targeting Ignition spec
  3.5.0-experimental
- Support and documentation for `grub` section moved to OpenShift
  4.18.0-experimental spec.

### Misc. changes

- Roll back to Ignition spec 3.4.0 _(openshift 4.17.0)_

## Butane 0.21.0 (2024-06-06)

Starting with this release, Butane binaries are signed with the [Fedora 40
key](https://getfedora.org/security/).

### Features

- Support `storage.luks.clevis` (flatcar 1.2.0-exp)
- Stabilize OpenShift spec 4.16.0, targeting Ignition spec 3.4.0
- Add OpenShift spec 4.17.0-experimental, targeting Ignition spec
  3.5.0-experimental

## Butane 0.20.0 (2024-02-19)

Starting with this release, Butane binaries are signed with the [Fedora 39
key](https://getfedora.org/security/).

### Features

- Support s390x layouts in `boot_device` section (fcos 1.6.0-exp, openshift 4.16.0-exp)
- Stabilize OpenShift spec 4.15.0, targeting Ignition spec 3.4.0
- Add OpenShift spec 4.16.0-experimental, targeting Ignition spec
  3.5.0-experimental

### Misc. changes

- Require Go 1.20+

## Butane 0.19.0 (2023-10-03)

Starting with this release, Butane binaries are signed with the [Fedora 38
key](https://getfedora.org/security/).

### Breaking changes

- Spec implementations require a `FieldFilters()` method (Go API)
- Reports from `Unvalidated` functions can now include `json` paths (Go API)

### Features

- Add `-c`/`--check` option to check config without producing output
- Warn if config attempts to reuse partition by label _(fcos 1.6.0-exp,
  openshift 4.14.0)_
- Require `storage.filesystems.path` to start with `/etc` or `/var` if
  `with_mount_unit` is true _(fcos 1.6.0-exp, openshift 4.14.0)_
- Stabilize OpenShift spec 4.14.0, targeting Ignition spec 3.4.0
- Add OpenShift spec 4.15.0-experimental, targeting Ignition spec
  3.5.0-experimental
- Add new variant `fiot` for fedora-iot

### Bug fixes

- Fix line/column reporting for `http_headers` errors
- Fix line/column reporting for unsupported field errors _(r4e)_

### Misc. changes

- Add error structs for YAML unmarshal errors, unknown config versions (Go API)
- Roll back to Ignition spec 3.4.0 _(openshift 4.14.0)_

### Docs changes

- Document consequence of setting `systemd.units.mask` to false
- Document `grub` section _(openshift 4.15.0-exp)_
- Document `/dev/disk/by-id/coreos-boot-disk` _(fcos, openshift 4.11.0+)_
- Don't claim to support generating swap units _(openshift 4.8.0 - 4.13.0)_
- Document `key_file` `compression` field  _(openshift 4.8.0 - 4.9.0)_
- Document support for special mode bits and `arn` URLs _(r4e 1.1.0+)_
- Improve rendering of spec docs on docs site

## Butane 0.18.0 (2023-03-24)

### Breaking changes

- Remove deprecated `rhcos` variant

### Features

- Support offline Tang provisioning via pre-shared advertisement _(fcos 1.5.0+,
  openshift 4.14.0-exp)_
- Support local file embedding for SSH keys and systemd units _(fcos 1.5.0+,
  flatcar 1.1.0+, openshift 4.14.0-exp, r4e 1.1.0+)_
- Allow enabling discard passthrough on LUKS devices _(fcos 1.5.0+,
  flatcar 1.1.0+, openshift 4.14.0-exp)_
- Allow specifying arbitrary LUKS open options _(fcos 1.5.0+,
  flatcar 1.1.0+, openshift 4.14.0-exp)_
- Allow specifying user password hash _(openshift 4.13.0+)_
- Stabilize Fedora CoreOS spec 1.5.0, targeting Ignition spec 3.4.0
- Stabilize Flatcar spec 1.1.0, targeting Ignition spec 3.4.0
- Stabilize OpenShift spec 4.13.0, targeting Ignition spec 3.2.0
- Stabilize RHEL for Edge spec 1.1.0, targeting Ignition spec 3.4.0
- Add Fedora CoreOS spec 1.6.0-experimental, targeting Ignition spec
  3.5.0-experimental
- Add Flatcar spec 1.2.0-experimental, targeting Ignition spec
  3.5.0-experimental
- Add OpenShift spec 4.14.0-experimental, targeting Ignition spec
  3.5.0-experimental
- Add RHEL for Edge spec 1.2.0-experimental, targeting Ignition spec
  3.5.0-experimental

### Bug fixes

- Use systemd default dependencies in mount units for Tang-backed LUKS volumes
- Allow setting `storage.trees.local` to the `--files-dir` directory

### Misc. changes

- Roll back to Ignition spec 3.2.0 _(openshift 4.13.0)_
- Drop `extensions` section _(fcos 1.5.0+, openshift 4.13.0+)_
- Drop `LuksOption` and `RaidOption` types _(Go API for fcos 1.5.0+,
  flatcar 1.1.0+, openshift 4.14.0-experimental)_
- Require Go 1.18+

### Docs changes

- Document that `hash` fields describe decompressed data
- Clarify spec docs for `files`/`luks` `hash` fields
- Document SSH key file path used by OpenShift 4.13+ _(openshift)_
- Document command to generate GRUB password hashes

## Butane 0.17.0 (2023-01-04)

Starting with this release, Butane binaries are signed with the [Fedora 37
key](https://getfedora.org/security/).

### Features

- Add RHEL for Edge (`r4e`) spec 1.0.0 and 1.1.0-experimental, targeting
  Ignition spec 3.3.0 and 3.4.0-experimental respectively

### Bug fixes

- Fix version string in release container

## Butane 0.16.0 (2022-10-14)

### Features

- Stabilize OpenShift spec 4.12.0, targeting Ignition spec 3.2.0
- Add OpenShift spec 4.13.0-experimental, targeting Ignition spec
  3.4.0-experimental
- Ship aarch64 macOS binary in GitHub release artifacts

### Misc. changes

- Roll back to Ignition spec 3.2.0 _(openshift 4.12.0)_
- Require Go 1.17+
- test: Check docs on macOS and Windows if dependencies are available

### Docs changes

- Document `passwd.users.should_exist` and `passwd.groups.should_exist` fields
  _(fcos 1.2.0+, flatcar, rhcos)_
- Clarify spec docs for `files`/`directories`/`links` `group` fields
- Document that `user`/`group` fields aren't applied to hard links

## Butane 0.15.0 (2022-06-23)

Starting with this release, Butane binaries are signed with the [Fedora 36
key](https://getfedora.org/security/).

### Breaking changes

- Return selected `compression` field value from `MakeDataURL()` _(Go API)_

### Features

- Add Flatcar spec 1.0.0 and 1.1.0-experimental, targeting Ignition spec
  3.3.0 and 3.4.0-experimental respectively
- Stabilize OpenShift spec 4.11.0, targeting Ignition spec 3.2.0
- Add OpenShift spec 4.12.0-experimental, targeting Ignition spec
  3.4.0-experimental
- Add arm64 support to container
- Add GRUB password support _(fcos 1.5.0-exp, openshift 4.12.0-exp)_
- Add `TranslationSet` `AddFromCommonObject()` and `Map()` methods _(Go API)_

### Bug fixes

- Set `compression` field for uncompressed `inline`/`local` resources, fixing
  provisioning failure when merged with a compressed parent resource
- Fix local file inclusion on Windows
- Fix `build` script on Windows

### Misc. changes

- Derive container from Fedora image to support use in multi-stage builds
- Fail if setuid/setgid/sticky mode bits specified _(openshift 4.10.0+)_
- Update to Ignition 2.14.0
- Roll back to Ignition spec 3.2.0 _(openshift 4.11.0)_

### Docs changes

- Support `arn` URL scheme _(fcos 1.5.0-exp, openshift 4.12.0-exp)_
- Document support status of setuid/setgid/sticky mode bits in each spec
- Document support for `gs` URLs _(openshift 4.8.0+)_
- Document support for `compression` field  _(openshift 4.8.0 - 4.9.0)_
- Correctly document supported URL schemes _(openshift 4.10.0)_
- examples: Use containerized `mkpasswd`
- Convert `NEWS` to Markdown and move to docs site

## Butane 0.14.0 (2022-01-27)

Starting with this release, Butane binaries are signed with the [Fedora 35
key](https://getfedora.org/security/).

### Breaking changes

- Drop `TranslateBytesOptions.Strict` field; callers should fail on
  non-empty reports instead _(Go API)_

### Features

- Stabilize OpenShift spec 4.10.0, targeting Ignition spec 3.2.0
- Add OpenShift spec 4.11.0-experimental, targeting Ignition spec
  3.4.0-experimental
- Warn on incorrect partition numbers for reserved labels _(fcos, openshift)_
- Require `storage.files.contents.source` URLs to use `data` scheme
  _(openshift)_
- Re-enable automatic and manual resource compression _(openshift 4.10.0+)_
- Add `extensions` section _(fcos 1.5.0-exp, openshift 4.11.0-exp)_

### Bug fixes

- Correctly fail on validation warnings if `--strict` is specified
- Statically link official Linux binaries

### Misc. changes

- Roll back to Ignition spec 3.2.0 _(openshift 4.10.0)_
- Add deprecation warning for `rhcos` variant
- Add reserved partitions to `aarch64`/`ppc64le` `boot_device.mirror` layouts
  _(fcos, openshift)_

### Docs changes

- Improve getting-started instructions for running in container
- Document availability of `gs` URL scheme
- Correctly document availability of `compression` fields
- Correctly document `ignition` section as optional
- Add `with_mount_unit` `swap` support to migration guide _(fcos 1.4.0)_
- Document build process and contribution flow for release binaries

## Butane 0.13.1 (2021-08-04)

### Misc. changes

- Roll back to Ignition spec 3.2.0, since 3.3.0 support didn't make
  it into OpenShift 4.9. No 3.3.0 features were permitted in this
  config version, so this shouldn't break configs. _(openshift 4.9.0)_
- Send `--help` output to stdout
- Drop support for Go 1.13 and 1.14

### Docs changes

- Correctly snake-case `ignition.proxy` fields

## Butane 0.13.0 (2021-07-13)

### Features

- Stabilize Fedora CoreOS spec 1.4.0, targeting Ignition spec 3.3.0
- Add Fedora CoreOS spec 1.5.0-experimental, targeting Ignition spec
  3.4.0-experimental
- Stabilize OpenShift spec 4.9.0, targeting Ignition spec 3.3.0
- Add OpenShift spec 4.10.0-experimental, targeting Ignition spec
  3.4.0-experimental
- Support `none` filesystem format _(fcos 1.4.0+)_

### Bug fixes

- Correctly track input line/column in `kernel_arguments` section

### Misc. changes

- Deprecate `rhcos` 0.1.0 spec in favor of `openshift` variant
- Disable `kernel_arguments` section in favor of `openshift.kernel_arguments`
  _(openshift 4.9.0+)_
- Convert `ClevisCustom.Config`, `ClevisCustom.Pin`, `Link.Target`, and
  `Raid.Level` Go fields to pointers _(fcos 1.4.0+, openshift 4.9.0+)_

### Docs changes

- Document default value for Clevis `threshold`

## Butane 0.12.1 (2021-06-10)

### Bug fixes

- Disable automatic resource compression _(openshift 4.8.0,
  openshift 4.9.0-exp)_

### Misc. changes

- Fail if file compression specified _(openshift 4.8.0, openshift 4.9.0-exp)_

## Butane 0.12.0 (2021-06-08)

Starting with this release, Butane binaries are signed with the [Fedora 34
key](https://getfedora.org/security/).

### Features

- Add `kernel_arguments` section _(fcos 1.4.0-exp, openshift 4.9.0-exp)_

### Bug fixes

- Fix incorrect config paths in validation reports on 386 architecture

### Misc. changes

- Fail on `btrfs` filesystem format _(openshift 4.8.0, openshift 4.9.0-exp)_
- Add comment to MachineConfig output noting that the config is
  machine-generated

## Butane 0.11.0 (2021-04-05)

### Breaking changes

- Rename project to Butane and binary to `butane`
- Change package path to `github.com/coreos/butane` _(Go API)_
- Remove `translate.AddIdentity()` in favor of `translate.MergeP()` _(Go API)_

### Features

- Add OpenShift spec 4.8.0, targeting Ignition spec 3.2.0
- Output MachineConfig unless `-r`/`--raw` specified _(openshift 4.8.0)_
- Error on Ignition fields discouraged by OpenShift _(openshift 4.8.0)_
- Add `metadata` section for MachineConfig metadata _(openshift 4.8.0)_
- Add `openshift` section for MachineConfig configuration _(openshift 4.8.0)_
- Set appropriate LUKS cipher if `openshift.fips` enabled _(openshift 4.8.0)_
- Add OpenShift spec 4.9.0-experimental, targeting Ignition spec
  3.3.0-experimental

### Misc. changes

- Remove RHEL CoreOS spec 0.2.0-experimental
- Refactor translation tracking for report entries
- Add undocumented `-D`/`--debug` option to report translation map

### Docs changes

- Provide separate config upgrade guide for each variant
- Document `storage.filesystems.resize`
- Fix filesystem resize example in upgrade docs
- Document default for `storage.filesystems.wipe_filesystem`

## FCCT 0.10.0 (2021-02-01)

### Features

- Create systemd `swap` unit when `with_mount_unit` is enabled on swap area
  _(fcos 1.4.0-exp, rhcos 0.2.0-exp)_

### Bug fixes

- Drop erroneous EFI partition in `boot_device.mirror` `ppc64le` layout
- Fix panic translating `boot_device` when config is invalid

## FCCT 0.9.0 (2021-01-05)

### Bug fixes

- Avoid ESP RAID desynchronization by creating independent ESP filesystems

### Docs changes

- Clarify semantics of `systemd.units.name`
- Correctly document `storage.filesystems.path` as optional
- Fix nesting of `storage.luks` and `storage.trees` sections
- Move codebase layout info from README to developer docs
- Recommend container image or distro package over standalone binary

## FCCT 0.8.0 (2020-12-04)

Starting with this release, Butane binaries are signed with the [Fedora 33
key](https://getfedora.org/security/).

### Breaking changes

- Restructure Go API

### Features

- Stabilize Fedora CoreOS spec 1.3.0, targeting Ignition spec 3.2.0
- Add Fedora CoreOS spec 1.4.0-experimental, targeting Ignition spec
  3.3.0-experimental
- Add RHEL CoreOS spec 0.1.0, targeting Ignition spec 3.2.0
- Add RHEL CoreOS spec 0.2.0-experimental, targeting Ignition spec
  3.3.0-experimental
- Add `boot_device` section for configuring boot device LUKS and mirroring
  _(fcos 1.3.0, rhcos 0.1.0)_

### Bug fixes

- Fix `systemd-fsck@.service` dependencies in generated mount units

### Misc. changes

- Warn if file/dir modes appear to have been specified in decimal
- Validate input in translation functions taking Go structs _(Go API)_
- Allow registering external translators _(Go API)_
- Allow specs to derive from other specs _(Go API)_

### Docs changes

- Document Clevis `custom` and LUKS `wipe_volume` fields
- Add LUKS and mirroring examples
- Add password authentication example

## FCCT 0.7.0 (2020-10-23)

### Features

- Stabilize FCC spec 1.2.0, targeting Ignition spec 3.2.0
- Add FCC spec 1.3.0-experimental, targeting Ignition spec
  3.3.0-experimental
- Add `storage.luks` section for creating LUKS2 encrypted volumes
  _(1.2.0)_
- Add `resize` field for modifying partition size _(1.2.0)_
- Add `should_exist` field for deleting users & groups _(1.2.0)_
- Add `NoResourceAutoCompression` translate option to skip automatic
  compression _(Go API)_

### Docs changes

- Switch to GitHub Pages

## FCCT 0.6.0 (2020-05-28)

Starting with this release, Butane binaries are signed with the [Fedora 32
key](https://getfedora.org/security/).

### Features

- Stabilize FCC spec 1.1.0, targeting Ignition spec 3.1.0
- Add FCC spec 1.2.0-experimental, targeting Ignition spec
  3.2.0-experimental
- Add `inline` field to TLS certificate authorities and config merge and
  replace _(1.1.0)_
- Add `local` field for embedding contents from local file _(1.1.0)_
- Add `storage.trees` section for embedding local directory trees _(1.1.0)_
- Auto-select smallest encoding for `inline` or `local` contents _(1.1.0)_
- Add `http_headers` field for specifying HTTP headers on fetch _(1.1.0)_

### Bug fixes

- Include mount options in generated mount units _(1.1.0)_
- Validate uniqueness constraints within FCC sections
- Omit empty values from output JSON
- Append newline to output

### Docs changes

- Document support for CA bundles in Ignition >= 2.3.0
- Document support for `sha256` resource verification _(1.1.0)_
- Clarify semantics of `overwrite` and `mode` fields

## FCCT 0.5.0 (2020-03-23)

### Breaking changes

- Previously, command-line options could be preceded by a single dash
  (`-strict`) or double dash (`--strict`).  Accept only the double-dash form.

### Features

- Accept input filename directly on command line, without `--input`
- Add short equivalents of command-line options

### Bug fixes

- Fail if unexpected non-option arguments are specified

### Misc. changes

- Deprecate `--input` and hide it from `--help`
- Document `files[].append[].inline` property
- Update docs for switch to Fedora signing keys

## FCCT 0.4.0 (2020-01-24)

### Features

- Add `mount_options` field to filesystem entry

### Misc. changes

- Add `release` tag to container of latest release
- Vendor dependencies

## FCCT 0.3.0 (2020-01-23)

### Features

- Add v1.1.0-experimental spec
- Add `with_mount_unit` field to generate mount unit from filesystem entry

### Bug fixes

- Report warnings and errors to stderr, not stdout
- Truncate output file before writing
- Fix line and column reporting

### Misc. changes

- Document syntax of inline file contents
- Document usage of published container image

## FCCT 0.2.0 (2019-07-24)

### Features

- Add `--version` flag
- Add Dockerfile and build containers automatically on quay.io

### Bug fixes

- Fix validation of paths for files and directories
- Fix `--output` flag handling

### Misc. changes

- Add tests for the examples in the docs
- Add travis integration

## FCCT 0.1.0 (2019-07-10)

Initial Release of FCCT. While the golang API is not stable, the Fedora CoreOS
Configuration language is. Configs written with version 1.0.0 will continue to
work with future releases of FCCT.
