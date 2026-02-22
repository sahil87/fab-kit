# Intake: git-pr Shipped Sentinel and Status Commit

**Change**: 260222-trdc-git-pr-shipped-sentinel-and-status-commit
**Created**: 2026-02-22
**Status**: Draft

## Origin

> Conversational discussion about race conditions in the pipeline orchestrator's ship-completion detection. The current flow has a TOCTOU problem: `git-pr` writes the PR URL to `.status.yaml` (which `run.sh` polls via `stageman is-shipped`), but that `.status.yaml` update is never committed or pushed — so the PR doesn't contain its own shipped URL, and the branch is dirty when the next dependent change branches from it.

## Why

1. **Dirty branch state**: When `run.sh` detects `shipped` in `.status.yaml` and marks the change `done`, the next dependent change branches from the parent's tip. But the parent branch hasn't committed the `.status.yaml` update with the PR URL — so either the uncommitted change leaks into the child worktree or is lost entirely.

2. **Race condition**: `run.sh` polls `stageman is-shipped` (checks `.status.yaml` for a non-empty `shipped` array). The signal appears the moment `git-pr` writes to `.status.yaml`, but the subsequent commit+push hasn't happened yet. If `run.sh` immediately dispatches the next change, it branches from a stale tip.

3. **Incomplete PR**: The PR itself doesn't contain the `.status.yaml` entry recording its own URL. While self-referential, this makes the change folder's committed state incomplete — the shipped metadata is only present locally.

## What Changes

### 1. git-pr: Second commit+push after recording shipped URL

After Step 4 (Record Shipped) writes the PR URL to `.status.yaml` via `stageman ship`, `git-pr` performs an additional commit+push cycle:

- `git add fab/changes/{name}/.status.yaml`
- `git commit -m "Record shipped URL in .status.yaml"`
- `git push`

This ensures the branch tip includes the shipped metadata and the PR diff contains the full final state.

### 2. git-pr: Write `.shipped` sentinel file as final action

After all git operations (including the second commit+push) are complete, `git-pr` writes a sentinel file:

```
fab/changes/{name}/.shipped
```

Contents: the PR URL (same value written to `.status.yaml`). Written atomically:
```bash
echo "$PR_URL" > "$change_dir/.shipped.tmp"
mv "$change_dir/.shipped.tmp" "$change_dir/.shipped"
```

This file is gitignored and never committed. Its sole purpose is providing a race-free filesystem signal that all git operations for this change are fully complete.

### 3. .gitignore: Add `.shipped` pattern

Add to `.gitignore`:
```
fab/changes/**/.shipped
fab/changes/**/.shipped.tmp
```

### 4. run.sh: Poll for `.shipped` sentinel instead of `stageman is-shipped`

In `poll_change()`'s `shipping` state (line ~407), replace the `stageman is-shipped` check with a file-existence check:

```bash
# Current:
if bash "$STAGEMAN" is-shipped "$status_file" 2>/dev/null; then

# New:
local shipped_sentinel="$wt_path/fab/changes/$resolved_id/.shipped"
if [[ -f "$shipped_sentinel" ]]; then
```

This polls for the sentinel file, which only exists after all commits and pushes are done.

### 5. git-pr skill update

Update `fab/.kit/skills/git-pr.md` to add:
- **Step 4b**: After Step 4 (Record Shipped), commit and push the `.status.yaml` update
- **Step 4c**: Write the `.shipped` sentinel file as the very last action

The sentinel write is unconditional — happens in both orchestrated and manual flows. If nobody polls for it, the file sits harmlessly (gitignored, cleaned up with worktree deletion).

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document git-pr's new post-ship commit+push and sentinel write
- `fab-workflow/pipeline-orchestrator`: (modify) Document sentinel-based ship detection replacing `stageman is-shipped` polling

## Impact

- `fab/.kit/skills/git-pr.md` — add Steps 4b and 4c
- `fab/.kit/scripts/pipeline/run.sh` — change `poll_change()` shipping detection
- `.gitignore` — add sentinel patterns
- `stageman.sh` — `is-shipped` command remains (backward compat) but is no longer used by the orchestrator

## Open Questions

- None — design was fully discussed and agreed before intake.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Sentinel file location is `fab/changes/{name}/.shipped` | Discussed — user explicitly chose change folder over `fab/pipeline/` after considering both | S:95 R:90 A:95 D:95 |
| 2 | Certain | Sentinel is gitignored, never committed | Discussed — agreed it's an orchestration breadcrumb, not a change artifact | S:95 R:95 A:95 D:95 |
| 3 | Certain | git-pr writes sentinel unconditionally (no orchestration-mode flag) | Discussed — cheap operation, decouples git-pr from orchestration awareness | S:90 R:95 A:90 D:95 |
| 4 | Certain | Second commit+push happens after writing shipped URL to .status.yaml | Discussed — makes PR contain the shipped metadata and branch clean | S:90 R:85 A:90 D:90 |
| 5 | Confident | Sentinel content is the PR URL (not just a touch) | Atomic mv pattern discussed; URL content provides debugging value | S:75 R:90 A:80 D:85 |
| 6 | Confident | `stageman is-shipped` remains but is unused by orchestrator | Backward compat — other consumers may use it; removing is separate cleanup | S:70 R:90 A:85 D:80 |
| 7 | Confident | Commit message for the status update is simple/fixed (e.g., "Record shipped URL") | No discussion but low-stakes, easily changed | S:60 R:95 A:85 D:85 |
| 8 | Confident | run.sh polls sentinel by file existence, not content | Discussed — existence is the signal, content is bonus | S:80 R:90 A:85 D:80 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
