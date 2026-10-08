# Butane — Threat Model

Butane is a local tool (CLI and library) that turns human-readable Butane YAML into Ignition JSON (or OpenShift MachineConfig). It is written in Go and does not use network: remote `source` URLs in a config are copied into the output and fetched later by Ignition at boot.

Use this document when triaging CVEs and security reports. Check whether vulnerable code is reachable in the butane binary first; many issues in shared dependencies are Won't Fix for Butane if they only affect paths Butane never runs.

## In scope

Security issues we treat as Butane bugs:

- **Wrong or unsafe output** — transpilation or desugaring that produces incorrect, incomplete, or unexpectedly privileged Ignition or MachineConfig.
- **Validation gaps** — configs that should be rejected but pass, including inline/local merge handling and variant-specific spec filters.
- **`--files-dir` safety** — reading or embedding local files outside the intended directory or with unsafe path handling.
- **Transpile-time parsing** — bugs on the path from input to output, including YAML/JSON handling, `data:` URL compression, and shared validation libraries when Butane actually invokes them.

## Out of scope (usually Won't Fix)

Issues we typically do not fix in Butane alone:

- **Denial of service** — crashes or extreme resource use from malicious or oversized input.
- **Remote fetch at boot** — HTTP/TLS or fetch-related CVEs that only run in Ignition when it downloads `source` URLs, unless the same code path is reachable in the butane binary during transpile.
- **Pass-through remote config** — problems that only manifest when Ignition merges or fetches remote `ignition.config.merge` `source` URLs (triage those with Ignition).
