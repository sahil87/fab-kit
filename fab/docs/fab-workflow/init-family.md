# Init Family

**Domain**: fab-workflow

## Overview

The init family is a suite of three commands that manage the lifecycle of Fab's initialization artifacts (`config.yaml` and `constitution.md`). While `/fab-init` handles the one-time structural bootstrap, the family commands provide ongoing management: creation, updates, and validation.

| Command | Purpose |
|---------|---------|
| `/fab-init constitution` | Create or amend `constitution.md` with semantic versioning |
| `/fab-init config` | Create or update `config.yaml` interactively, preserving comments |
| `/fab-init validate` | Validate structural correctness of both files |

## Architecture

### Delegation from `/fab-init`

`/fab-init` delegates artifact creation to the family commands:

- Step 1a: If `config.yaml` doesn't exist → invokes `/fab-init config` in create mode
- Step 1b: If `constitution.md` doesn't exist → invokes `/fab-init constitution` in create mode

This ensures each command is the single source of truth for its artifact's generation logic. `/fab-init` retains ownership of structural orchestration (directories, symlinks, `.gitignore`).

### Independence

Each family command operates independently — they can be invoked directly without going through `/fab-init`. This supports two workflows:

1. **Initial setup**: `/fab-init` orchestrates everything (delegates to family commands internally)
2. **Ongoing management**: User invokes family commands directly as project evolves

## Key Design Decisions

### Consolidated Skill with Subcommands
All three commands are subcommands within a single `fab-init.md` skill file. Each subcommand has its own behavior section, sharing the same `model_tier` and frontmatter.

### Config Updates Use String Replacement
`/fab-init config` uses targeted string replacement rather than full YAML parse-and-rewrite. This preserves the heavily-commented `config.yaml` format at the cost of slightly less structural safety.

### Validate Is Read-Only
`/fab-init validate` only checks and reports — it never modifies files. Fix suggestions are provided but the user applies them (directly or via the other family commands).

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260213-3tyk-merge-fab-init-subcommands | 2026-02-13 | Consolidated from separate commands into subcommands of `/fab-init` — single skill file, updated architecture description |
| 260212-h9k3-fab-init-family | 2026-02-12 | Initial creation — three family commands (constitution, config, validate) with delegation from `/fab-init` |
