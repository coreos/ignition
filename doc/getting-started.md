# Getting Started with Ignition #

Ignition uses a JSON configuration file to represent the set of changes to be made. The format of this config is detailed [in the spec][config spec]. Please refer to Ignition [config examples] to know how to start writing your own config files. One of the most important parts of this config is the version number. This **must** match the version number accepted by Ignition. If the config version isn't accepted by Ignition, Ignition will fail to run and prevent the machine from booting. This can be seen by inspecting the console output of the failed instance. For more information, check out the [troubleshooting section](#troubleshooting).

## How Ignition is Being Executed ##

One of the main difference between Ignition and [Cloud-Config] is Ignition will run only once at the very first boot. GRUB checks the disks for the dummy disk UUID and sets `coreos.first_boot=1` parameter accordingly. This parameter is being processed by [systemd-generators] every boot inside [initramfs] and whenever this parementer is set systemd adds `ignition.target` as one of the `initrd.target` requirement to run Ignition. In case when you use [PXE][supported platforms] deployment `coreos.first_boot=1` parameter needs to be set manually otherwise Ignition won't run.

## Providing a Config ##

Ignition will choose where to look for configuration based on the underlying platform. A list of [supported platforms] and metadata sources is provided for reference.

The configuration must be passed to Ignition through the designated data source. Please refer to Ignition [config examples] to know how to start writing your own config files.

## Troubleshooting ##

The single most useful piece of information needed when troubleshooting is the log from Ignition. Ignition runs in multiple stages so it's easiest to filter by the syslog identifier: `ignition`. When using systemd, this can be accomplished with the following command:

```
journalctl --identifier=ignition
```

In the vast majority of cases, it will be immediately obvious why Ignition failed. If it's not, inspect the config that Ignition dumped into the log. This shows how Ignition interpreted the supplied configuration. The user-provided config may have a misspelled section or maybe an incorrect hierarchy.

[config spec]: configuration.md
[supported platforms]: supported-platforms.md
[config examples]: https://github.com/coreos/docs/blob/master/ignition/examples.md
[Cloud-Config]: https://github.com/coreos/coreos-cloudinit/blob/master/Documentation/cloud-config.md
[systemd-generators]: http://www.freedesktop.org/software/systemd/man/systemd.generator.html
[initramfs]: https://www.kernel.org/doc/Documentation/filesystems/ramfs-rootfs-initramfs.txt
