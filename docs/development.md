---
nav_order: 11
---

# Development
{: .no_toc }

1. TOC
{:toc}

A Go 1.20+ [environment](https://golang.org/doc/install) and the `blkid.h` headers are required.

```sh
# Debian/Ubuntu
sudo apt-get install libblkid-dev

# RPM-based
sudo dnf install libblkid-devel
```

## Development notes

See also the [Ignition rationale](rationale.md).

### Code structure

The [frontend](https://github.com/coreos/ignition/tree/main/config) handles config parsing and validation which need not run on the target system.  The [backend](https://github.com/coreos/ignition/tree/main/internal) performs the configuration of the target system.  The frontend is a stable library API that is used by other programs, so existing frontend API cannot be changed without bumping the Ignition major version.

### Adding functionality

New config directives should only be added if the desired behavior cannot reasonably be achieved with existing directives.  User-friendly wrappers for existing syntax ("sugar") should be handled by [Butane](https://github.com/coreos/butane).

New behavior should only be added in the current experimental spec.  If new functionality is backported to older specs, and a config using an older spec comes to depend on that functionality, then it won't be obvious that the config will not work on all Ignition versions supporting that spec.  It's not always possible to follow this restriction, since the backend doesn't know what config version the user specified.  Where possible, use config validation to prevent the backend from seeing config directives that a spec version doesn't support (for example, values of the filesystem `format` field).

New functionality added to a config spec must be declarative: it must describe what should exist, not what Ignition should do.  In particular, field names should not include verbs.  When adding functionality, carefully think through the interactions between features, which can be non-trivial.  Features should be orthogonal, minimal, and low-level; making them user-friendly is the responsibility of Butane sugar.

When reprovisioning an existing node, the config may want to reuse existing disks and filesystems without reformatting them.  Config directives should support detecting and reusing an existing object (RAID volume, filesystem, etc.) if its properties match those specified in the config.

Ignition specs should not include distro-specific functionality such as package management.  Features may require support from the distro (for example, setting kernel arguments), but such features should be broadly applicable.  Distro-specific options such as support for SELinux, or paths to external binaries, can be configured at build time in the [`distro`](https://github.com/coreos/ignition/blob/main/internal/distro/distro.go) package.  Distro-specific glue (e.g. support for reformatting the root filesystem) should be implemented outside the Ignition codebase, in Dracut modules that run between Ignition stages (see below).

Ideally, functionality should not be added to an experimental spec in the same Ignition release that the spec is stabilized.  Doing so prevents users from trying out the functionality before we commit to maintaining it.

### Modifying existing functionality

Bugfixes can be backported to older specs if working configs will not be affected and the current behavior is unintended.  New config validations can (and should) be backported if the prohibited behavior always would have failed anyway.

Existing config semantics can only be changed (without adding a flag field) if the spec major version is bumped.  This might be appropriate e.g. to clean up some awkward syntax or change some default behavior.  However, altogether removing functionality is very costly.  The spec 2.x to 3.x transition was difficult because some 2.x configs cannot be represented in spec 3.x, so users were required to manually update their configs.  If a major version bump only rearranges functionality but doesn't remove any, Ignition can automatically translate the previous major version to the current one, and no [flag day](https://en.wikipedia.org/wiki/Flag_day_(computing)) will be required.

### Validation and failures

Ignition is a low-level tool and does not attempt to prevent configs from doing unreasonable things.

- Config validation should fail when it's clear from inspection that Ignition will be unable to perform the requested provisioning.  This must occur when the config is contradictory (e.g. specifies the same filesystem object as both a file and a directory) and thus would break the declarativeness of the spec, and may occur where the problem will produce a routine error at runtime (e.g. invalid arguments to `mkfs`).
- Ignition must fail at runtime when it is dynamically unable to perform the requested provisioning.
- If a config is implementable but will render the system non-functional, Ignition should execute the config anyway.  Any user-friendly detection of unreasonable configs should happen in Butane.

### Platforms

Platform providers should continue retrying config fetch until it's clear whether the user has provided an Ignition config.  If Ignition eventually timed out when fetching a config, then a slow network device or block device (on a very large machine or a heavily-loaded VM host) could cause Ignition to prematurely fail, or to continue booting without applying the specified config.

Platform providers must allow the user not to provide a config, e.g. to boot an exploratory OS instance and use Afterburn to inject SSH keys.

Ignition must never read from config providers that aren't under the control of the platform, since this could allow config injection from unintended sources.  For example, `169.254.169.254` is a link-local address and could easily be spoofed on platforms that don't specially handle that address.  As a corollary, the platform ID must always be explicitly set by the OS image, never guessed.

### Network access

All config fields that cause network accesses, directly or indirectly, should be added to the `fetch-offline` needs-net detector.

Any network accesses performed by subprocesses should be mimicked by Ignition before starting the subprocess.  This ensures that Ignition's retry logic is used, so Ignition doesn't improperly fail if the network is still coming up.

### Execution stages

Ignition execution is divided into stages to allow other OS functionality to run in the middle of provisioning:

1. `fetch-offline` stage
1. OS enables networking if required
1. `fetch` stage
1. OS examines fetched config and e.g. copies root filesystem contents to RAM
1. `disks` stage
1. OS mounts root filesystem
1. `mount` stage
1. OS does any preprocessing of configured filesystems, including copying root filesystem contents back
1. `files` stage
1. OS does any postprocessing of provisioned system
1. `umount` stage

New stages should only be created when OS hooks would otherwise need to run in the middle of a stage.  Note that various external projects hardcode the list of Ignition stages.

### Security

Ignition must always provide secure defaults, and does not provide config directives that support or encourage unsafe behavior.  For example, Ignition does not support disabling HTTPS certificate checks, nor seeding the system entropy pool from potentially deterministic sources.  Similarly, the LUKS `discard` option exists because users may legitimately want to trade off some security for hardware longevity, but Ignition defaults to the secure option.

Users might put secrets in Ignition configs.  On many platforms, userdata is accessible to unprivileged programs at runtime, potentially leaking those secrets.  When the userdata is accessible via a network service, users can configure firewall rules to prevent such access, but this may not be possible for hypervisors that expose userdata through a kernel interface.  Where possible, platform providers should provide a `DelConfig` method allowing Ignition to delete the userdata from the platform after provisioning is complete.

## Modifying the config spec

Install [schematyper](https://github.com/idubinskiy/schematyper) to generate Go structs from JSON schema definitions.

```sh
go get -u github.com/idubinskiy/schematyper
```

Modify `config/v${LATEST_EXPERIMENTAL}/schema/ignition.json` as necessary. This file adheres to the [json schema spec](http://json-schema.org/).

Run the `generate` script to create `config/vX_Y/types/schema.go`. Once a configuration is stabilized (i.e. it is no longer `-experimental`), it is considered frozen. The json schemas used to create stable specs are kept for reference only and should not be changed.

```sh
./generate
```

Add whatever validation logic is necessary to `config/v${LATEST_EXPERIMENTAL}/types`, modify the translator at `config/v${LATEST_EXPERIMENTAL}/translate/translate.go` to handle the changes if necessary, and update `config/v${LATEST_EXPERIMENTAL/translate/translate_test.go` to properly test the changes.

Finally, make whatever changes are necessary to `internal` to handle the new spec.

## Vendor

Ignition uses go modules. Additionally, we keep all of the dependencies vendored in the repo. This has a few benefits:
 - Ensures modification to `go.mod` is intentional, since `go build` can update it without `-mod=vendor`
 - Ensures all builds occur with the same set of sources, since `go build` will only pull in sources for the targeted `GOOS` and `GOARCH`
 - Simplifies packaging in some cases since some package managers restrict network access during compilation.

After modifying `go.mod` run `make vendor` to update the vendor directory.

Group changes to `go.mod`, `go.sum` and `vendor/` in their own commit; do not make code changes and vendoring changes in the same commit.

## Testing

Ignition uses three different test frameworks:

- Unit tests (`./test`) validate functionality that only affects internal program state.
- Blackbox tests validate config directives that affect the target disk.
- Fedora CoreOS [kola tests](https://coreos.github.io/coreos-assembler/kola/) validate functionality that interacts with platforms (e.g. config fetching) or the rest of the OS.  kola tests may be [internal](https://github.com/coreos/coreos-assembler/tree/main/mantle/kola/tests/ignition) or [external](https://github.com/coreos/fedora-coreos-config/tree/testing-devel/tests/kola/ignition).

### Running blackbox tests

```sh
./build_blackbox_tests
sudo sh -c 'PATH=$PWD/bin/amd64:$PATH ./tests.test'
```

To run a subset of the blackbox tests, pass a regular expression into `-test.run`. As an example:

```
sudo sh -c 'PATH=$PWD/bin/amd64:$PATH ./tests.test -test.run TestIgnitionBlackBox/Preemption.*'
```

You can get a list of available tests to run by passing the `-list` option, like so:

```
sudo sh -c 'PATH=$PWD/bin/amd64:$PATH ./tests.test -list'
```

### Blackbox test host system dependencies

The following packages are required by the blackbox tests:

* `util-linux`
* `dosfstools`
* `e2fsprogs`
* `btrfs-progs`
* `xfsprogs`
* `gdisk`
* `coreutils`
* `mdadm`
* `libblkid-devel`

### Writing blackbox tests

To add a blackbox test create a function which yields a `Test` object. A `Test` object consists of the following fields:

- `Name`: `string`
- `In`: `[]Disk` object, which describes the Disks that should be created before Ignition is run.
- `Out`: `[]Disk` object, which describes the Disks that should be present after Ignition is run.
- `MntDevices`: `MntDevice` object, which describes any disk related variable replacements that need to be done to the Ignition config before Ignition is run. This is done so that disks which are created during the test run can be referenced inside of an Ignition config.
- `SystemDirFiles`: `[]File` object which describes the Files that should be written into Ignition's system config directory before Ignition is run.
- `Config`: `string` type where the specific config version should be replaced by `$version` and will be updated before Ignition is run.
- `ConfigMinVersion`: `string` type which describes the minimum config version the test should be run with. Copies of the test will be generated for every version, inside the same major version, that is equal to or greater than the specified ConfigMinVersion. If the test should run only once with a specfic config version, leave this field empty and replace $version in the `Config` field with the desired version.

The test should be added to the init function inside of the test file. If the test module is being created then an `init` function should be created which registers the tests and the package must be imported inside of `tests/registry/registry.go` to allow for discovery.

UUIDs may be required in the following fields of a `Test` object: `In`, `Out`, and `Config`. Replace all GUIDs with GUID varaibles which take on the format `$uuid<num>` (e.g. $uuid123). Where `<num>` must be a positive integer. GUID variables with identical `<num>` fields will be replaced with identical GUIDs. For example, look at [tests/positive/partitions/zeros.go](https://github.com/coreos/ignition/blob/main/tests/positive/partitions/zeros.go).

## Releasing Ignition

Create a new [release checklist](https://github.com/coreos/ignition/issues/new?labels=kind/release&template=release-checklist.md) and follow the steps there.

## The build process

Note that the `build` script included in this repository is a convenience script only and not used for the actual release binaries. Those are built using an `ignition.spec` maintained in [Fedora rpms/ignition](https://src.fedoraproject.org/rpms/ignition). (The `ignition-validate` [container](https://quay.io/repository/coreos/ignition-validate) is built by the `build_for_container` script, which is not further described here.)
This build process uses the [go-rpm-macros](https://pagure.io/go-rpm-macros) to set up the Go build environment and is subject to the [Golang Packaging Guidelines](https://docs.fedoraproject.org/en-US/packaging-guidelines/Golang/).

Consult the [Package Maintenance Guide](https://docs.fedoraproject.org/en-US/package-maintainers/Package_Maintenance_Guide/) and the [Pull Requests Guide](https://docs.fedoraproject.org/en-US/ci/pull-requests/) if you want to contribute to the build process.

In case you have trouble with the aforementioned standard Pull Request Guide, consult the Pagure documentation on the [Remote Git to Pagure pull request](https://docs.pagure.org/pagure/usage/pull_requests.html#remote-git-to-pagure-pull-request) workflow.

## Marking an experimental spec as stable

Create a new [stabilization checklist](https://github.com/coreos/ignition/issues/new?template=stabilize-checklist.md) and follow the steps there.
