# Intake: Delegate fab-switch Name Resolution to Shell

**Change**: 260216-jmy4-DEV-1044-switch-shell-name-resolution
**Created**: 2026-02-16
**Status**: Draft

## Origin

> User observed that `/fab-switch 260216-ymvx-DEV-1043-envrc-line-sync` (full folder name) repeatedly returned "No active changes found" despite the folder existing in `fab/changes/`. The shorter partial name `/fab-switch 260216-ymvx-DEV-1043` succeeded. Root cause analysis revealed that the Haiku model executing the `/fab-switch` skill is doing string matching in-context rather than delegating to the existing `resolve-change.sh` shell script, and it fails on long hyphenated slugs.

## Why

1. **Problem**: The `/fab-switch` skill instructs the LLM (Haiku tier) to scan `fab/changes/` folder names and perform case-insensitive substring matching itself — in its reasoning. Small models are unreliable at exact string comparison for long hyphenated slugs (e.g., 40+ character folder names), causing false negatives where valid exact matches are not recognized.

2. **Consequence**: Users cannot reliably switch to changes by full folder name — a core workflow operation. Tab-completion and copy-paste of full names fail silently. The skill reports "no active changes" when the change clearly exists, forcing users to guess shorter partial names.

3. **Approach**: A shell function `resolve_change()` already exists at `fab/.kit/scripts/lib/resolve-change.sh` that handles exact + substring matching correctly using bash string operations. The batch script (`batch-fab-switch-change.sh`) already uses it. The fix is to have the `/fab-switch` skill call this shell function for name resolution instead of reimplementing it in-prompt, making resolution deterministic regardless of model tier.

## What Changes

### Replace in-prompt matching with shell-based resolution

Both the skill file (`fab/.kit/skills/fab-switch.md`) and the agent definition (`.claude/agents/fab-switch.md`) currently instruct the LLM to:

1. Scan `fab/changes/` (exclude `archive/`)
2. Match `<change-name>` against folder names (case-insensitive substring) in its reasoning

This will be replaced with instructions to call `resolve-change.sh` via Bash:

```bash
source fab/.kit/scripts/lib/resolve-change.sh
resolve_change "fab" "<change-name>"
echo "$RESOLVED_CHANGE_NAME"
```

The skill prompt will be updated to:
- **Argument Flow**: Call `resolve-change.sh` with the user-provided argument. On success (exit 0), use `$RESOLVED_CHANGE_NAME`. On failure (exit 1), read stderr for the diagnostic message and act accordingly (no match → list all, multiple matches → present options from stderr).
- **No Argument Flow**: Still handled by the skill — list all folders (this is a presentation concern, not a resolution one). The skill can `ls fab/changes/` and filter out `archive/`.

### Files affected

1. **`fab/.kit/skills/fab-switch.md`** — Update "Argument Flow" section to delegate to `resolve-change.sh` instead of describing in-prompt matching logic. Update "No Argument Flow" to keep the listing behavior. Update "Context Loading" to note shell script dependency.

2. **`.claude/agents/fab-switch.md`** — Mirror the same changes (this file is a copy of the skill with agent-specific frontmatter).

### What stays the same

- `resolve-change.sh` itself — no modifications needed, it already handles all cases correctly
- The switch flow (writing `fab/current`, branch integration, output format)
- The `--blank`, `--branch`, `--no-branch-change` flags
- Error handling table (conditions and actions remain the same, just the resolution mechanism changes)

## Affected Memory

- `fab-workflow/preflight`: (modify) Document that `/fab-switch` now uses `resolve-change.sh` for name resolution (previously it was only used by `preflight.sh` and `batch-fab-switch-change.sh`)

## Impact

- **`fab/.kit/skills/fab-switch.md`** — Argument Flow section rewritten
- **`.claude/agents/fab-switch.md`** — Argument Flow section rewritten (mirror of skill)
- **`fab/.kit/scripts/lib/resolve-change.sh`** — No changes (already correct)
- **`fab/.kit/scripts/lib/preflight.sh`** — No changes
- **Downstream skills** — No impact; `/fab-switch` output contract unchanged

## Open Questions

(none — root cause is clear and the fix path is straightforward)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use existing `resolve-change.sh` without modification | Shell script already handles exact + substring matching correctly; battle-tested by `preflight.sh` and `batch-fab-switch-change.sh` | S:95 R:90 A:95 D:95 |
| 2 | Certain | Keep No Argument Flow as LLM-driven listing | Listing all changes with stages for user selection is a presentation concern, not a resolution one; the shell script doesn't cover this case | S:90 R:95 A:90 D:90 |
| 3 | Certain | Mirror changes in both skill and agent files | Both files contain the same Argument Flow logic and must stay in sync | S:95 R:95 A:95 D:95 |
| 4 | Confident | Multiple-match handling via stderr parsing | `resolve-change.sh` prints comma-separated match list to stderr on multiple matches; skill can parse and present options from this. Alternative: call a separate listing script, but stderr output is already structured enough | S:80 R:85 A:75 D:70 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
