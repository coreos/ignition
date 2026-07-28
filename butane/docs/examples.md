---
nav_order: 3
---

# Examples
{: .no_toc }

1. TOC
{:toc}

Here you can find a bunch of simple examples for using Butane, with some explanations about what they do. The examples here are in no way comprehensive, for a full list of all the options present in Butane check out the [configuration specification][spec].

## Users and groups

This example modifies the existing `core` user and sets its SSH key by providing it inline.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
passwd:
  users:
    - name: core
      ssh_authorized_keys:
        - key1
```

This example creates one user, `user1` and sets up one SSH public key for the user. The user is also given the home directory `/home/user1`, but it's not created, the user is added to the `wheel` and `plugdev` groups, and the user's shell is set to `/bin/bash`.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
passwd:
  users:
    - name: user1
      ssh_authorized_keys:
        - key1
      home_dir: /home/user1
      no_create_home: true
      groups:
        - wheel
        - plugdev
      shell: /bin/bash
```

### Using password authentication

You can use a Butane config to set a password for a local user. Building on the previous example, we can configure the `password_hash` for one or more users:

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
passwd:
  users:
    - name: user1
      ssh_authorized_keys:
        - key1
      password_hash: $y$j9T$aUmgEDoFIDPhGxEe2FUjc/$C5A...
      home_dir: /home/user1
      no_create_home: true
      groups:
        - wheel
        - plugdev
      shell: /bin/bash
```

To generate a secure password hash, use `mkpasswd` from the `whois` package.  Your Linux distro may ship a different `mkpasswd` implementation; you can ensure you're using the correct one by running it from a container:

```
$ podman run -ti --rm quay.io/coreos/mkpasswd --method=yescrypt
Password:
$y$j9T$A0Y3wwVOKP69S.1K/zYGN.$S596l11UGH3XjN...
```

The `yescrypt` hashing method is recommended for new passwords. For more details on hashing methods, see `man 5 crypt`.

For more information, see the Fedora CoreOS documentation on [Authentication][fcos-auth-docs].

### SSH keys from local files

As shown in the previous examples you can inline multiple SSH public keys per user directly in the Butane config. Additionally, you can embed keys from local files at transpile time. This example creates a `core` user and configures its SSH keys from local files. The file paths are relative to the directory specified with the `--files-dir` command-line option, which must be provided.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.5.0
passwd:
  users:
    - name: core
      ssh_authorized_keys_local:
        - id_rsa.pub
        - id_ed25519.pub
```

Different combinations for providing SSH keys are possible. You can provide inline ones together with file references and the files may also contain multiple keys (one per line). However, keep in mind that overall the keys must be unique. Check the [configuration specification][spec] for details.

## Storage and files

### Files

This example creates a file at `/opt/file` with the contents `Hello, world!`, permissions 0644 (so readable and writable by the owner, and only readable by everyone else), and the file is owned by user uid 500 and gid 501.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
storage:
  files:
    - path: /opt/file
      contents:
        inline: Hello, world!
      mode: 0644
      user:
        id: 500
      group:
        id: 501
```

This example fetches a gzip-compressed file from `http://example.com/file2`, makes sure that the _uncompressed_ contents match the provided sha512 hash, and writes it to `/opt/file2`.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
storage:
  files:
    - path: /opt/file2
      contents:
        source: http://example.com/file2
        compression: gzip
        verification:
          hash: sha512-4ee6a9d20cc0e6c7ee187daffa6822bdef7f4cebe109eff44b235f97e45dc3d7a5bb932efc841192e46618f48a6f4f5bc0d15fd74b1038abf46bf4b4fd409f2e
      mode: 0644
```

This example creates a file at `/opt/file3` whose contents are read from a local file `local-file3` on the system running Butane. The path of the local file is relative to a _files-dir_ which must be specified via the `-d`/`--files-dir` option to Butane.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
storage:
  files:
    - path: /opt/file3
      contents:
        local: local-file3
      mode: 0644
```

### Directory trees

Consider a directory tree at `~/conf/tree` on the system running Butane:

```
file
overridden-file
directory/file
directory/symlink -> ../file
```

This example copies that directory tree to `/etc/files` on the target system. The ownership and mode for `overridden-file` are explicitly set by the config. All other filesystem objects are owned by `root:root`, directory modes are set to 0755, and file modes are set to 0755 if the source file is executable or 0644 otherwise. The example must be transpiled with `--files-dir ~/conf`.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
storage:
  trees:
    - local: tree
      path: /etc/files
  files:
    - path: /etc/files/overridden-file
      mode: 0600
      user:
        id: 500
      group:
        id: 501
```

### Filesystems and partitions

This example creates a single partition spanning all of the `sdb` device then creates a btrfs filesystem on it to use as `/var`. Finally, it creates the mount unit for systemd so it gets mounted on boot.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
storage:
  disks:
    - device: /dev/sdb
      wipe_table: true
      partitions:
        - number: 1
          label: var
  filesystems:
    - path: /var
      device: /dev/disk/by-partlabel/var
      format: btrfs
      wipe_filesystem: true
      label: var
      with_mount_unit: true
```

### Swap areas

This example creates a swap partition spanning all of the `sdb` device, creates a swap area on it, and creates a systemd swap unit so the swap area is enabled on boot.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.4.0
storage:
  disks:
    - device: /dev/sdb
      wipe_table: true
      partitions:
        - number: 1
          label: swap
  filesystems:
    - device: /dev/disk/by-partlabel/swap
      format: swap
      wipe_filesystem: true
      with_mount_unit: true
```

### LUKS encrypted storage

This example creates three LUKS2 encrypted storage volumes: one unlocked with a static key file, one with a TPM2 device via Clevis, and one with a network Tang server via Clevis. Volumes can be unlocked with any combination of these methods, or with a custom Clevis PIN and CFG. If a key file is not specified for a device, an ephemeral one will be created.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.2.0
storage:
  luks:
    - name: static-key-example
      device: /dev/sdb
      key_file:
        inline: REPLACE-THIS-WITH-YOUR-KEY-MATERIAL
    - name: tpm-example
      device: /dev/sdc
      clevis:
        tpm2: true
    - name: tang-example
      device: /dev/sdd
      clevis:
        tang:
          - url: https://tang.example.com
            thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
  filesystems:
    - path: /var/lib/static_key_example
      device: /dev/disk/by-id/dm-name-static-key-example
      format: ext4
      label: STATIC-EXAMPLE
      with_mount_unit: true
    - path: /var/lib/tpm_example
      device: /dev/disk/by-id/dm-name-tpm-example
      format: ext4
      label: TPM-EXAMPLE
      with_mount_unit: true
    - path: /var/lib/tang_example
      device: /dev/disk/by-id/dm-name-tang-example
      format: ext4
      label: TANG-EXAMPLE
      with_mount_unit: true
```

This example uses the shortcut `boot_device` syntax to configure an encrypted root filesystem unlocked with a combination of a TPM2 device and a network Tang server.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.3.0
boot_device:
  luks:
    tpm2: true
    tang:
      - url: https://tang.example.com
        thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
```

This example combines `boot_device` with a manually-specified filesystem `format` to create an encrypted root filesystem formatted with `ext4` instead of the default `xfs`.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.3.0
boot_device:
  luks:
    tpm2: true
storage:
  filesystems:
    - device: /dev/mapper/root
      format: ext4
```

This example uses the shortcut `boot_device` syntax to configure an encrypted root filesystem in s390x on the `dasda` DASD device unlocked with a network Tang server.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.6.0
boot_device:
  layout: s390x-eckd
  luks:
    device: /dev/dasda
    tang:
      - url: https://tang.example.com
        thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
```

This example uses the shortcut `boot_device` syntax to configure an encrypted root filesystem in s390x on the `sdb` zFCP device unlocked with a network Tang server.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.6.0
boot_device:
  layout: s390x-zfcp
  luks:
    device: /dev/sdb
    tang:
      - url: https://tang.example.com
        thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
```

This example uses the shortcut `boot_device` syntax to configure an encrypted root filesystem in s390x KVM unlocked with a network Tang server.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.6.0
boot_device:
  layout: s390x-virt
  luks:
    tang:
      - url: https://tang.example.com
        thumbprint: REPLACE-THIS-WITH-YOUR-TANG-THUMBPRINT
```

This example uses the shortcut `boot_device` syntax to configure an encrypted root filesystem in s390x on the `dasda` DASD device unlocked with a CEX card.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.6.0
boot_device:
  layout: s390x-eckd
  luks:
    device: /dev/dasda
    cex:
      enabled: true
```

### Mirrored boot disk

This example replicates all default partitions on the boot disk across multiple disks, allowing the system to survive disk failure.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.3.0
boot_device:
  layout: x86_64
  mirror:
    devices:
      - /dev/sda
      - /dev/sdb
```

This example configures a mirrored boot disk with a TPM2-encrypted root filesystem, overrides the size of the root partition replicas, and adds a mirrored `/var` partition which consumes the remainder of the disks.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.3.0
boot_device:
  layout: x86_64
  luks:
    tpm2: true
  mirror:
    devices:
      - /dev/sda
      - /dev/sdb
storage:
  disks:
    - device: /dev/sda
      partitions:
        - label: root-1
          size_mib: 10240
        - label: var-1
    - device: /dev/sdb
      partitions:
        - label: root-2
          size_mib: 10240
        - label: var-2
  raid:
    - name: md-var
      level: raid1
      devices:
        - /dev/disk/by-partlabel/var-1
        - /dev/disk/by-partlabel/var-2
  filesystems:
    - device: /dev/md/md-var
      path: /var
      format: xfs
      wipe_filesystem: true
      with_mount_unit: true
```

## systemd units

This example adds a drop-in for the `serial-getty@ttyS0` unit, turning on autologin on `ttyS0` by overriding the `ExecStart=` defined in the default unit. More information on systemd dropins can be found in [the systemd docs][dropins].

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
systemd:
  units:
    - name: serial-getty@ttyS0.service
      dropins:
        - name: autologin.conf
          contents: |
            [Service]
            TTYVTDisallocate=no
            ExecStart=
            ExecStart=-/usr/sbin/agetty --autologin core --noclear %I $TERM
```

This example creates a new systemd unit called `hello.service`, enables it so it will run on boot, and defines the contents to simply echo `"Hello, World!"`.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
systemd:
  units:
    - name: hello.service
      enabled: true
      contents: |
        [Unit]
        Description=A hello world unit!
        [Service]
        Type=oneshot
        RemainAfterExit=yes
        ExecStart=/usr/bin/echo "Hello, World!"
        [Install]
        WantedBy=multi-user.target
```

This example specifies a systemd unit (`example.service`) and a dropin (`proxy.conf`) to be read from local files at transpile time. The file paths are relative to the directory specified with the `--files-dir` command-line option, which must be provided.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.5.0
systemd:
  units:
    - name: example.service
      contents_local: example.service
    - name: rpm-ostreed.service
      dropins:
          - name: proxy.conf
            contents_local: example.conf
```

## Podman Quadlets

This example defines a simple podman quadlet that runs a container. Butane will place the quadlet file in the appropriate directory and Podman's systemd generator will create the corresponding systemd service.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.8.0-experimental
systemd:
  quadlets:
    - name: sleepy.container
      rootful: true
      contents: |
        [Container]
        Image=quay.io/fedora/fedora:latest
        Exec=sleep infinity
        [Install]
        WantedBy=multi-user.target
```

## GRUB password

This example adds a superuser to GRUB and sets a password. Users without the given username and password will not be able to access GRUB command line, modify kernel command-line arguments, or boot non-default OSTree deployments. Password hashes can be generated with `grub2-mkpasswd-pbkdf2`.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.5.0
grub:
  users:
    - name: root
      password_hash: grub.pbkdf2.sha512.10000.874A958E5264...
```
## Merging Ignition Configs

Butane supports merging Ignition configurations directly into Butane configs using different methods.

This example embeds an Ignition config directly into the Butane config:

<!-- butane-config -->
```yaml
variant: fcos
version: 1.6.0
ignition:
  config:
    merge:
    - inline: '{"ignition": {"version": "3.5.0"}}'
```

This example merges an Ignition from a local file. The file directory can be specified using the `-d/--files` option when running Butane.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.6.0
ignition:
  config:
    merge:
    - local: ignition.ign
```

This example fetches and merges an Ignition config from a remote URL.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.6.0
ignition:
  config:
    merge:
    - source: https://example.com/sample.ign
```

Note that if Butane encounters an Ignition version it does not know about (usually because it is from a newer release), it will show a warning and will continue processing. 
This lets you embed newer Ignition versions in Butane configs without blocking on the validation by Butane.

[spec]: specs.md
[dropins]: https://www.freedesktop.org/software/systemd/man/systemd.unit.html#Description
[fcos-auth-docs]: https://docs.fedoraproject.org/en-US/fedora-coreos/authentication
