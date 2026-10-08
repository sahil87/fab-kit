# Plan: Copilot Review Request at Ship + Drop the `/fab-archive` Nudge

**Change**: 261008-or8g-ship-copilot-request-drop-archive-nudge
**Intake**: `intake.md`

## Requirements

### Ship: Copilot Review Request

#### R1: `/git-pr` requests a Copilot review at ship
`src/kit/skills/git-pr.md` SHALL carry a new step, positioned strictly between Step 4b (Finish Ship Stage) and Step 4c (Commit and Push Status Update), that requests a Copilot review on the PR via `gh pr edit --add-reviewer copilot-pull-request-reviewer`. The step MUST NOT be placed in Step 3c (PR create), because `_pipeline.md` § PR Boundary Procedure's apply-exit open reuses Step 3c to open the draft PR immediately after apply — a request there would review pre-review code.

- **GIVEN** a change reaching the ship stage with an OPEN PR
- **WHEN** `/git-pr` completes Step 4b
- **THEN** a Copilot review is requested on that PR before Step 4c runs
- **AND** the apply-exit draft open (Steps 3a/3a-ter/3b/3c only) requests nothing

#### R2: The request is idempotent across re-runs
The step SHALL probe for an existing Copilot review request or submitted review before requesting, and skip silently when either is present. The pending-request half MUST use the REST surface `gh api repos/{owner}/{repo}/pulls/{number}/requested_reviewers`, NOT `gh pr view --json reviewRequests` — the GraphQL-backed field omits bot reviewers, so a GraphQL probe reads empty and re-requests on every run. The two surfaces render the bot under different logins and both MUST be matched: `Copilot` under REST `requested_reviewers`, `copilot-pull-request-reviewer` on `reviews[].author.login` (prefix match, since the login may carry a `[bot]` suffix).

- **GIVEN** a PR where Copilot is already a requested reviewer
- **WHEN** `/git-pr` is re-run on that PR
- **THEN** the probe detects the pending request and skips, printing no new request
- **AND** no re-review is triggered (Constitution III)

- **GIVEN** a PR where Copilot has already submitted a review
- **WHEN** `/git-pr` is re-run
- **THEN** the probe matches `copilot-pull-request-reviewer` on the submitted-reviews surface and skips

#### R3: The request is best-effort and never blocks shipping
A failing request — no Copilot entitlement in the repo or org, a `gh` error, a network timeout — SHALL print a single warning line and continue. The step MUST NOT STOP, retry, poll, wait, or block Step 4c. It MUST NOT fire on the MERGED/already-shipped STOP path. `src/kit/skills/git-pr.md` § Rules SHALL gain a line stating the best-effort contract.

- **GIVEN** a repository with no Copilot entitlement
- **WHEN** ship runs the request step and `gh` returns an error
- **THEN** one warning line is printed and the ship completes successfully
- **AND** `progress.ship` still reaches `done`

### Routing: Remove the `/fab-archive` Nudge

#### R4: The Next-Steps state table drops every `/fab-archive` entry
`src/kit/skills/_preamble.md` § Next Steps Convention SHALL render these rows exactly: `hydrate` → `/git-pr` (default `/git-pr`); `ship` → `/git-pr-review` (default `/git-pr-review`); `review-pr (pass)` → no commands. The Precedence paragraph SHALL select `/git-pr-review` for the `ship: done` + `review-pr: pending` case instead of `/fab-archive`.

- **GIVEN** a change at `ship: done` with `review-pr: pending`
- **WHEN** a skill derives its `Next:` line from the state table
- **THEN** it renders `/git-pr-review`, never `/fab-archive`

#### R5: A no-command terminal state omits the `Next:` line
`_preamble.md` § Next Steps Convention SHALL document that a state with no available commands prints `Pipeline complete.` and omits the `Next:` line. The wording MUST NOT contradict `_pipeline.md`'s existing `Pipeline complete.` + `Next: {per state table}` rendering, which stays valid because both drivers' terminal states (`hydrate` for `/fab-ff`, `ship` for `/fab-fff`) still carry a command.

- **GIVEN** a change at `review-pr: done`
- **WHEN** a skill derives its ending
- **THEN** it prints `Pipeline complete.` with no `Next:` line

#### R6: `fab change switch` output matches the new table
`src/go/fab/internal/change/change.go`'s `Switch()` SHALL replace the single `allResolved` → `/fab-archive` branch with a two-way split: when all stages are resolved and `review-pr` is still `pending`, print bare `Next:        /git-pr-review`; when all stages are resolved and `review-pr` is `done` or `skipped`, print `Pipeline complete.` with no `Next:` line. The not-all-resolved branch is unchanged. The explanatory comment above the branch and the `defaultCommand` doc comment SHALL stop naming `/fab-archive`.

- **GIVEN** `.status.yaml` with every stage done and `review-pr: pending`
- **WHEN** `fab change switch` renders its block
- **THEN** the last line is `Next:        /git-pr-review`

- **GIVEN** `.status.yaml` with every stage done and `review-pr: done`
- **WHEN** `fab change switch` renders its block
- **THEN** the last line is `Pipeline complete.` and no `Next:` line appears

#### R7: Go tests pin the new strings
`src/go/fab/internal/change/change_test.go` SHALL update the three cases asserting `wantNext: "Next:        /fab-archive"` to the new expectations and rename them accordingly. The two negative cases (`review-pr active …`, `review-pr failed …`) keep their expectations. `go test ./internal/change/...` from the `src/go/fab` module root SHALL pass.

- **GIVEN** the updated `change.go`
- **WHEN** `go test ./internal/change/...` runs from `src/go/fab`
- **THEN** every case passes and no test still expects `/fab-archive`

#### R8: Kit skills carrying a routed `/fab-archive` Next line are swept
`src/kit/skills/fab-status.md` (the routing line) SHALL render the three-branch form: `Next: {stage} (via {command})` for a routing stage, `Next: /git-pr-review` while `review-pr` is `pending`, `Pipeline complete.` once it is `done`/`skipped`. `src/kit/skills/fab-adopt.md` SHALL render `Next: /git-pr-review`. `src/kit/skills/fab-switch.md` SHALL describe the new two-branch terminal output. Descriptive, non-routing references SHALL be preserved: `fab-operator.md` (post-merge maintenance) and `fab-continue.md` (Key Properties cross-reference).

- **GIVEN** a repo-wide `grep -rn "fab-archive"` after apply
- **WHEN** each hit is classified
- **THEN** no hit presents `/fab-archive` as a routed next command, and every descriptive reference survives

#### R9: Specs restating the routing are swept
`docs/specs/skills.md` SHALL be updated wherever it restates a `Next: /fab-archive` routing outcome. Other specs SHALL be audited: lines describing archive as the post-pipeline step a user runs stay; lines presenting it as a routed next command change. Generated index files SHALL be refreshed via `fab docs-index`, never hand-edited.

- **GIVEN** `docs/specs/skills.md` after apply
- **WHEN** its `/fab-continue` hydrate row and ship-landing lines are read
- **THEN** they name `/git-pr-review` or `Pipeline complete.`, not `/fab-archive`

### Non-Goals

- `fab help`'s flow diagram (`src/go/fab/cmd/fab/fab_help.go:159,160,179`) and the `"fab-archive": "Completion"` category mapping — on-demand discoverability, not a nudge
- The `/fab-archive` skill, `fab change archive`, and `fab batch archive` — unchanged
- Auto-running `fab batch archive` at ship — rejected during intake (uncommitted rename on a pushed PR; archive means merged; archived changes break `/git-pr-review <change>`)
- `fab switch --none` at ship — out of scope; MUST NOT be implemented or described as implemented
- Reviving `review-pr` as an automatic stage or the retired `review_tools` config
- Any polling, waiting, timeout budget, `--tool` flag, or config knob around the request

### Design Decisions

#### Ship-Time Copilot Request Replaces the Repository-Setup Premise
**Decision**: `/git-pr` requests a Copilot review once at ship, fire-and-forget, with an idempotence probe and a best-effort failure contract.
**Why**: change `261007-4p4z` rejected this option on the premise that cross-vendor review would come from the repository's own setup. That premise is false here: `repos/sahil87/fab-kit/rulesets/14613881` carries no `copilot_code_review` rule, and PR #690 — the first PR after 4p4z with no `/git-pr-review` run — received zero reviews. Change `gyp9` measured that 42% of PRs carry a Copilot fix, so the deleted request path was the sole source of that signal.
**Rejected**: a GitHub ruleset `copilot_code_review` rule (zero code, but it fires on *ready for review* and `/git-pr` leaves PRs as drafts, so it would not fire until the user manually un-drafts); reviving the `review-pr` stage or its 10-minute poll (the complexity 4p4z deleted, and not needed for a request).
*Introduced by*: 261008-or8g-ship-copilot-request-drop-archive-nudge

#### Archive Is Never a Routed Next Command
**Decision**: `/fab-archive` is removed from the Next-Steps routing table entirely rather than demoted from default.
**Why**: archiving is post-merge housekeeping whose timing fab cannot observe — the user merges by hand, sometimes days later, often in batches (PR #690 archived 9 changes at once). A routing entry that is only sometimes correct is precisely what produces the repeated "Should I run fab archive?" question. Archiving early also actively breaks the remaining manual stage: the resolver skips `archive/`, so `/git-pr-review <change>` hard-STOPs on an archived change.
**Rejected**: keeping `/fab-archive` listed but non-default (still mentioned on every completed change, so the prompt persists); adding a per-skill suppression rule (treats the symptom, leaves the wrong entry in the table).
*Introduced by*: 261008-or8g-ship-copilot-request-drop-archive-nudge

## Tasks

### Phase 1: Core Implementation — Ship Request

- [x] T001 Add the Copilot review request step to `src/kit/skills/git-pr.md`, strictly between Step 4b and Step 4c: gate (PR known, not the MERGED STOP path), REST `requested_reviewers` + `gh pr view --json reviews` idempotence probe matching both bot logins, the `gh pr edit --add-reviewer copilot-pull-request-reviewer` request, the best-effort warning path, and the `  ✓ review — …` success line matching the existing output style <!-- R1 -->
- [x] T002 Add the best-effort contract line to `src/kit/skills/git-pr.md` § Rules <!-- R3 -->

### Phase 2: Core Implementation — Routing Removal

- [x] T003 Update `src/kit/skills/_preamble.md` § Next Steps Convention: the three state-table rows (`hydrate`, `ship`, `review-pr (pass)`) and the Precedence paragraph's trailing selection <!-- R4 -->
- [x] T004 Document the no-command terminal-state rendering rule in `src/kit/skills/_preamble.md` § Next Steps Convention, worded so it does not contradict `_pipeline.md`'s existing `Pipeline complete.` + `Next:` rendering <!-- R5 -->
- [x] T005 Replace the `allResolved` branch in `src/go/fab/internal/change/change.go` `Switch()` with the two-way split, and update the explanatory comment above it plus the `defaultCommand` doc comment <!-- R6 -->
- [x] T006 Update the three `/fab-archive` cases in `src/go/fab/internal/change/change_test.go` to the new expectations and rename them <!-- R7 -->
- [x] T007 Run `go test ./internal/change/...` from the `src/go/fab` module root and confirm it passes <!-- R7 -->

### Phase 3: Sweep

- [x] T008 [P] Update the routing line in `src/kit/skills/fab-status.md` to the three-branch form <!-- R8 -->
- [x] T009 [P] Update `src/kit/skills/fab-adopt.md` (`Next: /git-pr-review`) and `src/kit/skills/fab-switch.md` (two-branch terminal output description) <!-- R8 -->
- [x] T010 Update `docs/specs/skills.md` wherever it restates a `Next: /fab-archive` routing outcome <!-- R9 -->
- [x] T011 Run a repo-wide `grep -rn "fab-archive"` and audit every remaining hit against the keep/change rule; leave historical records (`fab/changes/archive/**`, `docs/**/log*.md`, findings) untouched <!-- R8 -->

### Phase 4: Polish

- [x] T012 Refresh generated index files via `fab docs-index` if any swept file's description changed; never hand-edit a generated block <!-- R9 -->

## Execution Order

- T005 blocks T006 blocks T007 (code before tests before run)
- T003 blocks T004 (same section, sequential edits)
- T008–T009 are independent of each other and of Phase 2
- T011 runs after T001–T010 so the audit sees the final state

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `git-pr.md` carries the request step between Step 4b and Step 4c, and Step 3c is unmodified
- [ ] A-002 R2: The probe uses REST `requested_reviewers` for the pending half and matches both `Copilot` and `copilot-pull-request-reviewer`
- [ ] A-003 R3: The step prints one warning on failure, never STOPs, and § Rules states the best-effort contract
- [ ] A-004 R4: The three state-table rows and the Precedence paragraph contain no `/fab-archive`
- [ ] A-005 R5: The no-command terminal-state rule is documented and consistent with `_pipeline.md`
- [ ] A-006 R6: `change.go` renders `/git-pr-review` when review-pr is pending and `Pipeline complete.` when it is done/skipped
- [ ] A-007 R8: `fab-status.md`, `fab-adopt.md`, and `fab-switch.md` carry no routed `/fab-archive`
- [ ] A-008 R9: `docs/specs/skills.md` restates the new routing

### Behavioral Correctness

- [ ] A-009 R2: Re-running `/git-pr` on a PR that already has a Copilot request or review issues no second request
- [ ] A-010 R6: The not-all-resolved branch of `Switch()` is behaviourally unchanged
- [ ] A-011 R4: The skill-side state table and the Go `Switch()` output agree for every progress state

### Removal Verification

- [ ] A-012 R8: A repo-wide `grep -rn "fab-archive"` shows no remaining hit that presents it as a routed next command
- [ ] A-013 R7: No Go test still expects `Next:        /fab-archive`

### Scenario Coverage

- [ ] A-014 R7: `go test ./internal/change/...` passes from the `src/go/fab` module root
- [ ] A-015 R1: The apply-exit draft open path (Steps 3a/3a-ter/3b/3c) requests no review

### Edge Cases & Error Handling

- [ ] A-016 R3: The request does not fire on the MERGED/already-shipped STOP path
- [ ] A-017 R3: A repo with no Copilot entitlement still ships successfully

### Code Quality

- [ ] A-018 Pattern consistency: The new `git-pr.md` step matches the surrounding step structure and `  ✓ {thing} — {detail}` output style
- [ ] A-019 No unnecessary duplication: Existing probe/guard idioms in `git-pr.md` are reused rather than reinvented
- [ ] A-020 Deployed-content citation rule: The new `git-pr.md` step restates the two-login and REST-vs-GraphQL facts inline and cites no `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` path (Constitution V)
- [ ] A-021 Canonical sources only: Every edit lands in `src/kit/skills/*.md`; no deployed copy under `.agents/skills/` or `.claude/skills/` is edited
- [ ] A-022 Sibling sweeps: The whole sweep class was swept up front, not reactively after review
- [ ] A-023 CLI reference: Either `_cli-fab.md` needs no update (no signature change) or the required note was added — the call is made explicitly, not by omission

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- The intake's "Known environment hazard" (stale deployed `_preamble`) is already resolved: `fab/.fab-version` was bumped 2.28.3 → 2.28.7 and `fab sync` redeployed all 41 skills before apply started. Do not act on that warning.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Memory files are left to hydrate; apply touches only `src/kit/**`, `src/go/**`, and `docs/specs/**` | `_pipeline.md` Step 3 states hydrate "hydrates `docs/memory/`"; the intake's § Affected Memory is the hydrate input, not an apply task list | S:90 R:90 A:95 D:95 |
| 2 | Certain | FULL lane — 12 tasks against the hardcoded ≤ 5 light-lane threshold | `_pipeline.md` § Step 1 lane fork, counted from `## Tasks` | S:95 R:95 A:95 D:95 |
| 3 | Confident | `review-pr: skipped` renders the same terminal output as `done` | `allStagesResolved` already treats `skipped` as resolved and `change_test.go:764` pins a skipped path; no state-table row distinguishes them (intake assumption 10) | S:55 R:80 A:75 D:65 |
| 4 | Confident | The request step is numbered 4b-bis rather than renumbering 4c onward | Renumbering would churn every cross-reference to Step 4c in `git-pr.md` and `_pipeline.md`; the intake explicitly permits either, "provided it sits between 4b and 4c" | S:60 R:75 A:80 D:70 |
| 5 | Confident | `_cli-fab.md` needs no update — no command signature changes and the partial does not document `change switch`'s `Next:` output today | Intake assumption 14, verified by grep; A-023 forces the reviewer to confirm or overrule rather than let it pass silently | S:60 R:85 A:75 D:65 |

5 assumptions (2 certain, 3 confident, 0 tentative).
