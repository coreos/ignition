# Butane — gent rules

Local YAML to Ignition JSON transpiler (CLI/library). Written Go; check CVE reachability in the **butane binary** first (usually Won't Fix). No network I/O—remote `source` URLs pass through, Ignition fetches at boot.

## In scope
- transpile/desugar bugs (unsafe/wrong Ignition or MachineConfig); validation gaps (inline/local merge, spec filters)
- `--files-dir` path safety
- reachable bugs on transpile path (YAML/JSON, `data:` compression, shared validation libs)

## Not in scope
- DoS (Denial of Service)
- HTTP/TLS/CVE paths only used by Ignition fetch unless same path in butane binary. No transpile/local impact
