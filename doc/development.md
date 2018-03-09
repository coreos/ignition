# Development

A Go 1.7+ [environment](https://golang.org/doc/install) and the `blkid.h` headers are required.

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

Modify `schema/ignition.json` as necessary. This file adheres to the [json schema spec](http://json-schema.org/).

Run the `generate` script to create `internal/config/types/schema.go` and `config/v${LATEST_EXPERIMENTAL}/types/schema.go`. The first of the two files is used internally by Ignition, and the second is presented to library consumers and used for validation.

```sh
./generate
```

Add whatever validation logic is necessary to `config/v${LATEST_EXPERIMENTAL}/types`, modify the translator at `internal/config/translate.go` to handle the changes, and update `internal/config/translate_test.go` to properly test the changes.

Finally, make whatever changes are necessary to `internal` to handle the new spec.

## Vendor

Install [glide](https://github.com/Masterminds/glide) and [glide-vc](https://github.com/sgotti/glide-vc) to manage dependencies in the `vendor` directory.

```sh
go get -u github.com/Masterminds/glide
go get -u github.com/sgotti/glide-vc
```

Edit the `glide.yaml` file to update a dependency or add a new dependency. Then make vendor.

```sh
make vendor
```

## Running Blackbox Tests on Container Linux

Build both the Ignition & test binaries inside of a docker container, for this example it will be building from the ignition-builder-1.8 image and targeting the amd64 architecture.

```sh
docker run --rm -e TARGET=amd64 -v "$PWD":/usr/src/myapp -w /usr/src/myapp quay.io/coreos/ignition-builder-1.8 ./build_blackbox_tests
sudo -E PATH=$PWD/bin/amd64:$PATH ./tests.test
```

## Runnning Blackbox Tests on platforms other than Container Linux

Build Ignition and the test binaries with HELPERS=HOST to use the paths of the binaries from your host system instead of those found in Container linux. Then run blackbox tests. The subshell ensures the root PATH is used instead of your user's.

```sh
HELPERS=HOST ./build_blackbox_tests
sudo sh -c 'PATH=$PWD/bin/amd64:$PATH ./tests.test'
```

## Test Host System Dependencies

The following packages are required by the Blackbox Test:

* `util-linux`
* `dosfstools`
* `e2fsprogs`
* `btrfs-progs`
* `xfsprogs`
* `uuid-runtime`
* `gdisk`
* `coreutils`
* `mdadm`

## Writing Blackbox Tests

To add a blackbox test create a function which yields a `Test` object. A `Test` object consists of the following fields:

Name: `string`

In: `[]Disk` object, which describes the Disks that should be created before Ignition is run.

Out: `[]Disk` object, which describes the Disks that should be present after Ignition is run.

MntDevices: `MntDevice` object, which describes any disk related variable replacements that need to be done to the Ignition config before Ignition is run. This is done so that disks which are created during the test run can be referenced inside of an Ignition config.

OEMLookasideFiles: `[]File` object which describes the Files that should be written into the OEM lookaside directory before Ignition is run.

SystemDirFiles: `[]File` object which describes the Files that should be written into Ignition's system config directory before Ignition is run.

Config: `string`

The test should be added to the init function inside of the test file. If the test module is being created then an `init` function should be created which registers the tests and the package must be imported inside of `tests/registry/registry.go` to allow for discovery.

## Marking an experimental spec as stable

When an experimental version of the Ignition config spec (e.g.: `2.3.0-experimental`) is to be declared stable (e.g. `2.3.0`), there are a handful of changes that must be made to the code base. These changes should have the following effects:

- Any configs with a `version` field set to the previously experimental version will no longer pass validation. For example, if `2.3.0-experimental` is being marked as stable, any configs written for `2.3.0-experimental` should have their version fields changed to `2.3.0`, for Ignition will no longer accept them.
- A new experimental spec version will be created. For example, if `2.3.0-experimental` is being marked as stable, a new version of `2.4.0-experimental` will now be accepted, and start to accumulate new changes to the spec.
- Internally, any configs presented to Ignition will be translated into the new experimental spec before Ignition begins processing them. For example, if the new experimental spec is `2.4.0-experimental`, and Ignition is given a `2.3.0` config, it will be converted into a `2.4.0-experimental` config before any work is done.
- The new stable spec and the new experimental spec will be identical. The new experimental spec is a direct copy of the old experimental spec, and no new changes to the spec have been made yet, so initially the two specs will have the same fields and semantics.
- The HTTP `user-agent` header that Ignition uses whenever fetching an object and the HTTP `accept` header that Ignition uses whenever fetching a config will be updated to advertise the new stable spec.
- New features will be documented in the [migrating configs](doc/migrating-configs.md) documentation.
