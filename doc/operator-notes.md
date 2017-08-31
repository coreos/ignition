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

## Partition Reuse Semantics

The `wipePartitionEntry` and `shouldExist` flags control what Ignition will do when it encounters an existing partition. `wipePartitionEntry` specifies whether Ignition is permitted to delete partition entries in the partition table.  `shouldExist` specifies whether a partition with that number should exist or not (it is invalid to specify a partition should not exist and specify its attributes, such as `size` or `label`).

The following table shows the possible combinations of `shouldExist`, `wipePartitionEntry`, whether or not a partition with the specified number is present, and the action Ignition will take

| partition present | shouldExist | wipePartitionEntry | action Ignition takes
| ----------------- | ----------- | ------------------ | ---------------------
| false             | false       | false              | Do nothing
| false             | false       | true               | Do nothing
| false             | true        | false              | Create specified partition
| false             | true        | true               | Create specified partition
| true              | false       | false              | Fail
| true              | false       | true               | Delete existing partition
| true              | true        | false              | Check if existing partition matches the specified one, fail if it does not
| true              | true        | true               | Check if existing partition matches the specified one, delete existing partition and create specified partition if it does not match

## Partition Matching
A partition matches if all of the specified attributes (`label`, `start`, `size`, `uuid`, and `typeGuid`) are the same. Specifying `uuid` or `typeGuid` as an empty string is the same as not specifying them. When 0 is specified for start or size, Ignition checks if the existing partition's start / size match what they would be if all of the partitions specified were to be deleted, then recreated if `shouldExist` is true.

## Partition number 0
Specifying `Number` as 0 will use the next available partition number *after* all deletions occur. This means if partition 9 is specified with `shouldExist` as `false` and `wipePartitionEntry` as `true`, and another partition is specified to be created with `number` as `0`, it will get partition number 9. Partition number 0 should be avoided on a disk with partitions that specify `shouldExist` as 0. If `number` is not specified it will be treated as 0.

## Partition start 0
Specifying `start` as 0 will use the starting sector of the largest available block. It will *not* use the first available block large enough. If `start` is not specified and there is no existing partition, start will be treated as 0

## Partition size 0
Specifying `size` as 0 is only valid if start is also 0. If start and size are both zero, Ignition will use the starting sector and size of the largest available block. If `size` is not specified and there is no existing partition, it will be treated as 0.
