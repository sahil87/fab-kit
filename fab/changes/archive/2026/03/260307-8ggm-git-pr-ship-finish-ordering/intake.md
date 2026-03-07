# Intake: Fix git-pr Ship Finish Ordering

**Change**: 260307-8ggm-git-pr-ship-finish-ordering
**Created**: 2026-03-07
**Status**: Draft

## Origin

> User observed that every PR created via `/git-pr` (whether called directly or via `/fab-ff` and `/fab-fff`) leaves `.status.yaml` and `.history.jsonl` with uncommitted changes in the working tree. Screenshot showed the diff: `ship: active -> done`, `review-pr: pending -> active`, new `review-pr` stage_metrics entry, and updated `last_updated` — all uncommitted.

Conversational discussion traced the root cause to step ordering in `git-pr.md`. This is the same class of bug that `260222-trdc-git-pr-shipped-sentinel-and-status-commit` fixed (uncommitted `.status.yaml` after PR URL recording) — but reintroduced when Step 4d was added after the commit boundary.

## Why

1. **Dirty working tree after every PR**: The ship stage's `fab status finish` call mutates `.status.yaml` and `.history.jsonl` after the last `git commit && git push` in Step 4b. These uncommitted files persist in the working tree, polluting `git status` and potentially leaking into subsequent operations (rebases, branch switches, child worktree creation).

2. **Silent data loss risk**: If the next operation involves `git checkout`, `git stash`, or worktree cleanup, these uncommitted status updates can be discarded — losing the `ship: done` / `review-pr: active` transition and its stage_metrics timestamp. Downstream consumers (`/fab-status`, `/fab-switch`, `fab pane-map`) would then see stale state.

3. **Systematic — affects all PR paths**: This isn't a rare edge case. Every single PR creation through any path (`/git-pr` direct, `/fab-ff`, `/fab-fff`) triggers this because they all flow through the same `git-pr.md` skill.

## What Changes

### Reorder Step 4d before Step 4b in `git-pr.md`

The current step ordering in `fab/.kit/skills/git-pr.md`:

```
Step 4:  Record PR URL       → fab status add-pr (mutates .status.yaml)
Step 4b: Commit + push        → git add/commit/push .status.yaml
Step 4c: Write .pr-done       → echo to .pr-done (gitignored)
Step 4d: Finish ship stage    → fab status finish <change> ship (mutates .status.yaml + .history.jsonl)
         ^^^ NO COMMIT AFTER THIS
```

The fix reorders so that all `.status.yaml` mutations happen before the single commit+push:

```
Step 4:  Record PR URL       → fab status add-pr (mutates .status.yaml)
Step 4d: Finish ship stage   → fab status finish <change> ship (mutates .status.yaml + .history.jsonl)
Step 4b: Commit + push       → git add .status.yaml .history.jsonl / commit / push
Step 4c: Write .pr-done      → echo to .pr-done (gitignored, stays last)
```

Specific edits to `git-pr.md`:

1. **Move Step 4d** to immediately after Step 4 (before current Step 4b)
2. **Update Step 4b** to stage both `.status.yaml` AND `.history.jsonl` (currently only stages `.status.yaml`)
3. **Update Step 4b commit message** to reflect the broader scope (e.g., "Update ship status and record PR URL")
4. **Renumber steps** for clarity: 4 → 4a (record PR), 4b (finish ship), 4c (commit+push), 4d (write sentinel)

The `.pr-done` sentinel write (current Step 4c) stays last — it's gitignored and serves as a filesystem signal that all git operations are complete. Moving it after the commit is correct.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update git-pr step ordering documentation

## Impact

- `fab/.kit/skills/git-pr.md` — step reordering and staging list update
- `docs/specs/skills/SPEC-git-pr.md` — update if step ordering is documented there
- All PR creation paths (`/git-pr`, `/fab-ff`, `/fab-fff`) benefit automatically since they all delegate to `git-pr.md`

## Open Questions

- None — root cause and fix are well-understood from the discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Reorder steps rather than add a second commit+push | Discussed — consolidating into one commit is cleaner than adding another commit+push cycle. Single commit captures all status mutations atomically. | S:95 R:90 A:95 D:95 |
| 2 | Certain | Stage `.history.jsonl` in the commit alongside `.status.yaml` | `fab status finish` writes to both files via auto-logging. Both must be committed. | S:90 R:90 A:90 D:95 |
| 3 | Certain | Keep `.pr-done` sentinel as the last step | Discussed — it's gitignored and signals completion of all git operations. Must stay after commit+push. | S:90 R:95 A:95 D:95 |
| 4 | Certain | `finish ship` is best-effort (failure silently ignored) | Existing behavior — Step 4d already uses `2>/dev/null || true`. The commit step should handle the case where finish didn't produce changes (check via `git diff --cached --quiet`). | S:85 R:90 A:90 D:90 |
| 5 | Confident | Renumber steps as 4a/4b/4c/4d for clarity | Convention choice — renumbering avoids confusion with the old ordering. Low stakes, easily changed. | S:70 R:95 A:70 D:75 |
| 6 | Certain | No changes needed to `/fab-ff` or `/fab-fff` orchestration | They delegate to the git-pr skill behavior. Fixing the skill fixes all callers. | S:90 R:95 A:90 D:95 |
| 7 | Certain | The spec for git-pr may need a corresponding update | Constitution requires skill changes to update corresponding `SPEC-*.md` file. | S:85 R:90 A:90 D:90 |
| 8 | Certain | This is the same bug class as 260222-trdc | Archived change added Step 4b to fix uncommitted PR URL. Step 4d was added later, reintroducing the pattern. Fix follows the same principle: all mutations before the commit boundary. | S:90 R:90 A:90 D:95 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
