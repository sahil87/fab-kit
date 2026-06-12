# Plan: Skills-Review Batch 1 — Surgical Correctness + Idempotency Fixes

**Change**: 260611-9u91-skills-correctness-idempotency-fixes
**Intake**: `intake.md`

## Requirements

### Pipeline Dispatch & Status Ownership

#### R1: fab-continue dispatch table prescribes only valid state transitions (f005)
`src/kit/skills/fab-continue.md` Step 1 MUST NOT instruct `fab status start` calls that the CLI rejects. The intake-ready dispatch rows SHALL read "finish intake (auto-activates apply) → execute apply" (no `start apply`), and the review-fail row SHALL use `reset <change> apply fab-continue` (matching the Verdict section).

- **GIVEN** a change with intake `ready`
- **WHEN** an agent follows the Step 1 dispatch table
- **THEN** it runs `fab status finish <change> intake fab-continue` and proceeds directly to apply execution without any `start apply` call
- **AND** on review failure the table directs `fail <change> review` then `reset <change> apply fab-continue`

#### R2: Orchestrator owns all fab status transitions (f006)
When `/fab-ff` / `/fab-fff` dispatch fab-continue's behavior sections as subagents, the orchestrator MUST be the single owner of `fab status` transitions. The ff/fff subagent dispatch prompts (apply, review, hydrate) SHALL include the explicit instruction "do NOT run `fab status` commands; return results only". The hydrate driver inversion (fab-ff/fab-fff Step 3 currently having the subagent run `finish hydrate`) SHALL be resolved the same way: the orchestrator runs `fab status finish <change> hydrate fab-ff|fab-fff` after the subagent returns. `fab-continue.md`'s Apply, Review, and Hydrate behavior sections SHALL each carry the rule "when invoked as a subagent, skip the finish step / §Verdict — the orchestrator owns transitions"; the ship dispatch row's re-finish (git-pr already finishes ship internally) SHALL be guarded by the same rule.

- **GIVEN** `/fab-ff` dispatches fab-continue Apply Behavior as a subagent
- **WHEN** the subagent completes all tasks
- **THEN** it returns results only; the orchestrator (and only the orchestrator) runs `fab status finish <change> apply fab-ff`
- **AND** no double-finish CLI error occurs

- **GIVEN** `/fab-ff` dispatches Hydrate Behavior as a subagent
- **WHEN** hydration completes
- **THEN** the orchestrator runs `fab status finish <change> hydrate fab-ff` (driver matches the orchestrator, not the subagent)

#### R3: Deterministic review pass rule (f012)
`src/kit/skills/_review.md` Findings Merge step 4 MUST state the deterministic rule: **no must-fix findings (including zero findings) → review passes**. should-fix and nice-to-have findings are reported but never block. The mirror sentence at `docs/specs/skills.md` (pass/fail paragraph) SHALL state the same rule.

- **GIVEN** a merged findings set with zero must-fix findings (possibly zero findings total)
- **WHEN** the pass/fail determination runs
- **THEN** review passes deterministically — no agent discretion

#### R4: ff/fff intake-finish condition covers `ready` (f069/g1-6)
`fab-ff.md` Step 1 and `fab-fff.md` Step 1 SHALL replace "finish intake first if still active" with "if `progress.intake` is not `done`, finish intake" — `finish` accepts both `active` and `ready` (status.go), and `/fab-new` leaves intake at `ready`.

- **GIVEN** a change created by `/fab-new` (intake at `ready`)
- **WHEN** `/fab-ff` Step 1 runs
- **THEN** the agent finishes intake (because it is not `done`), auto-activating apply

#### R5: review-failed resume guard (g1-5)
`fab-continue.md` Step 1 and the Resumability sections of `fab-ff.md`/`fab-fff.md` SHALL state: "if `progress.review` is `failed`, run `fab status start <change> review` first" — invoking the review-specific failed→active transition that exists for interruption recovery.

- **GIVEN** an interruption left `progress.review` at `failed` (apply already `done`)
- **WHEN** `/fab-continue` (or `/fab-ff`/`/fab-fff`) re-enters
- **THEN** the skill runs `fab status start <change> review` to recover review to `active` before re-executing it

### CLI (Go)

#### R6: fab-help renders the canonical six-stage pipeline and maps all user-facing skills (f014 + g4-1)
`src/go/fab/cmd/fab/fabhelp.go` MUST replace the retired "Planning stages: spec → tasks" / "Execution stages: apply → review → hydrate" lines with the six-stage pipeline `intake → apply → review → hydrate → ship → review-pr`. `skillToGroupMap` MUST gain the four unmapped skills — `fab-proceed`, `fab-operator`, `git-branch`, `git-pr-review` — grouped per existing map semantics. `fabhelp_test.go` MUST be updated accordingly.

- **GIVEN** a user runs `fab fab-help`
- **WHEN** the WORKFLOW section renders
- **THEN** it shows `intake → apply → review → hydrate → ship → review-pr` and no `spec`/`tasks` stage
- **AND** `fab-proceed`, `fab-operator`, `git-branch`, `git-pr-review` render under mapped groups (not "Other")

#### R7: Actionable "No active change." errors (f124)
`src/go/fab/internal/resolve/resolve.go` zero-candidate and missing-changes-dir paths MUST append recovery guidance: `No active change. Run /fab-new <description> to start one, or /fab-switch to activate an existing one.` Tests MUST cover the new strings; the affected `_cli-fab.md` Common Error Messages rows are updated for the changed strings only (full table regeneration is batch 2 / f052).

- **GIVEN** no `.fab-status.yaml` symlink and zero candidate changes
- **WHEN** `fab preflight` / `fab resolve` runs without an override
- **THEN** stderr contains the `/fab-new` + `/fab-switch` guidance (honoring `_preamble.md`'s "stderr contains the specific error and suggested fix" promise)

### PR Skills

#### R8: git-pr-review single exit point + timeout outcome (f015/f016)
`src/kit/skills/git-pr-review.md` terminal **STOP**s in Steps 1, 2, and 4 SHALL be replaced with "go to Step 6 with outcome {success | failure | no-reviews}". Step 6 SHALL state it is the single exit point for all terminal paths after Step 0, and SHALL gain a fourth outcome class: *Copilot review requested but timed out (10 min)* → leave the review-pr stage `active` — no finish, no fail — keeping the "Re-run /git-pr-review to process when ready" message.

- **GIVEN** the Copilot poll exhausts 20 attempts without a review appearing
- **WHEN** the timeout path executes
- **THEN** the skill routes to Step 6 with the timeout outcome, runs neither `finish` nor `fail`, and the review-pr stage stays `active` so a re-run can process the review later

- **GIVEN** Step 1 finds no PR on the branch
- **WHEN** the terminal path executes
- **THEN** it goes to Step 6 with outcome `failure` (no bare STOP that skips status routing)

#### R9: git-branch rename guard (f100)
`src/kit/skills/git-branch.md` Step 4 SHALL rename the current local-only branch only when its name does not match another change folder under `fab/changes/` (mechanism: `fab change resolve <current-branch>` fails to match); when it does match another change, the skill SHALL create a new branch via `git checkout -b` instead (accepted caveat: the new branch inherits the old change's HEAD).

- **GIVEN** the user sits on change B's local-only (upstream-less) branch after `/fab-switch` to change A
- **WHEN** `/git-branch` runs for change A
- **THEN** change B's branch is left intact and a new branch is created with `git checkout -b`

- **GIVEN** the current local-only branch is a disposable non-change branch (e.g., a `wt create` random name that resolves to no change)
- **WHEN** `/git-branch` runs
- **THEN** the rename path (`git branch -m`) is used as before

### Runtime / Operator

#### R10: Watch dedup against known PLUS completed (f018)
`src/kit/skills/fab-operator.md` §7 Watches Tick Behavior step 2 SHALL deduplicate spawns against the union of the `known` and `completed` lists, closing the respawn loop that occurs when an item ID moves from `known` to `completed` at stop_stage while still matching the watch query.

- **GIVEN** a watch-spawned item reached `stop_stage` (its ID moved from `known` to `completed`) and the Linear issue still matches the watch query
- **WHEN** the next tick queries the source
- **THEN** the item is skipped (present in `completed`) and no second agent is spawned

### Setup & Status Display

#### R11: fab-setup bootstrap trigger fires for fab-init configs; fab_version preserved (f024)
`src/kit/skills/fab-setup.md` step 1a's trigger SHALL broaden to "missing OR raw template OR missing required fields `project.name`/`project.description`" (a `fab init`-created config contains only `fab_version`). The Config Pre-flight create-mode reference SHALL be updated consistently, and Config Create Mode SHALL preserve an existing `fab_version` key when writing from the scaffold template (the scaffold lacks it; config.go errors without it).

- **GIVEN** the canonical flow `fab init` → `/fab-setup` (config.yaml exists with only `fab_version`)
- **WHEN** bootstrap step 1a evaluates its trigger
- **THEN** Config Behavior runs in create mode (project name/description/source paths are collected)
- **AND** the written config.yaml retains the pre-existing `fab_version` key

#### R12: fab-status over-threshold channel is emoji + bold, not ANSI (f047)
`src/kit/skills/fab-status.md` SHALL replace the unsatisfiable "highlighted in yellow (terminal `\e[33m...\e[0m`)" mandate with the surviving channels: a warning-emoji (⚠️) prefix plus bold on the over-threshold Impact line, mirroring fab-operator's health-emoji convention (ANSI SGR is stripped by the render path).

- **GIVEN** `true_impact.net > 100` (or `excluding.net > 50` when present)
- **WHEN** the Impact line renders
- **THEN** it is prefixed with ⚠️ and bolded — no ANSI escape sequences are emitted

### Idempotency (Constitution III)

#### R13: fab-new/fab-draft re-run routes backlog/Linear-ID collisions to resume (g1-2)
In Step 3 of `fab-new.md` and `fab-draft.md`, when a detected backlog/Linear ID already has an existing non-archived change, the skill SHALL route to resume — pointing the user to `/fab-switch {name}` + `/fab-continue` (whose intake-active row regenerates a missing intake) — instead of surfacing the raw `Change ID already in use` error. The `fab change new` collision failure rows in both Error Handling tables SHALL map to that recovery guidance. Both skills SHALL document explicitly that a natural-language re-run intentionally creates a new change each run. `change.go` stays unchanged (safety net).

- **GIVEN** `/fab-new 9u91` is re-run while change `260611-9u91-...` already exists (non-archived)
- **WHEN** Step 3 detects the collision
- **THEN** the skill does not error; it reports the existing change and directs the user to `/fab-switch 260611-9u91-... ` then `/fab-continue`

- **GIVEN** `/fab-new "add oauth"` (natural language) is re-run
- **WHEN** Step 3 runs
- **THEN** a new change with a fresh random ID is created — documented as intentional

#### R14: Hydrate merges without duplication; idempotency claim covers hydrate (g1-3)
`fab-continue.md` Hydrate Behavior step 4 SHALL instruct: before appending to a target memory file, check for an existing entry referencing this change (by change name) and update it in place — the same merge-without-duplication contract as `docs-hydrate-memory.md` and `_review.md`'s "replaced in place (not duplicated)". The Key Properties "Idempotent?" row SHALL extend to cover hydrate (merge-without-duplication).

- **GIVEN** hydrate was interrupted after memory writes but before `fab status finish`
- **WHEN** hydrate re-runs
- **THEN** existing entries referencing this change are updated in place — no duplicate Changelog/Design Decision entries

#### R15: Generic fab-command failure rule (g2-2)
`_preamble.md` § Common fab Commands "Key behaviors" SHALL gain one sentence: any fab command not explicitly marked best-effort (`2>/dev/null || true`) that exits non-zero → STOP and surface stderr — deferring to explicit per-skill handling where a skill intentionally branches on non-zero exit.

- **GIVEN** `fab status finish <change> apply` exits non-zero mid-pipeline
- **WHEN** the invoking skill consults the preamble rule
- **THEN** it stops and surfaces stderr rather than proceeding with diverged state
- **AND** skills that intentionally branch on non-zero exits (fab-proceed, fab-discuss, git-pr, fab-archive) are unaffected by the generic rule

#### R16: Idempotency declarations on fab-new, fab-draft, git-pr (g1-7)
`fab-new.md` and `fab-draft.md` SHALL gain a standard Key Properties section (not just a row) declaring the R13 re-run semantics; `git-pr.md` SHALL gain a Key Properties section declaring its existing contract (re-run after ship is a no-op via the "already shipped" path).

- **GIVEN** an agent or user reads any of the three skill files
- **WHEN** they look for the re-run contract
- **THEN** a Key Properties section states the idempotency semantics explicitly

### Non-Goals

- f019 (review-failed dispatch row presenting the rework menu) — batch 4 (`szxd`)
- g3-4 (change-type inference vs PostToolUse hook alignment) — batch 2 (`uliv`)
- f052 (full Common Error Messages table regeneration) — batch 2; only rows whose strings R7 changes are touched
- All other batch 2–4 findings (staleness sweep, `_preamble` context diet, twins refactor)
- No `change.go` behavior change for R13 — resume routing is skill-level only
- No runtime/user-data migrations (no `.status.yaml`/config schema changes)

### Design Decisions

1. **Orchestrator owns transitions (R2)**: subagent prompts carry an explicit "do NOT run fab status" instruction and fab-continue's behavior sections carry a when-subagent skip rule — *Why*: single writer eliminates textually-mandated double finish/fail errors; matches `_review.md`'s "verdict transitions remain in each orchestrator's own file" — *Rejected*: subagent-owns (ff/fff drop their finish lines) — would put state writes furthest from the retry/rework loop that consumes them.
2. **Deterministic pass rule (R3)**: "no must-fix findings (including zero findings) → review passes" — *Why*: SPEC-_review.md already states this form; unattended post-intake operation requires zero discretion — *Rejected*: documenting discretion conditions (keeps nondeterminism).
3. **Resume routing for ID collisions (R13)**: skill-level detection + `/fab-switch`/`/fab-continue` pointer; `change.go` error kept as safety net — *Why*: honors Constitution III without a Go behavior change — *Rejected*: making `fab change new` itself idempotent (hides a real collision signal other callers rely on).

## Tasks

### Phase 1: Go fixes (CLI)

- [x] T001 Replace the retired pipeline lines in `src/go/fab/cmd/fab/fabhelp.go` with the six-stage pipeline `intake → apply → review → hydrate → ship → review-pr`, and add `fab-proceed`, `fab-operator`, `git-branch`, `git-pr-review` to `skillToGroupMap` per existing group semantics <!-- R6 -->
- [x] T002 Update `src/go/fab/cmd/fab/fabhelp_test.go`: extend the group-mapping test to the four new skills and add an assertion that the rendered pipeline string is the six-stage form (no spec/tasks) <!-- R6 -->
- [x] T003 [P] Append actionable guidance to the zero-candidate and missing-changes-dir `No active change.` errors in `src/go/fab/internal/resolve/resolve.go` <!-- R7 -->
- [x] T004 Add/extend tests in `src/go/fab/internal/resolve/resolve_test.go` covering the new error strings (no-changes-dir and zero-candidates paths) <!-- R7 -->
- [x] T005 Update the affected Common Error Messages rows in `src/kit/skills/_cli-fab.md` for the strings changed by T003 only <!-- R7 -->

### Phase 2: Core skill edits (canonical `src/kit/skills/`)

- [x] T006 Fix `src/kit/skills/fab-continue.md` Step 1 dispatch table: intake-ready rows → "finish intake (auto-activates apply) → execute apply"; review-fail row → `reset <change> apply fab-continue` <!-- R1 -->
- [x] T007 Add the when-invoked-as-subagent skip rule to `src/kit/skills/fab-continue.md` Apply Behavior (step finish), Review Behavior (§Verdict), and Hydrate Behavior (finish step); guard the ship-row re-finish with the same rule <!-- R2 -->
- [x] T008 Add "do NOT run `fab status` commands; return results only" to the subagent dispatch prompts in `src/kit/skills/fab-ff.md` and `src/kit/skills/fab-fff.md` (Steps 1–3), and move the hydrate finish to the orchestrator (`fab status finish <change> hydrate fab-ff|fab-fff` after the subagent returns) <!-- R2 -->
- [x] T009 Replace "review **may pass**" in `src/kit/skills/_review.md` Findings Merge step 4 with the deterministic rule "**no must-fix findings (including zero findings) → review passes**; should-fix and nice-to-have findings are reported but never block" <!-- R3 -->
- [x] T010 [P] Change "finish intake first if still active" to "if `progress.intake` is not `done`, finish intake" in `src/kit/skills/fab-ff.md` and `src/kit/skills/fab-fff.md` Step 1 <!-- R4 -->
- [x] T011 Add the review-failed resume guard ("if `progress.review` is `failed`, run `fab status start <change> review` first") to `src/kit/skills/fab-continue.md` Step 1 and the Resumability sections of `fab-ff.md`/`fab-fff.md` <!-- R5 -->
- [x] T012 Rework `src/kit/skills/git-pr-review.md`: route Steps 1, 2, 4 terminal STOPs to Step 6 with named outcomes; declare Step 6 the single exit point; add the fourth timeout outcome (leave stage active — no finish, no fail) <!-- R8 -->
- [x] T013 Add the rename guard to `src/kit/skills/git-branch.md` Step 4 (rename only when `fab change resolve <current-branch>` matches no other change; else `git checkout -b`), and mirror in Error Handling/Key Properties as needed <!-- R9 -->
- [x] T014 [P] Fix `src/kit/skills/fab-operator.md` Watches Tick Behavior step 2: deduplicate against `known` plus `completed` <!-- R10 -->
- [x] T015 [P] Broaden `src/kit/skills/fab-setup.md` step 1a trigger (missing OR raw template OR missing `project.name`/`project.description`), update the Config Pre-flight create-mode reference, and add fab_version preservation to Config Create Mode <!-- R11 -->
- [x] T016 [P] Replace the ANSI-yellow mandate in `src/kit/skills/fab-status.md` with ⚠️ prefix + bold on the over-threshold Impact line <!-- R12 -->
- [x] T017 Add backlog/Linear-ID collision resume routing to Step 3 of `src/kit/skills/fab-new.md` and `src/kit/skills/fab-draft.md`, map the `fab change new` collision failure rows to the recovery guidance, and document NL re-run = new change <!-- R13 --> <!-- rework: Linear-ID branch prescribed `fab change resolve {id}`, which matches folder names only — Linear IDs live in .status.yaml issues arrays, so the check could never fire; detect via an issues-array scan instead -->
- [x] T018 Add merge-without-duplication to `src/kit/skills/fab-continue.md` Hydrate step 4 and extend the Key Properties idempotency row to cover hydrate <!-- R14 -->
- [x] T019 [P] Add the generic non-best-effort fab-command failure rule to `src/kit/skills/_preamble.md` § Common fab Commands "Key behaviors" with the defer-to-explicit-per-skill-handling carve-out <!-- R15 -->
- [x] T020 Add Key Properties sections (with Idempotent? declarations) to `src/kit/skills/fab-new.md`, `src/kit/skills/fab-draft.md`, and `src/kit/skills/git-pr.md` <!-- R16 -->

### Phase 3: Spec mirrors & flagged spec lines

- [x] T021 Update `docs/specs/skills/SPEC-fab-continue.md` (dispatch wording, when-subagent rule, hydrate dedup) and `docs/specs/skills.md` dispatch line (~:298) <!-- R1 -->
- [x] T022 [P] Update `docs/specs/skills/SPEC-fab-ff.md` and `SPEC-fab-fff.md` (orchestrator-owns-transitions, intake-finish condition, review-failed guard) <!-- R2 -->
- [x] T023 [P] Update `docs/specs/skills/SPEC-_review.md` and `docs/specs/skills.md` pass-rule line (~:504) <!-- R3 -->
- [x] T024 [P] Update `docs/specs/skills/SPEC-git-pr-review.md` (single exit point + timeout outcome) <!-- R8 -->
- [x] T025 [P] Update `docs/specs/skills/SPEC-git-branch.md` (rename guard) <!-- R9 -->
- [x] T026 [P] Update `docs/specs/skills/SPEC-fab-operator.md` (dedup against known + completed) <!-- R10 -->
- [x] T027 [P] Update `docs/specs/skills/SPEC-fab-setup.md` (broadened 1a trigger, fab_version preservation) <!-- R11 -->
- [x] T028 [P] Update `docs/specs/skills/SPEC-fab-status.md` (emoji/bold channel) <!-- R12 -->
- [x] T029 [P] Update `docs/specs/skills/SPEC-fab-new.md`, `SPEC-fab-draft.md`, `SPEC-git-pr.md` (resume routing, idempotency declarations) <!-- R13 --> <!-- rework: mirrors restate the broken `fab change resolve {id}` mechanism for the Linear branch — update to the issues-array scan -->
- [x] T030 [P] Update `docs/specs/skills/SPEC-preamble.md` (generic failure rule) <!-- R15 -->

### Phase 4: Validation

- [x] T031 Run scoped Go tests (`go test ./cmd/fab/ ./internal/resolve/` from `src/go/fab`), then the full `go test ./...`; fix any failures <!-- R6 -->

### Phase 5: Rework cycle 1 (review findings, action: Fix code)

- [x] T032 Mirror the f100 rename guard into `src/kit/skills/fab-new.md` Step 11 Case 4 (rename only when the current branch resolves to no other change via `fab change resolve`; otherwise `git checkout -b` with the inherited-HEAD caveat), noting it is kept in sync with git-branch.md Step 4; update `docs/specs/skills/SPEC-fab-new.md` and the `docs/specs/skills.md` git-branch rollup line (~:612) <!-- R9 -->
- [x] T033 Add the timeout branch to the `src/kit/skills/fab-continue.md` review-pr dispatch row (stage deliberately left active → report and stop, no re-finish) and guard its finish with "only if still active" matching the ship row; mirror in `docs/specs/skills/SPEC-fab-continue.md` <!-- R8 -->
- [x] T034 Add the timeout outcome case to `src/kit/skills/fab-fff.md` Step 5 (review-pr deliberately left `active`; report "Review-PR pending (Copilot review requested, timed out waiting) — re-run /git-pr-review when ready" instead of "Pipeline complete."); mirror in `docs/specs/skills/SPEC-fab-fff.md` <!-- R8 -->
- [x] T035 Scope the "orchestrator owns/runs all transitions" wording to `/fab-continue`-behavior subagents (ship/review-pr remain self-managed by git-pr/git-pr-review) in the `src/kit/skills/fab-ff.md` and `src/kit/skills/fab-fff.md` Dispatch notes and the `docs/specs/skills/SPEC-fab-ff.md`/`SPEC-fab-fff.md` summaries <!-- R2 -->

## Execution Order

- T001 blocks T002; T003 blocks T004 and T005
- T006/T007 and T018 touch the same file (`fab-continue.md`) — run sequentially; T008/T010/T011 touch `fab-ff.md`/`fab-fff.md` — run sequentially
- Phase 3 mirror tasks depend on their Phase 1/2 counterparts being final
- T031 runs last (re-run after Phase 5 rework edits)
- Phase 5: T017/T029 (reworked) before their mirrors; T034/T035 both touch `fab-fff.md` — run sequentially

## Acceptance

### Functional Completeness

- [x] A-001 R1: fab-continue.md Step 1 contains no `start <change> apply` instruction; intake-ready rows say "finish intake (auto-activates apply)" and the review-fail row uses `reset <change> apply fab-continue`
- [x] A-002 R2: fab-ff.md and fab-fff.md subagent dispatch prompts (apply, review, hydrate) each contain "do NOT run `fab status` commands; return results only", and the hydrate finish is run by the orchestrator
- [x] A-003 R2: fab-continue.md Apply/Review/Hydrate behavior sections each carry the when-invoked-as-subagent skip rule, and the ship row's finish is guarded by it
- [x] A-004 R3: _review.md Findings Merge step 4 states "no must-fix findings (including zero findings) → review passes" with no "may pass" hedge
- [x] A-005 R4: fab-ff.md:52 / fab-fff.md:52 condition reads "if `progress.intake` is not `done`, finish intake"
- [x] A-006 R5: fab-continue Step 1 and ff/fff Resumability include the `progress.review == failed → fab status start <change> review` guard
- [x] A-007 R6: fabhelp.go renders `intake → apply → review → hydrate → ship → review-pr`; skillToGroupMap maps fab-proceed, fab-operator, git-branch, git-pr-review
- [x] A-008 R7: resolve.go zero-candidate and missing-dir errors include the `/fab-new` + `/fab-switch` guidance
- [x] A-009 R8: git-pr-review.md Steps 1/2/4 route terminal paths to Step 6; Step 6 declares itself the single exit point and has a fourth timeout outcome that leaves the stage active
- [x] A-010 R9: git-branch.md Step 4 renames only when the current branch resolves to no other change; otherwise creates a new branch
- [x] A-011 R10: fab-operator.md tick step 2 dedups against `known` plus `completed`
- [x] A-012 R11: fab-setup.md step 1a trigger includes missing `project.name`/`project.description`; Config Create Mode preserves an existing `fab_version` key; pre-flight reference consistent
- [x] A-013 R12: fab-status.md over-threshold Impact line spec uses ⚠️ prefix + bold; no ANSI escape mandate remains
- [x] A-014 R13: fab-new.md and fab-draft.md Step 3 route existing-ID collisions to `/fab-switch` + `/fab-continue`; error-table rows map to the recovery; NL re-run = new change is documented — **met (re-review, rework cycle 1)**: Step 3 now branches by ID type — backlog IDs keep `fab change resolve {id}`; Linear IDs scan `.status.yaml` `issues` arrays via `grep -l "{ISSUE_ID}" fab/changes/*/.status.yaml` (single-level glob excludes archive/), which can fire (issues arrays populated by `fab status add-issue`; template status.yaml + statusfile.go `Issues []string`); error tables split collision→resume vs other→stderr; SPEC mirrors match
- [x] A-015 R14: fab-continue.md Hydrate step 4 includes the check-then-update-in-place rule; Key Properties idempotency row covers hydrate
- [x] A-016 R15: _preamble.md Key behaviors includes the generic non-zero-exit STOP rule with the per-skill carve-out
- [x] A-017 R16: fab-new.md, fab-draft.md, git-pr.md each have a Key Properties section declaring their re-run contract

### Behavioral Correctness

- [x] A-018 R6: fabhelp_test.go asserts the six-stage pipeline string and the four new map entries; tests pass
- [x] A-019 R7: resolve_test.go covers both changed error paths; tests pass; only the affected _cli-fab.md error rows changed (full table regen deferred)
- [x] A-020 R8: the Copilot-timeout path can no longer be classed as "no reviews" → finish; the stage remains re-runnable

### Scenario Coverage

- [x] A-021 R2: tracing a full /fab-ff run through the edited files yields exactly one `finish` per stage, all driven by the orchestrator
- [x] A-022 R13: tracing `/fab-new <existing-backlog-id>` lands on resume guidance without surfacing `Change ID already in use`

### Edge Cases & Error Handling

- [x] A-023 R5: the interrupted fail→reset window (review `failed`, apply `done`) has a documented recovery path in all three orchestrators
- [x] A-024 R9: standalone/wt-create branches (resolving to no change) still take the rename path; only branches matching another change divert to checkout -b
- [x] A-025 R15: the four skills that intentionally branch on non-zero fab exits remain exempt via the carve-out wording

### Code Quality

- [x] A-026 Pattern consistency: skill edits follow existing file conventions (table forms, bold/emoji conventions, RFC 2119 usage); Go edits follow existing error/string and test patterns
- [x] A-027 No unnecessary duplication: shared rules stated once and referenced (e.g., when-subagent rule, generic failure rule in _preamble), not copy-pasted divergently
- [x] A-028 Readability over cleverness: one-line surgical edits preferred; no restructuring beyond the findings' scope (code-quality.md Principles)
- [x] A-029 No magic strings: Go error guidance reuses the exact skill names (/fab-new, /fab-switch) consistent with resolve.go's existing multi-candidate hint (code-quality.md Anti-Patterns)

### Documentation Accuracy

- [x] A-030 R1: every edited skill's SPEC-*.md mirror reflects the changed behavior (fab-continue, fab-ff, fab-fff, _review, _preamble, git-pr-review, git-pr, git-branch, fab-operator, fab-setup, fab-status, fab-new, fab-draft)
- [x] A-031 R3: docs/specs/skills.md flagged lines (~:298 dispatch wording, ~:504 pass rule) match the edited skill text

### Cross References

- [x] A-032 R7: _cli-fab.md error rows match the new resolve.go strings verbatim
- [x] A-033 R2: no remaining text in fab-ff.md/fab-fff.md instructs a subagent to run `fab status finish`; cross-file references (fab-continue ↔ git-pr ship finish) are consistent

### Rework cycle 1 (added during Fix code)

- [x] A-034 R9: fab-new.md Step 11 Case 4 carries the same rename guard as git-branch.md Step 4 (rename only when the current branch resolves to no other change; otherwise checkout -b with the inherited-HEAD caveat); SPEC-fab-new.md and the skills.md git-branch rollup line reflect the guard
- [x] A-035 R8: fab-continue.md review-pr dispatch row and fab-fff.md Step 5 handle the timeout outcome — stage deliberately left active, "Review-PR pending" reported instead of "Pipeline complete.", finish guarded by "only if still active"; SPEC mirrors match
- [x] A-036 R2: the ownership wording in fab-ff.md/fab-fff.md Dispatch notes and SPEC-fab-ff.md/SPEC-fab-fff.md summaries is scoped to `/fab-continue`-behavior subagents (ship/review-pr self-managed by git-pr/git-pr-review)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Findings report (read-only evidence): `/home/sahil/code/sahil87/fab-kit/docs/specs/findings/skills-review-2026-06-11.md` — line numbers are vs ae79e04c; all edits re-located by content

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | R6 group assignments: `fab-proceed` → Planning (orchestrates the pipeline like ff/fff), `fab-operator` → Maintenance (long-lived coordination utility), `git-branch` → Start & Navigate (branch setup alongside fab-new/fab-switch), `git-pr-review` → Completion (beside git-pr) | Intake assumption 10 delegates exact groups to existing map semantics; these are the closest semantic fits and trivially reversible | S:60 R:90 A:80 D:70 |
| 2 | Confident | R7 also updates the `No active changes found` row's wording only if its string changes — it does not (resolve.go:95 untouched), so only the bare `No active change.` rows gain a new row/fix text in _cli-fab.md | Intake scopes _cli-fab edits to "rows whose strings f124 changes"; resolve.go:95 string is unchanged | S:80 R:90 A:90 D:85 |
| 3 | Confident | R8 outcome label set is {success, failure, no-reviews, timeout}; Step 2's clean-finish stops ("reviews but no inline comments", "no automated reviewer") map to the no-reviews (finish) outcome, preserving today's finish semantics for those paths | Intake names three outcome labels plus the new timeout class; mapping the existing clean-finish paths to no-reviews keeps f016's fix scoped to the timeout path only | S:70 R:85 A:85 D:80 |
| 4 | Confident | R2 ship-row guard phrasing: the fab-continue ship dispatch row notes git-pr finishes ship internally and instructs finish only if the stage is still active (covered by the same when-subagent/guarded phrasing the intake prescribes) | Intake: "ship re-finish ... covered by the same when-subagent rule / guarded phrasing" | S:75 R:85 A:85 D:80 |
| 5 | Certain | Memory files are NOT written in this change — `## Affected Memory` is hydrate-stage work | Orchestrator instruction + pipeline contract | S:95 R:95 A:95 D:95 |
| 6 | Confident | R13 Linear-branch detection mechanism (rework cycle 1): scan non-archived changes' `.status.yaml` `issues` arrays via `grep -l "{ISSUE_ID}" fab/changes/*/.status.yaml` — the single-level glob naturally excludes `archive/{name}/`; R13's requirement text never prescribed the broken `fab change resolve` mechanism for Linear, so only the skill/mirror text changes | Reviewer's must-fix gives the grep as an example ("e.g."); exact command form is a judgment call; folder-prefix resolve stays correct for backlog IDs | S:80 R:90 A:85 D:85 |

6 assumptions (1 certain, 5 confident, 0 tentative).
