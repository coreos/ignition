# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

check() {
    if [[ $IN_KDUMP == 1 ]]; then
        return 1
    fi
}

# 1) Fix for https://bugzilla.redhat.com/show_bug.cgi?id=1918244
# On s390x systems with IBM 2810XIV discs multipath couldn't be configured
# because SCSI_IDENT_* udev properties are not set at boot time
# 2-) Fix for https://bugzilla.redhat.com/show_bug.cgi?id=1990506
# Missing symlinks to disk install
install() {
    inst_simple sg_inq
    inst_rules 61-scsi-sg3_id.rules
    inst_rules 63-scsi-sg3_symlink.rules
}
