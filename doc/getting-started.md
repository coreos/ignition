# Getting Started with Ignition #

Ignition uses a JSON config to represent the set of changes to be made. The
format of this config is detailed [in the spec][config spec]. One of the most
important parts of this config is the version number. This **must** match the
version number accepted by Ignition. If the config version isn't accepted by
Ignition, Ignition will fail to run and prevent the machine from booting. This
can be seen by inspecting the console output of the failed instance. For more
information, check out the [troubleshooting section](#troubleshooting).

[config spec]: configuration.md

## Providing a Config ##

Ignition will look in different places for its configuration and metadata,
depending on the platform on which it is running. A full list of the
[supported platforms] and their data sources is provided for reference.

The configuration will need to be passed to Ignition via the designated data
source.

[supported platforms]: supported-platforms.md

## Troubleshooting ##

The single most useful piece of information needed when troubleshooting is the
log from Ignition. Ignition runs in multiple stages so it's easiest to filter
by the syslog identifier: `ignition`. When using systemd, this can be
accomplished with the following command:

```
journalctl --identifier=ignition
```

In the vast majority of cases, it will be immediately obvious why Ignition
failed. If it's not, inspect the config that Ignition dumped into the log. This
shows how Ignition interpretted the supplied configuration. The user-provided
config may have a misspelled section or maybe an incorrect hierarchy.
