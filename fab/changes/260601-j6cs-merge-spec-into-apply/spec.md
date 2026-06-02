# Spec: Merge Spec Stage into Apply, Frontload SRAD to Intake

**Change**: 260601-j6cs-merge-spec-into-apply
**Created**: 2026-06-01
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`, `planning-skills.md`, `execution-skills.md`, `clarify.md`, `schemas.md`, `configuration.md`, `migrations.md`, `templates.md`, `kit-architecture.md`

## Non-Goals

- Removing or merging the `hydrate` stage — it stays (Constitution II: memory is source of truth). Pipeline goes 7→6, not 7→5.
- Renaming any artifact file (`plan.md` stays `plan.md`; no `apply.md`).
- Changing the `review`, `ship`, or `review-pr` stage behavior beyond their spec.md/stage-list references.
- Introducing a new runtime "bounce-back" mechanism — the intake gate is the guard (see R-GATE-3).
- Per-type divergence of the intake gate threshold — it is flat 3.0 for all types this change (the per-type *map* is retained for future use).

## Stage Model: Pipeline Shape

### Requirement: R-STAGE-1 — Six-stage pipeline
The pipeline SHALL consist of exactly six stages in order: `intake → apply → review → hydrate → ship → review-pr`. The `spec` stage MUST be removed from `StageOrder`, `AllowedStates`, `stageTransitions`, and the `status.yaml` template `progress` block.

#### Scenario: StageOrder reflects six stages
- **GIVEN** the `fab` binary built from this change
- **WHEN** `StageOrder` is enumerated
- **THEN** it equals `[intake, apply, review, hydrate, ship, review-pr]` with no `spec` entry
- **AND** `StageNumber("apply") == 2`, `NextStage("intake") == "apply"`

#### Scenario: A fresh change starts with a spec-free progress map
- **GIVEN** a new change created via `fab change new`
- **WHEN** its `.status.yaml` is read
- **THEN** the `progress` map contains keys for the six stages only, with no `spec` key

### Requirement: R-STAGE-2 — `spec` stage events hard-error
Any `fab status` event (`start`/`advance`/`finish`/`reset`/`skip`/`fail`) targeting the `spec` stage SHALL return a non-zero exit and a deprecation message mirroring the existing `tasks` branch in `validateStage`.

#### Scenario: finishing the removed spec stage errors
- **GIVEN** any change
- **WHEN** `fab status finish <change> spec` is run
- **THEN** the command exits non-zero
- **AND** prints a message of the form `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".`

### Requirement: R-STAGE-3 — Orphan `progress.spec` keys are tolerated
The binary SHALL NOT error when loading a `.status.yaml` that still carries a `progress.spec` key (un-migrated file). `Validate()` MUST skip the unknown key; `GetProgressMap()` MUST omit it; a subsequent `Save` MAY preserve the orphan key verbatim (raw-node passthrough). Only the migration removes it.

#### Scenario: un-migrated file loads and validates
- **GIVEN** a `.status.yaml` with `progress.spec: done` and the six other stages
- **WHEN** the new binary runs `Validate()` on it
- **THEN** no error is returned
- **AND** `CurrentStage`/`DisplayStage` derive correctly from the six known stages

## Confidence Gate & Scoring

### Requirement: R-GATE-1 — Single intake gate at flat 3.0
There SHALL be exactly one confidence gate, evaluated at intake before the pipeline proceeds. Its threshold MUST be 3.0 for all seven change types. The separate spec gate MUST be removed.

#### Scenario: intake gate uses per-type lookup returning 3.0
- **GIVEN** a change of any type
- **WHEN** `fab score --check-gate --stage intake <change>` runs
- **THEN** the threshold compared against is 3.0
- **AND** `CheckGate`'s intake branch obtains it via `getGateThreshold(changeType)` (not a hardcoded literal), so future per-type divergence is a data-only change

### Requirement: R-GATE-2 — `intake.md` is the sole scoring source; `indicative` retired
`fab score` SHALL default to `--stage intake` and read `intake.md`. The `confidence.indicative` flag MUST no longer be written. The binary MUST still tolerate reading a legacy `indicative: true` key without error (decode-tolerant), and MUST NOT spuriously strip it in a way that errors.

#### Scenario: score writes no indicative flag
- **GIVEN** a change with an `intake.md`
- **WHEN** `fab score --stage intake <change>` runs
- **THEN** `.status.yaml` `confidence` is updated
- **AND** no `indicative` key is written by this code path

#### Scenario: legacy indicative key tolerated on read
- **GIVEN** a `.status.yaml` carrying `confidence.indicative: true`
- **WHEN** the new binary loads it
- **THEN** no error occurs and the rest of the confidence block decodes normally

### Requirement: R-GATE-3 — The intake gate is the only "bounce" guard
There SHALL be no runtime mechanism inside apply that detects an SRAD Unresolved and resets to intake. Instead, a change whose intake fails the gate MUST be prevented from advancing: with `intake` not `done`, orchestrators that gate on `fab score --check-gate` MUST refuse to enter apply. The SRAD Critical Rule (Unresolved must be asked/bailed) applies at intake-time skills only (`/fab-new`, `/fab-clarify`).

#### Scenario: a sub-threshold intake blocks the pipeline (non-force)
- **GIVEN** a change whose intake scores below 3.0
- **WHEN** `/fab-fff` or `/fab-ff` runs without `--force`
- **THEN** the pipeline stops at the intake gate and does not enter apply

### Requirement: R-SCORE-1 — Single `expectedMin` table
`getExpectedMin` SHALL use a single `expectedMin` map seeded with `feat:7, refactor:6, fix:5` and a default of `3` for types without an explicit entry (`docs`, `test`, `ci`, `chore`). The separate `expectedMinIntake` map MUST be deleted.

#### Scenario: expected_min for refactor is 6
- **GIVEN** a change of type `refactor`
- **WHEN** scoring computes the `cover` factor
- **THEN** `expectedMin` is 6
- **AND** a `docs` change uses the default 3

### Requirement: R-SCORE-2 — `docs/specs/change-types.md` reconciled to Go values
The `expected_min` and gate-threshold tables in `docs/specs/change-types.md` SHALL be rewritten to match the Go source of truth (single intake `expectedMin` feat:7/refactor:6/fix:5/default-3; gate flat 3.0). Pre-existing code↔doc drift MUST be eliminated, not perpetuated.

#### Scenario: doc matches code
- **GIVEN** the updated `change-types.md`
- **WHEN** its `expected_min` table is compared to `score.go`
- **THEN** the values are identical

## Artifacts: spec.md Absorption

### Requirement: R-ART-1 — `spec.md` absorbed into `plan.md`
The `spec.md` template (`src/kit/templates/spec.md`) MUST be removed. Its requirement discipline (RFC-2119 + GIVEN/WHEN/THEN) SHALL live as a `## Requirements` section in `plan.md`. The canonical artifact set becomes `intake.md → plan.md → code`.

#### Scenario: apply entry produces a unified plan.md
- **GIVEN** a change at apply entry with no `plan.md`
- **WHEN** the Plan Generation Procedure runs
- **THEN** `plan.md` contains `## Requirements`, `## Tasks`, and `## Acceptance` sections
- **AND** no separate `spec.md` is created

### Requirement: R-ART-2 — No `[NEEDS CLARIFICATION]` in `plan.md`
The `## Requirements` section of `plan.md` MUST NOT contain `[NEEDS CLARIFICATION]` markers. Under-specified requirements encountered at apply SHALL be recorded as graded SRAD assumptions in `plan.md`'s `## Assumptions` section instead. `[NEEDS CLARIFICATION]` markers are an intake-only construct.

#### Scenario: ambiguity becomes an assumption, not a marker
- **GIVEN** the apply agent generating `## Requirements` encounters an under-specified point
- **WHEN** it resolves the point
- **THEN** it records a graded assumption (Certain/Confident/Tentative) in `## Assumptions`
- **AND** does not emit a `[NEEDS CLARIFICATION]` marker into `plan.md`

### Requirement: R-ART-3 — Plan template scrubbed of spec.md references
The `plan.md` template's `**Spec**: \`spec.md\`` frontmatter line and any Acceptance-derivation comments citing "spec.md" SHALL be removed or repointed to the in-file `## Requirements` section.

#### Scenario: template has no dangling spec.md reference
- **GIVEN** the updated `plan.md` template
- **WHEN** it is searched for `spec.md`
- **THEN** there are no remaining references to a `spec.md` file

### Requirement: R-TRACE-1 — Traceability annotations are REQUIRED
Each `## Tasks` item SHALL carry a `<!-- R# -->` trace annotation referencing the requirement it implements; each `## Acceptance` item SHALL name the requirement it accepts (e.g., `A-001 R2: {outcome}`). Cross-linking changes from OPTIONAL (current `_generation.md`) to REQUIRED.

#### Scenario: every task traces to a requirement
- **GIVEN** a generated `plan.md`
- **WHEN** its `## Tasks` items are inspected
- **THEN** each carries an `<!-- R# -->` annotation naming an existing requirement

## Go Binary

### Requirement: R-GO-1 — Score hook and artifact matcher drop spec.md
The PostToolUse score hook (`cmd/fab/hook.go`) MUST remove its `case "spec.md"` branch, and `internal/hooklib/artifact.go` `MatchArtifactPath` MUST drop `"spec.md"` from its recognized-artifact switch, in lockstep. This prevents a stray/leftover `spec.md` edit from firing `score.Compute(...,"spec")` and silently overwriting the authoritative intake confidence.

#### Scenario: writing a leftover spec.md does not trigger scoring
- **GIVEN** the new binary and a leftover `spec.md` in a change folder
- **WHEN** that `spec.md` is written/edited (PostToolUse fires)
- **THEN** `MatchArtifactPath` does not match it and no `score.Compute(...,"spec")` runs
- **AND** the change's `.status.yaml` confidence is unchanged

### Requirement: R-GO-2 — `set-confidence` indicative param/flag retired
`SetConfidence`/`SetConfidenceFuzzy` (in `internal/status/status.go`) SHALL drop their `indicative` parameter. The `--indicative` CLI flag on `fab status set-confidence`/`set-confidence-fuzzy` MAY be retained as an accepted-but-ignored no-op for one release (back-compat) but MUST NOT cause `indicative: true` to be written.

#### Scenario: set-confidence writes no indicative flag
- **GIVEN** the new binary
- **WHEN** `fab status set-confidence <change> ... <score>` runs (with or without `--indicative`)
- **THEN** `.status.yaml` gains no `indicative` key

### Requirement: R-GO-3 — `fab change list` output ABI updated deliberately
The `fab change list` row formatter (`internal/change/change.go`) emits a positional row `name:display_stage:display_state:score:indicative`. Retiring indicative SHALL update this output contract; `fab-switch.md`'s parser and the corresponding test MUST be updated in the same change. The decision (drop the 5th field vs. keep-always-empty) MUST be explicit.

#### Scenario: list output and switch parser agree
- **GIVEN** the new binary and updated `fab-switch.md`
- **WHEN** `fab change list` output is parsed by the fab-switch listing flow
- **THEN** the field count and positions match; no parse error

### Requirement: R-GO-4 — Go tests updated to the six-stage model
All Go tests asserting the seven-stage pipeline or the `spec` stage SHALL be updated, including (at minimum) `status_test.go`, `statusfile_test.go`, `preflight_test.go`, `log_test.go`, `change_test.go`, `hooklib/artifact_test.go`, `hook_test.go`, `true_impact_test.go`. `go test ./...` MUST pass.

#### Scenario: the suite is green
- **GIVEN** the implemented change
- **WHEN** `go test ./...` runs from `src/go/fab`
- **THEN** all tests pass with no references to a `spec` stage in assertions

## Skills

### Requirement: R-SKILL-1 — Spec generation merged into plan generation
`_generation.md` SHALL delete the standalone Spec Generation Procedure and fold requirement generation into the Plan Generation Procedure (one walk emitting `## Requirements` + `## Tasks` + `## Acceptance`). The procedure MUST read intake (not a separate spec.md) as input, and MUST include a one-release legacy `spec.md` ingestion path (fold a leftover `spec.md` into `## Requirements` if present and `plan.md` lacks them).

#### Scenario: plan generation reads intake, emits requirements
- **GIVEN** a change with `intake.md` and no `spec.md`
- **WHEN** the Plan Generation Procedure runs
- **THEN** `## Requirements` is generated from intake-derived design
- **AND** the procedure does not require a `spec.md` file to exist

### Requirement: R-SKILL-2 — Orchestrators drop spec step and both auto-clarifies
`fab-ff.md` and `fab-fff.md` MUST remove the standalone spec generation step, BOTH `/fab-clarify [AUTO-MODE]` invocations (post-spec and on-plan), and the spec gate. Their hardcoded `>= 3.0` gate strings and `/fab-clarify spec|plan` recovery hints SHALL be updated. The deepest rework tier "Revise spec → reset to spec stage" SHALL be redefined as "Revise requirements → edit `plan.md` `## Requirements` + downstream, re-run apply", keeping the rework tiers distinct.

#### Scenario: the orchestrator bracket runs without any clarify invocation
- **GIVEN** `fab-fff` post-change
- **WHEN** it runs the automated bracket
- **THEN** it dispatches no `/fab-clarify` subagent at any stage

### Requirement: R-SKILL-3 — `/fab-clarify` becomes intake-only
`fab-clarify.md` SHALL accept only the `intake` target. The `spec` and `plan` targets MUST be removed. The recompute-confidence step MUST be inverted: instead of skipping at intake, it MUST always run `fab score --stage intake <change>`.

#### Scenario: clarify rejects a plan target
- **GIVEN** the post-change `/fab-clarify`
- **WHEN** invoked with target `plan`
- **THEN** it does not operate on `plan.md` (target unsupported)
- **AND** invoked with no target at the intake stage, it scans `intake.md` and recomputes the intake score

### Requirement: R-SKILL-4 — Dependent skill/preamble references updated
`_preamble.md` (State Table, Confidence Scoring section ~L483–551, Skill Invocation Protocol auto-clarify mappings, Skill-Specific Autonomy Levels recompute cell, Context Loading, Memory File Lookup, Assumptions Summary), `fab-continue.md` (dispatch table, reset flow, scoring step, `tasks`-deprecation strings, rework tier), `git-pr.md` (stage list at ~L249, `{has_spec}`/Spec-URL removal), `fab-new.md`/`fab-draft.md` (Step 7 rename + indicative removal), `fab-status.md` (`(1/7)`→`(1/6)`), `fab-operator.md`, and `_cli-fab.md` (finish chain, score modes, set-confidence signatures) SHALL be updated to the six-stage, single-gate, no-indicative, intake-only-clarify model.

#### Scenario: no skill references a spec stage as live
- **GIVEN** all updated skill sources
- **WHEN** searched for spec-stage references (state-table rows, `finish ... spec`, `/fab-clarify spec`)
- **THEN** none remain except deprecation/error strings

## Docs & Memory

### Requirement: R-DOC-1 — Specs updated to six-stage model
`docs/specs/` files SHALL be updated: `overview.md`, `srad.md`, `change-types.md`, `skills.md`, `architecture.md`, `glossary.md`, `templates.md`, `user-flow.md`, plus the non-skill-coupled specs `SPEC-hooks.md`, `SPEC-preamble.md`, `SPEC-_review.md`, `assembly-line.md`, and `index.md`.

#### Scenario: SPEC-hooks no longer documents a spec.md score hook
- **GIVEN** the updated `SPEC-hooks.md`
- **WHEN** searched for the `spec.md → fab score` hook rule and the `intake.md|spec.md|plan.md` matcher
- **THEN** the spec.md hook entry is removed and the matcher lists `intake.md|plan.md`

### Requirement: R-DOC-2 — Memory updated at hydrate
At the hydrate stage, the affected memory files (`change-lifecycle`, `planning-skills`, `execution-skills`, `clarify`, `schemas`, `configuration`, `migrations`, `templates`, `kit-architecture`) SHALL be updated to reflect the six-stage model, single gate, spec.md absorption, and indicative retirement.

#### Scenario: memory reflects six stages
- **GIVEN** post-hydrate memory
- **WHEN** `change-lifecycle.md` is read
- **THEN** it describes six stages and the `intake.md → plan.md → code` artifact flow

## Migration

### Requirement: R-MIG-1 — Idempotent, archive-safe migration `1.9.7-to-1.10.0`
A migration file `src/kit/migrations/1.9.7-to-1.10.0.md` SHALL be created, shipped with the `src/kit/VERSION` bump to `1.10.0` in the same change. It MUST walk in-flight `fab/changes/**` excluding `archive/**`, be idempotent on re-run, and never touch archived changes.

#### Scenario: re-run is a no-op
- **GIVEN** a project already migrated
- **WHEN** the migration runs again
- **THEN** every change hits the idempotency sentinel and nothing is mutated

### Requirement: R-MIG-2 — Four-state spec.md→plan.md handling
The migration MUST handle four per-change states: (1) spec.md only — leave spec.md for on-apply ingestion, do NOT create a plan.md stub; (2) plan.md only — progress rewrite only; (3) both — merge spec.md body into plan.md `## Requirements` (annotate `<!-- migrated from spec.md -->`), leave spec.md with a "safe to delete" comment; (4) neither — progress rewrite only. Idempotency sentinel: skip the merge if plan.md already has a `## Requirements` heading or the migration marker.

#### Scenario: mid-spec change is not stubbed
- **GIVEN** an in-flight change with `spec.md` and no `plan.md`
- **WHEN** the migration runs
- **THEN** no `plan.md` stub is created (avoiding the resumability skip-guard deadlock)
- **AND** `spec.md` remains for the new Plan Generation Procedure to ingest on first apply

### Requirement: R-MIG-3 — Progress, directives, and indicative handled
The migration MUST drop `progress.spec` and fold its state into `apply` (active/ready → carry level; done/skipped → leave apply). It MUST relocate the four `stage_directives.spec` directives into `stage_directives.apply` (not silently drop them). It SHALL leave any `confidence.indicative` key on disk (binary tolerates it).

#### Scenario: spec directives relocated, not lost
- **GIVEN** a project config with four `stage_directives.spec` entries
- **WHEN** the migration runs
- **THEN** those entries appear under `stage_directives.apply`
- **AND** `stage_directives.spec` is removed

## Constitution

### Requirement: R-CONST-1 — Governance note
The constitution `Last Amended` date SHALL be bumped and a short rationale note recorded for the stage-model change. Whether a new normative clause is added is deferred (Open Question) — at minimum the amendment is dated and explained.

#### Scenario: constitution records the amendment
- **GIVEN** the updated `constitution.md`
- **WHEN** its Governance block is read
- **THEN** `Last Amended` reflects this change's date

## Deprecated Requirements

### Spec stage as a distinct pipeline stage
**Reason**: Empirical evidence (loom: spec median 2 min, ~1% rework, 32% of clarify) shows the spec stage is a near-pass-through; its requirement-capture work is absorbed into apply-entry.
**Migration**: `progress.spec` folded into `apply`; `spec.md` content folded into `plan.md` `## Requirements`; `fab status ... spec` hard-errors.

### Two-gate confidence model (intake fixed-3.0 + per-type spec gate)
**Reason**: With one manual stage (intake), a single gate suffices; the second gate scored an artifact (`spec.md`) that no longer exists as a stage boundary.
**Migration**: Single intake gate at flat 3.0 (≥ every old gate); spec gate removed.

### `confidence.indicative` flag
**Reason**: Intake scoring becomes authoritative, not indicative; the flag's distinction (indicative vs. spec-authoritative) is meaningless with one scoring source.
**Migration**: No longer written; tolerated on read; left on disk in un-migrated/archived files.

## Design Decisions

1. **Absorb `spec.md` into `plan.md` rather than keep it as a separate hidden file**
   - *Why*: After the gate moves to intake, nothing reads `spec.md` programmatically (verified: score.Compute/CheckGate, hook.go, hooklib/artifact.go, git-pr.md — all removed/repointed). A separate file would be generated, never machine-read, hidden — dead weight. One-pass co-generation is the strongest alignment guarantee.
   - *Rejected*: Keeping `spec.md` as a separate apply-entry artifact — reintroduces the seam the merge removes and leaves an unread file.

2. **Flat 3.0 intake gate for all types (not per-type 2.0/3.0)**
   - *Why*: The old model gated intake at a fixed 3.0 AND had a per-type spec gate. Collapsing to one gate at flat 3.0 keeps every type's bar ≥ both old gates — no silent relaxation. The per-type *map* is retained so future divergence is a one-line data change.
   - *Rejected*: Per-type single gate (fix/docs/etc. at 2.0) — would relax the entry bar for 5 of 7 types, presented as no-loss when it isn't.

3. **The intake gate IS the bounce-back valve; apply has no bail logic**
   - *Why*: A failing intake never reaches `done`, so gate-checking orchestrators can't enter apply. No new runtime mechanism needed; the SRAD Critical Rule already governs intake-time skills.
   - *Rejected*: A runtime "apply detects Unresolved → reset to intake" mechanism — unbuilt, unspecified, and redundant with the existing gate.

4. **Accept the lost independent assumption re-grade; compensate via the intake gate + review**
   - *Why*: The old spec stage re-graded intake assumptions (`_generation.md` step 6). One-pass co-generation loses this. Compensated by the flat-3.0 gate (≥ old bars), the ~1% spec-rework loom evidence, and requirement-correctness still caught at review.
   - *Rejected*: Adding an independent re-grade step at apply entry — adds back ceremony the merge is trying to remove; deferred unless review evidence warrants.

## Assumptions

<!-- SCORING SOURCE: fab score reads only this table. Grades reconciled with the SRAD
     composite formula (0.25·S + 0.30·R + 0.25·A + 0.20·D); Certain ≥85, Confident 60–84.
     Low Reversibility (45–70) on a cascading structural change is an honest signal, not a
     labeling error — most rows are Confident even when the decision is user-confirmed. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Six-stage pipeline; `spec` removed from StageOrder/AllowedStates/template; `spec` events hard-error | Confirmed from intake #1/#7; precedent tasks→apply. Composite ~82 (low R: cascades widely) | S:95 R:60 A:90 D:90 |
| 2 | Certain | Change type is `refactor` | Confirmed from intake #2. Composite 94.3 | S:98 R:90 A:95 D:95 |
| 3 | Confident | Single intake gate at flat 3.0 for all 7 types (≥ every old gate) | Confirmed from intake #3, refined to flat 3.0. Composite ~81 | S:95 R:55 A:90 D:90 |
| 4 | Confident | Stop writing `confidence.indicative` (tolerate on read); intake scoring authoritative | Confirmed from intake #4. Composite ~79 | S:90 R:55 A:88 D:88 |
| 5 | Certain | `fab score` default `--stage` → `intake`; spec.md read paths removed | Confirmed from intake #5. Composite 88.0 | S:92 R:80 A:92 D:90 |
| 6 | Confident | Idempotent migration `1.9.7-to-1.10.0` with four-state case table + VERSION bump | Confirmed/upgraded from intake #6; review-hardened. Composite ~72 | S:88 R:50 A:88 D:82 |
| 7 | Confident | Hard deprecation error for `spec` events, mirroring `tasks` | Confirmed from intake #7. Composite ~75 | S:78 R:65 A:85 D:78 |
| 8 | Confident | `spec.md` absorbed into `plan.md` `## Requirements`; spec.md template removed; full consumer set repointed | Confirmed from intake #12; consumer set verified by adversarial review (score/hook/artifact/git-pr). Composite ~80 | S:95 R:50 A:90 D:92 |
| 9 | Confident | No `[NEEDS CLARIFICATION]` in plan.md; under-spec → SRAD assumption | Confirmed from intake #9/#10; resolves §1a-vs-§4 contradiction. Composite ~78 | S:88 R:55 A:88 D:90 |
| 10 | Confident | Score hook + artifact matcher drop spec.md (corruption-path fix) | New (review finding #4, must-fix); verified hook.go:270 + artifact.go:80. Composite ~78 | S:90 R:60 A:85 D:82 |
| 11 | Confident | `fab change list` `:indicative` ABI updated with fab-switch parser + test in lockstep | New (review must-fix); verified change.go:289-293, fab-switch.md:97, change_test.go:339. Composite ~76 | S:88 R:55 A:82 D:80 |
| 12 | Confident | Orchestrators drop spec step + BOTH auto-clarifies; "Revise spec" tier → "Revise requirements" | Confirmed/expanded from intake #9; both ff/fff:58 and :66 verified. Composite ~74 | S:82 R:55 A:80 D:80 |
| 13 | Confident | `/fab-clarify` intake-only; recompute guard inverted to always score intake | Confirmed from intake #9; verified fab-clarify.md:172-174 skip-at-intake guard. Composite ~76 | S:85 R:55 A:85 D:82 |
| 14 | Confident | Trace annotations REQUIRED (`<!-- R# -->` on tasks; `R#` on acceptance) | Confirmed from intake #14. Composite ~75 | S:80 R:60 A:82 D:78 |
| 15 | Confident | Single `expectedMin` (feat:7/refactor:6/fix:5/default-3); change-types.md reconciled to Go | Confirmed from intake #15; review surfaced the docs/test/ci/chore fallback + doc drift. Composite ~74 | S:90 R:50 A:78 D:85 |
| 16 | Confident | Relocate `stage_directives.spec` (4 real directives) into `apply`, not drop | New (review should-fix #20); diverges from empty-`tasks` precedent. Composite ~73 | S:82 R:55 A:78 D:78 |
| 17 | Confident | Accept lost independent re-grade; compensated by flat-3.0 gate + ~1% rework evidence + review catch | Confirmed from intake #17; design tradeoff documented in Why. Composite ~74 | S:80 R:55 A:80 D:75 |
| 18 | Certain | Artifacts not renamed (`plan.md` stays) | Confirmed from intake #13. Composite 87.9 | S:92 R:80 A:90 D:92 |

18 assumptions (3 certain, 15 confident, 0 tentative, 0 unresolved).
