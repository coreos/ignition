---
name: stabilize-spec
description: Automate the Ignition config spec stabilization process
---

# Stabilize Spec

This skill performs a complete config spec stabilization by running the codified script.

## Prerequisites

Before running, ensure `schematyper` is installed:

```bash
cd /tmp
git clone https://github.com/idubinskiy/schematyper.git
cd schematyper
go mod init github.com/idubinskiy/schematyper
echo 'replace gopkg.in/alecthomas/kingpin.v2 => github.com/alecthomas/kingpin/v2 v2.4.0' >> go.mod
go mod tidy
go install .
```

Also install build dependencies (Fedora/RHEL): `sudo dnf install -y libblkid-devel`

## Usage

When invoked, gather the following from the user if not provided:

- **Minor version** (required): the minor version to stabilize, e.g. `7` for `v3_7_experimental -> v3_7`

Then execute:

```bash
.opencode/skills/stabilize-spec/run.sh <minor_version>
```

The script creates 8 atomic commits: renaming experimental to stable, creating the next experimental version, updating all imports across the codebase, updating tests, regenerating docs/schemas, and running build/test validation.

See `README.md` in this directory for detailed documentation.
