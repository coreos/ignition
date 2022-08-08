# Ignition

Ignition is the utility used by Fedora CoreOS and RHEL CoreOS to manipulate disks during the initramfs. This includes partitioning disks, formatting partitions, writing files (regular files, systemd units, etc.), and configuring users. On first boot, Ignition reads its configuration from a source of truth (remote URL, network metadata service, hypervisor bridge, etc.) and applies the configuration.

## Usage

Odds are good that you don't want to invoke Ignition directly. In fact, it isn't even present in the root filesystem. Take a look at the [Getting Started Guide][getting started] for details on providing Ignition with a runtime configuration.

## Contact

- Mailing list: [coreos@lists.fedoraproject.org](https://lists.fedoraproject.org/archives/list/coreos@lists.fedoraproject.org/)
- IRC: #[fedora-coreos](ircs://irc.libera.chat:6697/#fedora-coreos) on Libera.Chat
- Reporting bugs: [issues][issues]

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

To help triage or fix bugs, see the current [Ignition issues](https://github.com/coreos/ignition/issues/).

## Config Validation

To validate a config for Ignition there are binaries for a cli tool called `ignition-validate` available [on the releases page][releases]. There is also an ignition-validate container: `quay.io/coreos/ignition-validate`.

Example:
```
# This example uses podman, but docker can be used too
podman run --pull=always --rm -i quay.io/coreos/ignition-validate:release - < myconfig.ign
```

## Branches

There are two branches:
- `main`: the actively maintained version of Ignition, supporting config spec 3.x. Used by Fedora CoreOS, RHEL CoreOS, and Flatcar Container Linux.
- `spec2x`: the legacy branch of Ignition, supporting config spec 1 and 2.x. This branch is no longer maintained.

### Legacy ignition-dracut

In Ignition 2.5.0, the old [ignition-dracut](https://github.com/coreos/ignition-dracut) repository, containing scripts and systemd units for boot-time execution, was merged into Ignition itself. CoreOS-specific Dracut modules have moved to the [fedora-coreos-config](https://github.com/coreos/fedora-coreos-config) repository.

[getting started]: docs/getting-started.md
[issues]: https://github.com/coreos/ignition/issues/new/choose
[releases]: https://github.com/coreos/ignition/releases


