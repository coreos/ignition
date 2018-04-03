
# for Fedora
    dnf install -y dracut-network git-core
    dnf install -y --nogpgcheck --repofrompath 'copr,https://copr-be.cloud.fedoraproject.org/results/dustymabe/ignition/fedora-$releasever-$basearch/' ignition

# for centos
    yum install -y dracut-network git-core
    curl -L https://copr.fedorainfracloud.org/coprs/dustymabe/ignition/repo/epel-7/dustymabe-ignition-epel-7.repo > /etc/yum.repos.d/copr.repo
    yum install -y ignition
    rm /etc/yum.repos.d/copr.repo

git clone https://github.com/dustymabe/bootengine.git
rsync -avh ./bootengine/dracut/* /usr/lib/dracut/modules.d/

dracut --add 'url-lib network ignition usr-generator' --install /usr/bin/ignition --force --verbose -N

rm /etc/machine-id
touch /coreos_first_boot
add 'coreos.config.url="https://dustymabe.fedorapeople.org/base.ign" ip=eth0:dhcp rd.neednet=1 coreos.first_boot' to grub.cfg

reboot
