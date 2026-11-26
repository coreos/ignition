---
nav_order: 12
---

# ostree/bootc Integration

The `ignition-integration` subpackage (built when `WITH_INTEGRATION=1`) adds ostree/bootc-oriented dracut modules and systemd units on top of the core `30ignition` initramfs module. It is intended for distributions that ship Ignition in an ostree or bootc image (for example fedora-bootc).

Installing `ignition`, `ignition-grub` and `ignition-integration` is not enough to produce a bootable image. Disk layout, the `ignition.firstboot` kernel argument, a real-root `/boot` mount, and any initramfs logic needed after Ignition rewrites disks are **not** in these packages. The image or distro overlay must provide them.

This page covers what the subpackage ships and what the image builder must add. For generic distributor topics (config directories, the kargs helper), see [Distributor Notes](distributor-notes.md). For how Ignition finds a user config, see [Getting Started](getting-started.md).

## Packages

Install it in the image. `ignition-integration` depends on `ignition` and `ignition-grub`.

```
dnf install -y ignition-integration
```

| Package | Role |
| --- | --- |
| `ignition` | Initramfs binary and `30ignition` (fetch, disks, files, kargs) |
| `ignition-grub` | GRUB snippet that sets `$ignition_firstboot` when `/boot/ignition.firstboot` exists |
| `ignition-integration` | ostree transposefs, ostree `/var` mount, network helper, kmsg logging |

Regenerate the initramfs **after** these packages and any extra dracut modules are installed:

```
kver=$(ls /usr/lib/modules)
env DRACUT_NO_XATTR=1 dracut -vf /usr/lib/modules/$kver/initramfs.img "$kver"
```

## What `ignition-integration` provides

| Path | Role |
| --- | --- |
| `/usr/lib/dracut/modules.d/40ignition-ostree` | Before Ignition rewrites disks, copy root/boot/ESP into RAM (`transposefs`) and restore afterward; mount and populate ostree `/var` |
| `/usr/lib/dracut/modules.d/41ignition-network` | Enable networking for Ignition fetch when NetworkManager is in the initramfs |
| `/usr/lib/dracut/modules.d/01ignition-scsi-rules` | SCSI udev rules needed on some multipath disks |
| `/usr/lib/dracut/modules.d/99ignition-log-kmsg` | Send Ignition unit stdout to kmsg |
| `/usr/lib/dracut/dracut.conf.d/60-omit-nfs.conf` | Omit NFS from the initramfs (conflicts with `/var` handling) |
| `/usr/lib/systemd/system/ignition-write-issues.service` | Console/issue files from `/etc/.ignition-result.json` |
| `/usr/lib/systemd/system-preset/40-ignition.preset` | Enable `ignition-write-issues.service` and `ignition-delete-config.service` |
| `/usr/libexec/ignition-write-issues` | Helper for the issues unit |
| `/usr/lib/tmpfiles.d/run-issued.conf` | Create `/run/issue.d` before `ignition-write-issues` writes there |

`99ignition-log-kmsg` only captures Ignition's stdout, so logs land in kmsg only if Ignition is actually writing to stdout. Consumers who want logs in kmsg can add the karg `systemd.setenv=IGNITION_ARGS=--log-to-stdout`.

These units hook into `ignition-complete.target`. They do not define partition tables, a real-root `/boot` mount, first-boot kernel arguments, or how to find root and boot again after Ignition has changed the disks.

## What the image must provide

The following is not packaged. Each item is required unless noted.

### Disk layout and filesystem labels

`40ignition-ostree` and Ignition look up filesystems by label:

- `root` — root filesystem (typically XFS)
- `boot` — boot filesystem (typically ext4, XBOOTLDR)
- `EFI-SYSTEM` — ESP (vfat)

The BIOS boot partition carries no filesystem, so it cannot be found by a filesystem label. `40ignition-ostree`'s transposefs save step instead looks it up by **GPT partition name** via `/dev/disk/by-partlabel/BIOS-BOOT`. Make sure your partition table sets that partition name, not just a type GUID — image builders that only set the partition type and skip the name will leave `/dev/disk/by-partlabel/BIOS-BOOT` missing, and `ignition-ostree-transposefs-save.service` fails with `cat: /dev/disk/by-partlabel/BIOS-BOOT: No such file or directory`.

A GPT layout that matches Fedora CoreOS-style images:

| Partition | Size (example) | Type | Filesystem | Label |
| --- | --- | --- | --- | --- |
| BIOS-BOOT | 1 MiB | `21686148-6449-6E6F-744E-656564454649` | — | `BIOS-BOOT` (GPT partition name) |
| ESP | 127 MiB | `C12A7328-F81F-11D2-BA4B-00A0C93EC93B` | vfat | `EFI-SYSTEM` (filesystem label) |
| boot (XBOOTLDR) | 384 MiB | `BC13C2FF-59E6-4262-A352-B275FD6F7172` | ext4 | `boot` (filesystem label) |
| root | remainder | `4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709` (x86_64) | xfs | `root` (filesystem label) |

Reserve GPT extra padding at the end of the disk if you use CoreOS-style GPT header relocation (2048 sectors).

With `image-builder-cli`, put this in `/usr/lib/image-builder/bootc/disk.yaml` inside the container. Other image builders need an equivalent partition table. Each filesystem partition needs an explicit `payload.mountpoint`; without one, `image-builder` cannot tell which partition is root and fails with `failed to initialize bootc distro: missing required info: DefaultRootFs` instead of reading the root filesystem type from `disk.yaml`. The BIOS-BOOT partition needs the partition-level `label` field (the GPT partition name) — not `payload.label`, which only sets a filesystem label and does not apply here since this partition has no filesystem. `mount_configuration: "none"` keeps `image-builder` from baking its own fstab/mount units, since `/boot` gets a static fstab/`boot.mount` only from your own unit (below) and the ESP is mounted on demand rather than persistently:

```yaml
mount_configuration: "none"
partition_table:
  type: "gpt"
  partitions:
    - size: "1 MiB"
      type: "21686148-6449-6E6F-744E-656564454649"
      bootable: true
      label: "BIOS-BOOT"
    - size: "127 MiB"
      type: "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
      payload_type: "filesystem"
      payload:
        type: "vfat"
        label: "EFI-SYSTEM"
        mountpoint: "/boot/efi"
    - size: "384 MiB"
      type: "BC13C2FF-59E6-4262-A352-B275FD6F7172"
      payload_type: "filesystem"
      payload:
        type: "ext4"
        label: "boot"
        mountpoint: "/boot"
    - type: "4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709"
      payload_type: "filesystem"
      payload:
        type: "xfs"
        label: "root"
        mountpoint: "/"
```

If Ignition may rewrite `/boot`, do not bake a static fstab/`boot.mount` from the installer for that filesystem. For bootc:

```toml
[install]
root-mount-spec = "LABEL=root"
boot-mount-spec = ""
```

Write that to `/usr/lib/bootc/install/10mount-spec.toml` (or the equivalent install config). Provide your own `/boot` mount unit on the real root instead (below).

### First-boot kernel argument and stamp

Ignition runs only when `ignition.firstboot` is on the kernel command line. The initrd generator queues `ignition-complete.target` (fetch, disks, files, and the integration units attached to that target) only in that case. Without the argument, the initrd takes `ignition-subsequent.target` and Ignition does not run.

Neither `ignition` nor `ignition-integration` writes that argument. `ignition-grub` does not write a literal `ignition.firstboot` into BLS either. It defines the GRUB variable `$ignition_firstboot`, which is set only when a stamp file exists on the boot filesystem. GRUB's root is that filesystem, so the snippet checks `/ignition.firstboot`. From the OS, when `/boot` is that filesystem, the same file is `/boot/ignition.firstboot`.

Put `$ignition_firstboot` on the installed kernel command line. bootc-image-builder writes the first-boot stamp, and an empty Ignition config so the first-boot path still runs, from this customization:

```toml
[customizations.kernel]
append = "$ignition_firstboot ignition.platform.id=qemu systemd.gpt_auto=0"

[customizations.ignition.firstboot]
empty = true
```

`[customizations.ignition.firstboot]` is a bootc-image-builder customization, not an Ignition config field. `empty = true` creates `/boot/ignition.firstboot`. Do not create that file with `touch` while building the container.

- `$ignition_firstboot` — expanded by GRUB to `ignition.firstboot` when `/ignition.firstboot` exists on the boot filesystem
- `ignition.platform.id` — required; see [supported platforms](supported-platforms.md)
- `systemd.gpt_auto=0` — recommended with XBOOTLDR + composefs (see GPT auto-mount below)

After a successful first boot, stop passing `ignition.firstboot`. When the bootloader adds that argument because the stamp exists, remove the stamp and `/boot/ignition`. A typical unit:

```ini
[Unit]
Description=Mark Ignition Boot Complete
ConditionKernelCommandLine=ignition.firstboot
ConditionPathExists=!/run/ostree-live
ConditionPathExists=/boot/ignition.firstboot
RequiresMountsFor=/boot

[Service]
Type=oneshot
RemainAfterExit=yes
MountFlags=slave
# /boot/ignition may include a baked config:
# https://github.com/coreos/fedora-coreos-tracker/issues/889
# `rm` of the stamp fails the unit if it is already gone, so Ignition cannot run again unnoticed.
ExecStart=/bin/bash -c 'mount -o remount,rw /boot && rm -rf /boot/ignition && rm /boot/ignition.firstboot'

[Install]
WantedBy=basic.target
```

Nothing in `ignition` or `ignition-integration` ships this unit. Enable it from a systemd preset. `MountFlags=slave` gives the service a private writable `/boot` without making `/boot` writable for the rest of the system ([ostree#1265](https://github.com/ostreedev/ostree/issues/1265)).

### Persistent `/boot` mount

If the installer does not emit a `/boot` fstab entry, the real root will not mount `/boot` unless you add a unit. Firstboot-complete, bootupd, and BLS edits need `/boot`.

```ini
[Unit]
Description=Boot partition

[Mount]
What=/dev/disk/by-label/boot
Where=/boot
Type=ext4
Options=defaults

[Install]
WantedBy=local-fs.target
```

Adjust `Type=` if `/boot` is not ext4.

### systemd-gpt-auto-generator

An XBOOTLDR boot partition causes systemd to automount the ESP at `/efi`. That directory is often missing on a composefs root, so `efi.automount` fails. If Ignition recreates the ESP, the original `EFI-SYSTEM` label may also be gone.

fedora-coreos-config does not keep a persistent `/boot/efi` mount; bootupd mounts ESPs on demand. In a bootc image, remove or mask the generator and pass `systemd.gpt_auto=0`:

```
rm -f /usr/lib/systemd/system-generators/systemd-gpt-auto-generator
```

### Initramfs rebuild

Consumer dracut modules and `dracut.conf.d` snippets must be in the image **before** `dracut -f`. Until that rebuild, neither `ignition-integration` nor your own modules are in the initramfs.

### After Ignition rewrites disks

`40ignition-ostree` transposefs copies root, boot, and ESP into RAM so Ignition can repartition or reformat, then restores those filesystems onto the new devices for the rest of first boot. That is only the first-boot hand-off.

Anything that must still mount after the disks change is the image’s responsibility, via **consumer dracut modules** (not this subpackage):

- Remaining initramfs mounts (`sysroot`, `/sysroot/boot`, and any ESP the later stages need) must resolve on the devices Ignition just created.
- Subsequent boots must find the same filesystems (by label, UUID, or whatever your cmdline uses) so `/`, `/boot`, and related mounts succeed.

If the new layout needs extra assembly or kernel arguments (device-mapper, RAID, LUKS, multipath, and so on), inject or enable that from the initramfs after Ignition’s disk stage and persist whatever the next boot needs (for example BLS `options`). Fedora CoreOS does this with `rdcore rootmap` in `coreos-boot-edit`; that helper is not part of `ignition-integration`.

Do not assume `/sysroot/boot` is already the boot filesystem after transposefs restore; it is often only a directory on the restored root until something mounts the boot device there.

## Boot sequence

1. Image contains `ignition`, `ignition-integration`, `ignition-grub`, disk labels, `$ignition_firstboot` in BLS, the `/boot/ignition.firstboot` stamp, `/boot` mount unit, consumer dracut modules for post-disk mounts, and a rebuilt initramfs.
2. First boot: GRUB finds `/ignition.firstboot` on the boot filesystem and expands `$ignition_firstboot` to `ignition.firstboot`. Ignition runs. If the config rewrites disks, transposefs saves and restores root/boot/ESP; consumer modules make those devices mountable for the rest of the initramfs and for later boots.
3. Real root: firstboot-complete removes the stamp (and `/boot/ignition`).
4. Later boots: consumer-provided initramfs logic still locates root and boot on the layout Ignition left behind.

## Checklist

- [ ] Install `ignition-integration`
- [ ] Partition and filesystem labels `boot`, `root`, and ESP
- [ ] Root mounted by `LABEL=root`; no installer-generated `/boot` fstab if Ignition may rewrite boot
- [ ] `$ignition_firstboot` in the bootc-image-builder kernel append, and `[customizations.ignition.firstboot] empty = true` so the installer writes the stamp
- [ ] `ignition.platform.id` set for the target platform
- [ ] Unit to mount `/boot` on the real root
- [ ] Disable systemd GPT auto-mount if using XBOOTLDR + composefs
- [ ] Service that deletes the firstboot stamp after success
- [ ] Consumer dracut modules so root, boot, and related mounts still work after Ignition rewrites disks (first boot and later boots)
- [ ] Run `dracut` after all dracut content is in the image
