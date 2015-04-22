This is a brain dump of a configuration file.

```yaml
storage:
  disks:
    - device: "/dev/sda"
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
      partitions:
        - name: "raid.1.2"
          size: 10GB
    - device: "/dev/sdc"
      partitions:
        - name: "raid.1.3"
          size: 10GB

  raid:
    - name: "md0"
      level: stripe
      devices:
        - name: "/dev/disk/by-label/raid.1.1"
        - name: "raid.1.2"
      spares:
        - name: "raid.1.3"

  filesystems:
    - device: "/dev/disk/by-label/ROOT"
      format: btrfs
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
```
