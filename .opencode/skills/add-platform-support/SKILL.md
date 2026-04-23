---
name: add-platform-support
description: Automate adding new cloud provider/platform support to Ignition
---

# Add Platform Support

This skill adds a new cloud provider/platform to Ignition by running the codified script.

## Usage

When invoked, gather the following from the user if not provided as arguments:

- **Provider ID** (`--id`): lowercase alphanumeric, e.g. `hetzner`
- **Provider Name** (`--name`): display name, e.g. `Hetzner Cloud`
- **Metadata URL** (`--url`): URL to fetch config from, e.g. `http://169.254.169.254/hetzner/v1/userdata`
- **Documentation URL** (`--docs`): platform docs URL, e.g. `https://www.hetzner.com/cloud`
- **Description** (`--description`, optional): custom description for supported-platforms.md

Then execute:

```bash
.opencode/skills/add-platform-support/run.sh \
  --id <provider_id> \
  --name "<provider_name>" \
  --url "<metadata_url>" \
  --docs "<docs_url>"
```

The script handles everything: generating the provider Go file, updating the registry, docs, release notes, running validation, and creating the commit.

See `README.md` in this directory for detailed documentation.
