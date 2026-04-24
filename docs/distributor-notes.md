---
nav_order: 10
---

# Distributor Notes
{: .no_toc}

1. TOC
{:toc}

## Dracut Module

The distribution specific integration is responsible for ensuring that the ignition dracut module is included in the initramfs when necessary. This can be achieved by adding it as dependency of the dracut module containing the distribution integration, or by installing a dracut configuration file.

## System Configuration Directories

Ignition searches for base configs and user configs (`user.ign`) in three directories, checked in the following priority order:

| Directory | Purpose | Priority |
| --------- | ------- | -------- |
| `/run/ignition` | Runtime/volatile configuration | Highest |
| `/etc/ignition` | Local/admin configuration | Medium |
| `/usr/lib/ignition` | Vendor/distro defaults | Lowest |

For `user.ign`, the first directory containing the file wins. For `base.d/` and `base.platform.d/` fragments, files are collected from all three directories. When the same filename appears in multiple directories, the version from the highest-priority directory is used. Fragments are then merged in sorted filename order.

All three directory paths can be overridden at build time using linker flags (e.g. `-X github.com/coreos/ignition/v2/internal/distro.systemRuntimeConfigDir=/custom/path`) or at runtime via environment variables (`IGNITION_SYSTEM_RUNTIME_CONFIG_DIR`, `IGNITION_SYSTEM_LOCAL_CONFIG_DIR`, `IGNITION_SYSTEM_CONFIG_DIR`).

Scripts such as `coreos-ignition-setup-user` that need to write an Ignition config in the initramfs should use `/etc/ignition/` rather than `/usr/lib/ignition/`, since `/usr` may be mounted read-only under systemd v256+ `ProtectSystem=` defaults.

## Kernel Arguments

When Ignition is updating kernel arguments it will call out to a binary (defined in `internal/distro/distro.go` and overridable at build-time via overriding the `github.com/coreos/ignition/v2/internal/distro.kargsCmd` build flag). Ignition expects that the binary accepts `--should-exist` & `--should-not-exist` parameters. Should exist operations should append the argument if missing and should not exist should NOT fail if the argument is not present. The binary should also reboot the system if necessary.

As an example of the binary implementation look at [`examples/ignition-kargs-helper`](https://github.com/coreos/ignition/blob/main/examples/ignition-kargs-helper).

If your implementation of Ignition doesn't intend to ship kargs functionality the [`ignition-kargs.service` unit](https://github.com/coreos/ignition/blob/main/dracut/30ignition/ignition-kargs.service) should be disabled.
