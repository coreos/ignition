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
Similar to the filesystemd use semantics, there are three possibilties for partitions when Ignition runs

- The is no preexisting partition
- There is a preexisting partition of the correct size, starting sector, label, type_guid and guid.
- There is a preexisting partition of an incorrect size, starting sector, label, type_guid or guid.

In the first case, Ignition will always create the desired partition. Using a partition number of 0 specifies to use the next available partition number and will ensure there is no preexisting partition.

In the second two cases, the wipePartition flag controls Ignition's behavior. If it is set to true, Ignition will first delete the partition with the specified number and then attempt to create the new partition. The partition number must be specified and non-zero if wipePartition is set to true.

If wipePartition is false, Ignition will attempt to reuse the existing partition. If the partition has a matching label, uuid, type_guid, start sector, and size, Ignition will reuse the partition. If any of those are not specified, zero, or empty, they do not need to match for Ignition to reuse the partition. If the partition does not match the Ignition configuration, Ignition will fail, and the machine will fail to boot.

## Raid Reuse Semantics
Just like filesystems and partitions, there are three possibilities for raid arrays when Ignition runs

- The devices specified by the RAID array are unused
- The devices specified by the RAID array are used in an existing RAID array with the correct level, devices, and name
- The devices specified by the RAID array are used are either not a member of an existing array or are a member of an exisiting RAID array with incorrect level, devices, or name

An unused device is
 * Not part of an exising RAID array
 * A partition with no filesystem present
 * A whole-disk device with not partition table

In the first case, Ignition will always create the raid array.

In the second and third case the overwriteDevices flag controls Ignition's behavior. If it is set to true, Ignition will use the devices specified whether or not they are currently in a RAID array or have an existing filesystem or partition table.

If overwriteDevices is false, Ignition will attempt to use an existing RAID device. The existing RAID device must have the same name and raid level. Additionally it must be composed of exactly the devices specified and all of those devices must share the same array uuid. The number of spares is allowed to differ. If Ignition cannot reuse the existing RAID device, it will fail and the machine will fail to boot.
