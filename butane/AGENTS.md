# Butane Repository

**Butane** translates human-readable Butane Configs into machine-readable [Ignition](https://github.com/coreos/ignition) Configs for CoreOS-based operating systems.

**Variants**: `fcos` (Fedora CoreOS), `flatcar`, `openshift`, `r4e` (RHEL for Edge), `fiot` (Fedora IoT)

## Architecture

```
base/vX_Y/           # Distro-agnostic, targets Ignition versions (v0_7, v0_8_exp)
config/*/vX_Y/       # Distro-specific (fcos/v1_7, fcos/v1_8_exp, openshift/v4_21)
  ├── schema.go      # Butane config structs
  ├── translate.go   # Butane → Ignition translation
  └── validate.go    # Validation logic
internal/doc/        # Documentation generation (butane.yaml)
.opencode/           # Automation scripts and skills
```

**Rules**:
- Never import from `base/` packages directly
- Only experimental specs (`*_exp`) receive new features
- Stable specs are frozen (bug fixes only)

**Version Lifecycle**: Experimental → Stabilization → Stable → New Experimental

## Core Patterns

### Schema-Translate-Validate

Each spec implements three files:

**schema.go**: Go structs with YAML tags, optional fields use pointers (`*string`, `*bool`), auto-filtered fields use `` `butane:"auto_skip"` ``

**translate.go**: Main function `ToIgnX_YUnvalidated()` returns (config, translation set, report)

**validate.go**: Returns validation reports (warnings/errors), uses `config/common/errors.go` for error definitions

### Sugar Implementation

**Preferred**: Config merging via `baseutil.MergeTranslatedConfigs()` - desugared struct is merge parent, user config is child (allows user overrides)

**Alternative**: Direct struct modification (only if merging isn't expressive enough)

### Error Handling

Define in `config/common/errors.go`:
```go
var ErrExample = errors.New("message")
```

Use: `r.AddOnError(path.New("json", "field"), common.ErrExample)`

### Documentation

After spec changes:
1. Update `internal/doc/butane.yaml`
2. Run `./generate`
3. Update `docs/examples.md` and `docs/release-notes.md`

## Commit Conventions

**Format**: `component: description in imperative mood`

**Component patterns** (NO file extensions):
- `fcos translate: add warn on small partition`
- `fcos translate_test: add tests for detection`
- `docs: run generate`
- `*: update experimental specs`
- `base/v0_7_exp: stabilize to v0_7`

**Style**:
- Imperative tense ("add" not "added")
- Lowercase component and description
- Under 72 characters
- NO file extensions (.go, .md)

## Testing

**ALWAYS run before commit**: `./test`

Runs: `gofmt -d`, `go vet -composites=false`, `go test ./... -cover`, doc validation

**Test patterns**: Table-driven tests, positive/negative cases, edge cases, `*_test.go` files alongside code

## OpenCode Skills

Located in `.opencode/skills/`:

- **add-sugar**: `/add-sugar --spec fcos/v1_8_exp` - Scaffolds sugar features (schema, translate, validate, tests, docs)
- **stabilize-spec**: `/stabilize-spec --spec fcos/v1_8_exp` - Stabilizes experimental specs, creates new experimental
- **remove-feature**: `/remove-feature --spec fcos/v1_8_exp --field old_sugar` - Safely removes features

**Global skills**: `add-skill`, `commit-message`, `review-pr`

## Automation Scripts

`.opencode/scripts/`:

- **version-info.sh**: `eval "$(.opencode/scripts/version-info.sh config/fcos/v1_8_exp)"` - Sets PACKAGE, IS_EXP, DOTTED, SPEC_TYPE vars
- **preflight.sh**: `--exists`, `--experimental`, `--git-clean`, `--all-experimental` - Pre-flight checks

## Development Workflow

**Adding Features**:
1. Target experimental spec (`base/v0_8_exp` for distro-independent, `config/fcos/v1_8_exp` for distro-specific)
2. Use `/add-sugar --spec {spec}` or manual: update schema.go → translate.go → validate.go → tests → errors.go → doc/butane.yaml
3. Run `./generate`
4. Run `./test`
5. Commit: `git commit -m "component: description"`

**Stabilizing**: Use `/stabilize-spec` or follow [stabilization checklist](https://github.com/coreos/butane/issues/new?template=stabilize-checklist.md)

**Releases**: Follow [release checklist](https://github.com/coreos/butane/issues/new?template=release-checklist.md)

## Important Rules

**Translation**: Prefer config merging; desugared = parent, user = child

**Validation**: All errors in `config/common/errors.go`; use path tracking; distinguish errors (fatal) vs warnings

**Testing**: Run `./test` before every commit (non-negotiable)

**Documentation**: Run `./generate` after changes; keep examples and release notes updated

**Code Quality**: `gofmt` formatting, `golangci-lint` in CI, `go vet -composites=false`

## Resources

- Docs: `docs/getting-started.md`, `docs/specs.md`, `docs/development.md`
- Dependencies: Ignition (github.com/coreos/ignition), vcontext (github.com/coreos/vcontext)
- Main branch: `main`, CI: format/vet/tests/linting

## Quick Reference

- Read existing code before changes
- Use experimental specs (`*_exp`) for new features
- Use config merging for sugar unless impossible
- Check preflight: `.opencode/scripts/preflight.sh --all-experimental {spec}`
- Butane configs should be intuitive and human-friendly
