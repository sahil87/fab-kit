# Plan: git-pr-review Commit Status Updates

**Change**: 260602-7x8u-git-pr-review-commit-status
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### git-pr-review: Commit Review-PR Status Bookkeeping

#### R1: Commit Status Updates step
`git-pr-review.md` SHALL add a new step after Step 6 ("Update Review-PR Stage") that commits the `.status.yaml` and `.history.jsonl` writes produced by Step 6's `fab status finish`, mirroring `git-pr.md` Step 4c.

- **GIVEN** an active change was resolved in Step 0 AND Step 6's `fab status finish` ran (success / no-reviews path)
- **WHEN** the new step executes
- **THEN** it stages `fab/changes/{name}/.status.yaml` and `fab/changes/{name}/.history.jsonl`
- **AND** it commits with message `Update review-pr status` and pushes when staged changes exist
- **AND** it prints `  ✓ status — committed and pushed status updates (.status.yaml, .history.jsonl)` only when a commit was made

#### R2: Gate on success path only
The new step SHALL run only on the success / no-reviews path of Step 6 and SHALL be skipped silently otherwise (no active change, or the Step 6 `fail` path — the fail path MUST NOT commit a half-finished state).

- **GIVEN** no active change was resolved, OR Step 6 took the `fail` path
- **WHEN** control reaches the new step
- **THEN** the step is skipped silently with no commit, no push, and no output

#### R3: Idempotency preserved via staged-diff guard
The new step SHALL preserve `git-pr-review`'s idempotency rule using a `git diff --cached --quiet` guard so that a re-run finds nothing staged and is a silent no-op.

- **GIVEN** the status writes were already committed on a prior run (re-invocation)
- **WHEN** the step stages the files and runs `git diff --cached --quiet`
- **THEN** no commit or push occurs and nothing is printed (silent no-op)

#### R4: Best-effort push, commit mirrors git-pr
On commit/push failure, push handling SHALL be best-effort: a transient push failure SHALL be reported (logged) but SHALL NOT hard-STOP the skill or abort an otherwise-complete review cycle. The commit itself mirrors git-pr's pattern.

- **GIVEN** the commit succeeded but `git push` fails (e.g., transient network error)
- **WHEN** the push error is observed
- **THEN** the error is reported and the skill completes normally (no STOP, no stage `fail`)
- **AND** the local commit is retained (a later re-run / push reconciles it)

#### R5: Spec stays in sync
`docs/specs/skills/SPEC-git-pr-review.md` SHALL document the new step in its Flow diagram, Tools-used table, and `.status.yaml`-writes/bookkeeping description, consistent with how `SPEC-git-pr.md` documents Step 4c.

- **GIVEN** the constitution mandate "Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md`"
- **WHEN** `git-pr-review.md` gains the new step
- **THEN** `SPEC-git-pr-review.md` reflects the step in the Flow block and supporting tables
- **AND** the spec and skill remain internally consistent (same gating, same commit message, same print line)

### Non-Goals

- No new `fab` CLI surface — the step reuses existing `git add`/`git commit`/`git push` and the already-called `fab status finish`.
- No change to git-pr.md (the precedent) — this change only adds the symmetric step to git-pr-review.md.
- No change to Step 5's code-fix commit/push behavior (that path keeps its own fail-fast `git reset` semantics).

### Design Decisions

1. **Best-effort push, fail-fast-free**: The new step does NOT `git reset` or STOP on push failure — *Why*: git-pr-review's terminal-stage ethos is "don't abort a completed review cycle"; the status write is already durable as a local commit and is reconcilable on re-run. *Rejected*: git-pr-parity fail-fast STOP — defensible for symmetry but would let a transient network blip abort an otherwise-finished pipeline, which the intake explicitly leans against (Assumption #7). The commit half still mirrors git-pr; only the push half is softened.

## Tasks

### Phase 1: Skill Source

- [x] T001 Add a new "Step 6.5: Commit Status Updates" to `src/kit/skills/git-pr-review.md` after Step 6, mirroring `git-pr.md` Step 4c: gate on active-change-resolved + Step 6 success/no-reviews path (skip on no-change and on the `fail` path); `git add` the change's `.status.yaml` + `.history.jsonl`; `git diff --cached --quiet` guard; commit `Update review-pr status` + push when staged changes exist; print the `✓ status` line on commit. Resolve push failure as best-effort (report, do not STOP). <!-- R1 R2 R3 R4 -->
- [x] T002 Update the `## Rules` section of `src/kit/skills/git-pr-review.md` so the new step's best-effort push does not contradict the existing "Fail fast" / "No partial commits" rules (scope those rules to the code-fix path in Step 5, or note the status-commit exception). <!-- R4 -->

### Phase 2: Spec Sync

- [x] T003 Update `docs/specs/skills/SPEC-git-pr-review.md` to document the new step: add it to the Flow diagram (after Step 6), to the "Direct .status.yaml writes" / bookkeeping description as appropriate, and ensure the Tools-used table still covers the git operations. Match SPEC-git-pr.md's Step 4c treatment. <!-- R5 -->

### Phase 3: Consistency Verification

- [x] T004 Verify spec ↔ skill internal consistency: same gating (success path only), same commit message (`Update review-pr status`), same print line, idempotency guard documented, best-effort push reflected. No test harness exists for skill prose — this is a manual cross-read. <!-- R5 -->

## Execution Order

- T001 blocks T002 (Rules wording depends on the step it qualifies)
- T001 and T002 block T003 (spec mirrors the finalized skill)
- T003 blocks T004 (verification reads the finalized spec + skill)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `git-pr-review.md` has a new step after Step 6 that stages `.status.yaml` + `.history.jsonl`, commits `Update review-pr status`, pushes, and prints the `✓ status` line on commit.
- [x] A-002 R5: `SPEC-git-pr-review.md` documents the new step in its Flow diagram and supporting tables, consistent with SPEC-git-pr.md's Step 4c.

### Behavioral Correctness

- [x] A-003 R2: The new step is gated to run only on Step 6's success / no-reviews path; the `fail` path and the no-active-change path skip silently with no commit.
- [x] A-004 R3: A `git diff --cached --quiet` guard is present so a re-run with nothing staged is a silent no-op (idempotency preserved).
- [x] A-005 R4: Push failure is best-effort — the step reports the error but does not STOP the skill or fail the stage; the commit mirrors git-pr.

### Scenario Coverage

- [x] A-006 R1: Success path — given a resolved change and a successful Step 6 finish, staged status writes are committed and pushed and the print line appears.
- [x] A-007 R2: Fail path — given Step 6's `fail` branch, the step does not commit.

### Edge Cases & Error Handling

- [x] A-008 R3: Idempotent re-run — given the writes are already committed, the staged-diff guard yields no commit, no push, no output.
- [x] A-009 R4: Transient push failure — given a push error, the error is reported and the skill still completes (no hard STOP).

### Code Quality

- [x] A-010 Pattern consistency: New step follows git-pr.md Step 4c's structure, numbering style, and print-line format; surrounding git-pr-review.md conventions (best-effort status writes, idempotency rule) are honored.
- [x] A-011 No unnecessary duplication: Reuses the established Step 4c pattern and existing git/fab commands; invents no new mechanism.
- [x] A-012 Follow existing project patterns: Edits only `src/kit/skills/git-pr-review.md` (canonical source, never `.claude/skills/`) and the corresponding `SPEC-*.md`, per constitution.
- [x] A-013 No god functions / magic strings: Commit message and print line match the intake's exact specified strings; no ad-hoc variants introduced.

### Documentation Accuracy

- [x] A-014 R5: Spec and skill are internally consistent — same gating, commit message, print line, and best-effort-push semantics described in both.

### Cross-References

- [x] A-015 R5: Cross-references between git-pr-review.md, SPEC-git-pr-review.md, and the git-pr.md Step 4c precedent remain accurate.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- No bats/skill-prose test harness exists in this repo; the Go test suite covers only the `fab` CLI binary. "test-alongside" here = spec↔skill cross-read for internal consistency (T004).

## Deletion Candidates

- None — this change adds a new step (Step 6.5) and reconciles prose in the `## Rules` section / spec. It makes no existing code, file, function, or branch redundant or unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add a "Commit Status Updates" step after Step 6 in git-pr-review.md, mirroring git-pr.md Step 4c | Intake Assumption #1 (Certain); user explicitly chose this over alternatives; Step 4c is the established, tested precedent | S:98 R:80 A:95 D:95 |
| 2 | Certain | Also update SPEC-git-pr-review.md | Constitution mandates spec update for any skill-file change | S:98 R:70 A:98 D:98 |
| 3 | Confident | Gate on active-change-resolved + Step 6 success/no-reviews path; skip silently on no-change and on the `fail` path | Intake Assumption #4; direct parallel to git-pr Step 4c's "if Step 4a recorded a PR URL" gate; committing a half-finished `fail` state is wrong | S:80 R:75 A:85 D:80 |
| 4 | Confident | Commit message `Update review-pr status`; print line `  ✓ status — committed and pushed status updates (.status.yaml, .history.jsonl)` | Intake Assumption #5 + explicit print string in intake `## What Changes`; mirrors git-pr's status-commit line | S:90 R:90 A:90 D:90 |
| 5 | Confident | `git diff --cached --quiet` guard preserves idempotency (re-run = silent no-op) | Intake Assumption #6; git-pr-review's Rules mandate idempotency; same guard git-pr uses | S:85 R:80 A:90 D:85 |
| 6 | Tentative | Push failure is best-effort (report, do NOT STOP / do NOT `git reset` / do NOT fail the stage); commit half still mirrors git-pr | Intake Assumption #7 (Tentative) + the orchestrator prompt's explicit instruction to resolve toward best-effort for the push. git-pr-review's existing status writes are best-effort (`2>/dev/null \|\| true`) and Step 5.5 replies are best-effort; aborting a completed terminal-stage cycle over a transient push blip contradicts that ethos. The `fail`-path gate (Assumption #3) already prevents committing bad state, so a soft push is safe. Two valid readings remain (git-pr parity would STOP), hence Tentative. | S:55 R:75 A:62 D:50 |

6 assumptions (2 certain, 3 confident, 1 tentative).
