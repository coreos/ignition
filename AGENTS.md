# Ignition

First-boot provisioning utility for immutable Linux (Fedora CoreOS, RHEL CoreOS, Flatcar). Partitions disks, formats filesystems, writes files, configures users during initramfs.

## Tech Stack

- **Language**: Go 1.24+ (CGO enabled, requires `libblkid-dev`)
- **Module**: `github.com/coreos/ignition/v2`
- **Dependencies**: Vendored (`vendor/`). After `go.mod` changes run `make vendor`.
- **Testing**: Go `testing` + `testify` assertions
- **Binaries**: `ignition` (provisioning engine), `ignition-validate` (config validator)

## Architecture

```
config/                  # Frontend: stable library API (used by external programs)
  v3_0/ .. v3_6/         # Frozen stable specs - DO NOT MODIFY
  v3_7_experimental/     # Active development spec
    schema/              # JSON schema → run ./generate after changes
    types/               # Go types (schema.go is generated)
    translate/           # Version translation
internal/                # Backend: system configuration engine
  exec/stages/           # Execution stages (fetch-offline, fetch, disks, mount, files, umount)
  providers/             # 30+ cloud/platform providers (aws, azure, gcp, ...)
  resource/              # Resource fetching (HTTP, S3, GCS, TFTP, data URIs)
tests/                   # Blackbox integration tests
  positive/              # Tests that should succeed
  negative/              # Tests that should fail
docs/                    # Documentation (GitHub Pages/Jekyll)
dracut/                  # Dracut module for initramfs integration
```

## Build Commands

- `./build` - Build binaries to `bin/<arch>/`
- `./test` - Run all checks (license, gofmt, govet, unit tests, doc validation)
- `./generate` - Regenerate schema types and docs after JSON schema changes
- `./build_blackbox_tests` - Compile integration tests
- `make vendor` - Vendor dependencies (`go mod vendor && go mod tidy`)
- `make install` - Install binaries, dracut modules, systemd units

## Code Style

- **Formatting**: `gofmt` enforced (CI and `./test`)
- **Linting**: `golangci-lint` v2.11.3 in CI
- **License header**: Required on all `.go` files (Apache 2.0, 13-line header)
- **Imports**: stdlib, blank line, project packages, blank line, external deps
- **Naming**: PascalCase exported, camelCase unexported, snake_case filenames
- **Vendoring**: Changes to `go.mod`/`go.sum`/`vendor/` must be in their own commit

## Testing

- **Unit tests**: `./test` runs all checks including gofmt, govet, license, unit tests
- **Blackbox tests**: `./build_blackbox_tests && sudo sh -c 'PATH=$PWD/bin/amd64:$PATH ./tests.test'`
- **Subset**: `./tests.test -test.run TestIgnitionBlackBox/Preemption.*`
- **Pattern**: Table-driven tests with `struct{ in, out }` slices
- **Registration**: New test modules must be imported in `tests/registry/registry.go`

## Commit Conventions

**Format**: `subsystem: lowercase description`

**Examples**:
- `providers/hetzner: add Hetzner Cloud support`
- `config/v3_6: stabilize spec`
- `internal/exec/stages/files: fix permission handling`
- `docs: update supported platforms`
- `*: refactor provider interface`
- `tests: add filesystem blackbox tests`

**Style**: Imperative mood, lowercase after colon, no trailing period, ~50 char subject.
Body optional; use `Fixes #N` for issue references.

## Important Rules

1. **Frontend is stable API** -- `config/` is used by external programs. API changes require bumping Ignition major version.
2. **New features only in experimental spec** (`v3_7_experimental`). Never modify frozen specs (`v3_0`-`v3_6`).
3. **Declarative config only** -- describe desired state, not actions. No verbs in field names.
4. **Generated files**: `config/v3_*/types/schema.go` -- regenerate with `./generate`, never edit manually.
5. **Platform providers must**: retry config fetch, allow empty config, never guess platform ID.
6. **Network access fields** must be added to the `fetch-offline` needs-net detector.
7. **Execution stages are fixed** (fetch-offline, fetch, disks, mount, files, umount). External projects hardcode this list.
8. **Every platform must be documented** in `docs/supported-platforms.md` (enforced by `./test`).
9. **Schema changes**: After modifying `config/v3_7_experimental/schema/ignition.json`, run `./generate`. CI verifies this.
10. **Security**: Always provide secure defaults. No directives that encourage unsafe behavior.

## Resources

- [Development Guide](docs/development.md)
- [Supported Platforms](docs/supported-platforms.md)
- [Spec Stabilization Checklist](.github/ISSUE_TEMPLATE/stabilize-checklist.md)
- [Release Checklist](.github/ISSUE_TEMPLATE/release-checklist.md)
