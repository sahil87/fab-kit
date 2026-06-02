# Plan: Merge Spec Stage into Apply, Frontload SRAD to Intake

**Change**: 260601-j6cs-merge-spec-into-apply
**Status**: In Progress
**Intake**: `intake.md`

<!--
  AUTO-GENERATED at apply entry. This change is meta: it rewrites the pipeline
  that produces this very file. Requirements below are condensed from the
  authoritative spec.md (which has a 52-finding adversarial review folded in);
  R-IDs are preserved verbatim from spec.md for traceability. Each ## Tasks item
  carries a <!-- R-ID --> trace annotation; each ## Acceptance item names its R-ID.
-->

## Requirements

> Condensed from `spec.md` (authoritative). RFC-2119 keywords retained. R-IDs match spec.md.

### Stage Model

- **R-STAGE-1** — Pipeline SHALL be exactly six stages: `intake → apply → review → hydrate → ship → review-pr`. `spec` MUST be removed from `StageOrder`, `AllowedStates`, `stageTransitions`, and the `status.yaml` template `progress` block. `StageNumber("apply") == 2`, `NextStage("intake") == "apply"`.
- **R-STAGE-2** — Any `fab status` event targeting `spec` SHALL return non-zero with a deprecation message mirroring the `tasks` branch: `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".`
- **R-STAGE-3** — The binary SHALL NOT error loading a `.status.yaml` with an orphan `progress.spec` key. `Validate()` skips it; `GetProgressMap()` omits it; Save MAY preserve it (raw-node passthrough). Only the migration removes it.

### Confidence Gate & Scoring

- **R-GATE-1** — Exactly one confidence gate, at intake, threshold 3.0 for all seven types. The spec gate MUST be removed. `CheckGate`'s intake branch obtains the threshold via `getGateThreshold(changeType)` (not a literal).
- **R-GATE-2** — `fab score` SHALL default to `--stage intake` and read `intake.md`. `confidence.indicative` MUST no longer be written; legacy `indicative: true` MUST be tolerated on read.
- **R-GATE-3** — No runtime mechanism inside apply resets to intake on SRAD Unresolved. The intake gate is the only "bounce" guard; the SRAD Critical Rule applies at intake-time skills only.
- **R-SCORE-1** — `getExpectedMin` SHALL use a single `expectedMin` map (`feat:7, refactor:6, fix:5`, default `3`). `expectedMinIntake` MUST be deleted.
- **R-SCORE-2** — `docs/specs/change-types.md` `expected_min`/gate tables SHALL be rewritten to match Go (single intake `expectedMin`; gate flat 3.0). Pre-existing drift eliminated.

### Artifacts: spec.md Absorption

- **R-ART-1** — `spec.md` template MUST be removed. Requirement discipline (RFC-2119 + GIVEN/WHEN/THEN) lives as `## Requirements` in `plan.md`. Canonical artifact set: `intake.md → plan.md → code`.
- **R-ART-2** — `plan.md`'s `## Requirements` MUST NOT contain `[NEEDS CLARIFICATION]`. Under-spec at apply → graded SRAD assumption in `## Assumptions`.
- **R-ART-3** — `plan.md` template `**Spec**: spec.md` line and Acceptance-derivation comments citing spec.md SHALL be removed/repointed to `## Requirements`.
- **R-TRACE-1** — Trace annotations REQUIRED: each `## Tasks` item carries `<!-- R# -->`; each `## Acceptance` item names its `R#` (e.g., `A-001 R2: outcome`).

### Go Binary

- **R-GO-1** — `cmd/fab/hook.go` MUST remove its `case "spec.md"` branch; `internal/hooklib/artifact.go` `MatchArtifactPath` MUST drop `"spec.md"`, in lockstep.
- **R-GO-2** — `SetConfidence`/`SetConfidenceFuzzy` SHALL drop their `indicative` parameter. The `--indicative` CLI flag MAY be retained as an accepted-but-ignored no-op but MUST NOT write `indicative: true`.
- **R-GO-3** — `fab change list` output ABI (`name:display_stage:display_state:score:indicative`) SHALL be updated deliberately; `fab-switch.md`'s parser and the test MUST be updated in lockstep.
- **R-GO-4** — All Go tests asserting the seven-stage pipeline or the `spec` stage SHALL be updated. `go test ./...` MUST pass.

### Skills

- **R-SKILL-1** — `_generation.md` SHALL delete the standalone Spec Generation Procedure and fold requirement generation into Plan Generation (one walk → `## Requirements` + `## Tasks` + `## Acceptance`). Reads intake, includes a one-release legacy `spec.md` ingestion path.
- **R-SKILL-2** — `fab-ff.md`/`fab-fff.md` MUST remove the standalone spec step, BOTH `/fab-clarify [AUTO-MODE]` invocations, and the spec gate. `>= 3.0` strings and `/fab-clarify spec|plan` hints updated. "Revise spec" rework tier redefined as "Revise requirements".
- **R-SKILL-3** — `fab-clarify.md` SHALL accept only `intake`. `spec`/`plan` targets removed. Recompute step inverted: always run `fab score --stage intake`.
- **R-SKILL-4** — `_preamble.md`, `fab-continue.md`, `git-pr.md`, `fab-new.md`/`fab-draft.md`, `fab-status.md`, `fab-operator.md`, `_cli-fab.md` SHALL be updated to the six-stage, single-gate, no-indicative, intake-only-clarify model.

### Docs & Migration & Constitution

- **R-DOC-1** — `docs/specs/` updated: `overview`, `srad`, `change-types`, `skills`, `architecture`, `glossary`, `templates`, `user-flow`, `SPEC-hooks`, `SPEC-preamble`, `SPEC-_review`, `assembly-line`, `index`.
- **R-DOC-2** — At hydrate, affected memory files updated (six-stage, single gate, spec.md absorption, indicative retirement). *(Hydrate stage — NOT this apply.)*
- **R-MIG-1** — Migration `src/kit/migrations/1.9.7-to-1.10.0.md` SHALL be created, shipped with `src/kit/VERSION` bump to `1.10.0`. Walks `fab/changes/**` excluding `archive/**`, idempotent, never touches archived changes.
- **R-MIG-2** — Migration handles four per-change states (spec.md only → leave for ingestion, no stub; plan.md only → progress rewrite; both → merge into `## Requirements`; neither → progress rewrite). Idempotency sentinel: skip merge if plan.md already has `## Requirements` or the migration marker.
- **R-MIG-3** — Migration drops `progress.spec` (folding state into apply), relocates `stage_directives.spec` directives into `stage_directives.apply`, and leaves any `confidence.indicative` key on disk.
- **R-CONST-1** — Constitution `Last Amended` date bumped + short rationale note for the stage-model change. *(Hydrate/ship adjacent — handled here as a doc edit since constitution is a project file.)*

## Tasks

> Phases execute sequentially. Within a phase, `[P]`-marked tasks may run in parallel.
> Phase 1 (Go) and Phase 2 (templates/migration) must pass `go build`/`go test` before later phases.

### Phase 1: Go binary core + tests

- [x] T001 Remove `"spec"` from `StageOrder` in `src/go/fab/internal/statusfile/statusfile.go` (StageNumber/NextStage/GetProgressMap derive automatically). <!-- R-STAGE-1, R-STAGE-3 -->
- [x] T002 Remove `spec` from `AllowedStates` and `stageTransitions`; add a `spec`-deprecation branch to `validateStage` in `src/go/fab/internal/status/status.go` mirroring the `tasks` branch. <!-- R-STAGE-1, R-STAGE-2 -->
- [x] T003 In `src/go/fab/internal/status/status.go`, drop the `indicative bool` param from `SetConfidence`/`SetConfidenceFuzzy` and stop setting `Confidence.Indicative`. <!-- R-GO-2 -->
- [x] T004 In `src/go/fab/internal/statusfile/statusfile.go`, remove the `encodeConfidence` write block for `indicative` (keep the `*bool` decode-tolerant field so legacy `indicative: true` round-trips harmlessly). <!-- R-GO-2 -->
- [x] T005 In `src/go/fab/internal/change/change.go`: drop `spec` from `defaultCommand()`; update the `fab change list` row formatter to drop the `:indicative` 5th field (decision: drop, not keep-empty). Also removed the `(indicative)` display label. <!-- R-STAGE-1, R-GO-3 -->
- [x] T006 In `src/go/fab/internal/score/score.go`: replace `expectedMinIntake`/`expectedMinSpec` with a single `expectedMin` map (`feat:7, refactor:6, fix:5`, default `3`); set `gateThresholds` to flat 3.0 for all seven types; change `CheckGate` intake branch to `getGateThreshold(changeType)`; repoint `Compute`/`CheckGate` `else` branches from `spec.md` to `intake.md`; remove the `indicative := stage == "intake"` derivation and stop threading it; simplify `getExpectedMin` to intake-only. <!-- R-GATE-1, R-GATE-2, R-SCORE-1 -->
- [x] T007 In `src/go/fab/cmd/fab/score.go`: change the `--stage` flag default from `"spec"` to `"intake"`. <!-- R-GATE-2 -->
- [x] T008 In `src/go/fab/cmd/fab/hook.go`: remove the `case "spec.md"` branch in `artifactBookkeeping`. <!-- R-GO-1 -->
- [x] T009 In `src/go/fab/internal/hooklib/artifact.go`: drop `"spec.md"` from the `MatchArtifactPath` recognized-artifact switch (→ `intake.md`, `plan.md`). <!-- R-GO-1 -->
- [x] T010 In `src/go/fab/cmd/fab/status.go`: keep `--indicative` flag as accepted-but-ignored no-op on `set-confidence`/`set-confidence-fuzzy`; update calls to the new param-less `SetConfidence`/`SetConfidenceFuzzy` signatures. Also removed the `indicative:` output line from `get-confidence`. <!-- R-GO-2 -->
- [x] T011 Update Go tests to the six-stage model: status_test.go (spec→intake/apply, new spec-deprecation tests), statusfile_test.go (six stages, new orphan-spec test), preflight_test.go, log_test.go, change_test.go (4-field list ABI), hooklib/artifact_test.go (spec.md→rejected), true_impact_test.go, batch_archive_test.go, preflight.go output. <!-- R-GO-4 -->
- [x] T012 Run `go build ./...` and `go test ./...` from `src/go/fab`; ALL PASS. <!-- R-GO-4 -->

### Phase 2: templates + migration + VERSION

- [x] T020 In `src/kit/templates/status.yaml`: drop `spec: pending` from the `progress:` block. <!-- R-STAGE-1 -->
- [x] T021 Rewrite `src/kit/templates/plan.md`: removed `**Spec**: spec.md` frontmatter line; added a `## Requirements` section (RFC-2119 + GIVEN/WHEN/THEN, stable `R#` IDs, optional Non-Goals/Design Decisions/Deprecated Requirements, NO `[NEEDS CLARIFICATION]`); made trace annotations REQUIRED (`<!-- R# -->` on tasks, `R#` on acceptance); repointed Acceptance-derivation comments from spec.md to `## Requirements`. <!-- R-ART-1, R-ART-2, R-ART-3, R-TRACE-1 -->
- [x] T022 Removed the `src/kit/templates/spec.md` template file (git rm). <!-- R-ART-1 -->
- [x] T023 Created `src/kit/migrations/1.9.7-to-1.10.0.md`: idempotent, archive-safe, four-state case table, drops `progress.spec` folding into apply, relocates `stage_directives.spec` → `apply`, leaves `confidence.indicative` on disk, idempotency sentinel on `## Requirements`/migration marker. <!-- R-MIG-1, R-MIG-2, R-MIG-3 -->
- [x] T024 Bumped `src/kit/VERSION` from `1.9.7` to `1.10.0`. <!-- R-MIG-1 -->
- [x] T025 Bumped `fab/project/constitution.md` `Last Amended` to 2026-06-01 with a six-stage rationale note + Additional Constraints clause. <!-- R-CONST-1 -->

### Phase 3: skills (canonical `src/kit/skills/*.md` only)

- [x] T030 `_generation.md`: deleted the standalone Spec Generation Procedure; folded requirement generation into Plan Generation (one walk → `## Requirements` + `## Tasks` + `## Acceptance`); reads intake not spec.md; dropped "Keep the Spec link"; flipped trace annotations to REQUIRED; added a one-release legacy `spec.md` ingestion path. <!-- R-SKILL-1, R-TRACE-1 -->
- [x] T031 `_preamble.md`: dropped the `spec` State-Table row; updated Confidence Scoring section for single intake gate + no-indicative; removed both `/fab-clarify [AUTO-MODE]` mappings from "Currently Applicable"; updated Skill-Specific Autonomy "Recomputes confidence?" cell; dropped spec.md from Context Loading, Memory File Lookup, Assumptions Summary; updated Common fab Commands score row. <!-- R-SKILL-4, R-GATE-1, R-GATE-2 -->
- [x] T032 `fab-continue.md`: dropped `spec` dispatch rows (intake-ready starts apply); updated reset flow; removed the "Spec stage only" scoring step; repointed `tasks`/`spec`-deprecation strings to `/fab-continue apply` / `/fab-clarify intake`; redefined review-fail "Revise spec" → "Revise requirements"; Purpose/Arguments/Preconditions/Error-Handling six-stage. <!-- R-SKILL-4, R-STAGE-1 -->
- [x] T033 `fab-ff.md`: removed the spec generation step; deleted BOTH auto-clarify invocations; consolidated to single intake gate; redefined "Revise spec" → "Revise requirements"; updated `>= 3.0` strings and recovery hints. <!-- R-SKILL-2 -->
- [x] T034 `fab-fff.md`: same as T033 for the full pipeline. <!-- R-SKILL-2 -->
- [x] T035 `fab-clarify.md`: intake-only (dropped `spec`/`plan` targets); removed post-planning artifact-default logic; inverted the recompute guard to always run `fab score --stage intake`. <!-- R-SKILL-3 -->
- [x] T036 `_review.md`: repointed each spec.md touchpoint to `plan.md` `## Requirements`/`### Deprecated Requirements`; left `docs/specs/` "spec files" reference. <!-- R-SKILL-4 -->
- [x] T037 `_cli-fab.md`: updated finish transition chain (intake→apply); added spec-event deprecation note; `fab score` modes (single intake gate, flat 3.0); `--indicative` no-op note. <!-- R-SKILL-4, R-GATE-1 -->
- [x] T038 `git-pr.md`: "seven"→"six"; dropped `spec`; removed `{has_spec}`/Spec blob URL/`spec → Spec URL` row. <!-- R-SKILL-4 -->
- [x] T039 `fab-new.md`: renamed Step 7 to "Confidence"; dropped `indicative: true` persistence + spec-stage-overwrite sentences; updated output lines. <!-- R-SKILL-4 -->
- [x] T040 `fab-draft.md`: same indicative-section cleanup as T039. <!-- R-SKILL-4 -->
- [x] T041 `fab-status.md`: `(1/7)`→`(1/6)`, "out of 7"→"6", "Next: spec"→"Next: apply", removed indicative confidence-display variant. <!-- R-SKILL-4 -->
- [x] T042 `fab-operator.md`: Pipeline Reference six stages; example status + `stop_stage` enum drop `spec`. <!-- R-SKILL-4 -->
- [x] T043 `fab-switch.md`: list-format parser → `name:display_stage:display_state:score` (dropped `:indicative`); `({N}/8)`→`({N}/6)`; removed indicative suffix. <!-- R-GO-3 -->

### Phase 4: specs (`docs/specs/`)

- [x] T050 `docs/specs/overview.md`: stage count/table/mermaid → six stages (intake → apply → review → hydrate → ship → review-pr); artifact flow `intake.md → plan.md → code`; example workflow + quick-reference rows. <!-- R-DOC-1 -->
- [x] T051 `docs/specs/srad.md`: single intake gate (flat 3.0), single `expected_min`, lifecycle/recompute/storage updated; "after spec generation" → intake. <!-- R-DOC-1, R-GATE-1, R-GATE-2 -->
- [x] T052 `docs/specs/change-types.md`: rewrote `expected_min` table to match Go (`feat:7/refactor:6/fix:5/default-3`); gate thresholds → flat 3.0; Tier-1 PR-template "links to intake and plan"; lifecycle. <!-- R-DOC-1, R-SCORE-2 -->
- [x] T053 `docs/specs/skills.md`: artifact glossary, state table, fab-continue/ff/fff/clarify/apply/review/hydrate sections, status example → six-stage, single gate, intake-only clarify, spec.md absorption. <!-- R-DOC-1 -->
- [x] T054 `docs/specs/architecture.md`: directory tree (drop spec.md), `.status.yaml` progress examples, `stage_directives` (relocated spec directives → apply), constitution context, PR tiers. <!-- R-DOC-1, R-MIG-3 -->
- [x] T055 `docs/specs/glossary.md`: artifact/plan.md entry, Stage (6), Apply/Review/Hydrate stage numbers, skill entries, confidence gate, rework loop, markers. <!-- R-DOC-1 -->
- [x] T056 `docs/specs/templates.md`: status progress map (drop spec); folded spec.md section into a `## Requirements`-is-a-plan.md-section note; plan.md template + example carry `## Requirements` + trace annotations; hydration source → plan.md. <!-- R-DOC-1, R-ART-1 -->
- [x] T057 `docs/specs/user-flow.md`: all three mermaid diagrams + notes → six stages, intake-gated. <!-- R-DOC-1 -->
- [x] T058 `docs/specs/skills/SPEC-hooks.md`: removed the `spec.md → fab score` hook rule; matcher → `intake.md|plan.md`; dispatch tree + table. <!-- R-DOC-1, R-GO-1 -->
- [x] T059 `docs/specs/skills/SPEC-preamble.md`: context-loading layers (drop spec), confidence scoring bookkeeping → intake. <!-- R-DOC-1 -->
- [x] T060 `docs/specs/skills/SPEC-_review.md`: spec.md touchpoints → `plan.md` `## Requirements`/`### Deprecated Requirements`. <!-- R-DOC-1 -->
- [x] T061 `docs/specs/assembly-line.md`: "spec → tasks → apply" narration → six-stage. <!-- R-DOC-1 -->
- [x] T062 `docs/specs/index.md`: templates description (drop spec, add plan ## Requirements); stage count already 6. <!-- R-DOC-1 -->
- [x] T063 [extra] Coupled per-skill SPECs updated (constitution skill↔SPEC rule): SPEC-fab-ff, SPEC-fab-fff, SPEC-fab-continue, SPEC-fab-clarify, SPEC-fab-new, SPEC-fab-draft, SPEC-fab-status, SPEC-git-pr; plus superpowers-comparison.md pipeline row. <!-- R-DOC-1 -->

## Execution Order

- Phase 1 (T001–T012) before Phase 3 skill `_cli-fab`/`fab-switch` edits depend on the Go ABI decisions made in T005/T010 — keep Phases sequential.
- T005 (drop `:indicative` from list) and T043 (fab-switch parser) and T011 (change_test.go) are the lockstep ABI trio.
- T008 (hook.go) and T009 (artifact.go) are lockstep.
- T021/T022 (plan.md template + spec.md removal) before T030 (_generation merge) for consistent references.

## Acceptance

> Each item names the requirement it accepts.

### Functional Completeness

- [x] A-001 R-STAGE-1: `StageOrder == [intake, apply, review, hydrate, ship, review-pr]`; `StageNumber("apply")==2`; `NextStage("intake")=="apply"`; no `spec` anywhere in `StageOrder`/`AllowedStates`/`stageTransitions`/status.yaml template progress.
- [x] A-002 R-STAGE-2: `fab status finish <change> spec` (and start/advance/reset/skip/fail) exits non-zero with the spec-deprecation message.
- [x] A-003 R-STAGE-3: A `.status.yaml` carrying `progress.spec` loads, `Validate()` returns nil, and `GetProgressMap()` omits the orphan key.
- [x] A-004 R-GATE-1: `fab score --check-gate --stage intake <change>` compares against 3.0 for every type, obtained via `getGateThreshold`; no separate spec gate exists.
- [x] A-005 R-GATE-2: `fab score --stage intake` writes confidence with no `indicative` key; a legacy `indicative: true` decodes without error.
- [x] A-006 R-GATE-3: No apply-side Unresolved/reset-to-intake mechanism exists in the binary or skills.
- [x] A-007 R-SCORE-1: A single `expectedMin` map (`feat:7/refactor:6/fix:5/default-3`) drives `cover`; `expectedMinIntake` is gone.
- [x] A-008 R-ART-1: `src/kit/templates/spec.md` is removed; `plan.md` template carries `## Requirements`, `## Tasks`, `## Acceptance`.
- [x] A-009 R-TRACE-1: `plan.md` template tasks carry `<!-- R# -->` and acceptance items name `R#`; `_generation.md` marks cross-linking REQUIRED.
- [x] A-010 R-GO-1: `MatchArtifactPath` does not match `spec.md`; `hook.go` has no `case "spec.md"`.
- [x] A-011 R-GO-2: `SetConfidence`/`SetConfidenceFuzzy` have no `indicative` param; `--indicative` flag is accepted-but-ignored (writes no key).
- [x] A-012 R-GO-3: `fab change list` rows have the new field count; `fab-switch.md` parser and `change_test.go` match.
- [x] A-013 R-MIG-1: `src/kit/migrations/1.9.7-to-1.10.0.md` exists; `src/kit/VERSION == 1.10.0`.
- [x] A-014 R-MIG-2: Migration enumerates the four per-change states with the idempotency sentinel; the spec.md-only state creates no plan.md stub.
- [x] A-015 R-MIG-3: Migration drops `progress.spec` (folding into apply), relocates `stage_directives.spec` → `apply`, leaves `confidence.indicative` untouched.

### Behavioral Correctness

- [x] A-016 R-GATE-2: `cmd/fab/score.go` `--stage` default is `intake`.
- [x] A-017 R-SKILL-1: `_generation.md` has no standalone Spec Generation Procedure; Plan Generation emits `## Requirements` from intake and has a legacy spec.md ingestion path.
- [x] A-018 R-SKILL-2: `fab-ff.md`/`fab-fff.md` dispatch no `/fab-clarify` and have no spec gate; "Revise requirements" rework tier present.
- [x] A-019 R-SKILL-3: `fab-clarify.md` accepts only `intake` and always recomputes the intake score.
- [x] A-020 R-ART-2: `plan.md` template's `## Requirements` contains no `[NEEDS CLARIFICATION]`.

### Removal Verification

- [x] A-021 R-SKILL-4: No skill source references a live `spec` stage (state-table rows, `finish ... spec`, `/fab-clarify spec`) except deprecation/error strings.
- [x] A-022 R-DOC-1: No `docs/specs/` file (overview/srad/change-types/skills/architecture/glossary/templates/user-flow/SPEC-hooks/SPEC-preamble/SPEC-_review/assembly-line/index) documents a live spec stage, spec.md hook, or indicative flag.
- [x] A-023 R-SCORE-2: `change-types.md` `expected_min` values are identical to `score.go`.

### Scenario Coverage

- [x] A-024 R-GO-4: `go test ./...` from `src/go/fab` passes with no `spec`-stage assertions.
- [x] A-025 R-CONST-1: `constitution.md` `Last Amended` reflects this change's date with a rationale note.

### Code Quality

- [x] A-026 Pattern consistency: Go changes follow surrounding naming/error-handling/structure; skill/doc edits follow existing prose and table conventions.
- [x] A-027 No unnecessary duplication: existing utilities (e.g., `validateStage`, `getGateThreshold`, raw-node passthrough) reused rather than reimplemented; no magic strings for the deprecation message beyond the single literal.

### Cross-References & Documentation Accuracy

- [x] A-028 cross_references: skill↔SPEC coupling satisfied (every changed `skills/*.md` has its `SPEC-*.md` updated or the partial-exemption documented); `_cli-fab.md` updated for the Go ABI change.
- [x] A-029 documentation_accuracy: doc tables/diagrams reflect the actual six-stage code (no residual seven-stage / two-gate / indicative text in scope).

## Notes

- docs/memory/ updates (R-DOC-2) are HYDRATE-stage, NOT this apply run.
- `.claude/skills/` are gitignored deployed copies — never edit; only `src/kit/skills/*.md`.
- Run `go build ./...` and `go test ./...` after Phase 1 and again after Phase 2.
- **A-028 SPEC-coupling exemption (recorded):** `fab-operator.md` and `fab-switch.md` were modified, but `SPEC-fab-operator.md`/`SPEC-fab-switch.md` contain no spec-stage, stage-number, or `:indicative`-field content (verified by grep) — nothing in them went stale, so no SPEC edit was required. The constitution's skill↔SPEC coupling is satisfied vacuously for these two. All other changed skills had their SPEC-*.md updated.
- **Review cycle 1 (fix-docs):** addressed 5 must-fix documentation_accuracy drifts — `glossary.md` Ship/Review-PR renumbered 6/7→5/6; `skills.md` `/git-pr` "stage 7"→"stage 5" and `/git-pr-review` "stage 8"→"stage 6"; `templates.md` `progress` keys prose dropped `spec`; `templates.md` confidence narration repointed from "generates the spec" to intake scoring. <!-- rework: review must-fix doc-accuracy, cycle 1 -->
- **Review cycle 2 (fix-docs):** cycle-1 re-review caught one more residual drift the first sweep missed — `SPEC-fab-status.md:5` "out of 7 total stages" → "out of 6" (paired skill `fab-status.md` was already correct). Confirmed via exhaustive `grep -rE 'out of 7|stage 7|stage 8|/7\)|7-stage'` over the full `docs/specs/` tree (incl. `skills/`) that this was the LAST instance — zero remaining. <!-- rework: review must-fix doc-accuracy, cycle 2 -->`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab change list` ABI: DROP the `:indicative` 5th field entirely (not keep-always-empty) | spec R-GO-3 leaves the choice explicit; dropping is cleaner and fab-switch is updated in lockstep (the only consumer). Composite ~86 | S:90 R:75 A:90 D:88 |
| 2 | Certain | `--indicative` CLI flag kept as accepted-but-ignored no-op for one release | spec R-GO-2 explicitly permits this for back-compat; lowest-risk path. Composite ~88 | S:92 R:85 A:90 D:88 |
| 3 | Confident | `statusfile.go` keeps the `Indicative *bool` decode field; only the `encodeConfidence` write block is removed | spec §3 finding #7 mandates decode-tolerance; round-trips legacy files harmlessly. Composite ~82 | S:90 R:70 A:88 D:85 |
| 4 | Confident | Constitution `Last Amended` edit done in this apply run (project file, not memory) | R-CONST-1 is a date+note, not a memory hydrate; doing it here keeps it with the change. Composite ~78 | S:85 R:70 A:80 D:80 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
