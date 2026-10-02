check() {
    if [[ $IN_KDUMP == 1 ]]; then
        return 1
    fi

    # This module will only work with NetworkManager now.
    # We might add support for other later.
    if dracut_module_included "network-manager"; then
        return 0
    fi

    return 1
}

depends() {
    echo network
}

install_and_enable_unit() {
    unit="$1"; shift
    target="$1"; shift
    inst_simple "$moddir/$unit" "$systemdsystemunitdir/$unit"
    # note we `|| exit 1` here so we error out if e.g. the units are missing
    # see https://github.com/coreos/fedora-coreos-config/issues/799
    $SYSTEMCTL -q --root="$initdir" add-requires "$target" "$unit" || exit 1
}

install() {
    inst_simple "$moddir/ignition-enable-network.sh" \
        "/usr/sbin/ignition-enable-network"
    install_and_enable_unit "ignition-enable-network.service" \
        "ignition-complete.target"
}
