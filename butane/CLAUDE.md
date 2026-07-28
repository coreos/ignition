@AGENTS.md

## Claude Code Specific Workflows

This file imports the main repository specification from AGENTS.md and adds Claude Code-specific instructions.

### Task Management

When working on multi-step tasks, use `TaskCreate` to break down work and track progress.

### Agent Usage

- Use the **Explore** agent for broad codebase searches when simple Grep/Glob isn't enough
- Use **Plan** mode (if available) for complex changes affecting multiple files
- Delegate independent research to subagents to keep main context clean

### Testing

Always run `./test` before committing - this is non-negotiable and enforced by the repository.
