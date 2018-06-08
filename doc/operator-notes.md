# Operator Notes

## HTTP Backoff and Retry

When Ignition is fetching a resource over http(s), if the resource is unavailable Ignition will continually retry to fetch the resource with an exponential backoff between requests.

For a given retry attempt, Ignition will wait 10 seconds for the server to send the response headers for the request. If response headers are not received in this time, or an HTTP 5XX error code is received, the request is cancelled, Ignition waits for the backoff, and a new request is made.

Any HTTP response code less than 500 results in the request being completed, and either the resource will be fetched or Ignition will fail.

Ignition will initially wait 100 milliseconds between failed attempts, and the amount of time to wait doubles for each failed attempt until it reaches 5 seconds.

## EC2 and IAM roles

Ignition has support for fetching files over the S3 protocol. When Ignition is running in EC2, it supports using the IAM role given to the EC2 instance to fetch protected assets from S3. If IAM credentials are not successfully fetched, Ignition will attempt to fetch the file with no credentials.


## Filesystem-Reuse Semantics

When a Container Linux machine first boots, it's possible that an earlier installation or other process has already provisioned the disks. The Ignition config can specify the intended filesystem for a given device, and there are three possibilities when Ignition runs:

- There is no preexisting filesystem.
- There is a preexisting filesystem of the correct type, label, or UUID (e.g. the Ignition config says `/dev/sda` should be `ext4`, and it is `ext4`).
- There is a preexisting filesystem of an incorrect type, label, or UUID (e.g. the Ignition config says `/dev/sda` should be `ext4`, and it is `btrfs`).

In the first case, when there is no preexisting filesystem, Ignition will always create the desired filesystem.

In the second two cases, where there is a preexisting filesystem, Ignition's behavior is controlled by the `wipeFilesystem` flag in the `filesystem` section.

If `wipeFilesystem` is set to true, Ignition will always wipe any preexisting filesystem and create the desired filesystem. Note this will result in any data on the old filesystem being lost.

If `wipeFilesystem` is set to false, Ignition will then attempt to reuse the existing filesystem. If the filesystem is of the correct type, has a matching label, and has a matching UUID, then Ignition will reuse the filesystem. If the label or UUID is not set in the Ignition config, they don't need to match for Ignition to reuse the filesystem. Any preexisting data will be left on the device and will be available to the installation. If the preexisting filesystem is *not* of the correct type, then Ignition will fail, and the machine will fail to boot.

## SELinux

When using Ignition with distributions which have [SELinux][selinux] enabled, extra care must be taken to prevent Ignition from creating files that lack SELinux labels. Unfortunately, distributions do not typically include SELinux policies in the initramfs where Ignition runs, so any files, directories, and links created by Ignition don't receive the proper default SELinux labels.

A workaround for this issue is to use [`restorecon`][restorecon] in a oneshot systemd unit to relabel files that Ignition has touched. This unit can be set to run after the SELinux policies have loaded, but before services will try to use them.

An example of this unit is as follows:

```
[Unit]
Requires=systemd-udevd.target
After=systemd-udevd.target

Before=sssd.service
DefaultDependencies=no
ConditionFirstBoot=true

[Service]
Type=oneshot
ExecStart=/usr/sbin/restorecon /foo/bar /etc/test /etc/systemd/system/example.service /etc/passwd /etc/group /etc/shadow

[Install]
WantedBy=multi-user.target
```

This unit will vary based on the Ignition config it is being added to and the distribution that Ignition is running on. Notably the paths listed in the unit are all paths that Ignition caused to be modified or created, not just paths listed in `storage.files`. For example, if a new user is created then `/etc/passwd`, `/etc/shadow`, and `/etc/group` will all need to be relabeled.

If tooling is being used to generate Ignition configs, the tooling _should_ generate such a unit when creating a config for distributions which rely on SELinux.

[selinux]: https://selinuxproject.org/page/Main_Page
[restorecon]: https://linux.die.net/man/8/restorecon

## Partition Reuse Semantics

The `wipePartitionEntry` and `shouldExist` flags control what Ignition will do when it encounters an existing partition. `wipePartitionEntry` specifies whether Ignition is permitted to delete partition entries in the partition table.  `shouldExist` specifies whether a partition with that number should exist or not (it is invalid to specify a partition should not exist and specify its attributes, such as `size` or `label`).

The following table shows the possible combinations of whether or not a partition with the specified number is present, `shouldExist`, and `wipePartitionEntry`, and the action Ignition will take:

| Partition present | shouldExist | wipePartitionEntry | Action Ignition takes
| ----------------- | ----------- | ------------------ | ---------------------
| false             | false       | false              | Do nothing
| false             | false       | true               | Do nothing
| false             | true        | false              | Create specified partition
| false             | true        | true               | Create specified partition
| true              | false       | false              | Fail
| true              | false       | true               | Delete existing partition
| true              | true        | false              | Check if existing partition matches the specified one, fail if it does not
| true              | true        | true               | Check if existing partition matches the specified one, delete existing partition and create specified partition if it does not match

### Partition Matching
A partition matches if all of the specified attributes (`label`, `start`, `size`, `uuid`, and `typeGuid`) are the same. Specifying `uuid` or `typeGuid` as an empty string is the same as not specifying them. When 0 is specified for start or size, Ignition checks if the existing partition's start / size match what they would be if all of the partitions specified were to be deleted (if allowed by wipePartitionEntry), then recreated if `shouldExist` is true.

### Partition number 0
Specifying `number` as 0 will use the next available partition number. Partition number 0 is disallowed on disks with partitions that specify `shouldExist` as false. If `number` is not specified it will be treated as 0.

### Partition start 0
Specifying `start` as 0 will use the starting sector of the largest available block. This is not necessarily the first available block large enough.

### Unspecified partition start
If `start` is not specified and a partition with the same number exists, Ignition will use the start of the existing partition, unless wipePartitionEntry is set.
If `start` is not specified and there is no existing partition, or wipePartitionEntry is set, Ignition will use the starting sector of the largest block, as if `start` were set to 0.

### Partition size 0
Specifying `size` as 0 means the partition should span to the end of the largest available block. If the starting sector is not within the largest available block, Ignition will fail.

### Unspecified partition size
If `size` is not specified and a partition with the same number exists, it will use the value of the existing partition, unless wipePartitionEntry is set.
If `size` is not specified and there is no existing partition, or wipePartitionEntry is set, `size` act as if it were set to 0 and use the size of the largest block.
