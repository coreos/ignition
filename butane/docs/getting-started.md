---
nav_order: 2
---

# Getting started

Butane (formerly the Fedora CoreOS Config Transpiler) is a tool that consumes a Butane Config and produces an Ignition Config, which is a JSON document that can be given to a Fedora CoreOS machine when it first boots. Using this config, a machine can be told to create users, create filesystems, set up the network, install systemd units, and more.

Butane configs are YAML files conforming to Butane's schema. For more information on the schema, take a look at the [configuration specifications][spec].

### Getting Butane

`butane` can be run from a container image with `podman` or `docker`, installed from Fedora package repositories or downloaded as a standalone binary.

Using the official container images is the recommended option.

#### Container image

This example uses `podman`, but `docker` can also be used.

```bash
# Pull the container image release
podman pull quay.io/coreos/butane:release

# Run Butane using standard input and standard output
podman run --interactive --rm quay.io/coreos/butane:release \
       --pretty --strict < your_config.bu > transpiled_config.ign

# Run Butane using a file as input and standard output
podman run --interactive --rm --security-opt label=disable \
       --volume ${PWD}:/pwd --workdir /pwd quay.io/coreos/butane:release \
       --pretty --strict your_config.bu > transpiled_config.ign
```

You may also add the following alias in your shell configuration:

```
alias butane='podman run --rm --interactive         \
              --security-opt label=disable          \
              --volume "${PWD}":/pwd --workdir /pwd \
              quay.io/coreos/butane:release'
```

Alternatively you may also create a wrapper script at `~/.local/bin/butane`:

```bash
#!/bin/sh
exec podman run --rm --interactive         \
     --security-opt label=disable          \
     --volume "${PWD}":/pwd --workdir /pwd \
     quay.io/coreos/butane:release         \
     "${@}"
```

Make sure that `~/.local/bin` is in your `$PATH`, or choose another path like `/usr/local/bin`.

#### Distribution packages

`butane` is available from the Fedora package repositories:

```
$ sudo dnf install -y butane
```

#### Standalone binary

Download the latest version of `butane` and the detached signature from the [releases page](https://github.com/coreos/butane/releases). Verify it with gpg:

```
gpg --verify <detached sig> <butane binary>
```
You may need to download the [Fedora signing keys](https://fedoraproject.org/fedora.gpg) and import them with `gpg --import <key>` if you have not already done so.

New releases of `butane` are backwards compatible with old releases, and with the Fedora CoreOS Config Transpiler, unless otherwise noted.

### Writing and using Butane configs

As a simple example, let's use `butane` to set the authorized ssh key for the `core` user on a Fedora CoreOS machine.

<!-- butane-config -->
```yaml
variant: fcos
version: 1.1.0
passwd:
  users:
    - name: core
      ssh_authorized_keys:
        - ssh-rsa AAAAB3NzaC1yc...
```

In this above file, you'll want to set the `ssh-rsa AAAAB3NzaC1yc...` line to be your ssh public key (which is probably the contents of `~/.ssh/id_rsa.pub`, if you're on Linux).

If we take this file and give it to `butane`:

```
$ ./bin/amd64/butane example.bu

{"ignition":{"config":{"replace":{"source":null,"verification":{}}},"security":{"tls":{}},"timeouts":{},"version":"3.0.0"},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa ssh-rsa AAAAB3NzaC1yc..."]}]},"storage":{},"systemd":{}}
```

We can see that it produces a JSON file. This file isn't intended to be human-friendly, and will definitely be a pain to read/edit (especially if you have multi-line things like systemd units). Luckily, you shouldn't have to care about this file! Just provide it to a booting Fedora CoreOS machine and [Ignition][ignition], the utility inside of Fedora CoreOS that receives this file, will know what to do with it.

The method by which this file is provided to a Fedora CoreOS machine depends on the environment in which the machine is running. For instructions on a given provider, head over to the [list of supported platforms for Ignition][supported-platforms].

To see some examples for what else Butane can do, head over to the [examples][examples].

[spec]: specs.md
[ignition]: https://coreos.github.io/ignition/
[supported-platforms]: https://coreos.github.io/ignition/supported-platforms/
[examples]: examples.md
