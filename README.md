# Ignition

Ignition is the utility used by CoreOS Container Linux to manipulate disks during the initramfs. This includes partitioning disks, formatting partitions, writing files (regular files, systemd units, networkd units, etc.), and configuring users. On first boot, Ignition reads its configuration from a source of truth (remote URL, network metadata service, hypervisor bridge, etc.) and applies the configuration.

## Usage

Odds are good that you don't want to invoke Ignition directly. In fact, it isn't even present in the Container Linux root filesystem. Take a look at the [Getting Started Guide][getting started] for details on providing Ignition with a runtime configuration.

## Contact

- Mailing list: [coreos-dev](https://groups.google.com/forum/?hl=en#!forum/coreos-dev)
- IRC: #[coreos](irc://irc.freenode.org:6667/#etcd) on freenode.org
- Bugs: [issues][issues]

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

To help triage or fix bugs see the current [ignition issues](https://github.com/coreos/bugs/labels/component%2Fignition).

## Reporting Bugs

- To report a bug use the [bug tracker][issues]

[getting started]: doc/getting-started.md
[issues]: https://github.com/coreos/bugs/issues/new?labels=component/ignition
