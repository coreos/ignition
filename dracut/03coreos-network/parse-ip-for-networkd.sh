#!/bin/sh
#
# This script was mostly stolen from 40network/parse-ip-opts.sh.  Its
# actions are adapted to write .network files to /etc/systemd/network
# in the initramfs instead of using separate DHCP commands, etc.  Note
# the bashisms.
#

command -v getarg >/dev/null          || . /lib/dracut-lib.sh
command -v ip_to_var >/dev/null       || . /lib/net-lib.sh

if [ -n "$netroot" ] && [ -z "$(getarg ip=)" ] && [ -z "$(getarg BOOTIF=)" ]; then
    # No ip= argument(s) for netroot provided, defaulting to DHCP
    return;
fi

function mask2cidr() {
    local -i bits=0
    for octet in ${1//./ }; do
        for i in {0..8}; do
            [ "$octet" -eq $(( 256 - (1 << i) )) ] && bits+=$((8-i)) && break
        done
        [ $i -eq 8 -a "$octet" -ne 0 ] && warn "Bad netmask $mask" && return
        [ $i -gt 0 ] && break
    done
    echo $bits
}

# Check ip= lines
# XXX Would be nice if we could errorcheck ip addresses here as well
for p in $(getargs ip=); do
    ip_to_var $p

    # Empty autoconf defaults to 'dhcp'
    if [ -z "$autoconf" ] ; then
        warn "Empty autoconf values default to dhcp"
        autoconf="dhcp"
    fi

    # Convert the netmask to CIDR notation
    if [[ "x$mask" =~ ^x[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        cidr=$(mask2cidr "$mask")
    elif [ -n "$mask" -a "x${mask//[0-9]/}" = 'x' ]; then
        # The mask is already a prefix length (uint), so validate it
        [[ "x$ip" == x*:*:* && "$mask" -le 128 || "$mask" -le 32 ]] && cidr=$mask
    fi

    # Error checking for autoconf in combination with other values
    for autoopt in $(str_replace "$autoconf" "," " "); do
        case $autoopt in
            error) die "Error parsing option 'ip=$p'";;
            auto6|ibft|bootp|rarp|both) die "Sorry, ip=$autoopt is currenty unsupported";;
            none|off)
                [ -z "$ip" ] && \
                    die "For argument 'ip=$p'\nValue '$autoopt' without static configuration does not make sense"
                [ -z "$mask" ] && \
                    die "Sorry, automatic calculation of netmask is not yet supported"
                [ -z "$cidr" ] && \
                    die "For argument 'ip=$p'\nSorry, failed to convert netmask '$mask' to CIDR"
                ;;
            dhcp|dhcp6|on|any) \
                [ -n "$NEEDBOOTDEV" ] && [ -z "$dev" ] && \
                    die "Sorry, 'ip=$p' does not make sense for multiple interface configurations"
                [ -n "$ip" ] && \
                    die "For argument 'ip=$p'\nSorry, setting client-ip does not make sense for '$autoopt'"
                ;;
            *) die "For argument 'ip=$p'\nSorry, unknown value '$autoopt'";;
        esac
    done

    # Enough validation, write the network file
    # Count down so that early ip= arguments are overridden by later ones
    _net_file=/etc/systemd/network/10-dracut-cmdline-$(( 99 - _net_count++ )).network
    echo '[Match]' > $_net_file
    [ -n "$dev" ] && echo "Name=$dev" >> $_net_file
    echo '[Link]' >> $_net_file
    [ -n "$macaddr" ] && echo "MACAddress=$macaddr" >> $_net_file
    [ -n "$mtu" ] && echo "MTUBytes=$mtu" >> $_net_file
    echo '[Network]' >> $_net_file
    [ "x$autoconf" = xoff -o "x$autoconf" = xnone ] &&
        echo DHCP=no >> $_net_file || echo DHCP=yes >> $_net_file
    [ -n "$gw" ] && echo "Gateway=$gw" >> $_net_file
    [ -n "$dns1" ] && echo "DNS=$dns1" >> $_net_file
    [ -n "$dns2" ] && echo "DNS=$dns2" >> $_net_file
    echo '[Address]' >> $_net_file
    [ -n "$ip" ] && echo "Address=$ip/${cidr:-24}" >> $_net_file
    [ -n "$srv" ] && echo "Peer=$srv" >> $_net_file
    echo '[DHCP]' >> $_net_file
    [ -n "$hostname" ] && echo "Hostname=$hostname" >> $_net_file
done
