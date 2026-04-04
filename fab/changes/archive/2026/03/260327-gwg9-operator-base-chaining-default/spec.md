# Spec: Operator Base-Chaining Default

**Change**: 260327-gwg9-operator-base-chaining-default
**Created**: 2026-03-27
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Operator: Autopilot Queue Default Strategy

### Requirement: Stack-then-review as default autopilot behavior

When the operator processes an autopilot queue, it SHALL default to **stack-then-review** mode: all queued changes build on each other via implicit `depends_on` chaining, PRs are created but NOT merged until the user explicitly requests merging. This replaces the current merge-as-you-go default.

Specifically, for an autopilot queue of N changes processed in order, the operator SHALL implicitly set `depends_on: [<prev-change-id>]` on each change after the first (equivalent to `--base <prev-change>` for every consecutive pair). The existing cherry-pick dependency resolution mechanism handles the actual branch stacking.

#### Scenario: Default autopilot queue with three changes

- **GIVEN** the user says "run ab12 then cd34 then ef56 in autopilot"
- **WHEN** the operator sets up the autopilot queue
- **THEN** `cd34` gets `depends_on: [ab12]` and `ef56` gets `depends_on: [cd34]`
- **AND** each change is spawned with cherry-pick resolution of its deps
- **AND** on completion, the operator reports "ab12: PR ready. 1 of 3 complete." (not "merged")
- **AND** after all three complete, the operator reports a summary with all PR links and suggested merge order

#### Scenario: Single-item autopilot queue

- **GIVEN** the user says "run ab12 in autopilot"
- **WHEN** the operator processes the single-item queue
- **THEN** no `depends_on` is added (no predecessor)
- **AND** on completion, the operator reports "ab12: PR ready. Queue complete."

### Requirement: Autopilot completion summary

When all changes in a stack-then-review autopilot queue complete, the operator SHALL display a completion summary.

#### Scenario: Queue completion with three stacked changes

- **GIVEN** three changes (ab12, cd34, ef56) were processed in stack-then-review mode
- **WHEN** the last change (ef56) completes (PR created)
- **THEN** the operator displays:
  ```
  Queue complete. 3 PRs ready for review:
  1. ab12: <PR-URL-1> (base)
  2. cd34: <PR-URL-2> (depends on ab12)
  3. ef56: <PR-URL-3> (depends on cd34)
  Merge in order (1→2→3) when ready, or ask me to merge all.
  ```

### Requirement: User-requested merge of stacked queue

When the user requests merging of a completed stacked queue, the operator SHALL merge PRs in dependency order (base-first), waiting for CI to pass on each before proceeding to the next.

#### Scenario: Merge all in order

- **GIVEN** a completed stack-then-review queue with 3 PRs
- **WHEN** the user says "merge all" or "merge the queue"
- **THEN** the operator merges PR 1 (base), waits for CI pass
- **AND** then merges PR 2, waits for CI pass
- **AND** then merges PR 3, waits for CI pass
- **AND** reports each merge: "ab12: merged (1/3)", "cd34: merged (2/3)", "ef56: merged (3/3)"

#### Scenario: CI failure during ordered merge

- **GIVEN** the operator is merging a stacked queue in order
- **WHEN** CI fails on PR 2
- **THEN** the operator stops merging and reports: "cd34: CI failed. Merge halted at 1/3. Fix and retry."
- **AND** does not attempt to merge PR 3

### Requirement: Merge-on-complete opt-in

The operator SHALL support a `--merge-on-complete` flag (or natural language equivalent) that reverts to the previous merge-as-you-go behavior: merge each PR on completion, rebase next change onto `origin/main`.

#### Scenario: Explicit merge-on-complete autopilot

- **GIVEN** the user says "run ab12 then cd34 in autopilot --merge-on-complete"
- **WHEN** ab12 completes
- **THEN** the operator merges ab12's PR immediately
- **AND** rebases cd34 onto latest `origin/main`
- **AND** reports "ab12: merged. 1 of 2 complete. Starting cd34."

#### Scenario: Natural language merge-on-complete

- **GIVEN** the user says "run these in autopilot and merge as you go"
- **WHEN** the operator interprets the instruction
- **THEN** it enables merge-on-complete mode for this queue

### Requirement: Updated confirmation prompt

The autopilot confirmation prompt SHALL reflect the active queue mode.

#### Scenario: Default (stack-then-review) confirmation

- **GIVEN** the user requests an autopilot queue without `--merge-on-complete`
- **WHEN** the operator confirms the queue
- **THEN** the confirmation says "Confirm upfront (creates PRs — merge after review)."

#### Scenario: Merge-on-complete confirmation

- **GIVEN** the user requests an autopilot queue with `--merge-on-complete`
- **WHEN** the operator confirms the queue
- **THEN** the confirmation says "Confirm upfront (merges PRs on completion)."

### Requirement: Updated autopilot steps

The numbered autopilot steps in §6 SHALL be updated to reflect the stack-then-review default:

1. **Spawn** — create worktree (`--reuse` for respawns)
2. **Resolve dependencies** — cherry-pick `depends_on` entries into the worktree, then open agent tab and enroll
3. **Gate** — check confidence score. If below threshold, flag and wait
4. **Dispatch** — send `/fab-fff` (or appropriate command based on current stage)
5. **Monitor** — normal tick detection handles progress
6. **Record** — on completion, record branch in `branch_map`, collect PR URL
7. **Dispatch next** — spawn next change (with implicit `depends_on`), cherry-pick deps, dispatch
8. **Report** — `"ab12: PR ready. 1 of 3 complete. Starting cd34."`
9. **(After all complete) Summary** — list all PR links with merge order suggestion

When `--merge-on-complete` is active, steps 6–9 revert to the previous behavior: merge PR, rebase next onto `origin/main`, report merge.

#### Scenario: Step-by-step flow in default mode

- **GIVEN** an autopilot queue of [ab12, cd34]
- **WHEN** ab12 completes
- **THEN** the operator records ab12's branch in `branch_map` and collects the PR URL
- **AND** spawns cd34 with `depends_on: [ab12]`
- **AND** cherry-picks ab12's content into cd34's worktree
- **AND** dispatches `/fab-fff` for cd34
- **AND** reports "ab12: PR ready. 1 of 2 complete. Starting cd34."

### Requirement: Queue ordering table update

The queue ordering strategy table SHALL be updated to note that user-provided ordering now implies `--base` chaining by default (not just when `--base` is explicitly passed).

#### Scenario: User-provided ordering without explicit --base

- **GIVEN** the user says "run ab12 then cd34 then ef56 in autopilot"
- **WHEN** the operator sets up the queue
- **THEN** implicit `--base` chaining is applied: cd34 depends on ab12, ef56 depends on cd34
- **AND** the operator does not require the user to say `--base` explicitly

### Requirement: Failure handling in stack-then-review mode

The existing failure handling SHALL apply to stack-then-review mode with one modification: "Rebase conflict → skip" is replaced with the existing cherry-pick conflict handling since there are no rebase steps in the default mode.

#### Scenario: Cherry-pick conflict in stacked queue

- **GIVEN** an autopilot queue where cd34 depends on ab12
- **WHEN** cherry-picking ab12's content into cd34's worktree produces a conflict
- **THEN** the operator escalates (does not skip) per existing cherry-pick conflict policy

## Deprecated Requirements

### Merge-as-you-go as default autopilot behavior

**Reason**: Replaced by stack-then-review as the default. The merge-as-you-go behavior is preserved via `--merge-on-complete` opt-in.
**Migration**: Users who relied on the default merge behavior should use `--merge-on-complete` flag or natural language "merge as you go".

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use existing `depends_on` + cherry-pick mechanism for stacking | Confirmed from intake #1 — already implemented in operator7 §6 | S:90 R:90 A:95 D:95 |
| 2 | Certain | Skill-file-only change, no Go binary modifications | Confirmed from intake #2 — operator behavior lives in skill markdown | S:85 R:95 A:90 D:90 |
| 3 | Confident | Implicit `--base` chaining for all queued items after the first | Confirmed from intake #3 — backlog says "default to --base chaining" | S:80 R:70 A:80 D:75 |
| 4 | Confident | `--merge-on-complete` flag name for opt-in | Confirmed from intake #4 — descriptive, consistent with operator conventions | S:60 R:85 A:70 D:65 |
| 5 | Confident | Merge in dependency order on user request, with CI wait | Confirmed from intake #5 — natural for stacked branches, CI gate prevents broken merges | S:70 R:80 A:85 D:80 |
| 6 | Certain | No `.fab-operator.yaml` schema changes needed | Confirmed from intake #6 — `depends_on`, `branch_map` already support this | S:90 R:95 A:90 D:95 |
| 7 | Certain | Confidence-based and hybrid strategies remain unchanged | These strategies are for independent changes — stacking only applies to user-provided sequential ordering | S:85 R:90 A:90 D:90 |
| 8 | Confident | Natural language "merge as you go" maps to `--merge-on-complete` | Operator already interprets natural language for flags; consistent pattern | S:65 R:85 A:75 D:70 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
