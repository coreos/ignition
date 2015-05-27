This is a brain dump of a configuration file.

```yaml
version: 1

storage:
  disks:
    - device: "/dev/sda"
      wipe-table: true
      partitions:
        - label: "raid.1.1"
          number: 1
          type-guid: "EBD0A0A2-B9E5-4433-87C0-68B6B72699C7"
          start: 1MiB
          size: 10GiB
    - device: "/dev/sdb"
      wipe-table: true
      partitions:
        - label: "raid.1.2"
          number: 1
          size: 10GiB
    - device: "/dev/sdc"
      wipe-table: true
      partitions:
        - label: "raid.1.3"
          number: 1
          size: 10GiB

  raid:
    - name: "md0"
      level: stripe
      devices:
        - "/dev/disk/by-partlabel/raid.1.1"
        - "/dev/disk/by-partlabel/raid.1.2"
        - "/dev/disk/by-partlabel/raid.1.3"
      spares: 1

  filesystems:
    - device: "/dev/disk/by-partlabel/ROOT" # switch coreos' ext4 root to btrfs
      format: btrfs
      initialize: true
      format-options:
        - "--force"
        - "--label=ROOT"
      files:
        - path: "/home/core/bin/find-ip4.sh"
          permissions: 0755
          content: |
            #!/bin/sh
            get_ipv4() {
                IFACE="${1}"
                FILE="${2}"
                VARIABLE="${3}"

                local ip
                while [ -z "${ip}" ]; do
                    ip=$(ip -4 -o addr show dev "${IFACE}" scope global | gawk '{split ($4, out, "/"); print out[1]}')
                    sleep .1
                done

                echo "${ip}"
            }

            sed -i -e "/^${VARIABLE}=/d" "${FILE}"
            echo "${VARIABLE}=${ip}" >> "${FILE}"

systemd:
  units:
    - name: find-ips.service
      content: |
        [Unit]
        Requires: network-online.target
        After: network-online.target

        [Service]
        Type=oneshot
        ExecStart=/home/core/bin/find-ip4.sh enp0s7 /run/environment COREOS_PRIVATE_IPV4
        ExecStart=/home/core/bin/find-ip4.sh enp0s8 /run/environment COREOS_PUBLIC_IPV4
    - name: etcd.service
      enable: true
      drop-ins:
        - name: install.conf
          content: |
           [Unit]
           Requires=find-ips.service
           After=find-ips.service

           [Service]
           EnvironmentFile=/run/environment

           [Install]
           WantedBy=multi-user.target

networkd:
  units:
    - name: 00-eth0.network
      content: |
        [Match]
        Name=eth0

        [Address]
        Address=10.0.0.2/24

        [Route]
        Gateway=10.0.0.1

        [Network]
        DNS=4.4.4.4
```
