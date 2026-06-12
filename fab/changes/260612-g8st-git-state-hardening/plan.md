# Plan: Git-State Hardening for Autonomous Skills

**Change**: 260612-g8st-git-state-hardening
**Intake**: `intake.md`

## Requirements

All skill edits target canonical sources in `src/kit/skills/` (never `.claude/skills/`). Every touched skill carries a same-PR `docs/specs/skills/SPEC-*.md` mirror update (constitution). The shared design stance: *autonomous git paths must verify state before mutating and report failures honestly.*

### Ship Pipeline: git-pr State Hardening

#### R1: Detached-HEAD STOP before the autonomous commit path
`/git-pr` MUST detect a detached HEAD in Step 1 (empty `git branch --show-current` output; confirmable via `git symbolic-ref -q HEAD` exiting 1) and STOP ahead of the Step 2 branch guard — before any commit or push — with: `Cannot ship from a detached HEAD — check out a branch first (run /git-branch).`

- **GIVEN** a repo in detached-HEAD state with uncommitted changes
- **WHEN** `/git-pr` runs
- **THEN** it stops with the detached-HEAD message before Step 3
- **AND** no commit is created and no refspec-less `git push -u origin` is ever emitted

#### R2: Scoped autonomous staging (expected-area guard)
`/git-pr` Step 3a MUST NOT run `git add -A`. It SHALL stage tracked changes with `git add -u`, enumerate untracked files via `git status --porcelain` (`??` lines), and apply the expected-area guard: untracked files inside the project's expected write areas (each `source_paths` entry from `fab/project/config.yaml`, plus `docs/` and `fab/`) are staged; any untracked file outside those areas causes a STOP that lists the offending files. When `config.yaml` is absent, the expected areas are `docs/` and `fab/` only.

- **GIVEN** a dirty tree containing an untracked `scratch.log` at the repo root and modified files under `src/`
- **WHEN** `/git-pr` reaches Step 3a
- **THEN** it stops, listing `scratch.log`, without committing anything
- **GIVEN** a dirty tree whose only untracked files live under `src/`, `docs/`, or `fab/`
- **WHEN** Step 3a runs
- **THEN** tracked modifications and the in-area untracked files are staged and committed

#### R3: Branch on PR state (OPEN/CLOSED/MERGED)
`/git-pr` MUST read the `state` field already fetched in Step 1 (`gh pr view --json number,state,url`) and branch explicitly: **OPEN** → existing behavior (short-circuit "already shipped" when nothing else to do); **CLOSED** → treat as no PR for creation purposes and proceed to Step 3c to create a fresh PR; **MERGED** → STOP at Step 3 entry with new-change/branch guidance, before any commit/push.

- **GIVEN** a branch whose PR is CLOSED and new local commits
- **WHEN** `/git-pr` runs
- **THEN** it commits/pushes as needed and creates a fresh PR via `gh pr create`
- **GIVEN** a branch whose PR is MERGED
- **WHEN** `/git-pr` runs
- **THEN** it stops with guidance that new work needs a new change/branch, performing no git mutations

#### R4: Default-branch resolution in the guard and nudge
`/git-pr` MUST resolve the actual default branch — `git symbolic-ref --short refs/remotes/origin/HEAD` (strip `origin/`), falling back to `gh repo view --json defaultBranchRef -q .defaultBranchRef.name`, falling back to treating literal `main`/`master` as default — and use the resolved name in the Step 2 guard and the Step 1b mismatch nudge.

- **GIVEN** a repo whose default branch is `develop`
- **WHEN** `/git-pr` runs on `develop`
- **THEN** the Step 2 guard stops PR creation from the default branch
- **GIVEN** both resolution commands fail
- **WHEN** Step 2 evaluates
- **THEN** the guard falls back to literal `main`/`master`

### Review-PR: git-pr-review Failure Honesty

#### R5: Split commit-failure and push-failure handling
`/git-pr-review` Step 5 MUST handle the two failure modes separately. Commit fails → `git reset` to clear staged changes, print the error, STOP (no partial state — true here). Push fails → KEEP the commit, print the push error plus documented recovery (`git pull --rebase && git push`, then re-run `/git-pr-review`), and STOP **without posting replies** — no "Fixed" reply may cite an unpushed SHA. Step 6's exception text MUST be updated to match the split semantics.

- **GIVEN** Step 5's `git push` is rejected (e.g., non-fast-forward)
- **WHEN** the failure is handled
- **THEN** the fix commit remains on the local branch, recovery guidance is printed, and Step 5.5 replies are NOT posted

#### R6: Unpushed-commit re-run gate
Before Step 5 may declare "No changes needed", `/git-pr-review` MUST check for unpushed commits (`git log --oneline @{u}..HEAD`, treating a missing upstream as unpushed). If unpushed commits exist, push them first, then proceed to Step 5.5 replies citing the now-pushed SHA.

- **GIVEN** a prior run whose commit succeeded but whose push failed
- **WHEN** `/git-pr-review` re-runs and Step 5 finds a clean tree
- **THEN** the unpushed commit is detected and pushed, and replies cite a SHA that exists on the remote
- **AND** the fix is never permanently stranded

### Branch Management: git-branch / fab-new Step 11 Hardening

#### R7: Multi-match disambiguation STOP
When an explicit `/git-branch` argument fails `fab change resolve`, the skill MUST distinguish the failure by stderr (both exit 1): a multi-match (`Multiple changes match "x": <list>.`) → STOP and show the candidate list, creating no branch; a true no-match (`No change matches "x".`) → existing standalone fallback. No Go change.

- **GIVEN** an argument matching two change folders
- **WHEN** `/git-branch <arg>` runs
- **THEN** it stops listing the candidates and creates no branch
- **GIVEN** an argument matching no change
- **WHEN** `/git-branch <arg>` runs
- **THEN** the existing standalone fallback creates the literal branch

#### R8: Remote-only branches checked out with `--track`
Before creating a branch, `/git-branch` Step 4 (and the fab-new Step 11 twin table) MUST check `git rev-parse --verify "origin/{branch_name}"`; a branch existing only on origin is checked out via `git checkout --track "origin/{branch_name}"` instead of being recreated as a divergent local with `checkout -b`.

- **GIVEN** the target branch exists on `origin` but not locally
- **WHEN** `/git-branch` runs
- **THEN** the local branch is created tracking `origin/{branch_name}` with the remote's HEAD, not a divergent local

#### R9: Dirty-tree warning at branch creation
`/fab-new` Step 11 and `/git-branch` Step 4 (twin tables, keep-in-sync comment) MUST check `git status --porcelain` before the branch operation; when the tree is dirty and the action creates or renames a branch (`checkout -b` / `branch -m` rows), a non-blocking warning is appended to the report line: `Branch: {name} (created) — note: {N} uncommitted change(s) carried over from {old_branch}`. Warn, never stash-prompt (autonomy posture).

- **GIVEN** uncommitted changes and a `checkout -b` row match
- **WHEN** the branch is created
- **THEN** the report line carries the carried-over note and execution continues unblocked

#### R10: Same-change rename gap closed
In both twin tables, a local-only current branch whose `fab change resolve` result is the SAME change being branched (e.g., a worktree placeholder named with the change's own ID) MUST take the rename path (row 4 outcome), not fall through to row 5; only a different change's branch triggers the leave-intact `checkout -b` path.

- **GIVEN** current branch `g8st` (local-only) resolving to change `260612-g8st-git-state-hardening`, the change being branched
- **WHEN** the table evaluates
- **THEN** row 4 renames the branch to the full change name (`renamed from g8st`), leaving no stray placeholder branch

### Runtime: Operator Default-Branch Resolution + Fetch

#### R11: Operator cherry-pick/rebase resolve the default branch and fetch first
`/fab-operator` MUST replace its three hardcoded `origin/main` sites — the dependency cherry-pick range, the "Why `origin/main` as base" rationale block, and the `--merge-on-complete` rebase — with the resolved default branch (same chain as R4), and MUST run `git fetch origin` before the cherry-pick and rebase sequences so the base is current.

- **GIVEN** a target repo whose default branch is `master` and a stale local `origin/HEAD`
- **WHEN** the operator resolves a same-repo dependency
- **THEN** it fetches origin, resolves the default branch, and cherry-picks `origin/{default_branch}..<dep-branch>`
- **AND** `--merge-on-complete` rebases onto the resolved `origin/{default_branch}` after fetching

### Archive: Honest Claims + Backlog-Mark Recovery

#### R12: fab-archive dirty-tree disclosure
`fab-archive.md` MUST qualify its "safe to re-run" claim with explicit dirty-tree disclosure: archive/restore move tracked files and edit `fab/backlog.md` with no commit step — they leave uncommitted moves and backlog edits for the caller to commit. No autonomous commit step is added.

- **GIVEN** a reader of the skill's Purpose / Key Properties
- **WHEN** they assess re-run safety
- **THEN** the dirty tree the operation leaves behind is disclosed, not contradicted by the safety claim

#### R13: Backlog mark recovered on re-archive (Go)
`ArchiveWithBacklog` (`src/go/fab/internal/archive/archive.go`) MUST still attempt `backlog.MarkDone` when `Archive` returns `ErrAlreadyArchived`, so a re-run recovers a previously-failed backlog mark (`MarkDone` is already idempotent — `already` when marked). Exit-code semantics are untouched: `ErrAlreadyArchived` propagates unchanged and callers' soft-skip paths behave as today (the k4ge seam). A unit test covers the recovery; the matching `src/kit/skills/_cli-fab.md` row is updated (constitution).

- **GIVEN** a prior archive run that moved the folder but failed the backlog mark
- **WHEN** `ArchiveWithBacklog` runs again and `Archive` yields `ErrAlreadyArchived`
- **THEN** `backlog.MarkDone` is attempted and the backlog item flips to `[x]`
- **AND** the returned error is still `ErrAlreadyArchived` (soft-skip exit semantics unchanged)

### Documentation: SPEC Mirrors

#### R14: Six SPEC mirrors updated in the same PR
`docs/specs/skills/SPEC-{git-pr,git-pr-review,git-branch,fab-new,fab-archive,fab-operator}.md` MUST each be updated to reflect the behavior changes landed in its skill — mirroring only what changed.

- **GIVEN** any touched skill file
- **WHEN** its SPEC mirror is read after this change
- **THEN** the mirror's flow/summary matches the new behavior (no stale `git add -A`, unsplit failure handling, hardcoded `origin/main`, etc.)

### Non-Goals

- No autonomous commit step in `/fab-archive` — that would change blast radius and overlap `/git-pr`'s commit ownership (honest disclosure instead)
- No archive exit-code changes — that seam (including the CLI-unreachable soft skip) belongs to change k4ge
- No Go change for the git-branch disambiguation — stderr text is sufficient and verified
- No stash-prompt at branch creation — blocking prompts violate the no-questions/orchestrated autonomy posture
- No w7dp findings — w7dp branches after g8st merges (g8st owns the colliding sections now)
- No edits to `.claude/skills/` deployed copies

### Design Decisions

1. **Expected-area guard over hard stop / disclosure** (clarified): `git add -u` + untracked included only inside `source_paths`/`docs/`/`fab/` — *Rejected*: stop-on-any-untracked (blocks every change that legitimately creates files); keep-`add -A`-and-disclose (the sweep finding would remain live).
2. **Per-state PR policy** (clarified): OPEN → short-circuit; CLOSED → fresh PR (shipping intent is explicit — the user/orchestrator just invoked `/git-pr`); MERGED → STOP — *Rejected*: reopen-on-CLOSED; stop-on-both.
3. **Go backlog-recovery lands in g8st** (clarified): on `ErrAlreadyArchived`, still attempt `MarkDone`; exit codes untouched — *Rejected*: defer to k4ge.
4. **Warn, not stash-prompt** (clarified): the dirty-tree caveat is a non-blocking report-line note.
5. **Default-branch chain** (clarified): `git symbolic-ref --short refs/remotes/origin/HEAD` → `gh repo view --json defaultBranchRef` → literal `main`/`master`; operator adds `git fetch origin` before cherry-pick/rebase.

## Tasks

### Phase 1: git-pr Hardening

- [x] T001 Add detached-HEAD detection to `src/kit/skills/git-pr.md` Step 1 (empty `git branch --show-current`) and the STOP ahead of the Step 2 guard <!-- R1 -->
- [x] T002 Add default-branch resolution to `src/kit/skills/git-pr.md` Step 1 and use it in the Step 2 guard + Step 1b nudge <!-- R4 --> <!-- rework: skip the Step 1b nudge when the current branch is empty (detached HEAD) — today it prints "Note: branch '' doesn't match…" before the detached-HEAD STOP -->
- [x] T003 Replace `git add -A` in `src/kit/skills/git-pr.md` Step 3a with `git add -u` + the expected-area untracked guard (STOP with file list) <!-- R2 --> <!-- rework: swap order — the expected-area guard must evaluate and STOP before `git add -u` stages anything, so the STOP path leaves no staged index (verify-before-mutate) -->
- [x] T004 Branch `src/kit/skills/git-pr.md` on PR state: capture `state` in Step 1; MERGED STOP at Step 3 entry; CLOSED → fresh PR in 3c; OPEN keeps the short-circuit <!-- R3 --> <!-- rework: declare {number} and {url} as named values in Step 1's "Determine:" list (the MERGED STOP interpolates both); drop the redundant CLOSED mention in the "all other cases" line right after the dedicated CLOSED paragraph -->

### Phase 2: git-pr-review Failure Honesty

- [x] T005 Split `src/kit/skills/git-pr-review.md` Step 5 failure handling (commit-fail → reset+STOP; push-fail → keep commit + recovery + STOP without replies) and update Step 6's exception text <!-- R5 -->
- [x] T006 Add the unpushed-commit re-run gate to `src/kit/skills/git-pr-review.md` Step 5 (check `@{u}..HEAD`, no-upstream = unpushed; push then proceed to replies) <!-- R6 -->

### Phase 3: Branch-Creation Twins

- [x] T007 Add the multi-match disambiguation STOP to `src/kit/skills/git-branch.md` Step 2 (stderr-keyed; no-match keeps standalone fallback) <!-- R7 -->
- [x] T008 Add the remote-only `--track` checkout to `src/kit/skills/git-branch.md` Step 4 <!-- R8 -->
- [x] T009 Add the dirty-tree warning and the same-change rename fix to `src/kit/skills/git-branch.md` Step 4 (+ report-string updates) <!-- R9, R10 -->
- [x] T010 Mirror the same three edits into `src/kit/skills/fab-new.md` Step 11 table (remote-only row, dirty-tree note, same-change rename condition; keep-in-sync comments updated) <!-- R8, R9, R10 --> <!-- rework: fab-new-side only — exclude the change's own `fab/changes/{name}/` artifacts from the dirty-tree porcelain count; intake.md/.status.yaml always exist by Step 11, so the note currently fires on every run (constant noise). git-branch side is correct as-is -->

### Phase 4: Operator

- [x] T011 In `src/kit/skills/fab-operator.md`: resolve the default branch + `git fetch origin` before the same-repo cherry-pick sequence; replace `origin/main` in the cherry-pick range, the rationale block, and both `--merge-on-complete` rebase mentions <!-- R11 --> <!-- rework: the step-0 literal-fallback comment claims "literal main/master fallback" but the code assigns only default_branch=main — make the comment match the behavior (single literal main fallback) or probe origin/master before falling back -->

### Phase 5: Archive

- [x] T012 Qualify the re-run-safety claim in `src/kit/skills/fab-archive.md` with the dirty-tree disclosure (Purpose + Key Properties, both modes) <!-- R12 -->
- [x] T013 In `src/go/fab/internal/archive/archive.go` `ArchiveWithBacklog`: on `ErrAlreadyArchived`, re-resolve the folder and attempt `backlog.MarkDone` best-effort; error propagation unchanged <!-- R13 -->
- [x] T014 Add `TestArchiveWithBacklog_ReRunRecoversBacklogMark` to `src/go/fab/internal/archive/archive_test.go`; run `go test ./...` scoped to the package <!-- R13 --> <!-- rework: add a comment explaining why the test reconstructs the source folder to reach ErrAlreadyArchived — the natural archive-ok/mark-failed path is resolution-blocked until k4ge lands -->
- [x] T015 Update the `fab change archive` row in `src/kit/skills/_cli-fab.md` to document the soft-skip backlog re-attempt <!-- R13 -->

### Phase 6: Mirrors & Verification

- [x] T016 Update the six SPEC mirrors `docs/specs/skills/SPEC-{git-pr,git-pr-review,git-branch,fab-new,fab-archive,fab-operator}.md` to reflect each behavior change <!-- R14 --> <!-- rework: MUST-FIX — SPEC-git-pr.md:9 "Re-run contract" still says the already-shipped path requires "PR exists"; the skill now requires an OPEN PR (git-pr.md:147/:330) and the mirror's own State-hardening paragraph already says so. Also re-mirror the T003/T004/T010/T011 rework edits where they change documented behavior -->
- [x] T017 Run the full Go test suite (`go test ./...` from `src/go/fab`) and fix any failures <!-- R13 -->

## Execution Order

- T001–T004 touch the same file (git-pr.md) — sequential within Phase 1
- T010 depends on T008–T009 (mirrors their final wording)
- T014 depends on T013; T015 documents T013's behavior
- T016 depends on all prior skill edits; T017 runs last (full Go suite)

## Acceptance

### Functional Completeness

- [x] A-001 R1: git-pr.md detects detached HEAD (empty `git branch --show-current`) and STOPs with the prescribed message before any commit/push
- [x] A-002 R2: git-pr.md Step 3a stages via `git add -u` + expected-area guard; `git add -A` no longer appears in the skill
- [x] A-003 R3: git-pr.md reads the fetched PR `state` and implements OPEN short-circuit / CLOSED fresh-PR / MERGED STOP
- [x] A-004 R4: git-pr.md resolves the default branch via the symbolic-ref → gh → literal fallback chain and uses it in Step 2 and Step 1b
- [x] A-005 R5: git-pr-review.md Step 5 splits commit-fail (reset+STOP) from push-fail (keep commit, recovery guidance, STOP without replies); Step 6 exception text matches
- [x] A-006 R6: git-pr-review.md gates "No changes needed" on an unpushed-commit check (`@{u}..HEAD`, no-upstream = unpushed) and pushes before replying
- [x] A-007 R7: git-branch.md Step 2 STOPs with the candidate list on multi-match stderr; no-match keeps the standalone fallback
- [x] A-008 R8: git-branch.md Step 4 and fab-new.md Step 11 check `origin/{branch_name}` and use `git checkout --track` for remote-only branches
- [x] A-009 R9: both twin tables append the non-blocking dirty-tree note to create/rename report lines
- [x] A-010 R10: both twin tables rename when the current local-only branch resolves to the same change; only a different change's branch takes the leave-intact path
- [x] A-011 R11: fab-operator.md fetches origin and uses the resolved default branch at the cherry-pick range, rationale block, and both `--merge-on-complete` rebase sites
- [x] A-012 R12: fab-archive.md's re-run claim discloses the uncommitted moves + backlog edits; no autonomous commit step added
- [x] A-013 R13: `ArchiveWithBacklog` attempts `backlog.MarkDone` on `ErrAlreadyArchived`; new unit test passes; `_cli-fab.md` row updated
- [x] A-014 R14: all six SPEC mirrors reflect the new behavior

### Behavioral Correctness

- [x] A-015 R3: a CLOSED PR no longer short-circuits `/git-pr` as "already shipped"
- [x] A-016 R5: the false "(no partial state)" claim for push failures is gone — the commit is kept and disclosed
- [x] A-017 R7: an ambiguous explicit argument no longer creates a junk standalone branch with a false "No matching change found" message
- [x] A-018 R13: re-archive no longer returns before the backlog mark — recovery is possible without exit-code changes

### Scenario Coverage

- [x] A-019 R6: the stranded-fix scenario (commit ok, push failed, re-run) ends with the commit pushed and replies citing a remote SHA
- [x] A-020 R10: the worktree-placeholder scenario (branch named with the change's own ID) renames instead of leaving a stray branch

### Edge Cases & Error Handling

- [x] A-021 R2: the untracked-file STOP lists the offending paths; absent `config.yaml` degrades to `docs/` + `fab/` areas
- [x] A-022 R4: both resolution commands failing falls back to literal `main`/`master` (guard never silently passes)
- [x] A-023 R11: `git fetch origin` precedes both the cherry-pick sequence and the `--merge-on-complete` rebase

### Code Quality

- [x] A-024 Pattern consistency: skill edits preserve each file's section numbering, STOP-message style, table formats, and report-line conventions
- [x] A-025 No unnecessary duplication: the twin tables stay in sync via the existing keep-in-sync comments; default-branch chain stated once per consuming skill
- [x] A-026 Go change follows the package's existing error-wrapping and best-effort comment style; no exported-API changes
- [x] A-027 Test-alongside: the Go fix ships with its unit test in the same package

### Documentation Accuracy

- [x] A-028: keep-in-sync comments and case counts in fab-new.md/git-branch.md remain truthful after the table edits
- [x] A-029: fab-archive idempotency claims (skill + `_cli-fab.md`) match actual command behavior post-fix

### Cross References

- [x] A-030: SPEC mirrors and `_cli-fab.md` cross-references (e.g., Step 0b/3c names cited by `prmeta.go`, twin-table pointers) remain valid; no stale references to removed behavior

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — every superseded behavior was replaced in place, leaving nothing orphaned: `git add -A` → scoped `git add -u` + expected-area guard (git-pr.md Step 3a), the literal `main`/`master` guard → resolved `{default_branch}` with the literal check retained as the documented fallback (still load-bearing, not dead), the unconditional standalone-fallback message → stderr-keyed multi-match/no-match branch (git-branch.md Step 2), the unsplit commit/push failure line → split handling (git-pr-review.md Step 5), and the hardcoded `origin/main` operator sites → resolved-base + fetch. The previously fetched-but-never-read `gh pr view` `state`/`number` fields are now consumed (the change removed latent redundancy rather than creating any). No Go code, skill prose, or SPEC text became unused. Nothing to delete. *(Re-verified at cycle-1 re-review.)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Same-change rename-gap fix is in scope as part of item 7's table edit (row 4 renames when resolve returns the change being branched) | Explicitly directed by the apply dispatch; closes a real gap the worktree-placeholder flow hits | S:90 R:85 A:85 D:85 |
| 2 | Confident | Remote-only `--track` row mirrored into fab-new Step 11, not just git-branch Step 4 | The twins' keep-in-sync comments mandate identical tables; leaving them divergent would make the comment false (documentation_accuracy) | S:70 R:85 A:80 D:70 |
| 3 | Confident | Default-branch resolution inlined in each consuming skill (git-pr Step 1, operator Dependency Resolution) — no new `_` helper file | Two consumers, three lines; a helper would add loading cost for every other skill | S:75 R:80 A:75 D:70 |
| 4 | Confident | Without `config.yaml`, expected write areas degrade to `docs/` + `fab/` (conservative STOP elsewhere) | Matches the verify-before-mutate stance; the STOP message tells the user exactly what to stage | S:55 R:80 A:65 D:60 |
| 5 | Confident | Dirty-tree note scoped to create/rename rows (`checkout -b`/`branch -m`); checkout/no-op rows excluded | The finding is "uncommitted work rides into the NEW change's branch"; plain checkout either carries visibly or fails loudly | S:70 R:85 A:75 D:65 |
| 6 | Confident | git-branch failure branching keys on the distinguishing stderr phrases (`Multiple changes match` / `No change matches`), not the exact `ERROR:` prefix | Robust to CLI prefix formatting; phrases verified in `resolve.go` | S:80 R:90 A:85 D:80 |
| 7 | Confident | MERGED STOP placed at Step 3 entry — before the nothing-to-do check and any commit/push | Prevents committing/pushing onto a merged branch; intake mandates STOP but not placement | S:75 R:85 A:80 D:75 |
| 8 | Confident | `ErrAlreadyArchived` recovery re-resolves the folder via `resolve.ToFolder` and treats `MarkDone` as best-effort (its error not propagated; `ErrAlreadyArchived` returned unchanged) | Keeps caller soft-skip semantics byte-identical (exit codes are k4ge's seam); resolution already succeeded inside `Archive`, so the re-resolve is deterministic | S:80 R:80 A:80 D:70 |

8 assumptions (1 certain, 7 confident, 0 tentative).
