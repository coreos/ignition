---
nav_order: 9
---

# Release Notes

## Upcoming Ignition 2.26.0 (unreleased)

### Breaking changes

### Features

### Changes

### Bug fixes

- Include `groupmod` binary in initramfs ([#2190](https://github.com/coreos/ignition/pull/2190))

## Ignition 2.25.1 (2025-12-22)

### Bug fixes
- Fix OpenStack provider returning empty JSON instead of empty bytes when metadata has no config


## Ignition 2.25.0 (2025-12-11)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 43 key](https://getfedora.org/security/).

### Features

- The name for custom clevis pins is not validated by Ignition anymore, enabling the use of arbitrary custom pins _(3.6.0-exp)_
- Add NVIDIA BlueField provider

### Bug fixes

- Fix EnsureGroup to be idempotent when group already exists ([#2158](https://github.com/coreos/ignition/pull/2158))
- Fix invalid random source in FIPS 140-only mode in FIPS mode ([#2159](https://github.com/coreos/ignition/pull/2159))
- Only load kernel modules when actually necessary so that they can be built-in ([#2164](https://github.com/coreos/ignition/pull/2164))


## Ignition 2.24.0 (2024-10-14)

This version was actually released 2025-10-14, but changing the title now would
invalidate links to this entry here in the release notes.

### Features

- Add support for nocloud config fetching in kubevirt

### Bug fixes

- Fix occasional cex.key file removal
- Fix multipath partitioning: ignore DM holders when no partitions are mounted


## Ignition 2.23.0 (2025-09-10)

### Features

- Support UpCloud

### Changes

- Switch to aws-sdk-go-v2 for S3 fetches and EC2 interactions

### Bug fixes

- Fix fetch-offline for Oracle Cloud Infrastructure


## Ignition 2.22.0 (2025-07-08)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 42 key](https://getfedora.org/security/).

### Breaking changes

### Features

- Support Oracle Cloud Infrastructure

### Changes

- Rename ignition.cfg -> 05_ignition.cfg
- Support setting setuid/setgid/sticky mode bits _(3.6.0-exp)_
- Warn if setuid/setgid/sticky mode bits specified _(3.4.0 - 3.5.0)_
- Add initial TMT tests and a new workflow to execute tests on PRs

### Bug fixes

- Fix use of setuid/setgid/sticky mode bits


## Ignition 2.21.0 (2025-03-13)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 41 key](https://getfedora.org/security/).

### Breaking changes

### Features

- Add Azure blob support for fetching ignition configs
- Add a check for ignition config in vendor-data (proxmoxve)

### Changes

### Bug fixes

- Add `pkey_cca` kernel module to detect CEX domain for LUKS encryption


## Ignition 2.20.0 (2024-10-22)

### Features

- Support partitioning disk with mounted partitions
- Support Proxmox VE
- Support gzipped Akamai user_data
- Support IPv6 for single-stack OpenStack

### Changes

- The Dracut module now installs partx
- Mark the 3.5.0 config spec as stable
- No longer accept configs with version 3.5.0-experimental
- Create new 3.6.0-experimental config spec from 3.5.0

### Bug fixes

- Fix network race when phoning home on Equinix Metal
- Fix Akamai Ignition base64 decoding on padded payloads
- Fix Makefile GOARCH for loongarch64 ([#1942](https://github.com/coreos/ignition/pull/1942))
- Don't log to journal if not available


## Ignition 2.19.0 (2024-06-05)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 40 key](https://getfedora.org/security/).

### Features

- Support Akamai Connected Cloud (Linode)
- Support LUKS encryption using IBM CEX secure keys


## Ignition 2.18.0 (2024-03-01)

### Breaking changes

- Only include dracut module in initramfs if requested (see distributor notes
  for details)

### Features

- Support Scaleway

### Changes

- Require Go 1.20+

### Bug fixes

- Fix failure when config only disables units already disabled
- Retry HTTP requests on Azure on status codes 404, 410, and 429


## Ignition 2.17.0 (2023-11-20)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 39 key](https://getfedora.org/security/).

### Features

- Support the native Apple Hypervisor
- Support Hetzner Cloud
- A GRUB configuration suitable for use with https://github.com/coreos/bootupd
  can now be installed; use `make install-grub-for-bootupd` to install it

### Changes

- Require Go 1.19+

### Bug fixes

- Prevent races with udev after disk editing
- Don't fail to wipe partition table if it's corrupted


## Ignition 2.16.2 (2023-07-12)

### Bug fixes

- Fix Dracut module installation on arches other than x86 and aarch64


## Ignition 2.16.1 (2023-07-10)

### Bug fixes

- Fix build on 32-bit systems


## Ignition 2.16.0 (2023-06-29)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 38 key](https://getfedora.org/security/).

### Features

- Support Hyper-V platform
- Automatically generate spec docs

### Changes

- Clarify spec terminology for contents of CA bundles, files, and key files
- Improve rendering of spec docs on docs site

### Bug fixes

- Fix failure disabling nonexistent unit with systemd ≥ 252
- Don't relabel a mount point that already exists
- Document that `hash` fields describe decompressed data
- Clarify documentation of `passwordHash` fields
- Correctly document Tang `advertisement` field as optional

### Test changes

- Support and require xfsprogs ≥ 5.19 in blackbox tests


## Ignition 2.15.0 (2023-02-21)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 37 key](https://getfedora.org/security/).

### Features

- Support offline Tang provisioning via pre-shared advertisement _(3.4.0)_
- Allow enabling discard passthrough on LUKS devices _(3.4.0)_
- Allow specifying arbitrary LUKS open options _(3.4.0)_
- Ship aarch64 macOS ignition-validate binary in GitHub release artifacts

### Changes

- Mark the 3.4.0 config spec as stable
- No longer accept configs with version 3.4.0-experimental
- Create new 3.5.0-experimental config spec from 3.4.0
- Fail if files/links/dirs conflict with systemd units or dropins
- Warn if template for enabled systemd instance unit has no `Install` section
- Warn if filesystem overwrites partitioned disk
- Warn if `wipeTable` overwrites a filesystem that would otherwise be reused
- Warn if `user`/`group` specified for hard link
- Install ignition-apply in `/usr/libexec`
- Allow distros to add Ignition command-line arguments from a unit drop-in
- Convert `NEWS` to Markdown and move to docs site
- Require Go 1.18+

### Bug fixes

- Don't overwrite LUKS1 volume when `storage.luks.wipeVolume` is false
- Request network when custom Clevis config has `needsNetwork` set
- Fix creating LUKS volume with custom Clevis config that uses TPM2
- Avoid logging spurious error when a LUKS volume wasn't previously formatted
- Fix version string in ignition-validate release container
- Fix reproducibility of systemd preset file in ignition-apply output
- Document that `user`/`group` fields aren't applied to hard links
- Clarify spec docs for `files`/`directories`/`links` `group` fields


## Ignition 2.14.0 (12-May-2022)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 36 key](https://getfedora.org/security/).

### Features

- Support KubeVirt platform
- Support AWS `arn:` URLs for S3 objects and access points _(3.4.0-exp)_
- Support reading configs from Azure IMDS "user data"
- Support S3 fetch via IPv6
- Add `ignition-apply` entrypoint to apply an Ignition config in a container

### Changes

- Delete userdata after provisioning on VirtualBox and VMware by default
  (see operator notes for details) (GHSA-hj57-j5cw-2mwp, CVE-2022-1706)
- Support setting setuid/setgid/sticky mode bits _(3.4.0-exp)_
- Warn if setuid/setgid/sticky mode bits specified _(3.0.0 - 3.3.0)_
- Support UEFI Secure Boot on VMware
- Add arm64 support to ignition-validate container
- Document S3 fetch semantics in operator notes
- Document considerations for handling secrets in operator notes

### Bug fixes

- Fix disabling systemd units with pre-existing enablement symlinks
- Fix reuse of statically keyed LUKS volumes (2.12.0 regression)
- Fix `gs://` fetch in GCE instances configured without a service account
- Fix error reading VirtualBox guest properties that have flags
- Fix infinite loop if `-root` command-line argument is a relative path


## Ignition 2.13.0 (30-Nov-2021)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 35 key](https://getfedora.org/security/).

### Features

- Add Nutanix provider
- Switch VirtualBox provider to read from `/Ignition/Config` guest property

### Changes

- Improve QEMU `fw_cfg` read performance
- Warn when QEMU `fw_cfg` config is too large for reasonable performance
- Move Ignition report to `/etc/.ignition-result.json`
- Improve resilience to filesystem unmount failures
- Run `mkfs.fat` instead of its alias `mkfs.vfat`
- Refresh supported platform documentation

### Bug fixes

- Make `ignition.version` required in JSON schema _(3.4.0-exp)_
- Disallow null `noProxy` array entries in JSON schema _(3.4.0-exp)_


## Ignition 2.12.0 (05-Aug-2021)

### Features

- Support Azure generation 2 VMs
- Write info about Ignition’s execution to `/var/lib/ignition/result.json`

### Changes

- Access GCP metadata service by IP address to mitigate DNS poisoning
  attacks
- Document `storage.luks.clevis.threshold` default
- Document minimum Ignition release for each spec version

### Bug fixes

- Fix permissions of mountpoints inside user home directories
- Apply SELinux labels to newly-created `ext4` filesystems

### Internal changes

- Drop `ignition-setup-user.service` and `ignition-firstboot-complete.service`
  in favor of distro-provided code
- Persist some state between Ignition stages using a file in `/run`
- Add command-line flag specifying path to `neednet` flag file
- Drop `-clear-cache` command-line flag
- Fix reboot race in example kargs helper
- Drop support for Go 1.13 and 1.14


## Ignition 2.11.0 (25-Jun-2021)

### Breaking changes

- Convert `ClevisCustom.Config`, `ClevisCustom.Pin`, `LinkEmbedded1.Target`,
  and `Raid.Level` Go fields to pointers _(3.3.0)_

### Features

- Accept `none` in `storage.filesystems.format` _(3.3.0)_
- Add `ParseCompatibleVersion()` Go functions to parse any config up to
  the selected version
- Add `powervs` platform

### Changes

- Mark the 3.3.0 config spec as stable
- No longer accept configs with version 3.3.0-experimental
- Create new 3.4.0-experimental config spec from 3.3.0
- Report specific reason an existing LUKS device cannot be reused
- Validate that `storage.raid.devices` is non-empty
- Don't sequence `ignition-setup-user.service` before `multipathd.service`

### Bug fixes

- Fix misleading error message if spares are requested for a RAID level
  that doesn't support them


## Ignition 2.10.1 (29-Apr-2021)

### Bug fixes

- Fix file mode of `ignition-kargs-helper` script


## Ignition 2.10.0 (29-Apr-2021)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 34 key](https://getfedora.org/security/).

### Breaking changes

- Rename `Custom` struct to `ClevisCustom` _(3.3.0-exp)_
- Embed `Clevis` and `ClevisCustom` structs in parents _(3.3.0-exp)_
- Always include interior nodes in merge transcript

### Features

- Add kernel argument support _(3.3.0-exp)_

### Bug fixes

- Fix fetching userdata on AWS when IMDSv1 is disabled
- Fix creating Tang-based LUKS volumes before network is up
- Document `storage.filesystems.wipeFilesystem` default


## Ignition 2.9.0 (08-Jan-2021)

### Changes

- Require `storage.filesystems.format` if `wipeFilesystem` or `mountOptions`
  is specified
- Refactor code to address golangci-lint warnings

### Bug fixes

- Fix fetching configs from S3 resources when running on non-default AWS
  partitions
- Fix fetching userdata from IMDSv2 on AWS
- Fix crash on partitions with no number or label
- Correctly document `storage.filesystems.path` as optional
- Clarify documented semantics of `systemd.units.name`


## Ignition 2.8.1 (02-Dec-2020)

### Bug fixes

- Correctly merge config fields behind a struct pointer (e.g. `clevis`)


## Ignition 2.8.0 (25-Nov-2020)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 33 key](https://getfedora.org/security/).

### Features

- Support unmasking systemd units

### Changes

- Switch system base config from single file to `.d` directory
- Add Go merge API that produces a transcript of merge operations


## Ignition 2.7.0 (13-Oct-2020)

### Features

- Support resizing existing partitions _(3.2.0)_
- Support reusing LUKS devices not bound to Clevis _(3.2.0)_

### Changes

- Mark the 3.2.0 config spec as stable
- No longer accept configs with version 3.2.0-experimental
- Create new 3.3.0-experimental config spec from 3.2.0
- Require presence of a config source on CloudStack/OpenStack, and
  wait indefinitely for it to appear
- When executing in non-default AWS partitions (GovCloud or AWS
  China), fetch `s3://` resources from the same partition

### Bug fixes

- Fix bundled library unconditionally blocking for entropy at startup
- Fix config fetching on AzureStack
- Fix partition offset/length calculation on big-endian systems
- Fix premature logging of successful config fetch


## Ignition 2.6.0 (07-Aug-2020)

### Features

- Add `release` tag to ignition-validate container for latest release
- Support creating ephemeral LUKS volumes _(3.2.0-exp)_
- Support deleting users/groups _(3.2.0-exp)_

### Bug fixes

- Request network when needed on CloudStack/OpenStack


## Ignition 2.5.0 (23-Jul-2020)

### Changes

- Merge ignition-dracut into the Ignition repository

### Bug fixes

- Fix udev race determining filesystem type when creating filesystem


## Ignition 2.4.1 (16-Jul-2020)

### Changes

- Set LUKS key file directory to mode 700

### Bug fixes

- Fix nondeterministic config provider precedence causing fetch failures
- Don't relabel symlink to home directory, since it might not be writable
- Fix failure looking up users/groups


## Ignition 2.4.0 (13-Jul-2020)

### Features

- Support creating LUKS volumes with Clevis or static key file _(3.2.0-exp)_
- Support Google Cloud Storage (`gs://`) resource URLs
- Support AWS IMDSv2
- Allow specifying multiple CA certificates in one resource
- Add Azure Stack platform
- Allow OS to avoid starting network if the config doesn't need it

### Changes

- When creating a filesystem, run `wipefs` on target device first
- Warn if filesystem probe finds multiple filesystem signatures
- Don't warn about unset file/directory mode in config
- Fetch AWS metadata version 2019-10-01 instead of 2009-04-04
- Refactor SELinux relabeling

### Bug fixes

- Fix compressed CA certificates
- Fix hard links to files deeper than the hard link
- Write empty systemd dropin if requested
- Remember to relabel `/etc/systemd/system-preset`
- Ensure configs are only fetched during fetch stage
- Clarify docs about interaction between file contents and overwrite


## Ignition 2.3.0 (05-May-2020)

Starting with this release, ignition-validate binaries are signed with the
[Fedora 32 key](https://getfedora.org/security/).

### Features

- Allow specifying HTTP headers when fetching remote resources _(3.1.0)_
- Support compression for CA certs and merged/replaced configs _(3.1.0)_
- Support `sha256` verification hashes _(3.1.0)_
- Support compression for `data` URIs
- Log structured journal entry when user config is found
- Log structured journal entry when SSH keys are written

### Changes

- Unify `CaReference`, `ConfigReference`, `FileContents` structs into
  `Resource` _(3.1.0)_
- Mark the 3.1.0 config spec as stable
- No longer accept configs with version 3.1.0-experimental
- Create new 3.2.0-experimental config spec from 3.1.0

### Bug fixes

- Fix `ignition-validate` for config versions other than 3.0.0
- Fix config fetch and status reporting on Packet


## Ignition 2.2.1 (24-Mar-2020)

### Bug fixes

- Fix build failure on arches other than amd64, arm64, ppc64le, or s390x


## Ignition 2.2.0 (23-Mar-2020)

### Features

- Add Exoscale and Vultr providers
- On QEMU/s390x and QEMU/ppc64le, fetch Ignition config from a virtio
  block device (experimental)

### Changes

- Don't relabel `/root` and `/home`

### Bug fixes

- Fix enabling systemd instantiated services
- Fail if SSH keys cannot be written
- Fix partition creation on s390x


## Ignition 2.1.1 (13-Dec-2019)

### Bug fixes

- Fix panics when processes Ignition starts fail

### Features

- An ignition-validate container is now built and can be used instead of
  the ignition-validate binaries


## Ignition 2.1.0 (12-Dec-2019)

### Bug fixes

- Do not panic when filesystem paths are unspecified
- Specify the correct config version HTTP `Accept` headers when fetching
  configs
- Write the config cache file atomically
- Relabel symlinks for masking systemd units
- Fix bug where empty GPT labels were treated as errors
- Do not generate warnings if mode is unset for files with only an `append`
  section
- Validate HTTP(S) proxy urls in spec 3.1.0-experimental

### Features

- Ignition now logs the name of the stage it is running
- Ignition now relabels files directly instead of writing systemd units to
  do so. Requires Linux 5.4.0+ or a patch. See operator notes for more
  details
- Add optional `fetch` stage to cache the rendered config, but not apply
  any of it
- Add support for `aliyun` cloud
- Add support for zVM hypervisor
- Add support for specifying mount options for filesystems in spec
  3.1.0-experimental

### Dependency changes

- Ignition no longer needs the `chroot` or `id` binaries in the initramfs


## Ignition 2.0.1 (24-Jul-2019)

### Bug fixes

- Fix getting AWS region when networking is slow to come up
- Validate file/directory paths correctly


## Ignition 2.0.0 (03-Jun-2019)

### Bug fixes

- Use `/run/ignition/dev_aliases` instead of `/dev_aliases` when creating our
  own symlinks to devices in `/dev`

### Test changes

- Rename tests to use `dots.with.lowercase`

### Public Go API changes

- Replace `config/validate` api with `github.com/coreos/vcontext`
- `Validate()` functions in `config/*` now follow the `vcontext` validation
  interface


## Ignition 2.0.0-beta (26-Apr-2019)

### Features

- Add configuration spec 3.1.0-experimental
- Allow specifying HTTP(S) proxies in spec 3.1.0-experimental
- Validate hard links do not link to directories
- Validate paths do not include links specified in the config

### Bug fixes

- Include major version in `go.mod` correctly
- Fix SELinux relabeling of systemd unit files
- Update documentation for spec 3.0.0+

### Changes

- Remove all deprecated fields in configuration specs
- Remove `ec2` platform id in favor of `aws`
- Remove `pxe` platform as it is not a platform
- Fail if files, links, and directories conflict after symlink resolution
- Do not fail when writing directories or links if overwrite is false and a
  matching directory or link already exists


## Ignition 2.0.0-alpha (25-Mar-2019)

**NOTE**: This is an alpha release. While the spec is marked as stable (i.e no
"-experimental" suffix) we still reserve the right to change it until the
stable 2.0.0 release. However, we do not anticipate any backwards
incompatible changes aside from removing deprecated fields.

**NOTE**: In order to allow types from both the 2.x.y and 3.0.0 specs to be
vendored and imported in the same project, we are skipping version 1.0.0.
Go mod (and some other tools) treat v0.x and v1.x as the same when
importing packages with semantic import versioning.

### Features

- Ignition now understands config specification 3.0.0
- Configs are now merged instead of appended

### Changes

- Configs with version < 3.0.0 are now rejected
- Duplicate entries are now disallowed in lists
- Removal of almost all deprecated fields


## Ignition 0.30.0 (14-Dec-2018)

### Features

- Parallelize filesystem creation

### Changes

- Increase default config fetch timeout to 2 minutes

### Test changes

- Add `-list` option to list blackbox tests
- Skip backward compatibility tests with `-test.short`


## Ignition 0.29.1 (06-Dec-2018)

### Bug fixes

- When writing files, directories, or links, do not follow symlinks if they
  are the last path element


## Ignition 0.29.0 (30-Nov-2018)

### Features

- Add support for `?versionId` on `s3://` URLs

### Changes

- Mark the 2.3.0 config spec as stable
- No longer accept configs with version 2.3.0-experimental
- Create new 2.4.0-experimental config spec from 2.3.0

### Bug fixes

- Don't allow HTTPS connections to block on system entropy pool
- Relabel `/var/home` and `/var/roothome` when SELinux is enabled
- Fix race where files were relabeled after `systemd-sysctl.service`
- Do not run `udevadm settle` after the disks stage if the disks stage did
  nothing
- Allow writing relative symlinks
- Resolve absolute symlinks relative to specified filesystem instead of the
  initramfs root
- Report status to Packet as `running` instead of `succeeded`

### Test changes

- Fix race with `umount` when running blackbox tests


## Ignition 0.28.0 (22-Aug-2018)

### Features

- Refactor blackbox tests to allow testing disks with 4k sectors

### Bug fixes

- Correctly detect disks with 4k sectors when scanning existing partitions
- Fix race between HTTP backoff tests
- Set the minimum config versions in tests to the actual minimum required
- Relabel `/root` when SELinux relabeling is enabled


## Ignition 0.27.0 (09-Aug-2018)

### Features

- Ignition is now built as a Position Independent Executable (PIE)
- Blackbox tests now run against all spec versions (within the same major
  version) greater than their minimum version
- Ignition now reports its status when running on Packet
- Add a compile-time flag to enable SELinux file relabeling after boot

### Bug fixes

- Directories specified in both base and appended configs are always
  created with the permissions specified in the appended config
- Call `chdir()` after `chroot()` to silence static checkers


## Ignition 0.26.0 (11-June-2018)

### Features

- Support partition matching, specifying that a partition should not
  exist, and recreating existing partitions
- Fail blackbox tests when Ignition encounters critical-level logs

### Bug fixes

- Fix an issue in timeout logic causing http(s) requests to sometimes fail
- Do not log non-critical errors with `CRITICAL` log level


## Ignition 0.25.1 (22-May-2018)

### Bug fixes

- Fix an issue in timeout logic causing http(s) requests to sometimes fail


## Ignition 0.24.1 (22-May-2018)

### Bug fixes

- Fix an issue in timeout logic causing http(s) requests to sometimes fail


## Ignition 0.25.0 (17-May-2018)

### Features

 - Blackbox tests can now be run in parallel

### Changes

 - Remove Oracle Cloud Infrastructure support

### Bug fixes

 - No longer leave a stray file when appending to an existing file
 - Fix multiple blackbox test validation errors
 - Fix v1 config parsing to return `ErrUnknownVersion` if version is
   unrecognized


## Ignition 0.24.0 (06-Mar-2018)

### Features

- Warn when adding and enabling a systemd unit and there is no `Install`
  section in the unit contents
- Add highlights to reports generated by `Validate` functions on config
  structs

### Changes

- Move a helper validation function to the `config/validate` package
- Move unit validation helpers to `config/shared/validations`
- Add common error types to `config/shared/errors`, refactor `config/v*` to
  use these errors


## Ignition 0.23.0 (12-Mar-2018)

### Changes

- Latest experimental package has been moved from `config/types` to
  `config/v2_3_experimental`.
- Each `config` package's `Parse` function will now transparently handle any
  configs of a lesser version than itself (e.g. `config/v2_2` will handle a
  2.0.0 config).
- Validation in `config/v1` reworked to use `config/validate`.
- Common error types from the `config` package moved to `config/errors`.


## Ignition 0.22.0 (09-Feb-2018)

### Changes

- Mark the 2.2.0 config spec as stable
- No longer accept configs with version 2.2.0-experimental
- Create new 2.3.0-experimental config spec from 2.2.0


## Ignition 0.21.0 (26-Jan-2018)

### Features

- Add support for networkd drop-ins
- Add new program, `ignition-validate`, for validating Ignition configs
- Add `overwrite` field to `files`, `directories`, and `links` sections for
  deleting preexisting items at the node's path
- Add `options` field to `raid` section for specifying arbitrary `mdadm`
  options
- Add `append` field to `files` section for appending to preexisting files
- Add support for specifying additional certificate authorities to use when
  fetching objects over HTTPS

### Changes

- Validate that partition labels don't contain colons, as `sgdisk` will
  silently truncate the label
- Remove `-validate` flag from Ignition that was introduced in 0.20.0
- Warn when the mode for a file or directory is unset
- Log retries of HTTP fetches at `info` loglevel so messages appear on console

### Bug fixes

- Fix issue where unspecified fields in an appended config could "unset"
  fields specified in a config earlier in the chain
- Use timeouts specified in a config when fetching other configs referenced
  by it


## Ignition 0.20.1 (12-Jan-2018)

### Changes

- Add support for fetching S3 objects from non-default AWS partitions when
  running in one such partition


## Ignition 0.20.0 (13-Dec-2017)

### Features

- Add `validate` flag for validating Ignition configs without running any
  stages
- Add support for reading user configs from initramfs

### Changes

- Move `update-ssh-keys` from dependency into internal library
- Move constants such as paths for invoked binaries into dedicated package
  to allow for easy overriding at link time
- Read base and default configs from initramfs instead of hardcoding them
- Use the golang DNS resolver instead of the default glibc DNS resolver


## Ignition 0.19.0 (22-Sep-2017)

### Features

- Add support for CloudStack network metadata
- Add blackbox tests for TFTP URLs
- Remove dependency on `kpartx` for blackbox tests

### Changes

- Stop adding extra quotes around GECOS field when creating users

### Bug fixes

- Fix regression in validation logic causing inaccurate line and column
  reporting
- Fix regression in validation logic where JSON syntax errors were not
  reported correctly
- Add warning if a non-existent filesystem is specified when creating links
  and directories
- Fix udev race causing systemd units depending on the Ignition disks stage and
  a device unit to fail when no filesystems are created
- Fix udev race where symlinks are deleted before Ignition can create its own
  copy


## Ignition 0.18.0 (08-Sep-2017)

### Features

- On VMWare allow guest variables to override values specified in the OVF
  environment
- Add partial support for CloudStack
- Add blackbox tests
- Add support for Oracle OCI provider

### Changes

- Chmod pre-existing directories to match defined permissions in config
- Chown pre-existing links to match defined owner in config
- Add `--homehost any` arguments to `mdadm` raid creation to ensure consistent
  device name under `/dev/md`
- On GCE, don't bind-mount `docker` binary into Google Cloud SDK container
- On GCE, remove `gcutil` alias

### Bug fixes

- Properly error out when a user or group set by name in the config cannot
  be resolved to an id
- Fix typo in `gcloud` alias preventing connection to the docker daemon in
  some cases
- Fix partition number validation where multiple partitions on a disk were
  unable to specify 0 for the next available partition number


## Ignition 0.17.2 (28-Jul-2017)

### Bug fixes

- Fix failure to create files/directories/links on correct filesystem
- Fix failure to force filesystem creation when legacy `force` flag was set
- Prevent VFAT filesystem creation from unconditionally overwriting existing
  filesystem
- Fix deprecation warning on `enable` field in OEM systemd units
- Fix failure where hard link targets would be on incorrect filesystem,
  causing creation to fail
- Fix incorrect filesystem UUID check when deciding whether to reuse
  existing filesystem, causing Ignition to fail


## Ignition 0.17.1 (05-Jul-2017)

### Bug fixes

- Fix failure when user data was not provided on EC2 and GCE
- Fix failure to fetch user data on packet.net


## Ignition 0.17.0 (30-Jun-2017)

### Features

- Add support for S3 fetching and IAM role credential use in EC2
- Add `enabled` flag to services to allow disabling services
- Add new `vagrant-virtualbox` oem

### Changes

- Mark 2.1.0 as stable
- No longer accept 2.1.0-experimental configs
- Create new 2.2.0-experimental spec from 2.1.0

### Bug fixes

- Mask `user-configdrive.service` and `user-configvirtfs.service` on
  `brightbox` and `openstack` to prevent cloudinit from running a second time
- Use value given in `root` flag everywhere, instead of hard coding `/sysroot`


## Ignition 0.16.0 (16-Jun-2017)

### Experimental (2.1.0-experimental)

- Fix TFTP URL validation
- Fix nil pointer dereference when uid or gid for a file is unspecified
- Add support for VFAT filesystem creation
- Fix `raid` device validation

### Changes

- Validate length of filesystem labels
- Remove all OEM etcd v0 drop-in units
- Remove `xendom0` OEM

### Features

- Add support for VMware's OVF environment
- Add support for VirtualBox OEM


## Ignition 0.15.0 (23-May-2017)

### Experimental (2.1.0-experimental)

- Define the Ignition Config schema in a JSON Schema file. Generate golang
  structs from this file
- Add partition GUID to the filesystem object, create or modify the
  partition as appropriate
- Add support for `swap` filesystems
- Add support for links, both symbolic and hard
- Deprecate the user level `create` object, add relevant fields directly to
  the user object
- Add support for referencing users and groups by name when creating files,
  directories, and links
- Deprecate the filesystem level `create` object, add relevant fields directly
  to the filesystem object
- Add support for reusing existing filesystems, toggled via the new
  `wipeFilesystem` field in the filesystem object
- Add filesystem UUID and label to the filesystem object
- Correctly handle timeouts, instead of ignoring timeout settings in the
  Ignition config

### Bug fixes

- Fix file path validation on Windows
- On Brightbox correctly fetch the config, instead of failing with a noop
- Fix a race with udev events which could cause filesystem creation to fail

### Changes

- Modify existing users, instead of attempting to create them

### Features

- Support for TFTP URLs


## Ignition 0.14.0 (13-Mar-2017)

### Changes

- Update the services for the Azure OEM
- Update the services for the BrightBox OEM
- Update the services for the EC2 OEM
- Update the services for the OpenStack OEM
- Update the services for the Packet OEM
- Update the services for the VMware OEM


## Ignition 0.13.0 (01-Mar-2017)

### Bug fixes

- Read from both the config-drive and metadata service when using the
  OpenStack provider
- Properly reports errors encountered while creating files
- Fix GCE `gcloud` alias to properly invoke the container

### Features

- Add support for experimental features via a newer config spec
- Allow file provider's config path to be overridden
- Perform basic syntactic validation on the contents of systemd units

### Experimental (2.1.0-experimental)

- Add ability to explicitly create directories
- Add configuration for HTTP-related timeouts


## Ignition 0.12.1 (14-Dec-2016)

### Bug fixes

- Enable `coreos-metadata-sshkeys` on Packet
- Assert validity of `data` URLs during config validation


## Ignition 0.12.0 (29-Nov-2016)

### Features

- Allow kernel command-line parameter to override OEM config


## Ignition 0.11.2 (07-Oct-2016)

### Bug fixes

- Correctly set the partition typecode

### Changes

- Update the services for the GCE OEM


## Ignition 0.11.1 (20-Sep-2016)

### Bug fixes

- Fix potential deadlock when waiting for multiple disks


## Ignition 0.11.0 (07-Sep-2016)

### Features

- Add support for DigitalOcean
- Add experimental support for OpenStack


## Ignition 0.10.1 (26-Aug-2016)

### Bug fixes

- Fix handling of `oem://` URLs
- Use stable symlinks when operating on devices
- Retry failed requests when fetching Packet userdata
- Log the raw configurations instead of the parsed result


## Ignition 0.10.0 (23-Aug-2016)

### Features

- Add support for QEMU Firmware Configuration Device


## Ignition 0.9.2 (15-Aug-2016)

### Bug fixes

- Do not retry HTTP requests that result in non-5xx status codes


## Ignition 0.9.1 (11-Aug-2016)

### Bug fixes

- Properly validate `data` URLs


## Ignition 0.9.0 (11-Aug-2016)

### Features

- Add detailed configuration validation

### Bug fixes

- Add retry to all HTTP requests
- Fix potential panic when parsing certain URLs


## Ignition 0.8.0 (26-Jul-2016)

### Features

- Add support for Packet


## Ignition 0.7.1 (13-Jul-2016)

### Bug fixes

- Interpret files without a URL to be empty instead of invalid
- HTTP fetches time out while waiting for response header instead of body
- Stream remote assets to disk instead of loading them into memory

### Changes

- Improve configuration validation


## Ignition 0.7.0 (15-Jun-2016)

### Features

- Allow HTTPS URLs

### Bug fixes

- Don't overwrite existing data when formatting `ext4` unless `force` is set
- Ensure service unit in `/etc` doesn't exist before masking
- Capture and log stdout of subprocesses

### Changes

- Drop YAML tags from the config package


## Ignition 0.6.0 (18-May-2016)

### Features

- All URL schemes (currently `http`, `oem`, and `data`) are now supported
  everywhere a URL can be provided
- Add base OEM and default user configurations for GCE


## Ignition 0.5.0 (04-May-2016)

### Features

- Add support for GCE

### Bug fixes

- Write files after users and home directories are created

### Changes

- Strip support for EC2 SSH keys (these are handled by coreos-metadata now)
- Add OEM-specific base configs and execute even if user config is empty


## Ignition 0.4.0 (05-Apr-2016)

### Features

- Update the config spec to v2.0.0 (see the migration guide for more info)
  - v1 configs will be automatically translated to v2.0.0
- Add HTTP `User-Agent` and `Accept` headers to all requests

### Changes

- Use Go's vendor directory for all dependencies
- Split source into a public `config` package and `internal`


## Ignition 0.3.3 (25-Mar-2016)

### Bug fixes

- Fix compilation errors when building for ARM
- Properly fetch configs from EC2


## Ignition 0.3.2 (17-Mar-2016)

### Bug fixes

- Properly decode VMware guest variables before parsing config

### Changes

- Move config structures from `config` package to `config/types`


## Ignition 0.3.1 (02-Mar-2016)

### Bug fixes

- Allow building on non-AMD64 architectures

### Changes

- Major refactoring of the internal processing of OEMs and providers


## Ignition 0.3.0 (24-Feb-2016)

### Features

- Add support for VMware


## Ignition 0.2.6 (13-Jan-2016)

### Features

- Improve validation of `storage.filesystems` options

### Bug fixes

- Properly zap GPT tables when they are partially valid


## Ignition 0.2.5 (06-Jan-2016)

### Bug fixes

- Recognize and ignore gzipped cloud-configs


## Ignition 0.2.4 (19-Nov-2015)

### Bug fixes

- Correctly escape device unit names


## Ignition 0.2.3 (17-Nov-2015)

### Features

- Provide logging to pinpoint JSON errors in invalid configs

### Bug fixes

- Ensure that `/mnt/oem` exists before mounting
- Remove `/sysroot/` prefix from alternate config path


## Ignition 0.2.2 (20-Oct-2015)

### Bug fixes

- Mount the oem partition for `oem://` schemes when needed


## Ignition 0.2.1 (15-Oct-2015)

### Bug fixes

- Allow empty CustomData on Azure


## Ignition 0.2.0 (29-Sep-2015)

### Features

- Added support for Azure
- Added support for formatting partitions as `xfs`

### Bug fixes

- Removed online timeout for EC2


## Ignition 0.1.6 (09-Sep-2015)

### Features

- `--fetchtimeout` becomes `--online-timeout`
- `--online-timeout` of 0 now represents infinity
- Added recognition of `interoute` OEM

### Documentation

- Examples have been removed and supported platforms added
- Various minor cleanups

### Bug fixes

- Ensure added SSH keys are newline terminated

### Build system changes

- Fix `gofmt` invocation from test script to fail when appropriate


## Ignition 0.1.5 (28-Aug-2015)

### Bug fixes

- Disable EC2 provider for now


## Ignition 0.1.4 (27-Aug-2015)

### Features

- Add support for `oem://` scheme config urls

### Documentation

- Added guides
- Updated config specification

### Bug fixes

- Add `DefaultDependencies=false` to `WaitOnDevices()` transient unit
- Updated JSON configuration keys to match style

### Build system changes

- Added script for tagging releases


## Ignition 0.1.3 (11-Aug-2015)

### Features

- Add support for ssh keys on EC2
- Log version at runtime

### Bug fixes

- Log ssh keys as they are added
- Various small cleanups

### Build system changes

- Derive version from `git describe` at build time
- Use `bash` build and test scripts instead of `make`


## Ignition 0.1.2 (22-Jul-2015)

### Bug fixes

- Fix validation of drop-in names
- Properly handle a lack of userdata on EC2


## Ignition 0.1.1 (22-Jul-2015)

### Bug fixes

- Ignore empty configs
- Ignore unsupported CoreOS OEMs
- Panic on incorrect OEM flag configurations


## Ignition 0.1.0 (14-Jul-2015)

### Features

- Initial release of Ignition!
- Support for disk partitioning, partition formatting, writing files,
  RAID, systemd units, networkd units, users, and groups.
- Supports reading the config from a remote URL (via
  `config.coreos.url`) or from the Amazon EC2 metadata service.
