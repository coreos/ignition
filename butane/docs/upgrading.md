---
has_children: true
nav_order: 5
has_toc: false
---

# Upgrading configs

Occasionally, changes are made to Butane configuration specifications that break backward compatibility or add new functionality. While this is not a concern for running machines, since Ignition only runs one time during first boot, it is a concern for those who maintain configuration files.

Ignition 2.27.0 and later reads Butane configs directly, so you do not need to keep a transpiled Ignition JSON copy alongside a Butane config, unless the config references local files. The guides below describe changes to the Butane specs.

For details about changes in new versions of Butane config specs, see the guide for your specific config variant:

- [Fedora CoreOS](upgrading-fcos.md) (`fcos`)
- [Flatcar](upgrading-flatcar.md) (`flatcar`)
- [OpenShift](upgrading-openshift.md) (`openshift`)
- [RHEL for Edge](upgrading-r4e.md) (`r4e`)
