---
title: Config Spec v3.4.0-experimental
parent: Configuration specifications
nav_order: 50
---

# Configuration Specification v3.4.0-experimental

_NOTE_: This pre-release version of the specification is experimental and is subject to change without notice or regard to backward compatibility.

The Ignition configuration is a JSON document conforming to the following specification, with **_italicized_** entries being optional:

* **ignition** (object): metadata about the configuration itself.
  * **version** (string): the semantic version number of the spec. The spec version must be compatible with the latest version (`3.4.0-experimental`). Compatibility requires the major versions to match and the spec version be less than or equal to the latest version. `-experimental` versions compare less than the final version with the same number, and previous experimental versions are not accepted.
  * **_config_** (objects): options related to the configuration.
    * **_merge_** (list of objects): a list of the configs to be merged to the current config.
      * **source** (string): the URL of the config. Supported schemes are `http`, `https`, `s3`, `arn`, `gs`, `tftp`, and [`data`][rfc2397]. Note: When using `http`, it is advisable to use the verification option to ensure the contents haven't been modified.
      * **_compression_** (string): the type of compression used on the config (null or gzip). Compression cannot be used with S3.
      * **_httpHeaders_** (list of objects): a list of HTTP headers to be added to the request. Available for `http` and `https` source schemes only.
        * **name** (string): the header name.
        * **_value_** (string): the header contents.
      * **_verification_** (object): options related to the verification of the config.
        * **_hash_** (string): the hash of the config, in the form `<type>-<value>` where type is either `sha512` or `sha256`.
    * **_replace_** (object): the config that will replace the current.
      * **source** (string): the URL of the config. Supported schemes are `http`, `https`, `s3`, `arn`, `gs`, `tftp`, and [`data`][rfc2397]. Note: When using `http`, it is advisable to use the verification option to ensure the contents haven't been modified.
      * **_compression_** (string): the type of compression used on the config (null or gzip). Compression cannot be used with S3.
      * **_httpHeaders_** (list of objects): a list of HTTP headers to be added to the request. Available for `http` and `https` source schemes only.
        * **name** (string): the header name.
        * **_value_** (string): the header contents.
      * **_verification_** (object): options related to the verification of the config.
        * **_hash_** (string): the hash of the config, in the form `<type>-<value>` where type is either `sha512` or `sha256`.
  * **_timeouts_** (object): options relating to `http` timeouts when fetching files over `http` or `https`.
    * **_httpResponseHeaders_** (integer) the time to wait (in seconds) for the server's response headers (but not the body) after making a request. 0 indicates no timeout. Default is 10 seconds.
    * **_httpTotal_** (integer) the time limit (in seconds) for the operation (connection, request, and response), including retries. 0 indicates no timeout. Default is 0.
  * **_security_** (object): options relating to network security.
    * **_tls_** (object): options relating to TLS when fetching resources over `https`.
      * **_certificateAuthorities_** (list of objects): the list of additional certificate authorities (in addition to the system authorities) to be used for TLS verification when fetching over `https`. All certificate authorities must have a unique `source`.
        * **source** (string): the URL of the certificate bundle (in PEM format). The bundle can contain multiple concatenated certificates. Supported schemes are `http`, `https`, `s3`, `arn`, `gs`, `tftp`, and [`data`][rfc2397]. Note: When using `http`, it is advisable to use the verification option to ensure the contents haven't been modified.
        * **_compression_** (string): the type of compression used on the certificate (null or gzip). Compression cannot be used with S3.
        * **_httpHeaders_** (list of objects): a list of HTTP headers to be added to the request. Available for `http` and `https` source schemes only.
          * **name** (string): the header name.
          * **_value_** (string): the header contents.
        * **_verification_** (object): options related to the verification of the certificate.
          * **_hash_** (string): the hash of the certificate, in the form `<type>-<value>` where type is either `sha512` or `sha256`.
  * **_proxy_** (object): options relating to setting an `HTTP(S)` proxy when fetching resources.
    * **_httpProxy_** (string): will be used as the proxy URL for HTTP requests and HTTPS requests unless overridden by `httpsProxy` or `noProxy`.
    * **_httpsProxy_** (string): will be used as the proxy URL for HTTPS requests unless overridden by `noProxy`.
    * **_noProxy_** (list of strings): specifies a list of strings to hosts that should be excluded from proxying. Each value is represented by an `IP address prefix (1.2.3.4)`, `an IP address prefix in CIDR notation (1.2.3.4/8)`, `a domain name`, or `a special DNS label (*)`. An IP address prefix and domain name can also include a literal port number `(1.2.3.4:80)`. A domain name matches that name and all subdomains. A domain name with a leading `.` matches subdomains only. For example `foo.com` matches `foo.com` and `bar.foo.com`; `.y.com` matches `x.y.com` but not `y.com`. A single asterisk `(*)` indicates that no proxying should be done.
* **_storage_** (object): describes the desired state of the system's storage devices.
  * **_disks_** (list of objects): the list of disks to be configured and their options. Every entry must have a unique `device`.
    * **device** (string): the absolute path to the device. Devices are typically referenced by the `/dev/disk/by-*` symlinks.
    * **_wipeTable_** (boolean): whether or not the partition tables shall be wiped. When true, the partition tables are erased before any further manipulation. Otherwise, the existing entries are left intact.
    * **_partitions_** (list of objects): the list of partitions and their configuration for this particular disk. Every partition must have a unique `number`, or if 0 is specified, a unique `label`.
      * **_label_** (string): the PARTLABEL for the partition.
      * **_number_** (integer): the partition number, which dictates its position in the partition table (one-indexed). If zero, use the next available partition slot.
      * **_sizeMiB_** (integer): the size of the partition (in mebibytes). If zero, the partition will be made as large as possible.
      * **_startMiB_** (integer): the start of the partition (in mebibytes). If zero, the partition will be positioned at the start of the largest block available.
      * **_typeGuid_** (string): the GPT [partition type GUID][part-types]. If omitted, the default will be 0FC63DAF-8483-4772-8E79-3D69D8477DE4 (Linux filesystem data).
      * **_guid_** (string): the GPT unique partition GUID.
      * **_wipePartitionEntry_** (boolean) if true, Ignition will clobber an existing partition if it does not match the config. If false (default), Ignition will fail instead.
      * **_shouldExist_** (boolean) whether or not the partition with the specified `number` should exist. If omitted, it defaults to true. If false Ignition will either delete the specified partition or fail, depending on `wipePartitionEntry`. If false `number` must be specified and non-zero and `label`, `start`, `size`, `guid`, and `typeGuid` must all be omitted.
      * **_resize_** (boolean) whether or not the existing partition should be resized. If omitted, it defaults to false. If true, Ignition will resize an existing partition if it matches the config in all respects except the partition size.
  * **_raid_** (list of objects): the list of RAID arrays to be configured. Every RAID array must have a unique `name`.
    * **name** (string): the name to use for the resulting md device.
    * **level** (string): the redundancy level of the array (e.g. linear, raid1, raid5, etc.).
    * **devices** (list of strings): the list of devices (referenced by their absolute path) in the array.
    * **_spares_** (integer): the number of spares (if applicable) in the array.
    * **_options_** (list of strings): any additional options to be passed to mdadm.
  * **_filesystems_** (list of objects): the list of filesystems to be configured. `device` and `format` need to be specified. Every filesystem must have a unique `device`.
    * **device** (string): the absolute path to the device. Devices are typically referenced by the `/dev/disk/by-*` symlinks.
    * **format** (string): the filesystem format (ext4, btrfs, xfs, vfat, swap, or none).
    * **_path_** (string): the mount-point of the filesystem while Ignition is running relative to where the root filesystem will be mounted. This is not necessarily the same as where it should be mounted in the real root, but it is encouraged to make it the same.
    * **_wipeFilesystem_** (boolean): whether or not to wipe the device before filesystem creation, see [the documentation on filesystems](operator-notes.md#filesystem-reuse-semantics) for more information. Defaults to false.
    * **_label_** (string): the label of the filesystem.
    * **_uuid_** (string): the uuid of the filesystem.
    * **_options_** (list of strings): any additional options to be passed to the format-specific mkfs utility.
    * **_mountOptions_** (list of strings): any special options to be passed to the mount command.
  * **_files_** (list of objects): the list of files to be written. Every file, directory and link must have a unique `path`.
    * **path** (string): the absolute path to the file.
    * **_overwrite_** (boolean): whether to delete preexisting nodes at the path. `contents.source` must be specified if `overwrite` is true. Defaults to false.
    * **_contents_** (object): options related to the contents of the file.
      * **_compression_** (string): the type of compression used on the contents (null or gzip). Compression cannot be used with S3.
      * **_source_** (string): the URL of the file contents. Supported schemes are `http`, `https`, `tftp`, `s3`, `arn`, `gs`, and [`data`][rfc2397]. When using `http`, it is advisable to use the verification option to ensure the contents haven't been modified. If source is omitted and a regular file already exists at the path, Ignition will do nothing. If source is omitted and no file exists, an empty file will be created.
      * **_httpHeaders_** (list of objects): a list of HTTP headers to be added to the request. Available for `http` and `https` source schemes only.
        * **name** (string): the header name.
        * **_value_** (string): the header contents.
      * **_verification_** (object): options related to the verification of the file contents.
        * **_hash_** (string): the hash of the contents, in the form `<type>-<value>` where type is either `sha512` or `sha256`.
    * **_append_** (list of objects): list of contents to be appended to the file. Follows the same stucture as `contents`
      * **_compression_** (string): the type of compression used on the contents (null or gzip). Compression cannot be used with S3.
      * **_source_** (string): the URL of the contents to append. Supported schemes are `http`, `https`, `tftp`, `s3`, `arn`, `gs`, and [`data`][rfc2397]. When using `http`, it is advisable to use the verification option to ensure the contents haven't been modified.
      * **_httpHeaders_** (list of objects): a list of HTTP headers to be added to the request. Available for `http` and `https` source schemes only.
        * **name** (string): the header name.
        * **_value_** (string): the header contents.
      * **_verification_** (object): options related to the verification of the appended contents.
        * **_hash_** (string): the hash of the contents, in the form `<type>-<value>` where type is either `sha512` or `sha256`.
    * **_mode_** (integer): the file's permission mode. Note that the mode must be properly specified as a **decimal** value (i.e. 0644 -> 420). Setuid/setgid/sticky bits are supported. If not specified, the permission mode for files defaults to 0644 or the existing file's permissions if `overwrite` is false, `contents.source` is unspecified, and a file already exists at the path.
    * **_user_** (object): specifies the file's owner.
      * **_id_** (integer): the user ID of the owner.
      * **_name_** (string): the user name of the owner.
    * **_group_** (object): specifies the file's group.
      * **_id_** (integer): the group ID of the group.
      * **_name_** (string): the group name of the group.
  * **_directories_** (list of objects): the list of directories to be created. Every file, directory, and link must have a unique `path`.
    * **path** (string): the absolute path to the directory.
    * **_overwrite_** (boolean): whether to delete preexisting nodes at the path. If false and a directory already exists at the path, Ignition will only set its permissions. If false and a non-directory exists at that path, Ignition will fail. Defaults to false.
    * **_mode_** (integer): the directory's permission mode. Note that the mode must be properly specified as a **decimal** value (i.e. 0755 -> 493). Setuid/setgid/sticky bits are supported. If not specified, the permission mode for directories defaults to 0755 or the mode of an existing directory if `overwrite` is false and a directory already exists at the path.
    * **_user_** (object): specifies the directory's owner.
      * **_id_** (integer): the user ID of the owner.
      * **_name_** (string): the user name of the owner.
    * **_group_** (object): specifies the directory's group.
      * **_id_** (integer): the group ID of the group.
      * **_name_** (string): the group name of the group.
    * **_contents_** (object): options related to the contents of the directory. If specified, `overwrite` must be `true`. Directories populated from an archive own all files under it. This means that specifying files, directories, and links under the path of this directory always result in a conflict error during config validation.
      * **archive** (string): format of the archive to extract into the directory. Must be `tar`. If `tar` is specified, the source must be a USTAR, PAX, or GNU tarball. Only regular files, directories, and links (both hard links and symlinks) are extracted, other file types are ignored and emit a warning. Note that for `tar` archives, sparse files are not supported and processing an archive with one will result in an error.
      * **_compression_** (string): the type of compression used on the archive (null or gzip). Compression cannot be used with S3.
      * **_source_** (string): the URL of the archive to extract. Supported schemes are `http`, `https`, `tftp`, `s3`, `arn`, `gs`, and [`data`][rfc2397]. When using `http`, it is advisable to use the verification option to ensure the contents haven't been modified. If source is omitted and a directory already exists at the path, Ignition will do nothing. If source is omitted and no directory exists, an empty directory will be created.
      * **_httpHeaders_** (list of objects): a list of HTTP headers to be added to the request. Available for `http` and `https` source schemes only.
        * **name** (string): the header name.
        * **_value_** (string): the header contents.
      * **_verification_** (object): options related to the verification of the archive file.
        * **_hash_** (string): the hash of the archive file, in the form `<type>-<value>` where type is either `sha512` or `sha256`.
  * **_links_** (list of objects): the list of links to be created. Every file, directory, and link must have a unique `path`.
    * **path** (string): the absolute path to the link
    * **_overwrite_** (boolean): whether to delete preexisting nodes at the path. If overwrite is false and a matching link exists at the path, Ignition will only set the owner and group. Defaults to false.
    * **_user_** (object): specifies the owner for a symbolic link. Ignored for hard links.
      * **_id_** (integer): the user ID of the owner.
      * **_name_** (string): the user name of the owner.
    * **_group_** (object): specifies the group for a symbolic link. Ignored for hard links.
      * **_id_** (integer): the group ID of the group.
      * **_name_** (string): the group name of the group.
    * **target** (string): the target path of the link
    * **_hard_** (boolean): a symbolic link is created if this is false, a hard one if this is true.
  * **_luks_** (list of objects): the list of luks devices to be created. Every device must have a unique `name`.
    * **name** (string): the name of the luks device.
    * **device** (string): the absolute path to the device. Devices are typically referenced by the `/dev/disk/by-*` symlinks.
    * **_keyFile_** (string): options related to the contents of the key file.
      * **_compression_** (string): the type of compression used on the contents (null or gzip). Compression cannot be used with S3.
      * **_source_** (string): the URL of the contents to append. Supported schemes are `http`, `https`, `tftp`, `s3`, `arn`, `gs`, and [`data`][rfc2397]. When using `http`, it is advisable to use the verification option to ensure the contents haven't been modified.
      * **_httpHeaders_** (list of objects): a list of HTTP headers to be added to the request. Available for `http` and `https` source schemes only.
        * **name** (string): the header name.
        * **_value_** (string): the header contents.
      * **_verification_** (object): options related to the verification of the key file.
        * **_hash_** (string): the hash of the contents, in the form `<type>-<value>` where type is either `sha512` or `sha256`.
    * **_label_** (string): the label of the luks device.
    * **_uuid_** (string): the uuid of the luks device.
    * **_options_** (list of strings): any additional options to be passed to the cryptsetup utility.
    * **_wipeVolume_** (boolean): whether or not to wipe the device before volume creation, see [the documentation on filesystems](operator-notes.md#filesystem-reuse-semantics) for more information.
    * **_clevis_** (object): describes the clevis configuration for the luks device.
      * **_tang_** (list of objects): describes a tang server. Every server must have a unique `url`.
        * **url** (string): url of the tang server.
        * **thumbprint** (string): thumbprint of a trusted signing key.
      * **_tpm2_** (bool): whether or not to use a tpm2 device.
      * **_threshold_** (int): sets the minimum number of pieces required to decrypt the device. Default is 1.
      * **_custom_** (object): overrides the clevis configuration. The `pin` & `config` will be passed directly to `clevis luks bind`. If specified, all other clevis options must be omitted.
        * **pin** (string): the clevis pin.
        * **config** (string): the clevis configuration JSON.
        * **_needsNetwork_** (bool): whether or not the device requires networking.
* **_systemd_** (object): describes the desired state of the systemd units.
  * **_units_** (list of objects): the list of systemd units. Every unit must have a unique `name`.
    * **name** (string): the name of the unit. This must be suffixed with a valid unit type (e.g. "thing.service").
    * **_enabled_** (boolean): whether or not the service shall be enabled. When true, the service is enabled. When false, the service is disabled. When omitted, the service is unmodified. In order for this to have any effect, the unit must have an install section.
    * **_mask_** (boolean): whether or not the service shall be masked. When true, the service is masked by symlinking it to `/dev/null`. When false, the service is unmasked by deleting the symlink to `/dev/null` if it exists.
    * **_contents_** (string): the contents of the unit.
    * **_dropins_** (list of objects): the list of drop-ins for the unit. Every drop-in must have a unique `name`.
      * **name** (string): the name of the drop-in. This must be suffixed with ".conf".
      * **_contents_** (string): the contents of the drop-in.
* **_passwd_** (object): describes the desired additions to the passwd database.
  * **_users_** (list of objects): the list of accounts that shall exist. All users must have a unique `name`.
    * **name** (string): the username for the account.
    * **_passwordHash_** (string): the encrypted password for the account.
    * **_sshAuthorizedKeys_** (list of strings): a list of SSH keys to be added as an SSH key fragment at `.ssh/authorized_keys.d/ignition` in the user's home directory. All SSH keys must be unique.
    * **_uid_** (integer): the user ID of the account.
    * **_gecos_** (string): the GECOS field of the account.
    * **_homeDir_** (string): the home directory of the account.
    * **_noCreateHome_** (boolean): whether or not to create the user's home directory. This only has an effect if the account doesn't exist yet.
    * **_primaryGroup_** (string): the name of the primary group of the account.
    * **_groups_** (list of strings): the list of supplementary groups of the account.
    * **_noUserGroup_** (boolean): whether or not to create a group with the same name as the user. This only has an effect if the account doesn't exist yet.
    * **_noLogInit_** (boolean): whether or not to add the user to the lastlog and faillog databases. This only has an effect if the account doesn't exist yet.
    * **_shell_** (string): the login shell of the new account.
    * **_shouldExist_** (boolean) whether or not the user with the specified `name` should exist. If omitted, it defaults to true. If false, then Ignition will delete the specified user.
    * **_system_** (bool): whether or not this account should be a system account. This only has an effect if the account doesn't exist yet.
  * **_groups_** (list of objects): the list of groups to be added. All groups must have a unique `name`.
    * **name** (string): the name of the group.
    * **_gid_** (integer): the group ID of the new group.
    * **_passwordHash_** (string): the encrypted password of the new group.
    * **_shouldExist_** (boolean) whether or not the group with the specified `name` should exist. If omitted, it defaults to true. If false, then Ignition will delete the specified group.
    * **_system_** (bool): whether or not the group should be a system group. This only has an effect if the group doesn't exist yet.
* **_kernelArguments_** (object): describes the desired kernel arguments.
  * **_shouldExist_** (list of strings): the list of kernel arguments that should exist.
  * **_shouldNotExist_** (list of strings): the list of kernel arguments that should not exist.

[part-types]: http://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_type_GUIDs
[rfc2397]: https://tools.ietf.org/html/rfc2397
