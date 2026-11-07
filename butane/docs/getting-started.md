---
nav_order: 2
---

# Getting started

Butane (formerly the Fedora CoreOS Config Transpiler) is Ignition's human-readable configuration format. Butane configs are YAML files that describe how a machine should be provisioned: users, filesystems, networking, systemd units, and more. Ignition 2.27.0 and later reads Butane configs directly and transpiles them at boot; for older Ignition versions, the `butane` CLI converts a Butane config into an Ignition JSON config that can be given to the machine.

Butane configs are YAML files conforming to Butane's schema. For more information on the schema, take a look at the [configuration specifications][spec].

With Ignition 2.27.0 and later, producing a separate Ignition JSON file is optional for configs that do not reference local files. The `butane` CLI is still useful for validating configs before provisioning and for `--files-dir`, which embeds local files into the generated Ignition config.

### Getting the Butane CLI

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

Download the latest version of `butane` and the detached signature from the [releases page](https://github.com/coreos/ignition/releases). Verify it with gpg:

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

### Processing files with an external tool

Butane can send selected input and local files through an external program before it translates them. This is useful for tools such as [gomplate](https://gomplate.ca/) that render templates. Install the processor separately and make its executable available in the same environment as Butane.

Both `--file-processor` and `--file-processor-allowed-ext` are required when using this feature, and both values must be non-empty. The processor is run directly, without a shell or extra command-line arguments. Butane writes the selected file to the processor's standard input and uses its standard output as the file contents. The processor inherits Butane's working directory, environment, and standard error. Each file is processed in a separate invocation. Generate values that must be shared across files beforehand and supply them through the environment or shared processor configuration.

For example, to render the root Butane config and local `.tmpl` files with gomplate:

```sh
butane --files-dir . \
  --file-processor gomplate \
  --file-processor-allowed-ext bu,tmpl \
  config.bu > config.ign
```

The allowed values are comma-separated filename suffixes. Matching is case-sensitive, and a leading dot is optional; multi-part suffixes such as `tar.gz` are supported. The special value `*` matches all named files. The special value `stdin` enables processing the root config when it is read from standard input. The `*` value alone does not enable stdin processing; use `--file-processor-allowed-ext='*,stdin'` to select both. For example:

```sh
cat config.bu | butane --files-dir . \
  --file-processor gomplate \
  --file-processor-allowed-ext bu,stdin > config.ign
```

With stdin input, Butane passes the complete input document through the processor before parsing it. Named root input files are processed when their suffix matches. This includes inline text in the root config, since the complete document is processed as a whole. Selected local files are processed when Butane reads them, including local file contents, SSH keys, systemd units, drop-ins, Quadlet files, and regular file payloads found under tree sources. Files whose suffix does not match are read unchanged. The existing `--files-dir` rules for local resources remain in effect.

Butane does not split processor settings into an executable and arguments. If a processor needs fixed flags or a config path, create a small executable wrapper and name that wrapper with `--file-processor`, for example:

```sh
#!/bin/sh
exec gomplate --config /etc/my-gomplate.yaml
```

The processor must produce valid Butane YAML when invoked for a root `.bu` file. A processor failure stops translation and is reported as an error; its standard error is passed through. Processing applies when Butane reads files on the host. It is not enabled automatically when Ignition reads a Butane config natively at boot.

[spec]: specs.md
[ignition]: https://coreos.github.io/ignition/
[supported-platforms]: https://coreos.github.io/ignition/supported-platforms/
[examples]: examples.md
