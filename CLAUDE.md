@AGENTS.md

## Claude Code Specific Workflows

### Task Management

When working on multi-step tasks, use TodoWrite to break down work and track progress.

### Agent Usage

- Use the **Explore** agent for broad codebase searches when simple Grep/Glob isn't enough
- Delegate independent research to subagents to keep main context clean

### Pre-Commit Checks

Run `./test` before committing any changes. This validates license headers, gofmt, govet, unit tests, and documentation.

### Skills

This repo has OpenCode skills available:
- `add-platform-support` -- Add new cloud provider support
- `release_note_update` -- Update release notes
- `stabilize-spec` -- Stabilize experimental config spec (8-commit process)
