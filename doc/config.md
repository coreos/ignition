This is a brain dump of a configuration file.

```yaml
storage:
  disks:
    - device: "/dev/sda"
      wipe-table: true
      partitions:
        - label: "raid.1.1"
          number: 1
          first-sector: 1
          last-sector:
          type:
            hex-code:
            guid:
          size: 10GB
    - device: "/dev/sdb"
      wipe-table: true
      partitions:
        - name: "raid.1.2"
          size: 10GB
    - device: "/dev/sdc"
      wipe-table: true
      partitions:
        - name: "raid.1.3"
          size: 10GB

  raid:
    - name: "md0"
      level: stripe
      devices:
        - "/dev/disk/by-partlabel/raid.1.1"
        - "/dev/disk/by-partlabel/raid.1.2"
      spares:
        - "/dev/disk/by-partlabel/raid.1.3"

  filesystems:
    - device: "/dev/disk/by-partlabel/ROOT" # switch coreos' ext4 root to btrfs
      format: btrfs
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
    - device: "/dev/md0"
      name: "vms"
      format: ext4
      files:
        - path: "/images/sparse.img"
          permissions: 0640
          allocate:
            - type: sparse
            - size: 10GB
    - device: "vms:/images/sparse.img"
      format: ext4
      format-options:
        - "-E"
        - "lazy_itable_init=0,lazy_journal_init=0"

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
```
