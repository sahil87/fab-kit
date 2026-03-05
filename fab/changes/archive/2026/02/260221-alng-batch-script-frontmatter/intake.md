# Intake: Batch Script Frontmatter for fab-help Discovery

**Change**: 260221-alng-batch-script-frontmatter
**Created**: 2026-02-21
**Status**: Draft

## Origin

> Add shell-comment frontmatter (# --- delimited, with name and description fields) to the three batch-*.sh scripts so fab-help.sh can discover them. Extend frontmatter.sh to parse shell-comment frontmatter. Add a scan loop in fab-help.sh that globs batch-*.sh, extracts name/description, and renders them under a new "Batch Operations" group (centralized mapping in fab-help.sh, consistent with how skills are grouped today).

Discussion-mode decision: user chose Option 1 (shell-comment frontmatter) over sidecar files, and explicitly chose centralized grouping in `fab-help.sh` over self-declared groups.

## Why

The three `batch-*.sh` scripts (`batch-fab-switch-change`, `batch-fab-archive-change`, `batch-fab-new-backlog`) are useful operational tools but invisible to `/fab-help`. Users have no way to discover them from the help output — they have to know the scripts exist and browse the filesystem. Adding frontmatter and a scan loop in `fab-help.sh` brings them into the same discovery surface as skills, under a dedicated "Batch Operations" group.

## What Changes

### 1. Shell-comment frontmatter in each `batch-*.sh`

Add a `# ---` delimited block near the top of each script (after the shebang and `set -euo pipefail`) containing `name` and `description` fields:

```bash
#!/usr/bin/env bash
# ---
# name: batch-fab-switch-change
# description: "Open tmux tabs in worktrees for one or more changes"
# ---
set -euo pipefail
```

The three scripts and their descriptions:

| Script | name | description |
|--------|------|-------------|
| `batch-fab-switch-change.sh` | `batch-fab-switch-change` | Open tmux tabs in worktrees for one or more changes |
| `batch-fab-archive-change.sh` | `batch-fab-archive-change` | Archive multiple completed changes in one session |
| `batch-fab-new-backlog.sh` | `batch-fab-new-backlog` | Create worktree tabs from backlog items |

### 2. Extend `frontmatter.sh` with a shell-comment parser

Add a new function `shell_frontmatter_field` to `fab/.kit/scripts/lib/frontmatter.sh` that parses `# key: value` lines between `# ---` markers. Same interface as `frontmatter_field` — takes `<file>` and `<field_name>`, returns the unquoted value.

The parser should:
- Match `# ---` as the opening/closing delimiter (not `---`)
- Strip the leading `# ` from each line before extracting the key/value
- Handle quoted and unquoted values the same way `frontmatter_field` does

### 3. Add batch script scan loop to `fab-help.sh`

In `fab-help.sh`:

1. Add `"Batch Operations"` to `group_order` and add the three batch scripts to a `batch_to_group` (or reuse `skill_to_group`) mapping
2. Add a second scan loop after the skills scan that globs `"$kit_dir"/scripts/batch-*.sh`, calls `shell_frontmatter_field` for `name` and `description`, and collects them into the same rendering data structures
3. Render them using the same `format_entry` function, but display the raw script name (no `/` prefix) since they're shell commands, not slash-commands

Display format example:
```
  Batch Operations
    batch-fab-switch-change    Open tmux tabs in worktrees for one or more changes
    batch-fab-archive-change   Archive multiple completed changes in one session
    batch-fab-new-backlog      Create worktree tabs from backlog items
```

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the shell-comment frontmatter convention and batch script discovery mechanism

## Impact

- **`fab/.kit/scripts/lib/frontmatter.sh`** — new function added; existing `frontmatter_field` unchanged
- **`fab/.kit/scripts/fab-help.sh`** — new group, new scan loop, wider alignment column (batch names are longer than skill names)
- **`fab/.kit/scripts/batch-fab-*.sh`** (3 files) — frontmatter block inserted, no behavioral changes
- No impact on skills, templates, or other scripts

## Open Questions

- None — approach was fully discussed and agreed before intake.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `# ---` delimiters for shell-comment frontmatter | Explicitly discussed and chosen as Option 1 | S:95 R:95 A:90 D:95 |
| 2 | Certain | Centralized grouping in fab-help.sh | User explicitly said "Let fab-help own the grouping" | S:95 R:90 A:90 D:95 |
| 3 | Confident | Add `shell_frontmatter_field` as a new function in frontmatter.sh | Same file, parallel function — keeps the parser library cohesive. Could also be a flag on the existing function, but a separate function is cleaner | S:75 R:90 A:85 D:75 |
| 4 | Confident | Display batch scripts without `/` prefix | They're shell commands run directly, not agent slash-commands. Using bare names avoids confusion | S:70 R:95 A:80 D:75 |
| 5 | Certain | Group name "Batch Operations" | Mentioned in discussion, no objection | S:85 R:95 A:85 D:90 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
