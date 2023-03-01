---
nav_order: 11
---

# Development
{: .no_toc }

1. TOC
{:toc}

A Go 1.18+ [environment](https://golang.org/doc/install) and the `blkid.h` headers are required.

```sh
# Debian/Ubuntu
sudo apt-get install libblkid-dev

# RPM-based
sudo dnf install libblkid-devel
```

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

## Running Blackbox Tests

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

## Test Host System Dependencies

The following packages are required by the Blackbox Test:

* `util-linux`
* `dosfstools`
* `e2fsprogs`
* `btrfs-progs`
* `xfsprogs`
* `gdisk`
* `coreutils`
* `mdadm`
* `libblkid-devel`

## Writing Blackbox Tests

To add a blackbox test create a function which yields a `Test` object. A `Test` object consists of the following fields:

Name: `string`

In: `[]Disk` object, which describes the Disks that should be created before Ignition is run.

Out: `[]Disk` object, which describes the Disks that should be present after Ignition is run.

MntDevices: `MntDevice` object, which describes any disk related variable replacements that need to be done to the Ignition config before Ignition is run. This is done so that disks which are created during the test run can be referenced inside of an Ignition config.

SystemDirFiles: `[]File` object which describes the Files that should be written into Ignition's system config directory before Ignition is run.

Config: `string` type where the specific config version should be replaced by `$version` and will be updated before Ignition is run.

ConfigMinVersion: `string` type which describes the minimum config version the test should be run with. Copies of the test will be generated for every version, inside the same major version, that is equal to or greater than the specified ConfigMinVersion. If the test should run only once with a specfic config version, leave this field empty and replace $version in the `Config` field with the desired version.

The test should be added to the init function inside of the test file. If the test module is being created then an `init` function should be created which registers the tests and the package must be imported inside of `tests/registry/registry.go` to allow for discovery.

UUIDs may be required in the following fields of a Test object: In, Out, and Config. Replace all GUIDs with GUID varaibles which take on the format `$uuid<num>` (e.g. $uuid123). Where `<num>` must be a positive integer. GUID variables with identical `<num>` fields will be replaced with identical GUIDs. For example, look at [tests/positive/partitions/zeros.go](https://github.com/coreos/ignition/blob/main/tests/positive/partitions/zeros.go).

## Releasing Ignition

Create a new [release checklist](https://github.com/coreos/ignition/issues/new?labels=kind/release&template=release-checklist.md) and follow the steps there.

## The build process

Note that the `build` script included in this repository is a convenience script only and not used for the actual release binaries. Those are built using an `ignition.spec` maintained in [Fedora rpms/ignition](https://src.fedoraproject.org/rpms/ignition). (The `ignition-validate` [container](https://quay.io/repository/coreos/ignition-validate) is built by the `build_for_container` script, which is not further described here.)
This build process uses the [go-rpm-macros](https://pagure.io/go-rpm-macros) to set up the Go build environment and is subject to the [Golang Packaging Guidelines](https://docs.fedoraproject.org/en-US/packaging-guidelines/Golang/).

Consult the [Package Maintenance Guide](https://docs.fedoraproject.org/en-US/package-maintainers/Package_Maintenance_Guide/) and the [Pull Requests Guide](https://docs.fedoraproject.org/en-US/ci/pull-requests/) if you want to contribute to the build process.

In case you have trouble with the aforementioned standard Pull Request Guide, consult the Pagure documentation on the [Remote Git to Pagure pull request](https://docs.pagure.org/pagure/usage/pull_requests.html#remote-git-to-pagure-pull-request) workflow.

## Marking an experimental spec as stable

Create a new [stabilization checklist](https://github.com/coreos/ignition/issues/new?template=stabilize-checklist.md) and follow the steps there.
