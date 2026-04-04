# Intake: Operator Base-Chaining Default

**Change**: 260327-gwg9-operator-base-chaining-default
**Created**: 2026-03-27
**Status**: Draft

## Origin

> Backlog item [gwg9]: "fab-operator7: Default to --base chaining for multi-phase autopilot queues instead of merge-as-you-go. When the operator runs a queue of dependent changes, it should stack branches (cherry-pick deps) and let the user review all PRs before merging — not auto-merge each PR on completion. Only merge-as-you-go when the user explicitly requests it. This prevents rebase conflicts and gives the user review control. Current behavior: operator merges each PR immediately on completion then rebases the next change onto origin/main."

One-shot initiation from backlog.

## Why

The current autopilot queue behavior in `/fab-operator7` merges each PR immediately on completion and rebases the next queued change onto `origin/main` (steps 5–7 in §6 Autopilot). This causes two problems:

1. **Rebase conflicts** — when change B depends on change A, rebasing B onto a freshly-merged `origin/main` can produce conflicts that the cherry-pick dependency resolution already handled. The rebase re-linearizes commits that were carefully cherry-picked, creating unnecessary friction.
2. **No review control** — the user has no opportunity to review all PRs holistically before any of them merge. Once the first PR merges, the commits are in `main` and the remaining PRs must deal with the consequences. The user might want to adjust a later change based on seeing the full set together.

By defaulting to `--base` chaining (branch stacking), all changes in a queue build on top of each other via cherry-pick deps, all PRs are created but none are merged until the user explicitly decides. This gives the user full review control over the entire queue before any code hits `main`.

## What Changes

### Default autopilot queue strategy

The autopilot queue in `fab/.kit/skills/fab-operator7.md` §6 Autopilot currently has this flow:

```
5. Monitor
6. Merge — on completion, merge PR from operator's shell
7. Rebase next — rebase next queued change onto latest origin/main. On conflict: flag, skip
8. Cleanup — optionally delete worktree after merge
9. Report — "ab12: merged. 1 of 3 complete. Starting cd34."
```

This changes to a **stack-then-review** default:

```
5. Monitor
6. Stack — on completion, record branch in branch_map and proceed to next queued item
7. Dispatch next — spawn next change, cherry-pick deps (existing mechanism), dispatch /fab-fff
8. Report — "ab12: PR ready. 1 of 3 complete. Starting cd34."
9. (After all complete) Summary — "Queue complete. 3 PRs ready for review: [links]. Merge when ready."
```

The existing `--base <prev-change>` flag already implies `depends_on` and triggers cherry-pick resolution. The change makes this the **default** for all autopilot queues — every queued change after the first implicitly gets `depends_on: [<prev-change-id>]` unless the user specifies an explicit ordering strategy.

### Merge-as-you-go opt-in

A new `--merge-on-complete` flag (or equivalent phrasing in natural language like "merge as you go") reverts to the current behavior: merge each PR on completion, rebase next onto `origin/main`.

### Queue completion behavior

When all changes in a stacked queue complete:
- Operator reports a summary with all PR links
- Operator suggests merge order (base-first)
- User can merge individually or ask operator to merge all in order
- Operator merges in dependency order when asked, waiting for CI to pass on each

### Confirmation prompt update

The current autopilot confirmation says "Confirm upfront (merges PRs)." This changes to reflect the new default: "Confirm upfront (creates PRs — merge after review)." When `--merge-on-complete` is used: "Confirm upfront (merges PRs on completion)."

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator7 autopilot queue behavior to document the stack-then-review default

## Impact

- **`fab/.kit/skills/fab-operator7.md`** — §6 Autopilot section: rewrite steps 5–9, update queue ordering table, update confirmation text, add `--merge-on-complete` flag
- **`docs/specs/skills/SPEC-fab-operator7.md`** — if it exists, update to reflect new default
- No code changes (Go binary) — this is a skill-file-only change
- No migration needed — `.fab-operator.yaml` schema unchanged (depends_on already exists)

## Open Questions

- None — the backlog description is specific and the mechanism (cherry-pick deps via `depends_on` + `--base`) already exists.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use existing `depends_on` + cherry-pick mechanism for stacking | Already implemented in operator7 §6 dependency resolution | S:90 R:90 A:95 D:95 |
| 2 | Certain | Skill-file-only change, no Go binary modifications | Backlog describes behavior change in the operator skill, not new CLI commands | S:85 R:95 A:90 D:90 |
| 3 | Confident | Implicit `--base` chaining for all queued items after the first | Backlog says "default to --base chaining" — simplest implementation is implicit depends_on for sequential queue entries | S:80 R:70 A:80 D:75 |
| 4 | Confident | `--merge-on-complete` flag name for opt-in to old behavior | Descriptive and consistent with operator conventions; easily renamed | S:60 R:85 A:70 D:65 |
| 5 | Confident | Operator merges in dependency order when user requests post-review merge | Natural consequence of stacked branches — base must merge first | S:70 R:80 A:85 D:80 |
| 6 | Certain | No `.fab-operator.yaml` schema changes needed | `depends_on` and `branch_map` already exist and support this flow | S:90 R:95 A:90 D:95 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
