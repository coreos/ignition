Butane: make `with_mount_unit` mount units require network access for
filesystems on LUKS devices whose custom Clevis pin sets `needs_network`,
matching the `_netdev` option Ignition writes to `/etc/crypttab`
