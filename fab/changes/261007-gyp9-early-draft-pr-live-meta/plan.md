# Plan: Early draft PR + live, refreshable `## Meta` block

**Change**: 261007-gyp9-early-draft-pr-live-meta
**Intake**: `intake.md`

## Requirements

### PR Lifecycle: Early Open and Boundary Pushes

#### R1: The draft PR opens at apply exit
The pipeline SHALL create the draft GitHub PR when the `apply` stage completes, not at `ship`.
The open MUST reuse `/git-pr`'s existing commit → rebase → push → `gh pr create --draft` mechanics
and MUST NOT mutate `progress.ship` or any `stage_metrics` entry.

- **GIVEN** a change whose `apply` stage has just finished and whose branch has no PR
- **WHEN** the orchestrator reaches the apply→review boundary
- **THEN** the working tree is committed, the branch is rebased onto `origin/{base_branch}`, pushed, and a draft PR is created against `base_branch`
- **AND** `progress.ship` remains `pending` and the `ship` stage's own transitions are untouched

#### R2: Later stage boundaries push without rebasing
At the `review`-pass and `hydrate` boundaries the orchestrator SHALL commit and push any new work,
but MUST NOT rebase. Only the apply-exit open (R1) and the `ship` stage rebase.
Consequently those two boundaries use a plain `git push` and MUST NOT force-push.

- **GIVEN** an open draft PR created at apply exit
- **WHEN** the review-pass or hydrate boundary runs and the tree has changes
- **THEN** a plain `git push` lands them with no history rewrite and no `--force-with-lease`
- **AND** when the tree is clean the boundary performs no commit and no push

#### R3: Rework cycles do not push
Pushes SHALL fire at stage boundaries only. An auto-rework cycle inside `review` MUST NOT push;
the push happens once when review finally passes.

- **GIVEN** a review that fails and enters the auto-rework loop for 3 cycles
- **WHEN** the cycles run
- **THEN** no push occurs during them, and exactly one push occurs at the eventual review-pass boundary

#### R4: The pre-fetch lease discipline holds on every rebase-push
Every push that follows a rebase SHALL use `--force-with-lease="<branch>:<lease_oid>"` where
`lease_oid` was captured **before** `git fetch origin`. The bare `--force-with-lease` flag MUST NOT
be used. The remote-branch divergence guard SHALL run at every rebasing boundary.

- **GIVEN** the apply-exit open, and later the ship stage, each rebasing onto the base
- **WHEN** either captures its lease
- **THEN** the capture happens before its own `git fetch origin`, independently per boundary
- **AND** if `origin/<branch>` moved since this checkout last fetched, the boundary STOPs without rewriting

#### R5: Workers gain no git or `gh` obligation
Commit, push, and PR sync SHALL be performed by the orchestrator after it reads a dispatched
worker's result. Stage workers' obligations remain exactly those in `_preamble.md`
§ Dispatch-Prompt Obligations.

- **GIVEN** a full-lane run where apply/review/hydrate dispatch to pane or headless workers
- **WHEN** a worker completes
- **THEN** the orchestrator — not the worker — commits, pushes, and syncs the PR
- **AND** no dispatch prompt names `git push`, `gh`, or `fab pr-sync`

### Meta Block: Marker Contract and Splice

#### R6: The rendered Meta block is marker-delimited
`fab pr-meta` SHALL emit the block wrapped in `<!-- fab pr-meta:start -->` and
`<!-- fab pr-meta:end -->` lines. The markers are part of the rendered output, so the existing
byte-stable render goldens extend to cover them.

- **GIVEN** any change with fab context
- **WHEN** `fab pr-meta <change> --type <t>` runs
- **THEN** stdout begins with the start marker and ends with the end marker
- **AND** the content between them is byte-identical to today's rendered block

#### R7: Splicing is a pure replace-between-markers operation
A splice function SHALL take an existing PR body plus a freshly rendered block and return the body
with everything between the markers replaced, leaving every other byte — including `## Summary`,
`## Changes`, and any human edit — identical.

- **GIVEN** a body containing marked Meta plus hand-edited prose below it
- **WHEN** the splice runs with a newly rendered block
- **THEN** only the marked region changes and the hand-edited prose is byte-identical
- **AND** the function is pure: no I/O, no network, unit-testable without `gh`

#### R8: First refresh adopts an unmarked body
The splice SHALL handle three input shapes. A body with markers takes the plain replace path. A
body with a bare `## Meta` heading and no markers SHALL have that whole section — heading through
the next top-level `## ` heading, or end of body — replaced by the marker-wrapped block. A body
with no `## Meta` at all SHALL have the marker-wrapped block prepended ahead of the existing body.

- **GIVEN** a PR body created before this change, carrying a bare unmarked `## Meta` section
- **WHEN** the splice runs for the first time
- **THEN** the old section is replaced in place by the marker-wrapped block — never duplicated
- **AND** the `## Summary` heading that follows it, and everything after, is byte-identical

#### R9: All splice paths are idempotent
Running the splice twice with unchanged inputs SHALL produce a byte-identical body on the second
run (Constitution III). The sync SHALL compare the spliced body against the current body and
perform no `gh pr edit` when they are equal.

- **GIVEN** any of the three input shapes in R8
- **WHEN** the splice runs twice with the same rendered block
- **THEN** the second output equals the first byte-for-byte
- **AND** the sync issues no `gh pr edit` call on the second run

### CLI: The PR-Sync Command

#### R10: `fab pr-sync` owns render + splice + apply
A new command `fab pr-sync <change> --type <type> [--issues <ids>]` SHALL render the Meta block,
read the PR's current body, splice, compare, and apply via `gh pr edit --body-file -` only when the
body changed. It SHALL exit non-zero without side effects when there is no fab context or no open
PR on the branch, mirroring `fab pr-meta`'s graceful-degradation contract.

- **GIVEN** a change with an open draft PR whose Meta block is stale
- **WHEN** `fab pr-sync <change> --type feat` runs
- **THEN** the PR body's Meta region is refreshed and the command reports what it did
- **AND** a second immediate run reports a no-op and issues no edit

#### R11: The CLI reference and tests ship with the command
The owning CLI reference partial `src/kit/skills/_cli-fab.md` SHALL document `fab pr-sync`
alongside `fab pr-meta`, and Go tests SHALL ship in the same change (Constitution Additional
Constraints).

- **GIVEN** the new command signature
- **WHEN** the change is reviewed
- **THEN** `_cli-fab.md` documents it and `prmeta` package tests cover render-with-markers, all three splice paths, and idempotency

### Orchestration: Boundary Wiring

#### R12: One shared boundary procedure, pointed at by every driver
`_pipeline.md` SHALL own a single **PR Boundary Procedure** defining the open and push/sync actions.
`fab-ff.md`, `fab-fff.md`, and `fab-continue.md` SHALL point at it rather than restating it
(`fab/project/code-quality.md` § Anti-Patterns: state a rule or point at its owner, never both).

- **GIVEN** the four orchestrator call sites
- **WHEN** the boundary behavior changes later
- **THEN** exactly one file needs editing
- **AND** no driver file restates the commit/push/sync mechanics

#### R13: Both lanes and both drivers get the behavior
The procedure SHALL run identically in the light lane (inline) and the full lane (after reading a
worker result). `/fab-ff` (terminal `hydrate`) SHALL open the PR at apply exit and push at the
review and hydrate boundaries, ending with an open draft PR that a later `/git-pr` finalizes; its
Purpose line SHALL be rewritten to stop claiming it runs no PR steps.

- **GIVEN** a `/fab-ff` run in either lane
- **WHEN** it completes its hydrate terminal
- **THEN** an open draft PR exists with a current Meta block
- **AND** `fab-ff.md`'s Purpose no longer reads "stopping before PR stages"

#### R14: `/git-pr` finalizes rather than creates
With a PR already open, `/git-pr` at the ship stage SHALL rebase, push, sync Meta, and finalize
(`fab status add-pr`, `finish ship`). Its Step 3d retrofit SHALL delegate to the splice so it
updates a stale block instead of skipping on presence.

- **GIVEN** ship runs on a branch whose PR opened at apply exit
- **WHEN** `/git-pr` reaches 3c/3d
- **THEN** it takes the existing-OPEN-PR path, syncs Meta, and records the PR URL
- **AND** a `/git-pr` run on a branch with no PR still creates one, exactly as today

### Memory Index: Refresh Reconciliation

#### R15: The memory-index refresh runs at the hydrate boundary
`fab docs-index docs/memory` SHALL run at the hydrate boundary, where `docs/memory/` is actually
written. The existing ship-time run in `/git-pr` Step 3a-bis SHALL be retained as an idempotent
backstop. Both sites MUST preserve the separate-commit discipline (never `--amend`), the
suppress-empty-commit guard, and the never-hand-merge rule for generated index/log files.

- **GIVEN** a hydrate stage that rewrote `docs/memory/` topic files
- **WHEN** the hydrate boundary runs
- **THEN** indexes regenerate and land in a separate `docs: refresh memory indexes` commit
- **AND** when nothing drifted, no commit is made

### Non-Goals

- The `review-pr` stage, `/git-pr-review`, and the Copilot request/poll machinery — a separate follow-on change owns them. Nothing here may remove or pre-empt them.
- Suppressing CI on intermediate pushes — intermediate failures are deliberate signal.
- Re-authoring `## Summary` / `## Changes` on refresh — they are written once at PR create.

### Design Decisions

#### PR-sync lands as a Go subcommand that owns the `gh` call
**Decision**: Add `fab pr-sync <change> --type <t> [--issues]` which renders, reads the body, splices, compares, and applies via `gh pr edit`. The splice itself is a pure exported function in `internal/prmeta`, unit-tested without network.
**Why**: Five call sites (`_pipeline.md` ×3 boundaries, `git-pr.md` 3d, `fab-continue.md`) each become one line. A skill-layer dance of `gh pr view` → `fab pr-meta --splice` → compare → `gh pr edit` would be four lines restated five times — exactly the drift mechanism `code-quality.md` § Anti-Patterns names. `internal/prmeta` already shells to `gh` for owner/repo and degrades gracefully when it is absent, so the dependency is not new.
**Rejected**: A `/git-pr --sync-meta` mode (a full ship pipeline whose contract a metadata-only mode strains); a new standalone skill (a third skill for a mechanical splice, more prose to keep in sync); Go-splices-only with `gh` left in the skill layer (keeps `gh` where it is, but duplicates the dance at five sites).
*Introduced by*: 261007-gyp9-early-draft-pr-live-meta

#### Rebase once at open and once at ship, not at every boundary
**Decision**: The apply-exit open rebases onto `origin/{base_branch}` and force-pushes with an explicit pre-fetch lease. The review and hydrate boundaries do neither — they commit and plain-push. The ship stage keeps its existing rebase as the final freshening.
**Why**: Rebasing at all four boundaries would rewrite history four times, force-push four times, and re-trigger CI on each rewrite while multiplying conflict exposure. `git-pr.md` 3a-ter's stated rationale — put the diff on the current base and surface conflicts to the agent that still has author context — is satisfied by rebasing once at the open; ship's rebase then freshens against anything that landed on the base during the run.
**Rejected**: Rebase at every boundary (maximum freshness, but four force-pushes and four conflict windows for one change); never rebase until ship (leaves the early CI run measuring a stale base, which is the signal the early PR exists to buy).
*Introduced by*: 261007-gyp9-early-draft-pr-live-meta

#### The apply-exit open is a non-stage orchestrator step
**Decision**: Opening the PR at apply exit does not touch `progress.ship`, `stage_metrics`, or the stage order. The `ship` stage keeps running `/git-pr` verbatim and finalizes the already-open PR.
**Why**: The decision taken was *when the PR opens*, not *where the ship stage lives*. Re-homing ship's transitions would cascade into `.status.yaml` semantics, the `pr-meta` Pipeline line, the operator's PR tracking, and the archive/done predicates — a large blast radius for no stated benefit.
**Rejected**: Re-scoping `ship` to apply exit (would make the Pipeline line and operator tracking lie about where the change is); adding a new stage for the open (a schema migration for a step that needs no state).
*Introduced by*: 261007-gyp9-early-draft-pr-live-meta

#### The splice is written fresh, not reused from `memoryindex`
**Decision**: Implement the marker constants and splice in `internal/prmeta`, following `internal/memoryindex/adoption.go`'s *convention* (HTML-comment marker pair, adopt-on-first-run) without calling its functions.
**Why**: `adoptNavigation` merges table rows with per-row seeding, loss detection, and tombstone handling — it is row-semantic, not span-semantic. The Meta splice is a flat span replacement. Reusing it would mean bending a row merger into a span replacer; the shared asset here is the convention, not the code.
**Rejected**: Exporting a generic splice from `memoryindex` for both callers (the two operations genuinely differ; a shared abstraction would have to carry row semantics the Meta block does not have).
*Introduced by*: 261007-gyp9-early-draft-pr-live-meta

## Tasks

### Phase 1: Go — Render and Splice

- [x] T001 Add `metaStart`/`metaEnd` marker constants and wrap `Render`'s output in them in `src/go/fab/internal/prmeta/prmeta.go` <!-- R6 -->
- [x] T002 Add an exported pure `Splice(body, rendered string) string` to `src/go/fab/internal/prmeta/` implementing the three input shapes (marked → replace; unmarked `## Meta` → span-replace to next top-level `##`; absent → prepend) <!-- R7 R8 -->
- [x] T003 Extend `src/go/fab/internal/prmeta/prmeta_test.go` — marker-inclusive render goldens, one case per splice path, idempotency (splice twice = byte-identical), and a case asserting `## Summary`/human prose survive byte-for-byte <!-- R6 R7 R8 R9 -->

### Phase 2: Go — The Command

- [x] T004 Add `src/go/fab/cmd/fab/pr_sync.go`: `fab pr-sync <change> --type <t> [--issues]` — gather, render, `gh pr view --json body`, splice, compare, `gh pr edit --body-file -` only on change; non-zero with no side effects when fab context or an open PR is missing <!-- R10 -->
- [x] T005 Register the command and add `src/go/fab/cmd/fab/pr_sync_test.go` covering flag wiring, the no-fab-context exit, and the no-op-on-equal-body path <!-- R10 R11 -->
- [x] T006 Run `gofmt -w` over every changed `.go` file, then `go test ./src/go/fab/internal/prmeta/... ./src/go/fab/cmd/fab/...` <!-- R11 -->

### Phase 3: Skill Wiring

- [x] T007 Add the **PR Boundary Procedure** section to `src/kit/skills/_pipeline.md` — the apply-exit open (commit, rebase, lease-push, create draft, sync) and the later-boundary action (commit, plain push, sync); state the no-rebase and no-push-during-rework rules here as the owner <!-- R1 R2 R3 R12 -->
- [x] T008 Wire the procedure into `_pipeline.md` Steps 1–3: apply exit → open; review-pass → push+sync; hydrate → push+sync; note it runs inline in the light lane and after result-read in the full lane, and that workers gain no obligation <!-- R5 R13 -->
- [x] T009 Reconcile the memory-index refresh: run `fab docs-index docs/memory` at the hydrate boundary in `_pipeline.md`, preserving the separate-commit, empty-commit-guard, and never-hand-merge rules; keep `git-pr.md` 3a-bis as the backstop <!-- R15 -->
- [x] T010 Update `src/kit/skills/git-pr.md`: 3c takes the existing-OPEN-PR path normally, and 3d delegates to `fab pr-sync` so it refreshes a stale block instead of skipping on `## Meta` presence <!-- R14 -->
- [x] T011 [P] Update `src/kit/skills/fab-fff.md` — point Steps 4–5 at the boundary procedure; ship now finalizes an open PR <!-- R12 R13 -->
- [x] T012 [P] Rewrite `src/kit/skills/fab-ff.md`'s Purpose and `{terminal}` framing — it opens the PR at apply exit and pushes at review/hydrate, ending with an open draft PR; it no longer "stops before PR stages" <!-- R13 -->
- [x] T013 [P] Update `src/kit/skills/fab-continue.md` — the single-stage advance path runs the boundary procedure for the stage it advances <!-- R12 R13 -->
- [x] T014 Update `src/kit/skills/_cli-fab.md` — document `fab pr-sync` in the section owning `fab pr-meta` <!-- R11 -->

### Phase 4: Sweep and Docs

- [x] T015 Update `src/kit/skills/fab-adopt.md` — the "ship-stage Meta retrofit" it relies on is now a refresh via `fab pr-sync` <!-- R14 -->
- [x] T016 [P] Sweep `docs/specs/skills.md`, `docs/specs/architecture.md`, `docs/specs/glossary.md` for "PR is created at ship", the write-once retrofit, and the `fab-ff` no-PR claim <!-- R12 R13 -->
- [x] T017 Grep repo-wide for `## Meta`, `pr-meta`, "retrofit", "created at ship", "Step 3d" and fix every surviving occurrence in the sweep class <!-- R12 -->
- [x] T018 Verify Constitution V compliance — no `src/kit/**` file added here cites `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` as an authority; run the Go guard test <!-- R11 -->

## Execution Order

- T001 → T002 → T003 (markers before splice before their tests)
- T002 blocks T004 (the command calls `Splice`)
- T004 → T005 → T006
- T007 blocks T008–T013 (the owner section must exist before anything points at it)
- T011, T012, T013 are independent of each other
- T016 is independent; T017 runs last in the sweep so it catches residue from T007–T016

## Acceptance

### Functional Completeness

- [x] A-001 R1: A draft PR exists against `base_branch` when `apply` finishes, with `progress.ship` still `pending` and `stage_metrics` untouched
- [x] A-002 R6: `fab pr-meta <change> --type <t>` emits start and end markers around content byte-identical to the pre-change render
- [x] A-003 R7: `Splice` is an exported pure function with no I/O or network calls
- [x] A-004 R10: `fab pr-sync` refreshes a stale Meta block and reports a no-op on an unchanged second run
- [x] A-005 R11: `_cli-fab.md` documents `fab pr-sync`, and Go tests ship for render-with-markers, all three splice paths, and idempotency
- [x] A-006 R12: `_pipeline.md` carries the PR Boundary Procedure and no driver file restates its mechanics
- [x] A-007 R15: `fab docs-index docs/memory` runs at the hydrate boundary and `git-pr.md` 3a-bis is retained

### Behavioral Correctness

- [x] A-008 R2: The review-pass and hydrate boundaries push without rebasing and without `--force-with-lease`
- [x] A-009 R3: A 3-cycle rework run pushes exactly once, at the review-pass boundary
- [x] A-010 R4: Every rebasing boundary captures its lease OID before its own `git fetch origin`; the bare `--force-with-lease` flag appears nowhere
- [x] A-011 R13: `fab-ff.md`'s Purpose no longer claims it runs no PR steps, and an ff run ends with an open draft PR
- [x] A-012 R14: `/git-pr` at ship takes the existing-OPEN-PR path and syncs Meta; on a branch with no PR it still creates one

### Scenario Coverage

- [x] A-013 R8: A pre-change body with a bare unmarked `## Meta` is adopted in place on first refresh — not duplicated — with following prose byte-identical
- [x] A-014 R9: Splicing twice with unchanged inputs yields a byte-identical body and issues no second `gh pr edit`
- [x] A-015 R5: No dispatch prompt in the changed skills names `git push`, `gh`, or `fab pr-sync`

### Edge Cases & Error Handling

- [x] A-016 R4: A boundary whose `origin/<branch>` moved since the last fetch STOPs without rewriting history
- [x] A-017 R10: `fab pr-sync` exits non-zero with no side effects when there is no fab context or no open PR
- [x] A-018 R2: A boundary with a clean tree performs no commit and no push
- [x] A-019 R15: A hydrate boundary where nothing drifted makes no memory-index commit

### Code Quality

- [x] A-020 Pattern consistency: New Go code follows the surrounding `internal/prmeta` and `cmd/fab` patterns; the marker constants mirror `memoryindex`'s naming
- [x] A-021 No unnecessary duplication: The splice is written once and called from every site; no call site restates the dance
- [x] A-022 Canonical source only: Every skill edit lands in `src/kit/`; no deployed copy under `.agents/skills/` or `.claude/skills/` is edited (Constitution V)
- [x] A-023 Owner-or-pointer: The boundary mechanics are stated in `_pipeline.md` only; the driver files point at it and do not restate it
- [x] A-024 R11: Deployed `src/kit/**` content cites no fab-kit-only path as an authority; the Go guard test passes (Constitution V)
- [x] A-025 **Partial**: Sibling sweep — `fab-ff` ↔ `fab-fff`, `skills.md` / `glossary.md` / `architecture.md`, and `fab-adopt.md` swept and verified at apply. The `/git-pr` memory file (`docs/memory/pipeline/execution-skills.md`, plus `change-lifecycle.md` / `schemas.md`) still carries the write-once-retrofit description — memory writes are the hydrate stage's scope by design (intake § Affected Memory); tracked as a should-fix finding for hydrate
- [x] A-026 Go changes ship tests: every changed `.go` file has accompanying test updates and `gofmt` is clean

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- The `review-pr` teardown is a deliberate non-goal here — a follow-on change owns it. Do not touch `/git-pr-review` or the Copilot machinery.

## Deletion Candidates

- None — this change adds new functionality (marker wrap, `Splice`, `fab pr-sync`, boundary wiring) without making existing code redundant; the old Step 3d prepend dance in `git-pr.md` was replaced in place, not orphaned, and `internal/memoryindex/adoption.go` stays in use for its own row-semantic navigation adoption.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab pr-sync` is a new top-level Go command owning render + splice + `gh pr edit`; `Splice` is a pure exported function in `internal/prmeta` | Intake assumption 6 delegated the pick with the tradeoff stated. The deciding factor is call-site count: five sites × a four-line `gh`/compare dance is precisely the restatement `code-quality.md` § Anti-Patterns forbids. `prmeta` already shells to `gh` and degrades gracefully, so no new dependency class | S:85 R:65 A:90 D:85 |
| 2 | Certain | Rebase fires only at the apply-exit open and at ship; review and hydrate boundaries plain-push | Not specified in the intake, which said only "commit and push" per boundary. Rebasing four times would force-push four times and open four conflict windows, while 3a-ter's own stated rationale (author context present) is met by rebasing once at the open. Ship's existing rebase covers base drift during the run | S:70 R:70 A:90 D:80 |
| 3 | Certain | The apply-exit open touches no `.status.yaml` progress or `stage_metrics`; `ship` keeps `/git-pr` verbatim | Promotes intake assumption 11 (Tentative) — the conservative reading is the one that leaves the Pipeline line, operator PR tracking, and done predicates truthful. Re-homing ship was never asked for and has a large blast radius | S:75 R:55 A:85 D:85 |
| 4 | Certain | The splice is written fresh in `prmeta`, following `memoryindex`'s marker convention but calling none of its code | Verified by reading `adoption.go`: `adoptNavigation` is row-semantic (per-row seeding, loss detection, tombstones) while the Meta splice is a flat span replacement. The intake's "reuse its splice helper where the shapes match" is satisfied by matching the convention — the shapes do not match at the code level | S:90 R:75 A:90 D:85 |
| 5 | Certain | The memory-index refresh runs at the hydrate boundary with `git-pr.md` 3a-bis retained as backstop | Intake assumption 9, unchanged. `fab docs-index docs/memory` is byte-stable and no-ops when nothing drifted, so the extra site is safe; hydrate is the only stage that writes `docs/memory/` | S:70 R:75 A:85 D:75 |
| 6 | Confident | Pushes fire at stage boundaries only; rework cycles inside review do not push | Intake assumption 8. Per-cycle pushes would multiply CI runs without adding signal the boundary push does not carry. Placement is one line of skill prose, cheap to re-tune if the CI signal during rework turns out to be wanted | S:75 R:70 A:75 D:70 |
| 7 | Confident | `## Summary` and `## Changes` are authored once at the apply-exit create and never refreshed | Intake assumption 13, stated outright by the user as part of the splice contract. The intake they derive from is stable by apply exit; an opt-in re-author stays a cheap later addition | S:80 R:70 A:80 D:75 |
| 8 | Confident | `fab-continue.md` runs the boundary procedure for whichever single stage it advances | Follows from intake assumption 3's orchestrator-ownership rule — `fab-continue` is named there as an orchestrator. The alternative (fab-continue advances stages without touching the PR) would make a stage-at-a-time run diverge from a bracket run for no reason | S:65 R:70 A:80 D:70 |
| 9 | Confident | `fab pr-sync` exits non-zero without side effects when fab context or an open PR is absent, mirroring `fab pr-meta` | Not specified; derived from the sibling command's documented contract ("Exits non-zero, emitting nothing, when there is no fab context, so /git-pr omits the Meta block exactly as before"). Matching the sibling is the least surprising choice and keeps call sites' `|| true` guards meaningful | S:60 R:80 A:85 D:75 |

9 assumptions (5 certain, 4 confident, 0 tentative).
