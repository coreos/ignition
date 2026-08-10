---
title: OpenShift
parent: Upgrading configs
nav_order: 3
---

# Upgrading OpenShift configs

Occasionally, changes are made to OpenShift Butane configs (those that specify `variant: openshift`) that break backward compatibility. While this is not a concern for running machines, since Ignition only runs one time during first boot, it is a concern for those who maintain configuration files. This document serves to detail each of the breaking changes and tries to provide some reasoning for the change. This does not cover all of the changes to the spec - just those that need to be considered when migrating from one version to the next.

{: .no_toc }

1. TOC
{:toc}

## From Version 4.21.0 to 4.22.0

There are no breaking changes between versions 4.21.0 and 4.22.0 of the
`openshift` configuration specification. Any valid 4.21.0 configuration can
be updated to a 4.22.0 configuration by changing the version string in the
config.

### Ignition version rollback

OpenShift spec 4.22.0 targets Ignition spec 3.5.0. Butane 0.28.0 stabilized the
4.22.0 spec targeting Ignition 3.6.0
([#704](https://github.com/coreos/butane/pull/704)), but OCP 4.22's Machine
Config Operator does not support Ignition 3.6.0
([OCPBUGS-90256](https://redhat.atlassian.net/browse/OCPBUGS-90256)). If you
transpiled a config with `version: 4.22.0` using Butane 0.28.0, you must
re-transpile it with an updated Butane to produce an Ignition config version
3.5.0 that will be understood by the MCO. Configs that use Ignition 3.6.0 only
features (such as setuid, setgid, or sticky bits in file modes, or `file_mode`,
`dir_mode`, `user`, or `group` on `storage.trees`) must be updated before
re-transpiling.

## From Version 4.20.0 to 4.21.0

There are no breaking changes between versions 4.20.0 and 4.21.0 of the `openshift` configuration specification. Any valid 4.20.0 configuration can be updated to a 4.21.0 configuration by changing the version string in the config.

## From Version 4.19.0 to 4.20.0

There are no breaking changes between versions 4.19.0 and 4.20.0 of the `openshift` configuration specification. Any valid 4.19.0 configuration can be updated to a 4.20.0 configuration by changing the version string in the config.

## From Version 4.18.0 to 4.19.0

There are no breaking changes between versions 4.18.0 and 4.19.0 of the `openshift` configuration specification. Any valid 4.18.0 configuration can be updated to a 4.19.0 configuration by changing the version string in the config.

The following is a list of notable new features.

### LUKS CEX support

The `luks` sections in `storage` and `boot_device` gained a `cex` field. If enabled, this will configure an encrypted root filesystem on a s390x system using IBM Crypto Express (CEX) card.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.19.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
openshift:
  kernel_arguments:
    - rd.luks.key=/etc/luks/cex.key
boot_device:
  layout: s390x-eckd
  luks:
    device: /dev/dasda
    cex:
      enabled: true
```

### Boot_Device Layouts s390x support

The `boot_device` section gained support for the following layouts `s390x-eckd`, `s390x-zfcp`, `s390x-virt`. This enables the use of the `boot_device` sugar for s390x systems.

The `s390x-eckd` layout enables configuration of an encrypted root filesystem for a DASD device.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.19.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
boot_device:
  layout: s390x-eckd
  luks:
    device: /dev/dasda
    tang:
      - url: https://tang.example.com
        thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
```

The `s390x-zfcp` layout enables configuration of an encrypted root filesystem for a zFCP device.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.19.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
boot_device:
  layout: s390x-zfcp
  luks:
    device: /dev/sdb
    tang:
      - url: https://tang.example.com
        thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
```

The `s390x-virt` layout enables configuration of an encrypted root filesystem for KVM.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.19.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
boot_device:
  layout: s390x-virt
  luks:
    tang:
      - url: https://tang.example.com
        thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
```

## From Version 4.17.0 to 4.18.0

There are no breaking changes between versions 4.17.0 and 4.18.0 of the `openshift` configuration specification. Any valid 4.17.0 configuration can be updated to a 4.18.0 configuration by changing the version string in the config.

## From Version 4.16.0 to 4.17.0

There are no breaking changes between versions 4.16.0 and 4.17.0 of the `openshift` configuration specification. Any valid 4.16.0 configuration can be updated to a 4.17.0 configuration by changing the version string in the config.

## From Version 4.15.0 to 4.16.0

There are no breaking changes between versions 4.15.0 and 4.16.0 of the `openshift` configuration specification. Any valid 4.15.0 configuration can be updated to a 4.16.0 configuration by changing the version string in the config.

## From Version 4.14.0 to 4.15.0

There are no breaking changes between versions 4.14.0 and 4.15.0 of the `openshift` configuration specification. Any valid 4.14.0 configuration can be updated to a 4.15.0 configuration by changing the version string in the config.

## From Version 4.13.0 to 4.14.0

There are no breaking changes between versions 4.13.0 and 4.14.0 of the `openshift` configuration specification. Any valid 4.13.0 configuration can be updated to a 4.14.0 configuration by changing the version string in the config.

The following is a list of notable new features, deprecations, and changes.

### Local SSH key and systemd unit references

SSH keys and systemd units are now embeddable via file references to local files The specified path is relative to a local _files-dir_, specified with the `-d`/`--files-dir` option to Butane. If no _files-dir_ is specified, this functionality is unavailable.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.14.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
systemd:
  units:
    - name: example.service
      contents_local: example.service
    - name: example-drop-in.service
      dropins:
        - name: example-drop-in.conf
          contents_local: example.service
passwd:
  users:
    - name: core
      ssh_authorized_keys_local:
        - id_rsa.pub
```

### Offline Tang provisioning

The `tang` sections in `storage.luks` and `boot_device.luks` gained a new `advertisement` field. If specified, Ignition will use it to provision the Tang server binding rather than fetching the advertisement from the server at runtime. This allows the server to be unavailable at provisioning time. The advertisement can be obtained from the server with `curl http://tang.example.com/adv`.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.14.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
boot_device:
  luks:
    tang:
      - url: https://tang.example.com
        thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
        advertisement: "{\"payload\": \"...\", \"protected\": \"...\", \"signature\": \"...\"}"
storage:
  luks:
    - name: luks-tang
      device: /dev/sdb
      clevis:
        tang:
          - url: https://tang.example.com
            thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
            advertisement: "{\"payload\": \"...\", \"protected\": \"...\", \"signature\": \"...\"}"
```

### LUKS discard

The `luks` sections in `storage` and `boot_device` gained a new `discard` field. If specified and true, the LUKS volume will issue discard commands to the underlying block device when blocks are freed. This improves performance and device longevity on SSDs and space utilization on thinly provisioned SAN devices, but leaks information about which disk blocks contain data.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.14.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
boot_device:
  luks:
    discard: true
    tpm2: true
storage:
  luks:
    - name: luks-tpm
      device: /dev/sdb
      discard: true
      clevis:
        tpm2: true
```

### LUKS open options

The `storage.luks` section gained a new `open_options` field. It is a list of options Ignition should pass to `cryptsetup luksOpen` when unlocking the volume. Ignition also passes `--persistent`, so any options that support persistence will be saved to the volume and automatically used for future unlocks. Any options that do not support persistence will only be applied to Ignition's initial unlock of the volume.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.14.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
storage:
  luks:
    - name: luks-tpm
      device: /dev/sdb
      open_options:
        - "--perf-no_read_workqueue"
        - "--perf-no_write_workqueue"
      clevis:
        tpm2: true
```

### Local SSH key and systemd unit references

SSH keys and systemd units are now embeddable via file references to local files. The specified path is relative to a local _files-dir_, specified with the `-d`/`--files-dir` option to Butane. If no _files-dir_ is specified, this functionality is unavailable.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.14.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
systemd:
  units:
    - name: example.service
      contents_local: example.service
    - name: example-drop-in.service
      dropins:
        - name: example-drop-in.conf
          contents_local: example.conf
passwd:
  users:
    - name: core
      ssh_authorized_keys_local:
        - id_rsa.pub
```

### Automatic generation of systemd swap units

The `with_mount_unit` field of the `filesystems` section can now be set to `true` if the `format` field is set to `swap`. Butane will generate a systemd swap unit for the specified swap area.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.14.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
storage:
  filesystems:
    - device: /dev/vdb1
      format: swap
      wipe_filesystem: true
      with_mount_unit: true
```

## From Version 4.12.0 to 4.13.0

There are no breaking changes between versions 4.12.0 and 4.13.0 of the `openshift` configuration specification. Any valid 4.12.0 configuration can be updated to a 4.13.0 configuration by changing the version string in the config.

The following is a list of notable new features, deprecations, and changes.

### User passwords

The `passwd.users` section enabled the `password_hash` field, which sets the password hash for an account. The `users` section continues to support only the `core` user.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.13.0
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
  name: core-password
passwd:
  users:
    - name: core
      password_hash: $y$j9T$nQ...
```

## From Version 4.11.0 to 4.12.0

There are no breaking changes between versions 4.11.0 and 4.12.0 of the `openshift` configuration specification. Any valid 4.11.0 configuration can be updated to a 4.12.0 configuration by changing the version string in the config.

## From Version 4.10.0 to 4.11.0

There are no breaking changes between versions 4.10.0 and 4.11.0 of the `openshift` configuration specification. Any valid 4.10.0 configuration can be updated to a 4.11.0 configuration by changing the version string in the config.

## From Version 4.9.0 to 4.10.0

There are no breaking changes between versions 4.9.0 and 4.10.0 of the `openshift` configuration specification. Any valid 4.9.0 configuration can be updated to a 4.10.0 configuration by changing the version string in the config. 

### Resource compression

Resource compression, which was disabled in all `openshift` specs in Butane 0.12.1, is re-introduced in this spec version. The `compression` field can be set to `gzip` to decompress gzip-compressed resources. In addition, Butane may automatically compress resources specified with `inline` or `local`.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.10.0
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
  name: config-openshift
storage:
  files:
    - path: /opt/file2
      contents:
        source: data:;base64,H4sIAAAAAAAC/zSQQY4bMQwE7/OKfsBgXpHccs0DGKntEJBIWSINP3+htfcmQECxq/74ZIeOlR3Vm08sDUhnnChuiyUYOSFVh66idgebxonFiuoHNVf3imAfPqFWtGpNC2SgyT+fBOONJrrcTSBNHykX8DcOmnZIRdf9eNJU+olH6oL5ipkVfHEWDQl1Q7YmvfgbreswXbpPfTN1gC9QULx3r/42eKTEBfzaTMkgdObkx1btmByT/2mVUwNqeHrLERLEc7uCaxFFW/tpRDBxy7tKHLYXYchUiZwX8PtVOIK5S1rASxEWCZQcWiUkYG4Y07XS4jzWjqWGkm3INoffblpUULk492/3tnfITqQVXJ+02a/jKwAA//+jjAk6wQEAAA==
        compression: gzip
      mode: 0644
```

## From Version 4.8.0 to 4.9.0

There are no functionality changes between versions 4.8.0 and 4.9.0 of the `openshift` configuration specification. Any valid 4.8.0 configuration can be updated to a 4.9.0 configuration by changing the version string in the config.

## From `rhcos` Version 0.1.0 to `openshift` Version 4.8.0

The new `openshift` config variant is intended to work both on the OpenShift Container Platform with RHEL CoreOS, and on OKD with Fedora CoreOS. The `rhcos` variant is no longer accepted by Butane.

The `openshift` 4.8.0 specification is not backward-compatible with the `rhcos` 0.1.0 specification. It adds new mandatory metadata fields and removes certain Ignition config fields. In addition, `openshift` configs are transpiled to an OpenShift [MachineConfig] rather than an Ignition config by default. A valid `rhcos` 0.1.0 configuration can be updated to an `openshift` 4.8.0 configuration by changing the variant and version strings and then correcting any errors reported during transpilation.

The following is a list of breaking changes and notable new features.

### MachineConfig generation

By default, Butane transpiles an `openshift` Butane config into an OpenShift [MachineConfig]. Butane produces an Ignition config if the `-r` or `--raw` option is specified on the Butane command line.

### Removed config fields

The config no longer allows certain Ignition config fields that are rejected or discouraged by the OpenShift [Machine Config Operator].

In the `storage` section, `directories` and `links` are removed, along with `append` in `files`. Local file trees referenced in `trees` must not contain symlinks.

In the `passwd` section, `groups` is removed. All fields in `users` are removed except for `name` (which must be set to `core`) and `ssh_authorized_keys`.

### MachineConfig metadata fields

The config gained a new top-level `metadata` section containing metadata for the generated [MachineConfig]. The mandatory `name` field specifies a [name for the Kubernetes MachineConfig resource][k8s-names]. The `labels` field specifies a map of key-value pairs to be applied to the MachineConfig resource as [Kubernetes labels][k8s-labels]. The `machineconfiguration.openshift.io/role` label is required.

The `metadata` section is ignored when generating a raw Ignition config using the `-r` or `--raw` option.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.8.0
metadata:
  name: minimal-config
  labels:
    machineconfiguration.openshift.io/role: worker
```

### MCO settings

The config gained a new top-level `openshift` section specifying [configuration][MCO settings] for the [Machine Config Operator]. The `extensions` field lists [RHCOS extension modules] to be installed on the node. The `fips` field enables [FIPS mode] when set to `true`. The `kernel_arguments` field specifies a list of [arguments][kernel arguments] to be added to the kernel command line. The `kernel_type` field can be set to `realtime` to use the [real-time kernel] on the node.

Fields in the `openshift` section are not included in a raw Ignition config generated using the `-r` or `--raw` option.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.8.0
metadata:
  name: config-openshift
  labels:
    machineconfiguration.openshift.io/role: worker
openshift:
  extensions:
    - usbguard
  fips: true
  kernel_arguments:
    - console=ttyS1,115200
  kernel_type: realtime
```

### FIPS configuration for LUKS

When the `fips` field in the `openshift` section is set to `true`, LUKS volumes specified in the config (but not in any referenced configs) are configured to use a cipher compatible with [FIPS 140-2]. This cipher is applied to LUKS volumes specified in the `luks` subsections of the `storage` and `boot_device` sections.

<!-- butane-config -->
```yaml
variant: openshift
version: 4.8.0
metadata:
  name: fips-luks
  labels:
    machineconfiguration.openshift.io/role: worker
openshift:
  fips: true
boot_device:
  luks:
    tpm2: true
```

[FIPS 140-2]: https://csrc.nist.gov/publications/detail/fips/140/2/final
[FIPS mode]: https://docs.openshift.com/container-platform/4.7/installing/installing-fips.html
[k8s-names]: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
[k8s-labels]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
[kernel arguments]: https://docs.openshift.com/container-platform/4.7/post_installation_configuration/machine-configuration-tasks.html#nodes-nodes-kernel-arguments_post-install-machine-configuration-tasks
[Machine Config Operator]: https://docs.openshift.com/container-platform/4.7/post_installation_configuration/machine-configuration-tasks.html#understanding-the-machine-config-operator
[MachineConfig]: https://docs.openshift.com/container-platform/4.7/post_installation_configuration/machine-configuration-tasks.html#machine-config-overviewpost-install-machine-configuration-tasks
[MCO settings]: https://docs.openshift.com/container-platform/4.7/post_installation_configuration/machine-configuration-tasks.html#what-can-you-change-with-machine-configs
[real-time kernel]: https://docs.openshift.com/container-platform/4.7/post_installation_configuration/machine-configuration-tasks.html#nodes-nodes-rtkernel-arguments_post-install-machine-configuration-tasks
[RHCOS extension modules]: https://docs.openshift.com/container-platform/4.7/post_installation_configuration/machine-configuration-tasks.html#rhcos-add-extensions_post-install-machine-configuration-tasks
