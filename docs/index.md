---
layout: default
nav_order: 1
---

# Ignition

Ignition is the utility used by Fedora CoreOS and RHEL CoreOS to manipulate disks during the initramfs. This includes partitioning disks, formatting partitions, writing files (regular files, systemd units, etc.), and configuring users. On first boot, Ignition reads its configuration from a source of truth (remote URL, network metadata service, hypervisor bridge, etc.) and applies the configuration.

## Usage

Odds are good that you don't want to invoke Ignition directly. In fact, it isn't even present in the root filesystem. Take a look at the [Getting Started Guide][getting started] for details on providing Ignition with a runtime configuration.

## Contact

- Mailing list: [coreos@lists.fedoraproject.org](https://lists.fedoraproject.org/archives/list/coreos@lists.fedoraproject.org/)
- IRC: #[fedora-coreos](irc://irc.freenode.org:6697/#fedora-coreos) on freenode.org
- Reporting bugs: [issues](https://github.com/coreos/ignition/issues/new/choose)

## Contributing

See [CONTRIBUTING][contributing] for details on submitting patches and the contribution workflow.

To help triage or fix bugs, see the current [Ignition issues](https://github.com/coreos/ignition/issues/).

[getting started]: getting-started.md
[contributing]: https://github.com/coreos/ignition/blob/master/CONTRIBUTING.md
