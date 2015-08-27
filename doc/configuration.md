# Configuration Specification #

The Ignition configuration is a JSON document conforming to the following
specification:

- **ignitionVersion** (integer): the version number of the spec. Must be `1`.
- **storage** (object): describes the desired state of the system's storage
                        devices.
  - **disks** (list of objects): the list of disks to be configured and their
                                 options.
    - **device** (string): the absolute path to the device. Devices are
                           typically referenced by the /dev/disk/by-* symlinks.
    - **wipeTable** (boolean): whether or not the partition tables shall be
                               wiped. When true, the partition tables are
                               erased before any further manipulation.
                               Otherwise, the existing entries are left intact.
    - **partitions** (list of objects): the list of partitions and their
                                        configuration for this particular disk.
      - **label** (string): the PARTLABEL for the partition.
      - **number** (integer): the partition number, which dictates it's
                              position in the partition table (one-indexed). If
                              zero, use the next available partition slot.
      - **size** (integer): the size of the partition (in sectors). If zero,
                            the partition will fill the remainder of the disk.
      - **start** (integer): the start of the partition (in sectors). If zero,
                             the partition will be positioned at the earliest
                             available part of the disk.
      - **typeGuid** (string): the GPT [partition type GUID][part-types].
  - **raid** (list of objects): the list of RAID arrays to be configured.
    - **name** (string): the name to use for the resulting md device.
    - **level** (string): the redundancy level of the array (e.g. linear,
                          raid1, raid5, etc.).
    - **devices** (list of strings): the list of devices (referenced by their
                                     absolute path) in the array.
    - **spares** (integer): the number of spares (if applicable) in the array.
  - **filesystems** (list of objects): the list of filesystems to be
                                       configured. Typically, one filesystem
                                       is configured per partition.
    - **device** (string): the absolute path to the device. Devices are
                           typically referenced by the /dev/disk/by-* symlinks.
    - **format** (string): the filesystem format (e.g. ext4, btrfs, etc.).
    - **create** (object): contains the set of options to be used when creating
                           the filesystem. A non-null entry indicates that the
                           filesystem shall be created.
      - **force** (boolean): whether or not the create operation shall
                             overwrite an existing filesystem.
      - **options** (list of strings): any additional options to be passed to
                                       the format-specific mkfs utility.
    - **files** (list of objects): the list of files, rooted in this particular
                                   filesystem, to be written.
      - **path** (string): the absolute path to the file.
      - **contents** (string): the contents of the file.
      - **mode** (integer): the file's permission mode. Note that the mode must
                            be properly specified as a **decimal** value
                            (i.e. 0644 -> 420).
      - **uid** (integer): the user ID of the owner.
      - **gid** (integer): the group ID of the owner.
- **systemd** (object): describes the desired state of the systemd units.
  - **units** (list of objects): the list of systemd units.
    - **name** (string): the name of the unit. This must be suffixed with a
                         valid unit type (e.g. "thing.service").
    - **enable** (boolean): whether or not the service shall be enabled. When
                            true, the service is enabled. In order for this to
                            have any effect, the unit must have an install
                            section.
    - **mask** (boolean): whether or not the service shall be masked. When
                          true, the service is masked by symlinking it to
                          /dev/null.
    - **contents** (string): the contents of the unit.
    - **dropins** (list of objects): the list of drop-ins for the unit.
      - **name** (string): the name of the drop-in. This must be suffixed with
                           ".conf".
      - **contents** (string): the contents of the drop-in.
- **networkd** (object): describes the desired state of the network units.
  - **units** (list of objects): the list of networkd units.
    - **name** (string): the name of the unit. This must be suffixed with a
                         valid unit type (e.g. "00-eth0.network").
    - **contents** (string): the contents of the unit.
- **passwd** (object): describes the desired additions to the passwd database.
  - **users** (list of objects): the list of accounts to be added.
    - **name** (string): the username for the account.
    - **passwordHash** (string): the encrypted password of the new account.
    - **sshAuthorizedKeys** (list of strings): a list of SSH keys to be added
                                               to the user's authorized_keys.
    - **create** (object): contains the set of options to be used when creating
                           the user. A non-null entry indicates that the user
                           account shall be created.
      - **uid** (integer): the user ID of the new account.
      - **gecos** (string): the GECOS field of the new account.
      - **homeDir** (string): the home directory of the new account.
      - **noCreateHome** (boolean): whether or not to create the user's home
                                    directory.
      - **primaryGroup** (string): the name or ID of the primary group of the
                                   new account.
      - **groups** (list of strings): the list of supplementary groups of the
                                      new account.
      - **noUserGroup** (boolean): whether or not to create a group with the
                                   same name as the user.
      - **noLogInit** (boolean): whether or not to add the user to the lastlog
                                 and faillog databases.
      - **shell** (string): the login shell of the new account.
  - **groups** (list of objects): the list of groups to be added.
    - **name** (string): the name of the group.
    - **gid** (integer): the group ID of the new group.
    - **passwordHash** (string): the encrypted password of the new group.

[part-types]: http://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_type_GUIDs
