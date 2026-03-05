# Intake: Add changeman.sh list Subcommand

**Change**: 260226-w4fw-add-changeman-list-subcommand
**Created**: 2026-02-26
**Status**: Draft

## Origin

> Add a `list` subcommand to changeman.sh that enumerates active changes with their stage and state, excluding archive. Output one structured line per change (name:stage:state). Useful for /fab-switch no-argument flow and /fab-status.

One-shot — clear requirements from conversation. The need was identified during a `/fab-switch` invocation where the agent incorrectly tried `changeman.sh list` (which doesn't exist), then had to fall back to manual `ls` + `.status.yaml` parsing. The user agreed that `list` should be a changeman subcommand.

## Why

1. **No structured enumeration of active changes**: `/fab-switch` (no-argument flow) and `/fab-status` both need to list changes with their stages, but `changeman.sh` has no `list` subcommand. Skills must manually scan `fab/changes/`, filter out `archive/`, and read each `.status.yaml` — duplicating logic that belongs in the change manager.

2. **Agent confusion**: LLMs naturally try `changeman.sh list` because the script owns all other change-level operations (`new`, `rename`, `resolve`, `switch`). Its absence leads to failed tool calls and fallback logic.

## What Changes

### 1. New `list` Subcommand

Add `changeman.sh list` that:

1. Scans `fab/changes/` for directories (excluding `archive/`)
2. For each change directory, reads `.status.yaml` to extract the current stage and state
3. Outputs one line per change in the format: `name:display_stage:display_state`
   - `name` = the folder name (e.g., `260226-tnr8-coverage-scoring-change-types`)
   - `display_stage` = derived from `stageman.sh display-stage` (the "where you are" stage)
   - `display_state` = the state of that stage (`active`, `done`, `pending`)
4. If no changes found, exit 0 with no output (empty result)
5. If a change directory has no `.status.yaml`, output `name:unknown:unknown` with a warning to stderr

Optional `--archive` flag to list archived changes instead (from `fab/changes/archive/`).

### 2. Update Help Text

Add `list` to `changeman.sh --help` output.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) document the new `list` subcommand

## Impact

- **`fab/.kit/scripts/lib/changeman.sh`**: New `list` subcommand and help text
- **`fab/.kit/skills/fab-switch.md`**: Can simplify no-argument flow to use `changeman.sh list` (future — not in this change)

## Open Questions

None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Output format `name:display_stage:display_state` one per line | Discussed — structured output for easy parsing by skills, matches changeman's existing stdout conventions | S:90 R:90 A:90 D:95 |
| 2 | Certain | Exclude archive by default, optional `--archive` flag for archived | Discussed — the primary use case is active changes; archive is a separate concern | S:85 R:90 A:90 D:90 |
| 3 | Certain | Use `stageman.sh display-stage` for stage derivation | Follows existing pattern — display-stage already encapsulates the "where you are" logic | S:85 R:85 A:95 D:90 |
| 4 | Confident | Empty output (exit 0) when no changes, not an error message | Convention — consistent with how other CLI tools handle empty results; callers check for empty stdout | S:75 R:90 A:80 D:80 |
| 5 | Confident | Missing `.status.yaml` outputs `name:unknown:unknown` with stderr warning | Defensive — don't fail the entire list because one change is corrupted | S:70 R:85 A:85 D:75 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
