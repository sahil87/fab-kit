# Plan: Refresh memory indexes post-commit in /git-pr (fix index date drift)

**Change**: 260620-o203-git-pr-refresh-memory-index
**Intake**: `intake.md`

## Requirements

<!-- Derived from the intake. This is a SKILL-PROSE + SPEC-MIRROR change: no Go
     code, no new CLI command, no migration. Canonical source is
     src/kit/skills/git-pr.md; the SPEC mirror docs/specs/skills/SPEC-git-pr.md
     is constitution-required (Additional Constraints). -->

### /git-pr: Post-commit memory-index refresh (sub-step 3a-bis)

#### R1: New sub-step 3a-bis between 3a (Commit) and 3b (Push)
`src/kit/skills/git-pr.md` SHALL define a new sub-step **3a-bis: Refresh Memory Indexes** in `### Step 3: Execute Pipeline`, positioned BETWEEN `#### 3a. Commit (if has_uncommitted)` and `#### 3b. Push (if has_unpushed or just committed)`. This is the only pipeline position where `git log` can see the change's own content commit before the push.

- **GIVEN** the git-pr skill source
- **WHEN** Step 3 is documented
- **THEN** a `#### 3a-bis. Refresh Memory Indexes` sub-step appears immediately after 3a and immediately before 3b
- **AND** the prose matches the surrounding heading conventions and `✓` progress-line format

#### R2: Dual gating — `{has_fab}` AND 3a-just-committed
The 3a-bis sub-step SHALL be gated on BOTH conditions and skipped entirely otherwise: (a) `{has_fab}` (the Step 0 variable) is true, AND (b) step 3a just committed this invocation (the `has_uncommitted` path ran). It SHALL NOT run on the "already shipped" / no-change re-run paths where 3a did not commit.

- **GIVEN** a `/git-pr` invocation
- **WHEN** `{has_fab}` is false (standalone use outside a fab project) OR 3a did not commit this invocation
- **THEN** 3a-bis is a silent no-op (skipped entirely), leaving standalone `/git-pr` behavior unchanged

#### R3: Byte-stable regen + diff-guarded separate follow-up commit
When gating passes, 3a-bis SHALL run `fab memory-index` (byte-stable; writes only `docs/memory/` index + log files; no-op when nothing drifted). If `docs/memory/` changed (`git diff --quiet -- docs/memory` exits non-zero) it SHALL `git add docs/memory` and make a SEPARATE follow-up commit `git commit -m "docs: refresh memory indexes"`. It SHALL NOT use `--amend`. When nothing drifted (`git diff --quiet -- docs/memory` exits 0) it SHALL make no commit (the guard suppresses an empty commit — Constitution III idempotency).

- **GIVEN** gating passed and 3a-bis runs `fab memory-index`
- **WHEN** `docs/memory/` has changes after the regen
- **THEN** a separate `docs: refresh memory indexes` commit is made (never `--amend`)
- **AND WHEN** `docs/memory/` is unchanged after the regen, no commit is made

#### R4: Fail-fast on regen-or-commit failure, no torn state
If the regen OR the follow-up commit fails, 3a-bis SHALL report the error and STOP. The 3a content commit is already made and intact; a failed refresh degrades to a benign stale-date index recoverable by re-running `fab memory-index` — never a torn state.

- **GIVEN** the regen or follow-up commit fails
- **WHEN** 3a-bis is executing
- **THEN** the error is reported and the skill STOPs, with the 3a content commit left intact

#### R5: Progress-line output only on follow-up commit
3a-bis SHALL print `  ✓ commit — "docs: refresh memory indexes"` ONLY when a follow-up commit was actually made, matching git-pr's existing `✓ <step>` progress-line convention. No line is printed when nothing drifted or when the sub-step was skipped.

- **GIVEN** 3a-bis ran a follow-up commit
- **WHEN** the sub-step completes
- **THEN** the `✓ commit — "docs: refresh memory indexes"` line is printed
- **AND WHEN** no follow-up commit was made, no such line is printed

#### R6: Rationale note in the skill prose
3a-bis SHALL include a rationale note stating: this is the first moment `git log` reports the real commit date; the step lives in ship (not hydrate) because hydrate is entirely pre-commit; it is a silent no-op when `/git-pr` runs standalone outside a fab project (`{has_fab}` false), so general-purpose standalone use is unaffected.

- **GIVEN** the 3a-bis sub-step prose
- **WHEN** read by a future maintainer
- **THEN** the why (date-drift fix, ship-not-hydrate, standalone no-op) is stated inline

#### R7: No push inside 3a-bis — 3b pushes both commits
3a-bis SHALL NOT push on its own. Because 3a-bis is positioned BEFORE 3b and 3b's existing trigger is "if has_unpushed or just committed", a commit made in 3a-bis is "just committed" and is naturally pushed by 3b together with the 3a content commit.

- **GIVEN** 3a-bis made a follow-up commit
- **WHEN** Step 3b runs
- **THEN** 3b pushes both the 3a content commit and the 3a-bis index-refresh commit together (3a-bis itself performs no push)

#### R8: Key Properties "Idempotent?" row amended
The `| Idempotent? | ... |` row in `## Key Properties` of `src/kit/skills/git-pr.md` SHALL be amended to note that 3a-bis is gated on 3a-having-just-committed (a re-run on the no-commit path skips it), and that even if reached, `fab memory-index` is byte-stable and the `git diff --quiet -- docs/memory` guard suppresses an empty follow-up commit.

- **GIVEN** the Key Properties table
- **WHEN** the Idempotent? row is read
- **THEN** it documents the 3a-bis re-run/byte-stable/diff-guard properties

### Specs: git-pr SPEC mirror (constitution-required)

#### R9: 3a-bis node mirrored into the SPEC Flow tree
`docs/specs/skills/SPEC-git-pr.md` SHALL mirror the 3a-bis node into the `## Flow` tree, positioned BETWEEN the `3a. Commit` node and the `3b. Push` node, with the rationale (byte-stable regen, diff-guarded separate commit, no `--amend`, first moment git log knows the real date, ship-not-hydrate, `{has_fab}`-false no-op, fail→report+STOP with 3a intact).

- **GIVEN** the SPEC Flow tree
- **WHEN** the 3a → 3b transition is read
- **THEN** a `3a-bis. Refresh Memory Indexes` node appears between them with the rationale

#### R10: SPEC summary rationale paragraph
`docs/specs/skills/SPEC-git-pr.md` SHOULD add a short rationale paragraph to its summary section describing the 3a-bis behavior and date-drift-fix rationale, consistent with how the SPEC documents other git-pr hardening (g8st, w7dp).

- **GIVEN** the SPEC summary section
- **WHEN** a reader scans the hardening lineage
- **THEN** the 3a-bis date-drift fix is described alongside g8st / w7dp

### Non-Goals

- No Go code change (`cmd/fab`, `internal/`), no new/changed CLI command, no `_cli-fab.md` update — `fab memory-index` already exists and is unchanged; this only adds a new caller in skill prose.
- No migration (skills redeploy via `fab sync`; no user-data restructuring).
- Does not change `fab memory-index` behavior, the hydrate-stage regen at Step 5, or the refuse-before-regen guard (glwc).
- Does not touch `log.md` behavior (freeze-on-write / append-only; does not drift) — targets the `index.md` "Last Updated" drift only.
- Does not run `fab sync` and does not edit the gitignored deployed copy `.claude/skills/git-pr.md` (Constitution V) — apply edits only the canonical source.

### Design Decisions

1. **Post-commit / pre-push in ship (3a-bis), not in hydrate**: regen at the only pipeline position where `git log` sees the change's own content commit — *Why*: hydrate is entirely pre-commit, so no in-hydrate position can stamp the real date — *Rejected*: regen at end of hydrate.
2. **Separate follow-up commit, not `git commit --amend`**: keeps 3a's authored content commit intact and reviewable; squash collapses the pair on merge anyway — *Rejected*: `--amend` (rewrites an already-made commit).
3. **`{has_fab}` gate, following git-pr's existing conditional-fab pattern (Steps 0a/4a/4c)** — *Rejected*: unconditional regen (would couple standalone `/git-pr` to fab).
4. **`git diff --quiet -- docs/memory` guard before the follow-up commit**: suppresses an empty commit when nothing drifted (Constitution III) — `fab memory-index` is byte-stable, so a no-drift regen produces no diff.
5. **Fail → report + STOP, leaving the 3a content commit intact** — a failed refresh degrades to a benign stale-date index recoverable by re-running `fab memory-index`; never a torn state.

## Tasks

### Phase 1: Skill source — sub-step 3a-bis

- [x] T001 In `src/kit/skills/git-pr.md`, add `#### 3a-bis. Refresh Memory Indexes` between `#### 3a. Commit (if has_uncommitted)` and `#### 3b. Push (if has_unpushed or just committed)`, with the dual gating (`{has_fab}` AND 3a-just-committed), the byte-stable `fab memory-index` regen, the diff-guarded separate follow-up commit (no `--amend`), fail→report+STOP semantics, the `✓ commit` progress line gated on a follow-up commit, and the inline rationale note <!-- R1 R2 R3 R4 R5 R6 R7 -->

### Phase 2: Skill source — Key Properties

- [x] T002 In `src/kit/skills/git-pr.md`, amend the `| Idempotent? | ... |` row in `## Key Properties` to note 3a-bis is gated on 3a-just-committed (re-run on the no-commit path skips it) and is byte-stable + diff-guarded even if reached <!-- R8 -->

### Phase 3: SPEC mirror (constitution-required)

- [x] T003 In `docs/specs/skills/SPEC-git-pr.md`, mirror the 3a-bis node into the `## Flow` tree between the `3a. Commit` node and the `3b. Push` node, with the rationale <!-- R9 -->
- [x] T004 In `docs/specs/skills/SPEC-git-pr.md`, add a short rationale paragraph to the summary section describing 3a-bis and its date-drift-fix rationale, consistent with the g8st / w7dp hardening notes <!-- R10 -->

## Execution Order

- T001 then T002 (same file, but logically independent edits; do T001 first as it establishes the sub-step the Idempotent? note refers to)
- T003 then T004 (same file; tree node first, then summary paragraph)
- Phase 1/2 (skill) and Phase 3 (SPEC) are independent files but conceptually paired — the SPEC mirror should reflect the final skill prose

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `src/kit/skills/git-pr.md` has a `#### 3a-bis. Refresh Memory Indexes` sub-step positioned between 3a (Commit) and 3b (Push)
- [ ] A-002 R2: The 3a-bis sub-step documents dual gating — both `{has_fab}` and 3a-just-committed — and is skipped entirely otherwise (silent no-op for standalone / no-change re-run paths)
- [ ] A-003 R3: 3a-bis runs `fab memory-index`, then makes a SEPARATE `docs: refresh memory indexes` commit only when `git diff --quiet -- docs/memory` exits non-zero; never uses `--amend`; makes no commit when nothing drifted
- [ ] A-004 R4: 3a-bis specifies fail → report the error and STOP, with the 3a content commit left intact (no torn state)
- [ ] A-005 R5: 3a-bis prints `  ✓ commit — "docs: refresh memory indexes"` only when a follow-up commit was made
- [ ] A-006 R6: 3a-bis includes the rationale note (first moment git log sees the real date; ship-not-hydrate; standalone `{has_fab}`-false no-op)
- [ ] A-007 R8: The Key Properties Idempotent? row is amended with the 3a-bis gating / byte-stable / diff-guard note
- [ ] A-008 R9: `docs/specs/skills/SPEC-git-pr.md` Flow tree has the 3a-bis node between the 3a Commit node and the 3b Push node, with rationale
- [ ] A-009 R10: `docs/specs/skills/SPEC-git-pr.md` summary section has a 3a-bis rationale paragraph consistent with g8st / w7dp

### Behavioral Correctness

- [ ] A-010 R7: 3a-bis performs no push of its own; the SPEC/skill reflect that 3b ("if has_unpushed or just committed") pushes both the 3a content commit and the 3a-bis follow-up commit together

### Scenario Coverage

- [ ] A-011 R2: Standalone `/git-pr` (`{has_fab}` false) is documented as unaffected — 3a-bis is a silent no-op
- [ ] A-012 R3: The no-drift path (byte-stable regen, no diff) is documented to make no commit

### Code Quality

- [ ] A-013 Pattern consistency: New prose matches git-pr.md's heading conventions (`#### 3x.` sub-steps) and the `  ✓ <step> — ...` progress-line format; SPEC additions match the existing Flow-tree node style
- [ ] A-014 No unnecessary duplication: The change reuses the existing `{has_fab}` variable (Step 0) and the existing `fab memory-index` command — no new variable or command introduced
- [ ] A-015 Documentation accuracy: Skill prose and SPEC mirror are mutually consistent and accurately describe the implemented behavior (config.yaml checklist.extra_categories: documentation_accuracy)
- [ ] A-016 Cross-references: The SPEC mirror correctly reflects the skill's 3a-bis placement and gating; the canonical source (not the deployed copy) is the file edited (config.yaml checklist.extra_categories: cross_references)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- The deployed copy `.claude/skills/git-pr.md` is regenerated by `fab sync` (not apply's job) and is NOT edited here.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | 3b push-ordering: 3a-bis performs no push; 3b's existing trigger "if has_unpushed or just committed" naturally pushes both the 3a content commit and the 3a-bis follow-up commit. | Open Question #10 confirmed against the actual 3b text in `src/kit/skills/git-pr.md` line 180: `#### 3b. Push (if has_unpushed or just committed)`. A commit made in 3a-bis is "just committed", so 3b pushes it. Intake graded this Confident; verifying against the live trigger upgrades it to Certain. | S:98 R:85 A:95 D:92 |
| 2 | Certain | Place 3a-bis as a `#### 3a-bis. Refresh Memory Indexes` heading (mirroring the `#### 3a.` / `#### 3b.` sibling heading level) and use the leading-two-space `  ✓ commit — ...` progress-line format used by 3a/3b/3c. | The surrounding sub-steps use `#### 3x.` headings and `  ✓ <step> — ...` lines verbatim; matching them is the lowest-surprise choice and is required by the "match surrounding prose style" directive. | S:96 R:90 A:95 D:92 |
| 3 | Confident | In the SPEC Flow tree, render the 3a-bis node at the same indentation level as the `3a. Commit` / `3b. Push` nodes (`│  ├─ 3a-bis. ...`), following the intake's illustrative excerpt. | The intake supplies an explicit example block for the node; the SPEC tree already uses this `│  ├─` shape for sibling sub-steps. Minor formatting latitude only. | S:88 R:85 A:90 D:85 |

3 assumptions (2 certain, 1 confident, 0 tentative).
