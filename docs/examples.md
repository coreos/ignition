---
layout: default
nav_order: 4
---

# Example Configs
{: .no_toc }

These examples are written in version 3.0.0 of the config. Ignition v2.0.0+ understands all configs with version 3.0.0+.

1. TOC
{:toc}

## Services

### Start Services

This config will write a single service unit (shown below) with the contents of an example service. This unit will be enabled as a dependency of multi-user.target and therefore start on boot.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.0.0" },
  "systemd": {
    "units": [{
      "name": "example.service",
      "enabled": true,
      "contents": "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"
    }]
  }
}
```

`example.service`:

```ini
[Service]
Type=oneshot
ExecStart=/usr/bin/echo Hello World

[Install]
WantedBy=multi-user.target
```

### Modify Services

This config will add a [systemd unit drop-in](https://www.freedesktop.org/software/systemd/man/systemd.unit.html#Description) to modify the existing service `systemd-journald` and sets its environment variable `SYSTEMD_LOG_LEVEL` to `debug`.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.0.0" },
  "systemd": {
    "units": [{
      "name": "systemd-journald.service",
      "dropins": [{
        "name": "debug.conf",
        "contents": "[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug"
      }]
    }]
  }
}
```

`systemd-journald.service.d/debug.conf`:

```ini
[Service]
Environment=SYSTEMD_LOG_LEVEL=debug
```
## Create Files on the Root Filesystem

In many cases it is useful to write files to the root filesystem. This example writes a single file to `/etc/someconfig` on the root filesystem. The contents of the file ("example file") are specified inline in the config using the [data URL scheme][rfc2397].

<!-- ignition -->
```json
{
  "ignition": { "version": "3.0.0" },
  "storage": {
    "files": [{
      "path": "/etc/someconfig",
      "mode": 420,
      "contents": { "source": "data:,example%20file%0A" }
    }]
  }
}
```

Paths are specified relative to the root filesystem of the system Ignition is configuring. Symlinks are followed as if Ignition was running from the final system. See the [operator notes][operator-notes] for more information about how Ignition follows symlinks.


## Reformat the /var Filesystem

### Btrfs

This example Ignition configuration will locate the device with the "VAR" filesystem label and reformat it to btrfs, recreating the filesystem label. The `wipeFilesystem` option is set to ensure that Ignition ignores any existing filesystem. This configuration also writes a file to `/var/example-asset`, fetching its contents from `https://example.com/asset`. Ignition mounts filesystems it creates at the specified `path` before creating anything on the filesystems, ensuring `/var/example-asset` is created on the newly created filesystem. Note that Ignition will not automatically create mount units or `/etc/fstab` entries for the filesystems it creates. In this case we assume the OS already has a mount unit or `/etc/fstab` entry for the `/var` filesystem by label.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.0.0" },
  "storage": {
    "filesystems": [{
      "device": "/dev/disk/by-label/VAR",
      "path": "/var",
      "format": "btrfs",
      "wipeFilesystem": true,
      "label": "VAR"
    }],
    "files": [{
      "path": "/var/example-asset",
      "mode": 420,
      "contents": {
        "source": "http://example.com/asset",
        "verification": { "hash": "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" }
      }
    }]
  }
}
```

The SHA512 sum of the file can be determined using `sha512sum`. SHA256 sums are also supported, and can be calculated using `sha256sum`.

## Create a RAID-enabled Data Volume

In many scenarios, it may be useful to have an external data volume. This config will set up a RAID0 ext4 volume, `data`, between two separate disks. It also writes a mount unit (shown below) which will automatically mount the volume to `/var/lib/data`.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.0.0" },
  "storage": {
    "disks": [
      {
        "device": "/dev/sdb",
        "wipeTable": true,
        "partitions": [{
          "label": "raid.1.1",
          "number": 1,
          "sizeMiB": 1024,
          "startMiB": 0
        }]
      },
      {
        "device": "/dev/sdc",
        "wipeTable": true,
        "partitions": [{
          "label": "raid.1.2",
          "number": 1,
          "sizeMiB": 1024,
          "startMiB": 0
        }]
      }
    ],
    "raid": [{
      "devices": [
        "/dev/disk/by-partlabel/raid.1.1",
        "/dev/disk/by-partlabel/raid.1.2"
      ],
      "level": "stripe",
      "name": "data"
    }],
    "filesystems": [{
      "device": "/dev/md/data",
      "path": "/var/lib/data",
      "format": "ext4",
      "label": "DATA"
    }]
  },
  "systemd": {
    "units": [{
      "name": "var-lib-data.mount",
      "enabled": true,
      "contents": "[Mount]\nWhat=/dev/md/data\nWhere=/var/lib/data\nType=ext4\n\n[Install]\nWantedBy=local-fs.target"
    }]
  }
}
```

`var-lib-data.mount`:

```ini
[Mount]
What=/dev/md/data
Where=/var/lib/data
Type=ext4

[Install]
WantedBy=local-fs.target
```

## Replace the Config with a Remote Config

In some cloud environments, there is a limit on the size of the config which may be provided to a machine. To work around this, Ignition allows configs to be replaced with the contents of an alternate, remote config. The following demonstrates this, using a SHA512 sum to verify the contents of the config.

<!-- ignition -->
```json
{
  "ignition": {
    "version": "3.0.0",
    "config": {
      "replace": {
        "source": "http://example.com/config.json",
        "verification": { "hash": "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" }
      }
    }
  }
}
```

The SHA512 sum of the config can be determined using `sha512sum`. SHA256 sums are also supported, and can be calculated using `sha256sum`.

## Set the Hostname

Setting the hostname of a system is as simple as writing `/etc/hostname`:

<!-- ignition -->
```json
{
  "ignition": { "version": "3.0.0" },
  "storage": {
    "files": [{
      "path": "/etc/hostname",
      "mode": 420,
      "overwrite": true,
      "contents": { "source": "data:,core1" }
    }]
  }
}
```

## Add Users

Users can be added to an OS with the `passwd.users` key which takes a list of objects that specify a given user. If you wanted to configure a user "systemUser" and a user "jenkins" you would do that as follows:

<!-- ignition -->
```json
{
  "ignition": { "version": "3.0.0" },
  "passwd": {
    "users": [
      {
        "name": "systemUser",
        "passwordHash": "$superSecretPasswordHash.",
        "sshAuthorizedKeys": [
          "ssh-rsa veryLongRSAPublicKey"
        ]
      },
      {
        "name": "jenkins",
        "uid": 1000
      }
    ]
  }
}
```

To add more users, configure them within the `users` list structure (`[...]`).

## Create a LUKS Volume

This config will set up a key-file based LUKS2 volume, `data`, put a filesystem on the volume, and write a mount unit (shown below) to automatically mount the volume to `/var/lib/data`.

<!-- ignition -->
```json
{
  "ignition": {"version": "3.2.0"},
  "storage": {
    "luks": [{
      "name": "data",
      "device": "/dev/sdb"
    }],
    "filesystems": [{
      "path": "/var/lib/data",
      "device": "/dev/disk/by-id/dm-name-data",
      "format": "ext4",
      "label": "DATA"
    }]
  },
  "systemd": {
    "units": [{
      "name": "var-lib-data.mount",
      "enabled": true,
      "contents": "[Mount]\nWhat=/dev/disk/by-label/DATA\nWhere=/var/lib/data\nType=ext4\n\n[Install]\nWantedBy=local-fs.target"
    }]
  }
}

```

`var-lib-data.mount`:

```ini
[Mount]
What=/dev/disk/by-label/DATA
Where=/var/lib/data
Type=ext4

[Install]
WantedBy=local-fs.target
```

[rfc2397]: http://tools.ietf.org/html/rfc2397
[operator-notes]: operator-notes.md
