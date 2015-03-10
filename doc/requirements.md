# Requirements #
This utility will be responsible for provisioning the system before boot. This
includes partitioning disks, formatting partitions, and creating files. These
features are discussed in more detail below.

The configuration for the utility will be provided by the user via a file on
the system, a URI to a remote file via a kernel parameter, or via a provider-
specific user data mechanism. To ease debugging, the configuration will be
human readable. The configuration will require versioning in order to remain
backward compatible. In the interest of explicitness, including the version in
the configuration will be required.

This utility will evaluate the configuration on the first boot only. In
addition to simplifying the mental model, this will make it clear to users that
this can not be used for configuration management.

## Features ##

### Users ###
- create users
 - name
 - password hash
 - ssh keys
 - home directory
 - groups
 - system
 - primary group
 - no user group
 - no create home
 - GECOS
 - no log init

### Disks ###
- partition disks
- create LVM volumes
- create RAID volumes
- format partitions
- optionally force reformat
- write files to filesystems
 - content
 - owner
 - permissions

### systemd Units ###
- write unit files
 - name
 - mask
 - enable
 - content
 - dropins
  - name
  - content

### networkd Units ###
- write networkd files
 - name
 - content

## OEM Metadata ##
Instance metadata (e.g. public and private addresses) will not be handled by
this utility. Instead a service, required by a metadata target, will discover
and expose the metadata during boot. Any service which requires that metadata
(e.g. etcd and fleet) will use systemd to express the requirement on the
metadata target. This will allow both the OS and the user to add services
that provide and depend on the metadata target.

Here is an example of a metadata-requiring service:

```
[Unit]
Description=etcd
Requires=coreos-metadata.target
After=coreos-metadata.target

[Service]
User=etcd
PermissionsStartOnly=true
EnvironmentFile=/run/coreos/metadata
Environment=ETCD_DATA_DIR=/var/lib/etcd
Environment=ETCD_NAME=%m
ExecStart=/usr/bin/etcd \
          --addr=${COREOS_IPV4_PUBLIC}:2379 \
          --peer-addr=${COREOS_IPV4_PRIVATE}:2380
Restart=always
RestartSec=10s
LimitNOFILE=40000
```

### coreos-metadata.target ###
Services providing metadata must install themselves with "RequiredBy" or
"WantedBy" under "coreos-metadata.target". The service must append the provided
environment variables to /run/coreos/metadata. On supported platforms (and if
the metadata exists), the OS will provide the services to fetch the following
environment variables:

 - COREOS_IPV4_PUBLIC
 - COREOS_IPV4_PRIVATE
 - COREOS_IPV6_PUBLIC
 - COREOS_IPV6_PRIVATE

## OEM-Specific Requirements ##

### Azure ###
#### Userdata ####
1. Mount the provisioning DVD
2. Parse the CustomData from ovf-env.xml on the DVD

#### Metadata ####
Functionality provided by wa-linux-agent. Requires python.

#### Network ####
DHCP

### CloudSigma ###
#### Userdata ####
Read from /dev/ttyS0

#### Metadata ####
Read from /dev/ttyS0

#### Network ####

### CloudStack ###
#### Metadata ####
Fetch from http://<DHCP Server>/latest/{public,local}-ipv4
Fetch from http://<DHCP Server>/latest/public-keys

#### Userdata ####
Fetch from http://<DHCP Server>/latest/user-data

#### Network ####
DHCP

### DigitalOcean ###
#### Metadata ####
Fetch from http://169.254.169.254/metadata/v1.json

#### Userdata ####
Fetch from http://169.254.169.254/metadata/v1/user-data or use metadata

#### Network ####
1. Parse hostname, nameservers, and interfaces from metadata
2. Use this information to write networkd units

### EC2-Compat ###
#### Metadata ####
Fetch from http://169.254.169.254/2009-04-04/meta-data

#### Userdata ####
Fetch from http://169.254.169.254/2009-04-04/user-data

#### Network ####
DHCP

### Exoscale ###
#### Metadata ####
Fetch from http://<DHCP Server>/latest/{public,local}-ipv4
Fetch from http://<DHCP Server>/latest/public-keys

#### Userdata ####
Fetch from http://<DHCP Server>/latest/user-data

#### Network ####
1. DHCP
2. ethtool -K eth0 tso off gso off

### GCE ###
#### Metadata ####
Fetch from http://169.254.169.254/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip
Fetch from http://169.254.169.254/computeMetadata/v1/instance/network-interfaces/0/ip
Fetch from http://169.254.169.254/computeMetadata/v1beta1/{project,instance}/attributes/sshKeys

#### Userdata ####
Fetch from http://169.254.169.254/computeMetadata/v1/{project,instance}/attributes/user-data

#### Network ####
Add "169.254.169.254 metadata" to /etc/hosts

### Hyper-V ###
#### Metadata ####
N/A

#### Userdata ####
N/A

#### Network ####
DHCP

### NiftyCloud ###
#### Metadata ####
Read SSH keys via vmtoolsd

#### Userdata ####
Read userdata via vmtoolsd

#### Network ####
DHCP (DNS is hardcoded to google)

### Rackspace ###
?

### Rackspace-OnMetal ###
#### Metadata ####
Read from /media/configdrive/openstack/<version>/meta_data.json

#### Userdata ####
Read from /media/configdrive/openstack/<version>/user_data

#### Network ####
1. Read from /media/configdrive/openstack/<version>/vendor_data.json
2. Parse hostname, nameservers, and interfaces from network_info
3. Use this information to write networkd units

### Vagrant ###
#### Metadata ####
N/A

#### Userdata ####
Read from /var/lib/coreos-vagrant/vagrantfile-user-data

#### Network ####
DHCP

### VMware ###
#### Userdata ####
Read from /media/configdrive/openstack/latest/user_data

#### Metadata ####
N/A

#### Network ####
DHCP

### PXE ###
#### Userdata ####
Provided via the CPIO or as a URL to the boot parameters

#### Metadata ####
N/A

#### Network ####
DHCP

### iPXE ###
#### Userdata ####
Provided via the CPIO or as a URL to the boot parameters

#### Metadata ####
N/A

#### Network ####
networkd units must be provided in the userdata

### Disk Install ###
#### Userdata ####
Read from /var/lib/coreos-install/user_data

#### Metadata ####
N/A

#### Network ####
DHCP or networkd units provided in the userdata
