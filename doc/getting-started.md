# Getting Started with Ignition #

## What is Ignition? ##

Ignition is a boot-time configuration utility that runs before the switch out of the initramfs during boot.  Mechanically-speaking, it is an executable in the initramfs that looks for configuration data (or a pointer to a URL for such data) in an environment-specific location and applies it to the machine before switch_root is called (see below under [Providing a Config](#providing-a-config) for details).

Ignition uses a JSON configuration file to represent the set of changes to be made. The format of this config is detailed [in the spec][config spec]. One of the most important parts of this config is the version number. This **must** match the version number accepted by Ignition. If the config version isn't accepted by Ignition, Ignition will fail to run and prevent the machine from booting. This can be seen by inspecting the console output of the failed instance. For more information, check out the [troubleshooting section](#troubleshooting).

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
