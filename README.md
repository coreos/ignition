# Ignition #

Ignition is the utility used by CoreOS Linux to manipulate disks during the
initramfs. This includes partitioning disks, formatting partitions, writing
files (regular files, systemd units, networkd units, etc.), and configuring
users. On first boot, Ignition reads its configuration from a source of truth
(remote URL, network metadata service, hypervisor bridge, etc.) and applies the
configuration.

**Ignition is under very active development!**

Please check out the [roadmap](ROADMAP.md) for information about the timeline.
Use the [bug tracker][issues] to report bugs, but please avoid the urge to
report lack of features for now.

The current scope of features can be found in the
[requirements documentation](doc/requirements.md).

[issues]: https://github.com/coreos/ignition/issues
