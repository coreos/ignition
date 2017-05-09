# Filesystem reuse semantics

When a Container Linux machine first boots, it's possible that an earlier installation or other process has already provisioned the disks. The Ignition config can specify the intended filesystem for a given device, and there are three possibilities when Ignition runs:

- There is no preexisting filesystem.
- There is a preexisting filesystem of the correct type (e.g. the Ignition config says `/dev/sda` should be `ext4`, and it is `ext4`).
- There is a preexisting filesystem of an incorrect type (e.g. the Ignition config says `/dev/sda` should be `ext4`, and it is `btrfs`).

In the first case, when there is no preexisting filesystem, Ignition will always create the desired filesystem.

In the second two cases, where there is a preexisting filesystem, Ignition's behavior is controlled by the `wipeFilesystem` flag in the `filesystem` section.

If `wipeFilesystem` is set to true, Ignition will always wipe any preexisting filesystem and create the desired filesystem. Note this will result in any data on the old filesystem being lost.

If `wipeFilesystem` is set to false, Ignition will then attempt to reuse the existing filesystem. If the filesystem is of the correct type (e.g. `ext4`, `btrfs`, `xfs`, etc.), then Ignition will reuse the filesystem. Any preexisting data will be left on the device and will be available to the installation. If the preexisting filesystem is *not* of the correct type, then Ignition will fail, and the machine will fail to boot.

When determining if a filesystem can be reused, *only* the filesystem type is considered. Any desired labels or other parameters on the filesystem are not checked. If `wipeFilesystem` is false, and a preexisting filesystem of the correct type exists, it's possible to get a machine without the desired filesystem labels.
