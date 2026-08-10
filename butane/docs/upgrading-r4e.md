---
title: RHEL for Edge
parent: Upgrading configs
nav_order: 4
---

# Upgrading RHEL for Edge configs

Occasionally, changes are made to RHEL for Edge Butane configs (those that specify `variant: r4e`) that break backward compatibility. While this is not a concern for running machines, since Ignition only runs one time during first boot, it is a concern for those who maintain configuration files. This document serves to detail each of the breaking changes and tries to provide some reasoning for the change. This does not cover all of the changes to the spec - just those that need to be considered when migrating from one version to the next.

{: .no_toc }

1. TOC
{:toc}

## From Version 1.0.0 to Version 1.1.0

There are no breaking changes between versions 1.0.0 and 1.1.0 of the `r4e` configuration specification. Any valid 1.0.0 configuration can be updated to a 1.1.0 configuration by changing the version string in the config.

The following is a list of notable new features.

### AWS S3 access point ARN support

The sections which allow fetching a remote URL now accept AWS S3 access point ARNs (`arn:aws:s3:<region>:<account>:accesspoint/<accesspoint>/object/<path>`) in the `source` field.

<!-- butane-config -->
```yaml
variant: r4e
version: 1.1.0
storage:
  files:
    - path: /etc/example
      mode: 0644
      contents:
        source: arn:aws:s3:us-west-1:123456789012:accesspoint/test/object/some/path
```

### Local SSH key and systemd unit references

SSH keys and systemd units are now embeddable via file references to local files. The specified path is relative to a local _files-dir_, specified with the `-d`/`--files-dir` option to Butane. If no _files-dir_ is specified, this functionality is unavailable.

<!-- butane-config -->
```yaml
variant: r4e
version: 1.1.0
systemd:
  units:
    - name: example.service
      contents_local: example.service
    - name: example-drop-in.service
      dropins:
        - name: example-drop-in.conf
          contents_local: example.conf
passwd:
  users:
    - name: core
      ssh_authorized_keys_local:
        - id_rsa.pub
```
