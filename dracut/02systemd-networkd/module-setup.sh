#!/bin/bash

# This is derived from upstream dracut 043

install() {
    inst_multiple -o \
        $systemdutildir/systemd-networkd \
        $systemdutildir/systemd-resolved \
        /etc/systemd/resolved.conf \
        ip

    inst_simple "$moddir/initrd-systemd-networkd.service" \
        "$systemdsystemunitdir/initrd-systemd-networkd.service"
    
    inst_simple "$moddir/initrd-systemd-resolved.service" \
        "$systemdsystemunitdir/initrd-systemd-resolved.service"
    
    inst_simple "$moddir/99-default.link" \
        "$systemdutildir/network/99-default.link"
    
    inst_simple "$moddir/zz-default.network" \
        "$systemdutildir/network/zz-default.network"

    # user/group required for systemd-networkd
    getent passwd systemd-network >> "$initdir/etc/passwd"
    getent group systemd-network >> "$initdir/etc/group"

    # user/group required for systemd-resolved
    getent passwd systemd-resolve >> "$initdir/etc/passwd"
    getent group systemd-resolve >> "$initdir/etc/group"

    # point /etc/resolv.conf @ systemd-resolved's resolv.conf
    ln -s ../run/systemd/resolve/resolv.conf "$initdir/etc/resolv.conf"

    _arch=$(uname -m)
    inst_libdir_file {"tls/$_arch/",tls/,"$_arch/",}"libnss_dns.so.*" \
                     {"tls/$_arch/",tls/,"$_arch/",}"libnss_mdns4_minimal.so.*" \
                     {"tls/$_arch/",tls/,"$_arch/",}"libnss_myhostname.so.*" \
                     {"tls/$_arch/",tls/,"$_arch/",}"libnss_resolve.so.*"
}
