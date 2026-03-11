# Spec: Operator Autopilot UC8

**Change**: 260310-1ttn-operator-autopilot-uc8
**Created**: 2026-03-11
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Persistent queue state — v1 uses conversation context only; file-backed queue deferred
- New CLI primitives — all required commands (pane-map, send-keys, runtime, status show) already exist
- Background polling/daemon behavior — operator remains user-driven, not event-driven
- Multi-repo coordination — out of scope per existing design decision

## Skill Structure: Heading and Use Case Updates

### Requirement: Use Cases Heading Rename

The heading "Seven Use Cases" in `fab/.kit/skills/fab-operator1.md` SHALL be renamed to "Use Cases".

#### Scenario: Heading is generic

- **GIVEN** the current skill file has heading `## Seven Use Cases`
- **WHEN** the change is applied
- **THEN** the heading reads `## Use Cases`
- **AND** no other references to "Seven Use Cases" remain in the skill file

### Requirement: UC8 Stub

A new UC8 entry SHALL be added to the Use Cases section in `fab/.kit/skills/fab-operator1.md`, positioned after UC7 (Notification surface).

The UC8 stub SHALL:
- Accept a list of changes (IDs, names, or "all idle")
- Reference ordering resolution via one of three strategies
- Note that this is destructive (merges PRs) and confirms the full queue at start
- Delegate to the Autopilot Behavior section for execution details

#### Scenario: UC8 appears in use cases

- **GIVEN** the skill file has UC7 (Notification surface) as the last use case
- **WHEN** the change is applied
- **THEN** UC8 (Autopilot) appears after UC7
- **AND** UC8 describes accepting a list of changes and delegating to the Autopilot Behavior section

### Requirement: Confirmation Model Update

A new row SHALL be added to the Confirmation Model table for the autopilot use case. The row SHALL classify autopilot as Destructive, requiring full queue confirmation at start, with no per-PR confirmation during execution.

#### Scenario: Confirmation model includes autopilot

- **GIVEN** the Confirmation Model table has rows for Read-only, Recoverable, and Destructive
- **WHEN** the change is applied
- **THEN** the Destructive row's examples include "Autopilot (merge after each success)"
- **AND** the existing Destructive examples are preserved

## Autopilot Behavior: Ordering Strategies

### Requirement: Three Ordering Strategies

The Autopilot Behavior section SHALL support three ordering strategies for the change queue:

1. **User-provided**: Run in the exact order given by the user
2. **Confidence-based**: Sort by confidence score descending (via `fab status show --all`)
3. **Hybrid**: User provides ordering constraints (partial order), operator sorts the rest by confidence as tiebreaker

#### Scenario: User-provided ordering

- **GIVEN** the user says "run bh45, qkov, ab12"
- **WHEN** the operator resolves ordering
- **THEN** the queue is [bh45, qkov, ab12] in that exact order

#### Scenario: Confidence-based ordering

- **GIVEN** the user says "run all my changes, highest confidence first"
- **WHEN** the operator resolves ordering
- **THEN** changes are sorted by confidence score descending
- **AND** the operator uses `fab status show --all` to obtain scores

#### Scenario: Hybrid ordering

- **GIVEN** the user says "bh45 before qkov, optimize the rest"
- **WHEN** the operator resolves ordering
- **THEN** bh45 appears before qkov in the queue
- **AND** remaining unconstrained changes are sorted by confidence

## Autopilot Behavior: Execution Loop

### Requirement: Per-Change Autopilot Loop

For each change in the resolved queue, the operator SHALL execute the following sequence:

1. **Spawn**: Create worktree via `wt create --non-interactive`
2. **Open tab**: `tmux new-window -n "fab-<id>" -c <worktree> "claude --dangerously-skip-permissions '/fab-switch <change>'"`
3. **Gate check**: Check confidence via `fab status show <change>`
   - confidence >= gate → `fab send-keys <change> "/fab-ff"`
   - confidence < gate → flag to user with score and threshold
4. **Monitor**: Poll `fab pane-map` + `fab runtime is-idle` on each user interaction
   - Stage reaches hydrate/ship → change succeeded
   - Review fails after rework budget → flag and skip
   - Agent idle >15min at non-terminal stage → nudge once, then flag
   - Pane dies → flag and skip
5. **On success**: `gh pr merge` from operator shell (destructive — already confirmed at start)
6. **Rebase next**: `fab send-keys <next-change> "git fetch origin main && git rebase origin/main"`
   - If conflict → flag to user, skip to next (never auto-resolve)
7. **Cleanup**: `wt delete` (optional, after merge)
8. **Progress**: Report one-line status

#### Scenario: Successful change through autopilot

- **GIVEN** change bh45 has confidence 4.2 (above gate)
- **WHEN** the operator processes bh45
- **THEN** a worktree is created, a tmux tab opened, `/fab-ff` dispatched
- **AND** on pipeline completion, the PR is merged from the operator shell
- **AND** the next change in queue is rebased on main

#### Scenario: Change below confidence gate

- **GIVEN** change qkov has confidence 2.1 (below feat gate 3.0)
- **WHEN** the operator's gate check runs
- **THEN** the operator flags: "qkov confidence 2.1, below feat gate (3.0). Run /fab-fff or skip?"
- **AND** the operator waits for user input before proceeding

#### Scenario: Full queue confirmation at start

- **GIVEN** the user requests autopilot for 3 changes
- **WHEN** the operator resolves the queue
- **THEN** the operator displays the full ordered queue with confidence scores
- **AND** asks for confirmation, noting that PRs will be merged (destructive)

## Autopilot Behavior: Failure Handling

### Requirement: Failure Matrix

The operator SHALL handle failures according to the following matrix:

| Failure | Action | Resume? |
|---------|--------|---------|
| Confidence below gate | Flag to user: run `/fab-fff` or skip | Wait for user input |
| Review fails (rework exhausted) | Flag, skip to next change | Yes |
| Rebase conflict | Flag, skip to next change | Yes |
| Agent pane dies | 1 respawn attempt, then flag and skip | Yes |
| Stage timeout (>30 min same stage) | Flag regardless of retry state | Yes |
| Total timeout (>2 hr per change) | Flag for review | Yes |

#### Scenario: Review fails after rework budget

- **GIVEN** change ab12's agent has exhausted the rework budget (3 cycles)
- **WHEN** the operator detects review failure
- **THEN** the operator flags to user and skips to the next change in queue
- **AND** the operator reports the skip in progress output

#### Scenario: Rebase conflict after merge

- **GIVEN** bh45 was merged and qkov is next
- **WHEN** rebase of qkov on main produces a conflict
- **THEN** the operator flags: "qkov has rebase conflicts after merging bh45. Resolve manually or skip?"
- **AND** the operator never auto-resolves conflicts

#### Scenario: Agent pane dies

- **GIVEN** the operator detects a change's pane is gone via pane map refresh
- **WHEN** the operator attempts 1 respawn
- **THEN** if respawn succeeds, execution continues
- **AND** if respawn fails, the operator flags and skips to next change

#### Scenario: Stage timeout

- **GIVEN** a change has been at the same stage for >30 minutes
- **WHEN** the operator's monitoring detects the timeout
- **THEN** the operator flags to user regardless of the agent's active/idle state

## Autopilot Behavior: Interruptibility

### Requirement: User Interrupt Commands

During autopilot, the operator SHALL support the following interrupt commands:

- `"stop after current"` — finish active change, halt queue
- `"skip <change>"` — remove from queue, proceed to next
- `"pause"` — stop sending new commands, running agents continue
- `"resume"` — pick up from where paused

The operator SHALL acknowledge interrupts immediately, even if an action is in progress.

#### Scenario: Stop after current

- **GIVEN** the operator is processing bh45 with qkov and ab12 remaining
- **WHEN** the user says "stop after current"
- **THEN** bh45 completes (merge if successful)
- **AND** qkov and ab12 are not started
- **AND** the operator reports: "Stopped. bh45 completed. qkov, ab12 not started."

#### Scenario: Skip a change

- **GIVEN** the operator is processing a queue with qkov pending
- **WHEN** the user says "skip qkov"
- **THEN** qkov is removed from the queue
- **AND** execution proceeds to the next change

#### Scenario: Pause and resume

- **GIVEN** the operator is in autopilot mode
- **WHEN** the user says "pause"
- **THEN** the operator stops sending new commands
- **AND** already-running agents continue their current work
- **WHEN** the user later says "resume"
- **THEN** the operator picks up from where it paused

## Autopilot Behavior: Resumability

### Requirement: State Reconstruction

If the operator session restarts, state SHALL be reconstructable from `fab pane-map`. Merged changes show as archived/shipped, in-progress changes show their current stage. The operator resumes from the first non-completed change.

#### Scenario: Operator session restarts mid-queue

- **GIVEN** the operator was processing a queue of 3 changes, bh45 already merged
- **WHEN** the operator session restarts
- **THEN** the operator reconstructs state from `fab pane-map`
- **AND** identifies bh45 as completed (archived/shipped stage)
- **AND** resumes from the next non-completed change

## Autopilot Behavior: Progress Reporting

### Requirement: Progress Reporting

After each change completes (success or skip), the operator SHALL output a one-line status. A final summary SHALL list all changes with outcomes.

#### Scenario: Per-change progress

- **GIVEN** the operator merges bh45 successfully (1st of 3)
- **WHEN** the merge completes
- **THEN** the operator reports: "bh45: merged. 1 of 3 complete. Starting qkov."

#### Scenario: Final summary

- **GIVEN** all changes in the queue have been processed
- **WHEN** the queue is complete
- **THEN** the operator outputs a summary listing each change with its outcome (merged, skipped, failed)

## Spec Alignment: Renumbering

### Requirement: Spec UC Renumbering

`docs/specs/skills/SPEC-fab-operator1.md` SHALL be updated to align use case numbering with the skill file:

- Spec's current UC7 (Sequential pipeline execution / autopilot) → UC8
- Spec's current UC8 (Notification surface) → UC7

This ensures the spec matches the skill's canonical ordering where UC7 = Notification surface and UC8 = Autopilot.

#### Scenario: Spec numbering matches skill

- **GIVEN** the spec currently has UC7 as autopilot and UC8 as notification surface
- **WHEN** the change is applied
- **THEN** the spec has UC7 as Notification surface and UC8 as Autopilot
- **AND** all internal cross-references within the spec are updated

## Design Decisions

1. **Approach B: UC8 stub + separate Autopilot Behavior section**
   - *Why*: Keeps the use cases list scannable while providing detailed behavior documentation. Mirrors the pattern of `/fab-continue` which has separate "Apply Behavior" and "Review Behavior" sections.
   - *Rejected*: Approach A (inline all details in UC8) — would make the use cases section unbalanced and hard to scan.

2. **Multi-worktree model: each change gets its own worktree + agent pane**
   - *Why*: Enables true parallelism — each agent has its own working tree and can make file changes without interference. Reuses the existing worktree infrastructure (`wt create`, `wt delete`).
   - *Rejected*: Single-pane sequential — slower and doesn't leverage the existing multi-worktree setup.

3. **Full autopilot with merge after each success**
   - *Why*: Merging after each success means subsequent changes rebase on a fresher main, reducing conflict accumulation. Matches the pipeline model where changes flow through to completion.
   - *Rejected*: Checkpoint-after-each — adds friction without benefit; the confirmation at start covers the destructive intent.

4. **Conversation-only queue state (v1)**
   - *Why*: Minimizes complexity. The operator already holds instructions in conversation context. File-backed queue is a clean upgrade path if context compression causes friction.
   - *Rejected*: File-backed queue from the start — premature complexity for v1.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Extend fab-operator1 skill, not a new skill | Confirmed from intake #1 — user explicitly chose operator extension | S:95 R:80 A:90 D:95 |
| 2 | Certain | Multi-worktree model (each change gets own worktree + pane) | Confirmed from intake #2 — user chose multi-worktree | S:95 R:70 A:85 D:95 |
| 3 | Certain | Full autopilot with merge after each success | Confirmed from intake #3 — user chose full autopilot | S:95 R:60 A:80 D:95 |
| 4 | Certain | All three ordering strategies | Confirmed from intake #4 — user explicitly requested all three | S:95 R:75 A:85 D:90 |
| 5 | Certain | Approach B: UC8 stub + separate Autopilot Behavior section | Confirmed from intake #5 — user chose Approach B | S:95 R:85 A:90 D:95 |
| 6 | Confident | Renumber spec UC7 → UC8 to match skill numbering | Confirmed from intake #6 — spec alignment is straightforward | S:70 R:90 A:80 D:75 |
| 7 | Certain | No new CLI primitives needed | Confirmed from intake #7 — verified all commands exist | S:80 R:85 A:90 D:80 |
| 8 | Certain | Conversation context sufficient for v1 queue state | Confirmed from intake #8 — user chose option A | S:95 R:60 A:55 D:50 |
| 9 | Certain | Spawn pattern: wt create + tmux new-window + claude --dangerously-skip-permissions | Confirmed from intake #9 — validated from batch script | S:95 R:70 A:90 D:90 |
| 10 | Confident | Timeout values: 30 min stage, 2 hr total, 15 min idle nudge | Intake specified these values; align with existing guardrails in spec | S:75 R:90 A:70 D:70 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
