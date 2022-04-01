---
nav_order: 8
---

# Supported Platforms

Ignition is currently only supported for the following platforms:

* [Alibaba Cloud] (`aliyun`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [Amazon Web Services] (`aws`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [Microsoft Azure] (`azure`)- Ignition will read its configuration from the custom data provided to the instance. Cloud SSH keys are handled separately.
* [Microsoft Azure Stack] (`azurestack`) - Ignition will read its configuration from the custom data provided to the instance. Cloud SSH keys are handled separately.
* [Brightbox] (`brightbox`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [CloudStack] (`cloudstack`) - Ignition will read its configuration from the instance userdata via either metadata service or config drive. Cloud SSH keys are handled separately.
* [DigitalOcean] (`digitalocean`) - Ignition will read its configuration from the droplet userdata. Cloud SSH keys and network configuration are handled separately.
* [Exoscale] (`exoscale`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [Google Cloud] (`gcp`) - Ignition will read its configuration from the instance metadata entry named "user-data". Cloud SSH keys are handled separately.
* [IBM Cloud] (`ibmcloud`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [KubeVirt] (`kubevirt`) - Ignition will read its configuration from the instance userdata via config drive. Cloud SSH keys are handled separately.
* Bare Metal (`metal`) - Use the `ignition.config.url` kernel parameter to provide a URL to the configuration. The URL can use the `http://`, `https://`, `tftp://`, `s3://`, or `gs://` schemes to specify a remote config.
* [Nutanix] (`nutanix`) - Ignition will read its configuration from the instance userdata via config drive. Cloud SSH keys are handled separately.
* [OpenStack] (`openstack`) - Ignition will read its configuration from the instance userdata via either metadata service or config drive. Cloud SSH keys are handled separately.
* [Equinix Metal] (`packet`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [IBM Power Systems Virtual Server] (`powervs`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [QEMU] (`qemu`) - Ignition will read its configuration from the 'opt/com.coreos/config' key on the QEMU Firmware Configuration Device (available in QEMU 2.4.0 and higher).
* [VirtualBox] (`virtualbox`) - Use the VirtualBox guest property `/Ignition/Config` to provide the config to the virtual machine.
* [VMware] (`vmware`) - Use the VMware Guestinfo variables `ignition.config.data` and `ignition.config.data.encoding` to provide the config and its encoding to the virtual machine. Valid encodings are "", "base64", and "gzip+base64". Guestinfo variables can be provided directly or via an OVF environment, with priority given to variables specified directly.
* [Vultr] (`vultr`) - Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.
* [zVM] (`zvm`) - Ignition will read its configuration from the reader device directly. The vmur program is necessary, which requires the vmcp and vmur kernel module as prerequisite, and the corresponding z/VM virtual unit record devices (in most cases 000c as reader, 000d as punch) must be set online.

Ignition is under active development, so this list may grow over time.

For most cloud providers, cloud SSH keys and custom network configuration are handled by [Afterburn].

[Alibaba Cloud]: https://www.alibabacloud.com/product/ecs
[Amazon Web Services]: https://aws.amazon.com/ec2/
[Microsoft Azure]: https://azure.microsoft.com/en-us/services/virtual-machines/
[Microsoft Azure Stack]: https://azure.microsoft.com/en-us/overview/azure-stack/
[BrightBox]: https://www.brightbox.com/cloud/servers/
[CloudStack]: https://cloudstack.apache.org/
[DigitalOcean]: https://www.digitalocean.com/products/droplets/
[Exoscale]: https://www.exoscale.com/compute/
[Google Cloud]: https://cloud.google.com/compute
[IBM Cloud]: https://www.ibm.com/cloud/vpc
[KubeVirt]: https://kubevirt.io
[Nutanix]: https://www.nutanix.com/products/ahv
[OpenStack]: https://www.openstack.org/
[Equinix Metal]: https://metal.equinix.com/product/
[IBM Power Systems Virtual Server]: https://www.ibm.com/products/power-virtual-server
[QEMU]: https://www.qemu.org/
[VirtualBox]: https://www.virtualbox.org/
[VMware]: https://www.vmware.com/
[Vultr]: https://www.vultr.com/products/cloud-compute/
[zVM]: http://www.vm.ibm.com/overview/

[Afterburn]: https://coreos.github.io/afterburn/
