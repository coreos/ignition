---
layout: default
nav_order: 6
---

# Upgrading Configs
{: .no_toc }

Occasionally, there are changes made to Ignition's configuration that break backward compatibility. While this is not a concern for running machines (since Ignition only runs one time during first boot), it is a concern for those who maintain configuration files. This document serves to detail each of the breaking changes and tries to provide some reasoning for the change. This does not cover all of the changes to the spec - just those that need to be considered when migrating from one version to the next.

1. TOC
{:toc}

## From Version 3.1.0 to 3.2.0

There are not any breaking changes between versions 3.1.0 and 3.2.0 of the configuration specification. Any valid 3.1.0 configuration can be updated to a 3.2.0 configuration by simply changing the version string in the config.

The following is a list of notable new features, deprecations, and changes.

### Partition resizing

The `partition` section gained a new `resize` field. When true, Ignition will resize an existing partition if it matches the config in all respects except the partition size.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.2.0" },
  "storage": {
    "disks": [{
      "device": "/dev/sda",
      "partitions": [{
        "label": "root",
        "sizeMiB": 16384,
        "resize": true
      }]
    }]
  }
}
```

### LUKS encrypted storage

Ignition now supports creating LUKS2 encrypted storage volumes. Volumes can be configured to allow unlocking with any combination of a TPM2 device via Clevis, network Tang servers via Clevis, and static key files. Alternatively, the Clevis configuration can be manually specified with a custom PIN and CFG. If a key file is not specified for a device, an ephemeral one will be created.

<!-- ignition -->
```json
{
  "ignition": {"version": "3.2.0"},
  "storage": {
    "luks": [{
      "name": "static-key-example",
      "device": "/dev/sdb",
      "keyFile": {
        "source": "data:,REPLACE-THIS-WITH-YOUR-KEY-MATERIAL"
      }
    },{
      "name": "tpm-example",
      "device": "/dev/sdc",
      "clevis": {
        "tpm2": true
      }
    },{
      "name": "tang-example",
      "device": "/dev/sdd",
      "clevis": {
        "tang": [{
          "url": "https://tang.example.com",
          "thumbprint": "REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT"
        }]
      }
    }],
    "filesystems": [{
      "path": "/var/lib/static_key_example",
      "device": "/dev/disk/by-id/dm-name-static-key-example",
      "format": "ext4",
      "label": "STATIC-KEY-EXAMPLE"
    },{
      "path": "/var/lib/tpm_example",
      "device": "/dev/disk/by-id/dm-name-tpm-example",
      "format": "ext4",
      "label": "TPM-EXAMPLE"
    },{
      "path": "/var/lib/tang_example",
      "device": "/dev/disk/by-id/dm-name-tang-example",
      "format": "ext4",
      "label": "TANG-EXAMPLE"
    }]
  },
  "systemd": {
    "units": [{
      "name": "var-lib-static_key_example.mount",
      "enabled": true,
      "contents": "[Mount]\nWhat=/dev/disk/by-label/STATIC-KEY-EXAMPLE\nWhere=/var/lib/static_key_example\nType=ext4\n\n[Install]\nWantedBy=local-fs.target"
    },{
      "name": "var-lib-tpm_example.mount",
      "enabled": true,
      "contents": "[Mount]\nWhat=/dev/disk/by-label/TPM-EXAMPLE\nWhere=/var/lib/tpm_example\nType=ext4\n\n[Install]\nWantedBy=local-fs.target"
    },{
      "name": "var-lib-tang_example.mount",
      "enabled": true,
      "contents": "[Mount]\nWhat=/dev/disk/by-label/TANG-EXAMPLE\nWhere=/var/lib/tang_example\nType=ext4\n\n[Install]\nWantedBy=remote-fs.target"
    }]
  }
}
```

### User/group deletion

The `passwd` `users` and `groups` sections have a new field `shouldExist`. If specified and false, Ignition will delete the specified user or group if it exists.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.2.0" },
  "passwd": {
    "users": [{
      "name": "core",
      "shouldExist": false
    }],
    "groups": [{
      "name": "core",
      "shouldExist": false
    }]
  }
}
```

### Google Cloud Storage URL support

The sections which allow fetching a remote URL now accept Google Cloud Storage (`gs://`) URLs in the `source` field.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.2.0" },
  "storage": {
    "files": [{
      "path": "/etc/example",
      "mode": 420,
      "contents": {
        "source": "gs://bucket/object"
      }
    }]
  }
}
```

## From Version 3.0.0 to 3.1.0

There are not any breaking changes between versions 3.0.0 and 3.1.0 of the configuration specification. Any valid 3.0.0 configuration can be updated to a 3.1.0 configuration by simply changing the version string in the config.

The following is a list of notable new features, deprecations, and changes.

### SHA-256 resource verification

All `verification.hash` fields now support the `sha256` hash type.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.1.0" },
  "storage": {
    "files": [{
      "path": "/etc/hosts",
      "mode": 420,
      "contents": {
        "source": "https://example.com/etc/hosts",
        "verification": {
          "hash": "sha256-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        }
      }
    }]
  }
}
```

### Compression support for certificate authorities and merged configs

The config `merge` and `replace` sections and the `certificateAuthorities` section now support gzip-compressed resources via the `compression` field. `gzip` compression is supported for all URL schemes except `s3`.

<!-- ignition -->
```json
{
  "ignition": {
    "version": "3.1.0",
    "config": {
      "merge": [{
        "source": "https://secure.example.com/example.ign.gz",
        "compression": "gzip"
      }]
    },
    "security": {
      "tls": {
        "certificateAuthorities": [{
          "source": "https://example.com/ca.pem.gz",
          "compression": "gzip"
        }]
      }
    }
  }
}
```

### Filesystem mount options

The `filesystems` section gained a new `mountOptions` field. It is a list of options Ignition should pass to `mount -o` when mounting the specified filesystem. This is useful for mounting btrfs subvolumes. This field only affects mounting performed by Ignition while it is running; it does not affect mounting of the filesystem by the provisioned system.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.1.0" },
  "storage": {
    "filesystems": [{
      "path": "/var/data",
      "device": "/dev/vdb1",
      "wipeFilesystem": false,
      "format": "btrfs",
      "mountOptions": [
        "subvolid=5"
      ]
    }]
  }
}
```

### Custom HTTP headers

The sections which allow fetching a remote URL &mdash; config `merge` and `replace`, `certificateAuthorities`, and file `contents` and `append` &mdash; gained a new field called `httpHeaders`. This field can be set to an array of HTTP headers which will be added to an HTTP or HTTPS request. Custom headers can override Ignition's default headers, and will not be retained across HTTP redirects.

During config merging, if a child config specifies a header `name` but not a corresponding `value`, any header with that `name` in the parent config will be removed.

<!-- ignition -->
```json
{
  "ignition": { "version": "3.1.0" },
  "storage": {
    "files": [{
      "path": "/etc/hosts",
      "mode": 420,
      "contents": {
        "source": "https://example.com/etc/hosts",
        "httpHeaders": [
          {
            "name": "Authorization",
            "value": "Basic YWxhZGRpbjpvcGVuc2VzYW1l"
          },
          {
            "name": "User-Agent",
            "value": "Mozilla/5.0 (compatible; MSIE 6.0; Windows NT 5.1)"
          }
        ]
      }
    }]
  }
}
```

### HTTP proxies

The `ignition` section gained a new field called `proxy`. It allows configuring proxies for HTTP and HTTPS requests, as well as exempting certain hosts from proxying.

The `httpsProxy` field specifies the proxy URL for HTTPS requests. The `httpProxy` field specifies the proxy URL for HTTP requests, and also for HTTPS requests if `httpsProxy` is not specified. The `noProxy` field lists specifiers of hosts that should not be proxied, in any of several formats:

- An IP address prefix (`1.2.3.4`)
- An IP address prefix in CIDR notation (`1.2.3.4/8`)
- A domain name, matching the domain and its subdomains (`example.com`)
- A domain name, matching subdomains only (`.example.com`)
- A wildcard matching all hosts (`*`)

IP addresses and domain names can also include a port number (`1.2.3.4:80`).

<!-- ignition -->
```json
{
  "ignition": {
    "version": "3.1.0",
    "proxy": {
      "httpProxy": "https://proxy.example.net/",
      "httpsProxy": "https://secure.proxy.example.net/",
      "noProxy": ["www.example.net"]
    }
  },
  "storage": {
    "files": [{
      "path": "/etc/hosts",
      "mode": 420,
      "contents": {
        "source": "https://example.com/etc/hosts"
      }
    }]
  }
}
```

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

<!-- ignition -->
```json
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

<!-- ignition -->
```json
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

Refer to [this doc in the `spec2x`](https://github.com/coreos/ignition/tree/spec2x/doc/migrating-configs.md) branch of this repository. That doc also describes specification version 2.4.0, a parallel development which shares some enhancements with spec 3.1.0.

[networkd-docs]: https://www.freedesktop.org/software/systemd/man/systemd-networkd.html#
[operator-notes]: operator-notes.md
