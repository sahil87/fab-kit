# Plan: Retire `review-pr` as a pipeline stage — ship becomes terminal

**Change**: 261007-4p4z-retire-review-pr-stage
**Intake**: `intake.md`

## Requirements

### State Machine: Ship Is Terminal

#### R1: `finish ship` SHALL NOT auto-activate `review-pr`
`status.Finish` currently auto-activates `NextStage(stage)` whenever that stage is `pending`. The
automatic pipeline SHALL end at `ship`. A named exported constant SHALL express the terminal so the
rule is greppable rather than an inline string.

- **GIVEN** a change whose `ship` stage is `active`
- **WHEN** `fab status finish <change> ship` runs
- **THEN** `ship` becomes `done` and `review-pr` stays `pending`
- **AND** finishing `intake`/`apply`/`review`/`hydrate` still auto-activates their successor exactly as before

#### R2: `review-pr` SHALL remain a fully valid stage
`review-pr` MUST stay in `statusfile.StageOrder`, `status.AllowedStates`, and
`status.stageTransitions`. No `.status.yaml` migration is introduced, and in-flight changes whose
`review-pr` is already `active` or `done` MUST keep validating.

- **GIVEN** an existing change with `review-pr: active` from before this change
- **WHEN** `fab preflight` runs on it
- **THEN** it validates with no error and the stage is still reachable
- **AND** `review-pr` remains a legal `--stage` and `--stop-stage` value

#### R3: `NextStage` SHALL keep its pure stage-order meaning
The guard SHALL live at the `Finish` call site, not inside `NextStage`. `NextStage` stays a
stage-order lookup so a future caller that genuinely wants ordering is not silently truncated.

- **GIVEN** `NextStage("ship")`
- **WHEN** it is called directly
- **THEN** it still returns `"review-pr"` — the order is unchanged
- **AND** only `Finish` declines to act on that result

### `/git-pr-review`: Manual Triage Only

#### R4: The Copilot request and poll SHALL be removed
`git-pr-review.md` Step 2 Phase 2 SHALL be deleted in full — the `code-review.md` § Review Tools
config check, the `gh pr edit --add-reviewer` request, and the 30s × 20 poll. The skill SHALL never
request a review.

- **GIVEN** `/git-pr-review <change>` invoked on a PR with no reviews
- **WHEN** it runs
- **THEN** it prints `No reviews on PR #{number}.` and exits cleanly, requesting nothing
- **AND** the stage finishes as a successful no-op

#### R5: The timeout budget and its outcome class SHALL be removed
The `awk` history-marker counting, the `timeouts = 0 / 1 / ≥2` branching, the fail-closed
unwritable-`.history.jsonl` path, the `review-gate-unavailable` reason, the synchronous-poll
discipline note, and the `--tool` flag with its Step 1.5 parsing SHALL all be deleted. Step 6's
outcome table SHALL carry three classes: `success`, `failure`, `no-reviews`.

- **GIVEN** the rewritten skill
- **WHEN** it is read end to end
- **THEN** no `timeout` outcome, no budget counting, and no `--tool` flag remain
- **AND** nothing in the repo still produces or consumes a `timeout` review-pr outcome

#### R6: Triage behavior SHALL be unchanged when reviews exist
Steps 3–5.5 — fetch, triage, fix, commit, push, reply, dedupe — SHALL behave exactly as today. The
unpushed-commit re-run gate and the reply-deduplication logic are untouched.

- **GIVEN** a PR carrying unprocessed Copilot or human inline comments
- **WHEN** `/git-pr-review <change>` runs
- **THEN** comments are triaged, fixes pushed, and replies posted as before

#### R7: Re-running on a review-less PR SHALL be a clean no-op
Constitution III. Repeated invocation produces the same result and performs no write.

- **GIVEN** a PR with no reviews
- **WHEN** `/git-pr-review` is run twice
- **THEN** both runs report the same message, make no commit, and leave state unchanged

### Orchestrators: Drop the Review-PR Step

#### R8: `/fab-fff`'s terminal SHALL become `ship`
`fab-fff.md` Step 5 SHALL be deleted along with the synchronous-poll directive and the two timeout
rows in its Error Handling table. `{terminal}` becomes `ship`. The driver SHALL report
`Pipeline complete.` after ship.

- **GIVEN** a `/fab-fff` run that reaches a successful ship
- **WHEN** ship finishes
- **THEN** the pipeline reports complete and dispatches no review-pr worker

#### R9: Shared bracket and preamble SHALL stop describing review-pr as automatic
`_pipeline.md` (`{terminal}` framing, Light Lane's inline ship/review-pr note), `_preamble.md` (the
review-pr result schema's `timeout` outcome, the self-managing-stages text, the Next-Steps state
table) and `fab-continue.md` (the ship row's "auto-activates review-pr") SHALL be corrected.
`fab-continue.md`'s **review-pr rows stay** — that is the manual path.

- **GIVEN** the updated skills
- **WHEN** a reader follows the Next-Steps table after ship
- **THEN** it points at `/fab-archive`, not `/git-pr-review`
- **AND** `/git-pr-review` is still documented as an available manual command

#### R10: `fab-adopt.md` SHALL stop landing in review-pr
Its "Step 6 — Land in review-pr" section, its `Next: /git-pr-review` closing line, its frontmatter
description, and its prose claims SHALL be updated. Its `skip`/`reset` cascade comments stay correct
and are left alone.

- **GIVEN** a completed `/fab-adopt` run
- **WHEN** it reports its next step
- **THEN** it names ship completion, not review-pr

#### R11: `_cli-fab.md`'s documented `finish` side-effect chain SHALL be corrected
Line ~147 documents `ship→review-pr` auto-activation. That becomes false. The owning CLI reference
partial MUST be updated (Constitution Additional Constraints).

- **GIVEN** `_cli-fab.md` § fab status
- **WHEN** the `finish` side-effect chain is read
- **THEN** it ends at ship and does not claim ship activates review-pr

### Operator: Swap the Hardcoded Predicate

#### R12: The operator terminus SHALL become `ship` (option (b))
`tickTerminusStage` SHALL change from `"review-pr"` to `"ship"`, with its doc comment and
`tickCompleted`'s comment reworded. The five prose restatements
(`fab-operator.md` ×3, `_cli-fab-operator.md` ×2) SHALL match. No `terminal_stage` field, no derived
`done` predicate, no `fab status is-done`, no `done:` JSON key, no migration. `stop_stage` is
unchanged.

- **GIVEN** a tracked pane item with a null `stop_stage` whose change reaches `ship: done`
- **WHEN** the operator ticks
- **THEN** the item's built-in predicate fires and the item completes
- **AND** an item with an explicit `stop_stage: review-pr` still parks there

#### R13: The `no-reviewer` recovery row SHALL be removed; `ci-failed` and `conflicting` stay
`fab-operator.md` § Review-PR Recovery's `no-reviewer` row exists to unstick a first-timeout stall
that can no longer occur. The other two rows remain valid and become more useful now that CI runs
from apply onward.

- **GIVEN** the updated operator skill
- **WHEN** § Review-PR Recovery is read
- **THEN** no `no-reviewer` row, no `scope.recovery == "resend"` budget, and no re-send action remain
- **AND** the `ci-failed` and `conflicting` rows are intact

### Rendering and Scaffold

#### R14: The PR Pipeline line SHALL end at ship
`prmeta.pipelineStages` SHALL drop `review-pr`, otherwise every PR body renders a trailing unticked
stage reading as perpetually unfinished. Byte-stable render goldens SHALL be updated.

- **GIVEN** a change at `ship: done`
- **WHEN** `fab pr-meta` renders
- **THEN** the Pipeline line ends `… → ship ✓` with no trailing review-pr segment

#### R15: The Copilot review-tools knob SHALL be removed from the scaffold
`src/kit/scaffold/fab/project/code-review.md`'s § Review Tools `copilot` entry configured a request
path that no longer exists. It SHALL be removed; the rest of the scaffold's review policy stays.

- **GIVEN** a freshly scaffolded project
- **WHEN** its `code-review.md` is read
- **THEN** no § Review Tools / `copilot` knob is present

### Governance and Documentation

#### R16: The Constitution's trailing clause SHALL be reworded without a version bump
`fab/project/constitution.md` § Additional Constraints' "everything after intake runs unattended
unless review-rework exhausts or PR feedback arrives" SHALL name `ship` as terminal and `review-pr`
as manually invoked, with a dated governance comment and **no** version bump.

- **GIVEN** the amended Constitution
- **WHEN** the governance block is read
- **THEN** a dated `4p4z` comment explains the change and **Version** still reads `1.8.0`
- **AND** the six-stage enumeration is intact, because `review-pr` remains in `StageOrder`

#### R17: Current-behavior docs SHALL be rewritten; historical records SHALL NOT
Files describing `review-pr` as current automatic behavior (specs, memory, site docs) SHALL be
corrected. Files recording it historically — migrations, `docs/specs/findings/*`, `srad-v1.md`,
generated `log.md` / `log.seed.md`, archived changes, and historical test fixtures — SHALL be left
verbatim.

- **GIVEN** the 115-file survey in the intake's Impact section
- **WHEN** the sweep runs
- **THEN** every rewrite-class file states ship-terminal truth
- **AND** no migration, finding, archived change, or generated log line is edited

### Non-Goals

- Any automatic Copilot request, including a fire-and-forget variant at ship — explicitly rejected by the user.
- Anything from #688 (early PR open, Meta markers, `fab pr-sync`) — shipped and settled.
- `fab-operator.md:733` / `:774`'s bare `git push --force-with-lease` — a real latent bug, but pre-existing and out of scope; recorded as a backlog follow-up.
- Option (a): `terminal_stage`, a derived `done` predicate, `fab status is-done`, a `done:` JSON key, or any `.status.yaml` migration.

### Design Decisions

#### The auto-advance guard lives at the Finish call site, behind a named constant
**Decision**: Add `const AutoAdvanceTerminal = "ship"` to `internal/statusfile` and guard the
auto-activation block in `status.Finish` with `if stage != sf.AutoAdvanceTerminal`. Leave
`NextStage` returning `"review-pr"` for `"ship"`.
**Why**: `NextStage` is a stage-order lookup and reads like one; truncating it would make a
general-purpose helper lie to any future caller. The guard belongs where the policy is — one call
site, one named constant that greps cleanly next to `StageOrder`.
**Rejected**: Making `NextStage("ship")` return `""` (fewer lines, but silently redefines a pure
order function and would mislead a later caller); dropping `review-pr` from `StageOrder` entirely
(forces a `.status.yaml` migration and breaks every in-flight change — exactly what option (b) was
chosen to avoid).
*Introduced by*: 261007-4p4z-retire-review-pr-stage

#### The operator keeps a hardcoded terminus, one stage over
**Decision**: `tickTerminusStage` becomes `"ship"`. No derived predicate, no new status field.
**Why**: User-decided 2026-10-08 with the tradeoff stated. The gain is a materially smaller change
with no schema migration.
**Rejected**: Option (a) — a `terminal_stage` field with a derived `done` predicate the operator
queries. It would decouple the operator from the pipeline's shape permanently, but costs a status
schema change and therefore a `src/kit/migrations/` file. The accepted cost of (b): a future
pipeline reshape is again operator surgery.
*Introduced by*: 261007-4p4z-retire-review-pr-stage

#### `/git-pr-review` never requests a review, even manually
**Decision**: Delete the request path entirely rather than keeping it behind the `--tool` flag.
**Why**: The user's stated principle — fab deciding when a review is requested "depends too much on
the user's flow matching what we think". A manual request path would re-introduce exactly the
coupling, and keeping `--tool` would keep the poll machinery alive to serve it.
**Rejected**: Retaining `--tool copilot` as an explicit opt-in request (keeps the 10-minute poll and
its failure modes alive for a path the user did not ask for). If a Copilot review is wanted, GitHub's
own repository-level automatic review setting is where that belongs — the repo decides, fab never knows.
*Introduced by*: 261007-4p4z-retire-review-pr-stage

## Tasks

### Phase 1: Go — State Machine and Rendering

- [x] T001 Add `AutoAdvanceTerminal = "ship"` to `src/go/fab/internal/statusfile/statusfile.go` beside `StageOrder`, documenting that `review-pr` is reachable only via explicit `fab status start` <!-- R1 R3 -->
- [x] T002 Guard the auto-activation block in `src/go/fab/internal/status/status.go` `Finish` with `if stage != sf.AutoAdvanceTerminal`, leaving `NextStage` untouched <!-- R1 R3 -->
- [x] T003 Change `tickTerminusStage` to `"ship"` in `src/go/fab/cmd/fab/operator_tick_start.go:166` and reword its doc comment plus `tickCompleted`'s comment at ~:459 <!-- R12 -->
- [x] T004 Drop `"review-pr"` from `pipelineStages` in `src/go/fab/internal/prmeta/prmeta.go:32` <!-- R14 -->
- [x] T005 Update Go tests: `internal/status` (finish-at-ship no longer activates review-pr; every other finish still does; review-pr still reachable via `start`), `internal/statusfile` (StageOrder intact, new const), `internal/prmeta` (Pipeline-line goldens), `cmd/fab/operator_tick_diff_test.go` (terminus) <!-- R1 R2 R12 R14 -->
- [x] T006 Run `gofmt -w` on every changed `.go` file, then `cd src/go/fab && go test -count=1 ./...` and `cd src/go/fab-kit && go test -count=1 ./...` <!-- R1 R12 R14 -->

### Phase 2: `/git-pr-review` Teardown

- [x] T007 Rewrite `src/kit/skills/git-pr-review.md`: delete Step 2 Phase 2, the timeout budget, `review-gate-unavailable`, the `timeout` outcome class, the synchronous-poll note, and the `--tool` flag + Step 1.5; Step 2 becomes detect-or-report, Step 6's table becomes three classes <!-- R4 R5 R7 -->
- [x] T008 Verify Steps 3–5.5 (fetch / triage / fix / commit / push / reply / dedupe) and the unpushed-commit re-run gate survive the rewrite unchanged <!-- R6 -->
- [x] T009 Update the skill's frontmatter `description` — it currently advertises the Copilot request and the 10-minute wait <!-- R4 R5 -->

### Phase 3: Orchestrator and Operator Wiring

- [x] T010 `src/kit/skills/fab-fff.md` — delete Step 5, the synchronous-poll directive, and the two timeout Error-Handling rows; `{terminal}` becomes `ship` <!-- R8 -->
- [x] T011 `src/kit/skills/_pipeline.md` — `{terminal}` framing and the Light Lane inline ship/review-pr note <!-- R9 -->
- [x] T012 `src/kit/skills/_preamble.md` — drop the review-pr result schema's `timeout` outcome, fix the self-managing-stages text, and repoint the Next-Steps state table rows at `/fab-archive` <!-- R9 -->
- [x] T013 `src/kit/skills/fab-continue.md` — the ship row's "(auto-activates review-pr)" and the review-pr row's timeout branch; **keep** both review-pr rows (the manual path) <!-- R9 -->
- [x] T014 `src/kit/skills/fab-adopt.md` — Step 6, the `Next:` line, frontmatter description, and prose claims; leave the `skip`/`reset` cascade comments alone <!-- R10 -->
- [x] T015 `src/kit/skills/_cli-fab.md` — correct the `finish` side-effect chain (line ~147) <!-- R11 -->
- [x] T016 `src/kit/skills/fab-operator.md` — swap the three predicate restatements (~281, ~531, ~812) to `ship`, and delete § Review-PR Recovery's `no-reviewer` row plus its `scope.recovery == "resend"` budget; **keep** `ci-failed` and `conflicting` <!-- R12 R13 -->
- [x] T017 `src/kit/skills/_cli-fab-operator.md` — the two predicate restatements (~139, ~204) <!-- R12 -->
- [x] T018 [P] `src/kit/skills/fab-ff.md` and `_srad.md` — sibling-sweep pass for any review-pr/terminal claim <!-- R8 R9 -->
- [x] T019 [P] `src/kit/scaffold/fab/project/code-review.md` — remove § Review Tools' `copilot` entry <!-- R15 -->

### Phase 4: Governance, Docs, Sweep

- [x] T020 `fab/project/constitution.md` — reword § Additional Constraints' trailing clause; add a dated `4p4z` governance comment; **Version stays 1.8.0** <!-- R16 -->
- [x] T021 Sweep `docs/specs/` rewrite class — `skills.md`, `overview.md`, `architecture.md`, `glossary.md`, `user-flow.md`, `stage-models.md`, `harness-adapters.md`, `operator.md`, `assembly-line.md` <!-- R17 -->
- [x] T022 [P] Sweep `docs/site/` — `workflows.md`, `skill.md`, `install.md`, `merge-topologies.md`; leave the synced `fkf.md` copy alone <!-- R17 -->
- [x] T023 Verify the do-not-rewrite class is untouched: `src/kit/migrations/*`, `docs/specs/findings/*`, `docs/specs/srad-v1.md`, generated `docs/memory/**/log.md` and `log.seed.md`, `fab/changes/archive/**`, historical Go test fixtures <!-- R17 -->
- [x] T024 Grep repo-wide for `review-pr`, `git-pr-review`, `review-gate-unavailable`, `--add-reviewer`, `no-reviewer`, and `copilot-pull-request-reviewer`; confirm every surviving hit is either the manual path or the do-not-rewrite class <!-- R17 -->
- [x] T025 Mark backlog `[vb4l]` obsolete, keep `[hf6x]` open, and add a new backlog entry for the bare `--force-with-lease` at `fab-operator.md:733`/`:774` <!-- R17 -->
- [x] T026 Verify Constitution V compliance — no `src/kit/**` file cites fab-kit-only paths as an authority (in particular, restate the terminus rule rather than pointing at `operator_tick_start.go`); run the `src/go/fab-kit` guard test <!-- R17 -->

## Execution Order

- T001 → T002 (the const before its use); T001–T004 before T005; T005 before T006
- Phase 2 is independent of Phase 1 and may run alongside it
- T010–T017 are independent of each other; T018, T019, T022 are `[P]`
- T024 runs after every other sweep task so it catches residue
- T026 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab status finish <change> ship` leaves `review-pr` `pending`; finishing any earlier stage still auto-activates its successor *(verified: `status.go` guard + `TestShipFinishDoesNotAutoActivateReviewPr`, module green)*
- [x] A-002 R2: `review-pr` is still in `StageOrder`, `AllowedStates`, and `stageTransitions`; an existing change with `review-pr: active` still validates under `fab preflight` *(verified: diff touches none of the three; `TestAutoAdvanceTerminal` pins the six-stage order)*
- [x] A-003 R4: `/git-pr-review` on a review-less PR reports `No reviews on PR #{number}.` and requests nothing *(verified: `git-pr-review.md` Step 2)*
- [x] A-004 R8: `/fab-fff` reports `Pipeline complete.` after ship and dispatches no review-pr worker *(verified: Step 5 deleted; Output section updated)*
- [x] A-005 R12: `tickTerminusStage` is `"ship"` and all five prose restatements match *(verified: `operator_tick_start.go:167`; `fab-operator.md` ×3, `_cli-fab-operator.md` ×2)*
- [x] A-006 R14: the rendered Pipeline line ends at `ship` with no trailing review-pr segment *(verified: `prmeta.go` + updated goldens, tests green)*
- [x] A-007 R15: the scaffold's `code-review.md` carries no § Review Tools / `copilot` knob *(verified: section deleted)*
- [x] A-008 R16: the Constitution's trailing clause names ship-terminal, carries a dated `4p4z` comment, and **Version** still reads `1.8.0` *(verified)*

### Behavioral Correctness

- [x] A-009 R3: `NextStage("ship")` still returns `"review-pr"`; only `Finish` declines to act on it *(verified: guard at the `Finish` call site; `TestNextStage` pins `ship → review-pr`)*
- [x] A-010 R6: triage, fix, commit, push, reply, dedupe, and the unpushed-commit re-run gate behave exactly as before when reviews exist *(verified: no diff hunks in Steps 3–5.5)*
- [x] A-011 R13: § Review-PR Recovery has no `no-reviewer` row and no `scope.recovery == "resend"` budget; `ci-failed` and `conflicting` are intact *(verified: section renamed PR Recovery; both rows kept)*
- [x] A-012 R9: the Next-Steps state table points at `/fab-archive` after ship, and `/git-pr-review` is still documented as an available manual command *(verified: `_preamble.md` state table)*
- [x] A-013 R10: `/fab-adopt` reports ship completion, not review-pr; its `skip`/`reset` cascade comments are unchanged *(verified: lines 94–99 untouched, Step 6 rewritten)*

### Removal Verification

- [x] A-014 R5: no `timeout` review-pr outcome, no budget counting, no `review-gate-unavailable`, no `--tool` flag, and no synchronous-poll directive survive anywhere *(verified: repo-wide greps clean outside the do-not-rewrite findings class)*
- [x] A-015 R4: no `gh pr edit --add-reviewer` and no `copilot-pull-request-reviewer` poll remain in `src/kit/**` *(verified: grep clean)*
- [x] A-016 R17: `src/kit/migrations/*`, `docs/specs/findings/*`, `srad-v1.md`, generated `log.md`/`log.seed.md`, and `fab/changes/archive/**` are byte-identical to before *(verified: no diff entries under any of those paths)*

### Scenario Coverage

- [x] A-017 R2: an operator item with an explicit `stop_stage: review-pr` still parks there *(verified: `StageOrder` and stop_stage semantics unchanged; accepted-selector list untouched)*
- [x] A-018 R7: running `/git-pr-review` twice on a review-less PR gives the same result and writes nothing *(verified: rewritten Idempotency paragraph; no-reviews path stages nothing)*
- [x] A-019 R12: a tracked pane item with a null `stop_stage` completes when its change reaches `ship: done` *(verified: `TestOperatorTickDiff_CompletionPredicateBranches` s010/s011, module green)*

### Edge Cases & Error Handling

- [x] A-020 R1: a change already at `review-pr: active` from before this change can still be finished manually *(verified: review-pr finish/fail transitions and auto-log rows unchanged)*
- [x] A-021 R2: `fab status start <change> review-pr` still transitions `pending → active` *(verified: exercised in `TestShipFinishDoesNotAutoActivateReviewPr`)*

### Code Quality

- [x] A-022 Pattern consistency: the new constant sits beside `StageOrder` and follows surrounding naming *(verified: `AutoAdvanceTerminal` in `statusfile.go`)*
- [x] A-023 No unnecessary duplication: the terminal is one named constant, not a repeated literal *(verified)*
- [x] A-024 Canonical source only: every skill edit lands in `src/kit/`; no deployed copy is edited (Constitution V) *(verified: no `.agents/skills/` / `.claude/skills/` entries in the diff)*
- [x] A-025 R26: no `src/kit/**` file cites a fab-kit-only path as an authority; the `src/go/fab-kit` guard test passes (Constitution V) *(verified: grep clean — no `operator_tick_start.go` pointer; guard test green)*
- [x] A-026 Sibling sweep: `fab-ff` ↔ `fab-fff` swept together, plus `skills.md` / `glossary.md` / `architecture.md` / `overview.md` swept; the `docs/memory` sweep is hydrate-stage scope and correctly untouched at review
- [x] A-027 Go changes ship tests: every changed `.go` file has test updates; `gofmt` clean; both modules green *(verified: `gofmt -l src/go/` empty; `go test -count=1 ./...` green in both modules)*
- [x] A-028 CLI ⇒ docs: `_cli-fab.md` and `_cli-fab-operator.md` reflect every changed documented behavior *(verified: finish side-effect chain and both predicate restatements updated)*

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- **This change ships under the OLD pipeline.** Its own `/fab-fff` run will still auto-activate `review-pr` after ship, because the installed `fab` binary predates this change. That is expected — do not "fix" it mid-run.

## Deletion Candidates

- None — this change is itself the teardown: it deletes the Copilot request/poll machinery, the timeout budget, the `--tool` flag, and the `no-reviewer` recovery row rather than leaving newly redundant code behind. No surviving symbol, file, or block was made redundant by the diff (the retained `review-pr` state-machine entries are deliberate — they serve the manual path). Borderline stale *prose* survivors (`_cli-fab.md:419`'s `review_tools` pointer, `probes.go:43` / `configref.go:626`'s review-pr mentions, `docs/specs/fkf.md:146`'s "at review-pr" rationale) are reported as review findings, not deletions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The auto-advance guard lives at the `Finish` call site behind `sf.AutoAdvanceTerminal`; `NextStage` keeps returning `"review-pr"` for `"ship"` | Verified `NextStage` has exactly ONE non-test caller (`status.go:180`), so either placement works mechanically. The call-site guard is chosen because `NextStage` is named and documented as a stage-order lookup; truncating it would make a general helper lie. A named constant keeps the rule greppable | S:80 R:85 A:90 D:85 |
| 2 | Certain | `review-pr` stays in `StageOrder` / `AllowedStates` / `stageTransitions`; no `.status.yaml` migration | Intake decision, and the reason it was chosen over deletion: removing it would force a migration and break in-flight changes. Also what keeps the Constitution's six-stage enumeration literally true | S:90 R:85 A:90 D:90 |
| 3 | Certain | Operator takes option (b) — `tickTerminusStage` → `"ship"` plus five prose restatements; no `terminal_stage`, no derived predicate, no migration | User-decided 2026-10-08, asked directly with the tradeoff stated and accepted (coupling reproduced one stage over, in exchange for a materially smaller change) | S:95 R:75 A:90 D:95 |
| 4 | Certain | The Constitution edit is wording-only with a dated governance comment and NO version bump | The on-point precedent is **j6cs** (260601) — a 7→6 pipeline-shape change, strictly larger than this one — which recorded exactly this treatment and no bump. udwv/jjg0/t513/yd9s/si4k agree but are pure-wording cases | S:85 R:80 A:85 D:85 |
| 5 | Certain | `--tool` is deleted rather than retained as a manual request path | Follows directly from the user's stated principle; retaining it would keep the entire poll and failure-mode machinery alive to serve a path the user did not ask for | S:85 R:70 A:85 D:90 |
| 6 | Certain | `fab-continue.md`'s review-pr rows are KEPT; only the auto-activation claim and the timeout branch change | `/git-pr-review` remains a real, invocable skill — removing its rows would make the manual path undiscoverable. Intake named this explicitly | S:85 R:85 A:85 D:85 |
| 7 | Confident | Historical records (migrations, findings, `srad-v1.md`, generated logs, archived changes, test fixtures) are left verbatim | Intake's survey split. Rewriting a migration or an archived change would falsify the record of what was true then; generated logs are append-only and byte-stable by contract, so hand-editing them is the never-hand-merge violation | S:75 R:70 A:85 D:80 |
| 8 | Confident | The scaffold's § Review Tools `copilot` entry is removed outright rather than left inert | It configures a request path that no longer exists anywhere; leaving it would advertise a knob with no effect, which is worse than absent. Scaffold-only, so no existing project's file is rewritten | S:70 R:75 A:80 D:75 |
| 9 | Confident | This change's own pipeline run still auto-activates `review-pr` after ship, and that is expected | The installed `fab` binary predates the guard — the same version-skew the previous change hit. Noted in `## Notes` so neither apply nor review treats it as a defect | S:80 R:85 A:80 D:70 |

9 assumptions (6 certain, 3 confident, 0 tentative).
