# Butane

## This repository has been archived

Butane has been merged into the [Ignition](https://github.com/coreos/ignition) repository. All future development, issues, and pull requests should be directed there.

The Butane source code now lives under the [`butane/`](https://github.com/coreos/ignition/tree/main/butane) subdirectory of the Ignition repository. Starting with Ignition 2.27.0, Ignition natively accepts Butane YAML configs at boot, and the `butane` CLI is built from the Ignition source tree.

For more information, see [coreos/fedora-coreos-tracker#2006](https://github.com/coreos/fedora-coreos-tracker/issues/2006).

---

## About Butane

Butane (formerly the Fedora CoreOS Config Transpiler, FCCT) translates human readable Butane Configs
into machine readable [Ignition](https://github.com/coreos/ignition) Configs. See the [getting
started](https://github.com/coreos/ignition/tree/main/butane/docs/getting-started.md) guide for how to use Butane and the [configuration
specifications](https://github.com/coreos/ignition/tree/main/butane/docs/specs.md) for everything Butane configs support.

For information on developing Butane, using it as a library, or understanding how the binaries released
are built, see the [development docs](https://github.com/coreos/ignition/tree/main/butane/docs/development.md).
