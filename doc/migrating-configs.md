# Migrating Between Configuration Versions

Occasionally, there are changes made to Ignition's configuration that break backward compatibility. While this is not a concern for running machines (since Ignition only runs one time during first boot), it is a concern for those who maintain configuration files. This document serves to detail each of the breaking changes and tries to provide some reasoning for the change. This does not cover all of the changes to the spec - just those that need to be considered when migrating from one version to the next.

## From Version 2.3.0 to 3.0.0

The 3.0.0 version of the configuration is fully incompatible with prior versions (i.e. v1, v2.x.0) of the config. The previous versions had bugs that are not representable as a 3.0.0 config, so Ignition does not support older versions.

### All deprecated fields are dropped

All deprecated fields have been dropped. Refer to this migration guide in the `spec2x` branch for how to migrate to their replacements first.

### Filesystems now specify path

Ignition now will mount filesystems at the mountpoint specified by `path` when running. Filesystems no longer have the `name` field and files, links, and directories no longer specify the filesystem by name. To create a file on a specified filesystem, ensure that the specified `path` for that filesystem is below the mountpoint for that filesystem. The following two configs are equivalent and both specify `/dev/disk/by-label/VAR` should be an XFS filesystem with an empty file named `example-file` at that filesystem's root. Note the path change in the files section to account for the filesystem's mountpoint.

```json
{
  "ignition": { "version": "2.3.0" },
  "storage": {
    "filesystems": [{
      "name": "var",
      "mount": {
        "device": "/dev/disk/by-label/VAR",
        "format": "xfs",
        "wipeFilesystem": true,
        "label": "var"
      }
    }],
    "files": [{
      "filesystem": "var",
      "path": "/example-file",
      "mode": 420,
      "contents": { "source": "" }
    }]
  }
}
```

```json ignition
{
  "ignition": { "version": "3.0.0" },
  "storage": {
    "filesystems": [{
      "path": "/var",
      "device": "/dev/disk/by-label/VAR",
      "format": "xfs",
      "wipeFilesystem": true,
      "label": "var"
    }],
    "files": [{
      "path": "/var/example-file",
      "mode": 420,
      "contents": { "source": "" }
    }]
  }
}
```

### Files now default to overwrite=false

Files do not overwrite existing files by default. If no source is specified, Ignition will simply ensure a file exists at that path, creating an empty file if necessary.

### File permissions now default to 0644

If file permissions are unspecified, the permissions default to 0644. If Ignition does not need to create or overwrite the file (i.e. `overwrite` is false, a file exists at that path, and `source` is nil), it will not change the mode, owner or group.

### Directories now create leading directories with mode 0755 and uid/gid 0

When a file, directory, or link is specified but the parent directory does not exist, Ignition will create the directory(ies) leading up to it. Previously, when a directory was specified, any leading directories were created with the same mode/uid/gid as the specified directory. Now they are created with mode 0755, uid 0, and gid 0. This ensures the behavior is consistent with files and links.

### Duplicate entries are no longer allowed

Configs cannot specify contradictory entries. This means a config cannot contain two file entries with the same `path`, or specify a path as both a link and a file.

### Configuration appending is replaced by configuration merging

`ignition.config.append` has been replaced by `ignition.config.merge`. Instead of appending entries from the child configs, Ignition merges them. Refer to the [operator notes][operator-notes] for more information.

### Files now have a list of sources to append

Files now have a list of contents to append instead of multiple entries with `append=true`. The following two configs are equivalent. Since `overwrite` is false and `contents.source` is unspecified, Ignition will first ensure a file exists at the path (creating it if necessary) and then append both contents to it. 

```json
{
  "ignition": { "version": "2.3.0" },
  "storage": {
    "files": [{
      "filesystem": "root",
      "path": "/example-file",
      "mode": 420,
      "append": true,
      "contents": { "source": "data:,hello" }
    },
    {
      "filesystem": "root",
      "path": "/example-file",
      "mode": 420,
      "append": true,
      "contents": { "source": "data:,world" }
    }]
  }
}
```

```json ignition
{
  "ignition": { "version": "3.0.0" },
  "storage": {
    "files": [{
      "path": "/example-file",
      "mode": 420,
      "append": [
        { "source": "data:,hello" },
        { "source": "data:,world" }
      ]
    }]
  }
}
```

### Networkd section is removed

The networkd section has been removed. Use the files section instead. Refer to the [networkd documentation][networkd-docs] for more information.

## From 2.x.0 to 2.3.0

Refer to [this doc in the `spec2x`](https://github.com/coreos/ignition/tree/spec2x/doc/migrating-configs.md) branch of this repository.

[networkd-docs]: https://www.freedesktop.org/software/systemd/man/systemd-networkd.html#
[operator-notes]: operator-notes.md
