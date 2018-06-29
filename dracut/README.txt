
# for Fedora (classic)
    dnf install -y dracut-network git-core gdisk
    dnf install -y --nogpgcheck --repofrompath 'copr,https://copr-be.cloud.fedoraproject.org/results/dustymabe/ignition/fedora-$releasever-$basearch/' ignition ignition-dracut
    dracut --force --verbose -N
    rm /etc/machine-id
    add 'ip=dhcp rd.neednet=1 enforcing=0 coreos.firstboot' to /boot/grub2/grub.cfg

# for Fedora Atomic Host
    rpm-ostree initramfs --enable
    rpm-ostree ex kargs --append 'ip=dhcp rd.neednet=1 enforcing=0 coreos.firstboot=1'
    curl -L https://copr.fedorainfracloud.org/coprs/dustymabe/ignition/repo/fedora-28/dustymabe-ignition-fedora-28.repo > /etc/yum.repos.d/copr.repo
    rpm-ostree install ignition ignition-dracut


# for CentOS (classic)
    yum install -y dracut-network git-core gdisk
    curl -L https://copr.fedorainfracloud.org/coprs/dustymabe/ignition/repo/epel-7/dustymabe-ignition-epel-7.repo > /etc/yum.repos.d/copr.repo
    yum install -y ignition ignition-dracut
    rm /etc/yum.repos.d/copr.repo
    dracut --force --verbose -N
    rm /etc/machine-id
    add 'ip=dhcp rd.neednet=1 enforcing=0 coreos.firstboot' to /boot/grub2/grub.cfg
