# Distro Integration

Ignition is designed to be run from an initramfs, only on first boot. Parts of Ignition need to run before the rootfs is mounted and parts need to run after. Ignition requires systemd be running in the initramfs as well. This makes integration with distros somewhat complicated. As an example, the configuration for integrating Ignition with [Fedora CoreOS](fcos) can be found in the [ignition-dracut](ignition-dracut) repo.

## Boot ordering

This section assumes knowledge of the systemd initramfs configuration described by `bootup(7)`.

Ignition is split into four stages: `disks`, `mount`, `files` and `umount`. The disks stage handles configuration that needs to be applied before filesystems are mounted (e.g. partitioning). The mount stage handles mounting filesystems specified in the `storage.filesystems` section and the umount stage unmounts them. The files stage handles creating everything else.

Generally, a boot sequence when Ignition runs looks something like:

1) `basic.target` reached.
1) Networking is started
1) Ignition `disks` stage runs. This is where the config is fetched (and then cached). Since the root fs may change, this needs to occur before the root filesystem is mounted
1) The root filesystem is mounted (this example will assume it is mounted at `/sysroot`)
1) If the system requires special bind mounts for directories, those are mounted (e.g. ostree systems' /var)
1) Ignition `mount` stage runs, creating any additional mountpoints. This needs to happen before the next step to ensure files created end up on the right filesystems.
1) Any filesystem prepopulation occurs. This may be things like running systemd-sysusers and systemd-tmpfiles
1) Ignition `files` stage runs
1) Ignition `umount` stage runs
1) Switch root

## Failure

If Ignition fails, the system should isolate to `emergency.target` and not proceed to switch root. Ignition should give the user the system they specified or system at all.

## The first boot is not special

Ignition has a `umount` stage to ensure that after Ignition runs the system continues booting as if it didn't. This is important to ensure the system will boot when Ignition does not run. Running Ignition should be a diversion in the initramfs, not a replacement.

## Base configs

Ignition supports supplying a "base config" that users' configs get merged into. This allows distros to do things like add default users or enable units. Distros should include a base config describing where any filesystems are mounted (e.g. if `/var` is a separate filesystem) to ensure that the `mount` stage mounts it and files are created where they should be.

## Link time overrides and features

Ignition has a handleful of link-time knobs. SELinux support can be enabled or disabled and the path of some helper utilities Ignition uses can be configured. See the [distro package][distro-package] for a complete list.

## Initramfs tools

See the [developer documentation][dev-doc] for a list of tools Ignition requires in the initramfs.
