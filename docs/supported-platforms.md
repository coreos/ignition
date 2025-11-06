---
nav_order: 8
---

# Supported Platforms

Ignition is currently supported for the following platforms:

* [Akamai Connected Cloud] (`akamai`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys and network configuration are handled separately.
* [Alibaba Cloud] (`aliyun`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [Apple Hypervisor] (`applehv`) - Ignition will read its configuration using an HTTP GET over a vsock connection with its host on port 1024.
* [Amazon Web Services] (`aws`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [Microsoft Azure] (`azure`)- Ignition will read its configuration from the custom data provided to the instance. Cloud SSH keys are handled separately.
* [Microsoft Azure Stack] (`azurestack`) - Ignition will read its configuration from the custom data provided to the instance. Cloud SSH keys are handled separately.
* [Brightbox] (`brightbox`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [CloudStack] (`cloudstack`) - Ignition will read its configuration from the instance userdata via either metadata service or config drive. Cloud SSH keys are handled separately.
* [DigitalOcean] (`digitalocean`) - Ignition will read its configuration from the droplet userdata. Cloud SSH keys and network configuration are handled separately.
* [Exoscale] (`exoscale`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [Google Cloud] (`gcp`) - Ignition will read its configuration from the instance metadata entry named "user-data". Cloud SSH keys are handled separately.
* [Hetzner Cloud] (`hetzner`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [Microsoft Hyper-V] (`hyperv`) - Ignition will read its configuration from the `ignition.config` key in pool 0 of the Hyper-V Data Exchange Service (KVP). Values are limited to approximately 1 KiB of text, so Ignition can also read and concatenate multiple keys named `ignition.config.0`, `ignition.config.1`, and so on.
* [IBM Cloud] (`ibmcloud`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [KubeVirt] (`kubevirt`) - Ignition will read its configuration from the instance userdata via `cloudInitConfigDrive` or `cloudInitNoCloud`. Cloud SSH keys are handled separately.
* Bare Metal (`metal`) - Use the `ignition.config.url` kernel parameter to provide a URL to the configuration. The URL can use the `http://`, `https://`, `tftp://`, `s3://`, `arn:`, or `gs://` schemes to specify a remote config.
* [Nutanix] (`nutanix`) - Ignition will read its configuration from the instance userdata via config drive. Cloud SSH keys are handled separately.
* [NVIDIA BlueField] (`nvidiabluefield`) - Ignition will read its configuration from the bootfifo sysfs interface from the mlxbf_bootctl platform driver.
* [OpenStack] (`openstack`) - Ignition will read its configuration from the instance userdata via either metadata service or config drive. Cloud SSH keys are handled separately.
* [Oracle Cloud Infrastucture] (`oraclecloud`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [Proxmox VE] (`proxmoxve`) - Ignition will read its configuration from the instance userdata via config drive. If there isn't any valid Ignition configuration in userdata it will check the vendordata next. Cloud SSH keys are handled separately.
* [Equinix Metal] (`packet`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [IBM Power Systems Virtual Server] (`powervs`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [QEMU] (`qemu`) - Ignition will read its configuration from the 'opt/com.coreos/config' key on the QEMU Firmware Configuration Device (available in QEMU 2.4.0 and higher).
* [Scaleway] (`scaleway`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [UpCloud] (`upcloud`) - Ignition will read its configuration from the instance userdata fetched from the metadata service (which is NOT enabled by default, make sure you enable it if you use custom images). Cloud SSH keys are handled separately.
* [VirtualBox] (`virtualbox`) - Use the VirtualBox guest property `/Ignition/Config` to provide the config to the virtual machine.
* [VMware] (`vmware`) - Use the VMware Guestinfo variables `ignition.config.data` and `ignition.config.data.encoding` to provide the config and its encoding to the virtual machine. Valid encodings are "", "base64", and "gzip+base64". Guestinfo variables can be provided directly or via an OVF environment, with priority given to variables specified directly.
* [Vultr] (`vultr`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [zVM] (`zvm`) - Ignition will read its configuration from the reader device directly. The vmur program is necessary, which requires the vmcp and vmur kernel module as prerequisite, and the corresponding z/VM virtual unit record devices (in most cases 000c as reader, 000d as punch) must be set online.

Ignition is under active development, so this list may grow over time.

For most cloud providers, cloud SSH keys and custom network configuration are handled by [Afterburn].

[Akamai Connected Cloud]: https://www.linode.com
[Alibaba Cloud]: https://www.alibabacloud.com/product/ecs
[Apple Hypervisor]: https://developer.apple.com/documentation/hypervisor
[Amazon Web Services]: https://aws.amazon.com/ec2/
[Microsoft Azure]: https://azure.microsoft.com/en-us/services/virtual-machines/
[Microsoft Azure Stack]: https://azure.microsoft.com/en-us/overview/azure-stack/
[BrightBox]: https://www.brightbox.com/cloud/servers/
[CloudStack]: https://cloudstack.apache.org/
[DigitalOcean]: https://www.digitalocean.com/products/droplets/
[Exoscale]: https://www.exoscale.com/compute/
[Google Cloud]: https://cloud.google.com/compute
[Hetzner Cloud]: https://www.hetzner.com/cloud
[Microsoft Hyper-V]: https://learn.microsoft.com/en-us/virtualization/hyper-v-on-windows/
[IBM Cloud]: https://www.ibm.com/cloud/vpc
[KubeVirt]: https://kubevirt.io
[Nutanix]: https://www.nutanix.com/products/ahv
[OpenStack]: https://www.openstack.org/
[Oracle Cloud Infrastucture]: https://www.oracle.com/cloud
[Proxmox VE]: https://www.proxmox.com/en/proxmox-virtual-environment/overview
[Equinix Metal]: https://metal.equinix.com/product/
[IBM Power Systems Virtual Server]: https://www.ibm.com/products/power-virtual-server
[QEMU]: https://www.qemu.org/
[Scaleway]: https://www.scaleway.com
[UpCloud]: https://www.upcloud.com
[VirtualBox]: https://www.virtualbox.org/
[VMware]: https://www.vmware.com/
[Vultr]: https://www.vultr.com/products/cloud-compute/
[zVM]: http://www.vm.ibm.com/overview/

[Afterburn]: https://coreos.github.io/afterburn/
