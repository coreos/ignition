---
nav_order: 7
---

# Operator Notes
{: .no_toc }

1. TOC
{:toc}

## HTTP Backoff and Retry

When Ignition is fetching a resource over http(s), if the resource is unavailable Ignition will continually retry to fetch the resource with an exponential backoff between requests.

For a given retry attempt, Ignition will wait 10 seconds for the server to send the response headers for the request. If response headers are not received in this time, or an HTTP 5XX error code is received, the request is cancelled, Ignition waits for the backoff, and a new request is made.

Any HTTP response code less than 500 results in the request being completed, and either the resource will be fetched or Ignition will fail.

Ignition will initially wait 100 milliseconds between failed attempts, and the amount of time to wait doubles for each failed attempt until it reaches 5 seconds.

## AWS S3 access

Ignition has built-in support for fetching resources from the Amazon Simple Storage Service (AWS S3). Several URL formats are supported:

| URL format | Supported specs | Semantics | Ignition behavior in Amazon EC2 instance | Ignition behavior outside EC2 |
| - | - | - | - | - |
| `s3://<bucket>/<object-path>` | 3.0.0+ | Fetch the object. | Fetch from the same AWS [partition](https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#arns-syntax) as the instance. Authenticate using the instance's IAM role, or fetch anonymously if no role is available. | Fetch anonymously from the `aws` (public AWS) partition. |
| `arn:<partition>:s3:::<bucket>/<object-path>` | 3.4.0+ | Fetch the object from the specified partition. | Authenticate using the instance's IAM role, or fetch anonymously if no role is available. | Fetch anonymously. |
| `arn:<partition>:s3:<region>:<account>:accesspoint/<access-point>/object/<object-path>` | 3.4.0+ | Fetch the object from the specified access point. Multi-region access points are not supported. | Authenticate using the instance's IAM role, or fail if no role is available. | Fail. Access points don't support anonymous access. |

Append `?versionId=<version>` to any of the URL formats to fetch the specified object version.

## Azure Blob access

Ignition supports fetching resources from Azure Blob Storage. The URL format for Azure Blob Storage is `https://<storageAccount>.blob.core.windows.net/<container>/<fileName>`. Ignition will recognize this format and attempt to authenticate using the [default Azure credential](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential) to fetch the resource via the [Azure Blob Storage API](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob#section-readme).

If the Azure storage blob is public, the resource can be fetched anonymously. For private storage blobs, the resource can only be fetched if valid credentials are available. To configure the credentials, ensure the environment has credentials with the necessary permissions to access the storage account and storage blob. One approach is to configure a managed identity with contributor access to the storage account and assign it to the VM during creation.

## HTTP headers

When fetching data from an HTTP URL for config references, CA references and file contents, additional headers can be attached to the request using the `httpHeaders` attribute. This allows downloading data from servers that require authentication or some additional parameters from your request.

Headers can be attached only when `source` has `http` or `https` scheme.

If multiple values are to be set for the same header, they must be separated by a comma. Example: `{"name": "Accept", "value": "text/html, application/json"}`.

If the remote HTTP server returns a redirect status code (3xx), then additional headers are not included in the redirected request.

If a specified header is one that Ignition sets by default, such as `Accept` or `User-Agent`, the specified value overrides Ignition's default.

## Filesystem-Reuse Semantics

When a machine first boots, it's possible that an earlier installation or other process has already provisioned the disks. The Ignition config can specify the intended filesystem for a given device, and there are three possibilities when Ignition runs:

- There is no preexisting filesystem.
- There is a preexisting filesystem of the correct type, label, or UUID (e.g. the Ignition config says `/dev/sda` should be `ext4`, and it is `ext4`).
- There is a preexisting filesystem of an incorrect type, label, or UUID (e.g. the Ignition config says `/dev/sda` should be `ext4`, and it is `btrfs`).

In the first case, when there is no preexisting filesystem, Ignition will always create the desired filesystem.

In the second two cases, where there is a preexisting filesystem, Ignition's behavior is controlled by the `wipeFilesystem` flag in the `filesystem` section.

If `wipeFilesystem` is set to true, Ignition will always wipe any preexisting filesystem and create the desired filesystem (or skip creation if the format is set to `none`). Note this will result in any data on the old filesystem being lost.

If `wipeFilesystem` is set to false, Ignition will then attempt to reuse the existing filesystem. If the filesystem is of the correct type, has a matching label, and has a matching UUID, then Ignition will reuse the filesystem. If the label or UUID is not set in the Ignition config, they don't need to match for Ignition to reuse the filesystem. Any preexisting data will be left on the device and will be available to the installation. If the preexisting filesystem is *not* of the correct type, then Ignition will fail, and the machine will fail to boot. Similarly, if the format is set to `none`, then any preexisting filesystem will cause Ignition to fail.

## Path Traversal and Following Symlinks

When resolving paths, Ignition follows symlinks on all but the last element of a path. This ensures existing symlinks on a filesystem can be overwritten while still following symlinks as expected. When writing files, links, or directories, Ignition does not allow following symlinks outside the specified filesystem. When writing files, links, or directories on the `root` filesystem, Ignition follows symlinks as if it were executing in that root; a symlink to `/etc` is followed to `/etc` on the `root` filesystem. When writing files, links, or directories to any other filesystem, Ignition fails if it tries to follow a symlink outside that filesystem.

## Making changes to `/proc`, `/sys`, `/dev`, `/tmp` or `/run`

To create files, directories or symlinks in `/proc`, `/sys` or `/dev`, you should use [sysctl.d config files](https://www.mankier.com/5/sysctl.d) or [udev rules](https://www.mankier.com/7/udev).

Similarly, to make changes under the `/tmp` or `/run` paths, you should use [tmpfiles.d config files](https://www.mankier.com/5/tmpfiles.d).

Those paths are expected to be mount points for temporary (`/tmp`) or virtual (`/proc`, `/sys`, `/dev`) filesystems and at the time Ignition runs, none of those paths are mounted. Using Ignition to write to those paths will thus not have the desired effect.

## SELinux

Ignition fully supports distributions which have [SELinux][selinux] enabled. It requires that the distribution ships the [`setfiles`][setfiles] utility. The kernel must be at least v5.5 or alternatively have [this patch](https://lore.kernel.org/selinux/20190912133007.27545-1-jlebon@redhat.com/T/#u) backported.

[selinux]: https://selinuxproject.org/page/Main_Page
[setfiles]: https://linux.die.net/man/8/setfiles

## Partition Reuse Semantics

The `wipePartitionEntry` and `shouldExist` flags control what Ignition will do when it encounters an existing partition. `wipePartitionEntry` specifies whether Ignition is permitted to delete partition entries in the partition table.  `shouldExist` specifies whether a partition with that number should exist or not (it is invalid to specify a partition should not exist and specify its attributes, such as `size` or `label`).

The following table shows the possible combinations of whether or not a partition with the specified number is present, `shouldExist`, and `wipePartitionEntry`, and the action Ignition will take:

| Partition present | shouldExist | wipePartitionEntry | Action Ignition takes
| ----------------- | ----------- | ------------------ | ---------------------
| false             | false       | false              | Do nothing
| false             | false       | true               | Do nothing
| false             | true        | false              | Create specified partition
| false             | true        | true               | Create specified partition
| true              | false       | false              | Fail
| true              | false       | true               | Delete existing partition
| true              | true        | false              | Verify existing partition matches the specified one, otherwise resize it if `resize` field is true and partition matches in all respects except size, otherwise fail
| true              | true        | true               | Check if existing partition matches the specified one, delete existing partition and create specified partition if it does not match

### Partition Matching
A partition matches if all of the specified attributes (`label`, `start`, `size`, `uuid`, and `typeGuid`) are the same. Specifying `uuid` or `typeGuid` as an empty string is the same as not specifying them. When 0 is specified for start or size, Ignition checks if the existing partition's start / size match what they would be if all of the partitions specified were to be deleted (if allowed by wipePartitionEntry), then recreated if `shouldExist` is true.

### Partition number 0
Specifying `number` as 0 will use the next available partition number. Partition number 0 is disallowed on disks with partitions that specify `shouldExist` as false. If `number` is not specified it will be treated as 0.

### Partition start 0
Specifying `start` as 0 will use the starting sector of the largest available block. This is not necessarily the first available block large enough.

### Unspecified partition start
If `start` is not specified and a partition with the same number exists, Ignition will use the start of the existing partition, unless wipePartitionEntry is set.
If `start` is not specified and there is no existing partition, or wipePartitionEntry is set, Ignition will use the starting sector of the largest block, as if `start` were set to 0.

### Partition size 0
Specifying `size` as 0 means the partition should span to the end of the largest available block. If the starting sector is not within the largest available block, Ignition will fail.

### Unspecified partition size
If `size` is not specified and a partition with the same number exists, it will use the value of the existing partition, unless wipePartitionEntry is set.
If `size` is not specified and there is no existing partition, or wipePartitionEntry is set, `size` act as if it were set to 0 and use the size of the largest block.

## Config Merging

Ignition supports fetching and merging multiple configs. This replaces the `append` functionality of the Ignition 2.x.0 specification. There are several rules that determine how configs get merged. When a child config is merged with a parent, generally the child config's values override the parent config's values.

### Child configs take precedence when specified

If a parent and child object are being merged, the fields in the child object take precedence over the fields in the parent config. If a field in the child object is not specified, the field from the parent is used instead.

### Most lists are deduplicated

All lists of objects have a field that uniquely identifies that object. If a child config contains an entry that matches an entry already specified in the parent config, those entries are merged. A few sections of the config are exempt from this behavior. See the [configuration specification][config-spec] for a complete listing. Generally the only lists that are simply appended are those that specify arguments to commands like `mkfs` or `mdadm`.

### Files, Directories, and Links are deduplicated across each other

Since files, directories, and links all describe filesystem entries can conflict, these lists are deduplicated across each other. This means a file in a child config can replace a link in the parent, or a directory in a child config can replace a file in the parent.

### Configs are merged in a depth first traversal

A child config can specify children of its own. Those children are merged into their parent config before that config is merged into its own parent. If a config specifies multiple children, those children are merged in the order they appear.

[config-spec]: configuration-v3_0.md

### HTTP headers merging

If names of the parent and child headers match, the result will be to replace the value of the parent header with that of the child.

If a child header has no value, the parent header with the same name will be removed.

## LUKS

Ignition has support for creating both purely key-file based LUKS2 devices as well as Tang/TPM2 backed (via clevis) devices.

If a key-file is not specified one will be generated for the device. Key-files will be stored at `/etc/luks/<deviceName>` (this path can be overridden via build flags).

Ignition generates entries in `/etc/crypttab` for each device and expects that the operating system has hooks to be able to unlock the device (e.x.: `systemd-cryptsetup-generator`).

### Clevis Based Devices

When creating clevis based devices to utilize Tang or TPM2 Ignition will use an [SSS Pin](https://github.com/latchset/clevis#pin-shamir-secret-sharing) and will create the relevant configuration JSON from the provided attributes.

## Secrets

We do not recommend storing secrets in Ignition configs. Many platforms allow unprivileged software in a VM (including software running in a container) to retrieve the Ignition config from a networked metadata service or local API. To avoid any possibility of leaking sensitive information, it's best to store secrets in a dedicated service such as [Hashicorp Vault](https://www.vaultproject.io/).

If you must store secrets in an Ignition config, strongly consider applying mitigations.  For example:

- Put secrets in a child Ignition config stored in a location under your control. Configure firewall rules to prevent unprivileged software from accessing this location. Merge the child config into your root config via an `ignition.config.merge` directive.
- On platforms with a networked instance metadata service (IMDS), configure firewall rules to prevent unprivileged software from contacting the metadata service.  Some software uses the instance metadata service for other purposes (such as determining network addresses), so this may not be practical without a transparent proxy.

### Automatic config deletion

On some platforms, Ignition 2.14.0 and later automatically deletes the Ignition config from VM metadata after provisioning succeeds.  This helps limit access by unprivileged software to sensitive information in the Ignition config.  This functionality is currently supported in VirtualBox and VMware VMs, and other platforms may be added in the future.

If you have external tools that require the Ignition config to remain available in VM metadata after provisioning, you can prevent automatic deletion by masking `ignition-delete-config.service`.  For example:

```json
{
  "ignition": {
    "version": "3.0.0"
  },
  "systemd": {
    "units": [
      {
        "name": "ignition-delete-config.service",
        "mask": true
      }
    ]
  }
}
```
