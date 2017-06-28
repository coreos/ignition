# Working with raid arrays

Ignition can be used to create and assemble a software raid array. This is done
with the `raid` section of the schema under `storage`. There's an example of how
to do this in the [examples document][1].

On the first boot, Ignition will make sure that the raid array gets assembled
after it's created. This is necessary for the array to be used. On subsequent
boots however, Ignition doesn't run and won't assemble the array.

The arrays can be detected and automatically assembled by [dracut][2] with a
little configuration. First and foremost, the partitions that are part of the
raid array must have the correct [type GUID][3] to signify that they are raid
partitions. This can be done by setting the `typeGuid` field in the Ignition
config to `A19D880F-05FC-4D3B-A006-743F0F84911E` for the partitions. This is
supported in Ignition configs of at least version 2.1.0.

With the correct type GUID set, the only remaining thing to be done is to append
the following snippet to the grub config:

```
set linux_append="rd.auto"
```

On a typical disk install, the grub config will be located at
`/usr/share/oem/grub.cfg`.

[1]: examples.md
[2]: http://man7.org/linux/man-pages/man7/dracut.cmdline.7.html
[3]: https://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_type_GUIDs
