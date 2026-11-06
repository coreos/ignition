---
nav_order: 1
---

# Butane

Butane is Ignition's human-readable configuration format (formerly the Fedora CoreOS Config Transpiler, FCCT). Butane configs are YAML files that describe how a machine should be provisioned; Ignition 2.27.0 and later reads them directly, transpiling them at boot. Butane is maintained in the [Ignition](https://github.com/coreos/ignition) repository, alongside the `butane` CLI, which produces Ignition JSON for older Ignition versions, validates configs ahead of time, and embeds local files. See the [getting
started](getting-started.md) guide for how to use the CLI and the [configuration specifications](specs.md)
for everything Butane configs support.
