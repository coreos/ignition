---
layout: default
nav_order: 9
---

# Distributor Notes
{: .no_toc}

1. TOC
{:toc}

## Kernel Arguments

When Ignition is updating kernel arguments it will call out to a binary (defined in `internal/distro/distro.go` and overridable at build-time via overriding the `github.com/coreos/ignition/v2/internal/distro.kargsCmd` build flag). Ignition expects that the binary accepts `--should-exist` & `--should-not-exist` parameters. Should exist operations should append the argument if missing and should not exist should NOT fail if the argument is not present. The binary should also reboot the system if necessary.

As an example of the binary implementation look at [`examples/ignition-kargs-helper`](https://github.com/coreos/ignition/blob/master/examples/ignition-kargs-helper).

If your implementation of Ignition doesn't intend to ship kargs functionality the [`ignition-kargs.service` unit](https://github.com/coreos/ignition/blob/master/dracut/30ignition/ignition-kargs.service) should be disabled.
