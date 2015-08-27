# Getting Started with Ignition #

Ignition uses a JSON config to represent the set of changes to be made. The
format of this config is detailed [in the spec][config spec]. One of the most
important parts of this config is the version number. This **must** match the
version number accepted by Ignition. If the config version isn't accepted by
Ignition, Ignition will fail to run and prevent the machine from booting. This
can be seen by inspecting the console output of the failed instance. For more
information, check out the [troubleshooting](#troubleshooting) section.

[config spec]: configuration.md

## Example Configs ##

Each of these examples is written in version 1 of the config. Always double
check to make sure that your config matches the version that Ignition is
expecting.

### Reformatting the Root Filesystem ###

This config will find the device with the filesystem label "ROOT" (the root
filesystem) and reformat it to btrfs, maintaining the filesystem label. The
force flag is needed here because CoreOS currently ships with an EXT4 root
filesystem. Without this flag, Ignition would recognize that there is existing
data and refuse to overwrite it.

```json
{
	"ignitionVersion": 1,
	"storage": {
		"filesystems": [
			{
				"device": "/dev/disk/by-label/ROOT",
				"format": "btrfs",
				"create": {
					"force": true,
					"options": [
						"--label=ROOT"
					]
				}
			}
		]
	}
}
```

### Starting Services ###

This config will write a single service unit with the contents of an example
service. This unit will be enabled as a dependency of multi-user.target and
therefore start on boot.

```json
{
	"ignitionVersion": 1,
	"systemd": {
		"units": [
			{
				"name": "example.service",
				"enable": true,
				"contents": "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"
			}
		]
	}
}
```

This instructs Ignition to create a single service unit named "example.service"
containing the following:

```
[Service]
Type=oneshot
ExecStart=/usr/bin/echo Hello World

[Install]
WantedBy=multi-user.target
```

### Static Networking ###

In this example, the network interface with the name "eth0" will be given the
IP address 10.0.1.7. A typical interface will need more configuration and can
use all of the options of a [network unit][network].

```json
{
	"networkd": {
		"units": [
			{
				"name": "00-eth0.network",
				"contents": "[Match]\nName=eth0\n\n[Network]\nAddress=10.0.1.7"
			}
		]
	}
}
```

This configuration will instruct Ignition to create a single network unit named
"00-eth0.network" with the contents:

```
[Match]
Name=eth0

[Network]
Address=10.0.1.7
```

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
