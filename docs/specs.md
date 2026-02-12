---
has_children: true
nav_order: 5
has_toc: false
---

# Configuration specifications

Ignition configurations must conform to a specific version of the configuration
specification schema, specified with the `ignition.version: X.Y.Z` field in the
configuration.

See the [Upgrading Configs](migrating-configs.md) page for instructions to
update a configuration to the latest specification.

## Stable specification versions

We recommend that you always use the latest **stable** specification to benefit
from new features and bug fixes. The following **stable** specification
versions are currently supported in Ignition:

- [v3.6.0](configuration-v3_6.md)
- [v3.5.0](configuration-v3_5.md)
- [v3.4.0](configuration-v3_4.md)
- [v3.3.0](configuration-v3_3.md)
- [v3.2.0](configuration-v3_2.md)
- [v3.1.0](configuration-v3_1.md)
- [v3.0.0](configuration-v3_0.md)

## Experimental specification versions

Do not use the **experimental** specification for anything beyond **development
and testing** as it is subject to change **without warning or announcement**.
The following **experimental** specification version is currently available in
Ignition:

- [v3.7.0-experimental](configuration-v3_7_experimental.md)

## Legacy spec 2.x configuration specifications

Documentation for the spec 1 and 2.x configuration specifications is available
in the legacy [`spec2x` branch](https://github.com/coreos/ignition/tree/spec2x/doc)
of Ignition. Those specification versions are used by older versions of RHEL
CoreOS and Flatcar Container Linux. This branch is no longer maintained.

## Specification versions and Ignition releases

This table lists, for each stable specification version, the first released
version of Ignition that supports it. To get all bug fixes, it is usually
recommended to use the latest Ignition release.

| Spec version | Ignition release |
|--------------|------------------|
| 3.0.0        | 2.0.0            |
| 3.1.0        | 2.3.0            |
| 3.2.0        | 2.7.0            |
| 3.3.0        | 2.11.0           |
| 3.4.0        | 2.15.0           |
| 3.5.0        | 2.20.0           |
| 3.6.0        | 2.26.0           |