# Config Management

**Domain**: fab-workflow

## Overview

`fab/config.yaml` is the factual project context file loaded by nearly every Fab skill. This doc covers how to maintain it over the project lifecycle using `/fab-init config`.

## Updating Config

### Interactive Menu

Run `/fab-init config` to see all editable sections:

1. `project` — name and description
2. `context` — tech stack and conventions
3. `source_paths` — implementation code directories
4. `stages` — pipeline stage definitions
5. `rules` — per-stage generation rules
6. `checklist` — extra quality categories
7. `git` — branch integration settings
8. `naming` — change folder naming format

### Direct Section Access

Skip the menu with `/fab-init config <section>` (e.g., `/fab-init config context`).

### Comment Preservation

Updates use targeted string replacement on the specific section being edited. Comments and formatting in other sections are preserved. This is important because `config.yaml` relies on inline comments for self-documentation.

## Validation

After manual edits or updates, run `/fab-init validate` to check structural correctness:

- Required fields present (`project.name`, `project.description`, `stages`, etc.)
- Stage `requires` references point to valid stage IDs
- No circular dependencies in the stage graph
- Valid YAML syntax

`/fab-init config` also validates automatically after each edit, offering to revert invalid changes.

## Common Workflows

| Scenario | Command |
|----------|---------|
| Add a new tech to the stack | `/fab-init config context` |
| Add a new source directory | `/fab-init config source_paths` |
| Add a custom checklist category | `/fab-init config checklist` |
| Change branch naming | `/fab-init config git` |
| Verify config after manual edit | `/fab-init validate` |

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260213-3tyk-merge-fab-init-subcommands | 2026-02-13 | Updated command references — `/fab-init config` is now a subcommand of `/fab-init` |
| 260212-h9k3-fab-init-family | 2026-02-12 | Initial creation — config lifecycle management with `/fab-init config` |
