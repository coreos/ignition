#!/bin/bash
# Installs systemd drop-ins so Ignition dracut units log to kmsg.

install() {
    services=(
        ignition-disks.service
        ignition-enable-network.service
        ignition-fetch-offline.service
        ignition-fetch.service
        ignition-files.service
        ignition-kargs.service
        ignition-mount.service
        ignition-ostree-mount-var.service
        ignition-ostree-populate-var.service
        ignition-ostree-transposefs-detect.service
        ignition-ostree-transposefs-restore.service
        ignition-ostree-transposefs-save.service
        ignition-remount-sysroot.service
    )
    for u in "${services[@]}"; do
        inst_dir "$systemdsystemunitdir/${u}.d"
        inst_simple "$moddir/10-stdout-kmsg.conf" \
            "$systemdsystemunitdir/${u}.d/10-stdout-kmsg.conf"
    done

    inst_simple "$moddir/00-journal-log-level-kmsg.conf" \
        "/etc/systemd/journald.conf.d/00-journal-log-level-kmsg.conf"
}
