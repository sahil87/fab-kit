# Plan: CLI Dispatch Skill Wiring (3d)

**Change**: 260702-aetz-cli-dispatch-skill-wiring
**Intake**: `intake.md`

## Requirements

<!-- Wiring-only markdown change. Requirements implement the FIXED contract in
     docs/specs/harness-adapters.md — they do NOT redefine it. Grouped by the
     dispatch-seam surface each governs. -->

### Dispatch Seam: CLI-adapter branch (canonical, `_preamble.md`)

#### R1: Canonical CLI-adapter dispatch procedure in `_preamble.md`
The `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution section SHALL be the single canonical home for the CLI-adapter dispatch procedure. It MUST branch on `spawn=` presence from the single existing `fab resolve-agent <stage> --alias` call: `spawn=` absent ⇒ native Agent-tool dispatch (byte-preserving in behavior); `spawn=` present ⇒ the CLI-adapter procedure. There MUST be NO fallback to `agent.spawn_command`. Dispatch sites in `_pipeline.md`/`fab-continue.md`/`fab-adopt.md` reference this section rather than restating the five-state machine.

- **GIVEN** a dispatch site about to dispatch stage S with a resolved `fab resolve-agent S --alias` result
- **WHEN** the result carries no `spawn=` line
- **THEN** the site dispatches natively via the Agent tool exactly as before this change (model alias param + effort-prompt instruction), with no new steps
- **AND WHEN** the result carries a `spawn=` line
- **THEN** the site follows the canonical CLI-adapter procedure in `_preamble.md` (start on stdin → poll → five-state handling), consulting no `agent.spawn_command` fallback

#### R2: CLI-adapter procedure steps (start / poll / five-state)
The canonical procedure SHALL specify: (a) `fab dispatch start <change> <stage>` with the full stage prompt on stdin (no `--timeout` in v1); (b) poll `fab dispatch status <change> <stage>` with `sleep 30` between polls until a terminal state; (c) five-state handling — `running` keep polling; `done` read `.fab-dispatch/{4-char-id}/{stage}-result.yaml` and proceed with the normal sequencer transition (a review verdict `fail` inside a `done` result is a review outcome, not a dispatch failure); `failed` surface `fab dispatch logs <change> <stage> --tail N` and stop per the stage's failure path; `failed (no-result)` NEVER treat as done, surface logs and stop; `orphaned` surface and stop with re-run guidance.

- **GIVEN** a CLI-dispatched stage
- **WHEN** `fab dispatch status` reports `done`
- **THEN** the site reads `{stage}-result.yaml` at `.fab-dispatch/{4-char-id}/{stage}-result.yaml` as the block's returned result and runs the normal finish/fail sequencer transition
- **AND WHEN** it reports `failed (no-result)`
- **THEN** the site surfaces logs and stops — it MUST NOT treat the clean exit as a completed stage

#### R3: CLI-path model/effort seams do not apply
Under CLI dispatch the `spawn=` command SHALL already embed the full model ID and substituted effort (via `internal/spawn`, even under `--alias`), so the Agent-tool seams (the `model` alias param and the imperative effort-prompt line) MUST NOT be applied on the CLI path — the profile rides the spawn command. Sites keep the single `--alias` call and branch; no second resolve call is made.

- **GIVEN** a resolved tier carrying a `spawn_command`
- **WHEN** the CLI path dispatches
- **THEN** neither the Agent `model` param nor the effort-prompt instruction is emitted for that dispatch; the model/effort ride the `spawn=` command already substituted by `fab resolve-agent`

#### R4: Compliance visibility extends to `spawn=`
Each dispatch site MUST surface the resolved `spawn=` line alongside the existing `model=`/`effort=` surfacing, so a CLI dispatch (or a `spawn=` line resolved but not honored) is visible in orchestrator output rather than silent.

- **GIVEN** a dispatch site running `fab resolve-agent <stage> --alias`
- **WHEN** it surfaces the resolved profile
- **THEN** it surfaces `model=/effort=/spawn=` (the `spawn=` line only when present), extending the existing compliance-visibility rule

#### R5: No cleanup wiring after `done`
The wiring SHALL add NO cleanup calls after a `done` dispatch. `.fab-dispatch/` cleanup stays archive-time deletion + explicit `fab dispatch clean` only (no automatic GC).

- **GIVEN** a CLI dispatch that reached `done`
- **WHEN** the sequencer proceeds past it
- **THEN** no `fab dispatch clean` (or any cleanup) call is emitted by the wiring

### Dispatch-prompt obligations (bind BOTH adapters)

#### R6: Result obligation in both adapters' prompts
Per `harness-adapters.md` § Dispatch-prompt obligations, every dispatched stage prompt (native AND CLI) MUST instruct the worker to produce `{stage}-result.yaml` — a real file at `.fab-dispatch/{4-char-change-id}/{stage}-result.yaml` for the CLI adapter, the structural return for the native adapter.

- **GIVEN** any dispatched stage prompt
- **WHEN** it is composed
- **THEN** it carries the result obligation appropriate to its adapter (file path for CLI, structural return for native)

#### R7: Standard subagent context + refresh epilogue in both adapters' prompts
Every dispatched stage prompt (native AND CLI) MUST carry the standard subagent context files instruction (`config.yaml`, `constitution.md`, optional `context.md`/`code-quality.md`/`code-review.md`) and MUST end with a terminal `fab status refresh` epilogue so the worker recomputes state from artifacts after finishing.

- **GIVEN** a CLI-path stage prompt handed to a worker on a fresh harness
- **WHEN** it is composed
- **THEN** it instructs the worker to read the standard subagent context files and ends with a `fab status refresh` epilogue

#### R8: Block-contract carve-out (transition prohibition + required refresh)
The universal block-contract line "do NOT run `fab status` commands; return results only" SHALL be refined at EVERY occurrence to prohibit `fab status` TRANSITION commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`) while REQUIRING the terminal `fab status refresh`. Refresh is a pull-based recompute, not a transition; the orchestrator still owns all transitions. The refinement MUST NOT loosen the "orchestrator owns all transitions" invariant.

- **GIVEN** every dispatch site carrying the old "do NOT run `fab status` commands" line
- **WHEN** the carve-out is applied
- **THEN** the line prohibits the six transition commands and requires the terminal `fab status refresh`, at every occurrence in the sweep class

### `{stage}-result.yaml` schema (3d-defined)

#### R9: Minimal result schema with status vs verdict split
The `{stage}-result.yaml` schema SHALL be documented (canonically beside the CLI-adapter procedure, or in the result-obligation prose) as a minimal YAML envelope mirroring each native block's return contract: common `stage`/`status`/`summary`; apply adds `failed_task`/`reason` on failure; review adds `verdict` (pass|fail) and `findings{must_fix,should_fix,nice_to_have}`; hydrate carries only the common envelope. The `status` (worker/infrastructure outcome) vs `verdict` (review outcome) split is load-bearing: a completed review with verdict `fail` is dispatch-state `done` (result present), and the orchestrator then takes the normal review-fail path.

- **GIVEN** a review worker that ran to completion and found a must-fix
- **WHEN** it writes `{stage}-result.yaml`
- **THEN** it records `status: success` (the review ran) and `verdict: fail` (the review outcome), and the orchestrator reads dispatch-state `done` and takes the normal review-fail transition

### Review nesting degradation (`_review.md`)

#### R10: Nesting degradation note — canonical + CLI-prompt
`review` is the one nesting stage. On a harness WITH sub-agent support the inward/outward reviewers + merge run as parallel sub-agents; on a harness WITHOUT sub-agent support the worker runs them sequentially inline in one context. Only concurrency degrades — the outcome contract (same merged findings + verdict) is identical. This note MUST live canonically in `_review.md` § Shared Review Dispatch AND be carried in the CLI-path review dispatch prompt (a cross-harness worker may never read fab's skill files beyond the prompt).

- **GIVEN** a review stage dispatched to a harness without sub-agent support
- **WHEN** the worker executes
- **THEN** it runs inward + outward + merge sequentially inline and returns the same merged findings + verdict a parallel run would produce

### Sibling & mirror sweep (constitution-required)

#### R11: SPEC mirrors updated in the same change
Every edited `src/kit/skills/*.md` SHALL have its `docs/specs/skills/SPEC-*.md` mirror updated in the same change: `SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, `SPEC-_review.md` (and `SPEC-fab-adopt.md` if fab-adopt.md is edited).

- **GIVEN** an edit to a `src/kit/skills/*.md` file in this change
- **WHEN** apply completes
- **THEN** the corresponding `SPEC-*.md` mirror reflects the CLI-adapter wiring

#### R12: Stale forward pointers repointed
The stale `3c/3d`/`follow-ups`/`not read here`/`3d's job`/`3d's business` forward pointers SHALL be updated to reflect the now-shipped wiring: `_preamble.md` line ~313, `stage-models.md` lines ~145 and ~284, and the memory files (`_shared/context-loading.md` line ~121, `runtime/dispatch.md`). A repo-wide grep MUST confirm no stale claim survives in the sweep class.

- **GIVEN** a repo-wide grep for `3c/3d`, `follow-ups (3c`, `not read here`, `3d's job`, `content is 3d`, `against which the 3d`
- **WHEN** apply completes
- **THEN** every non-archived occurrence in the sweep class reflects the landed wiring (or is an accurate historical design-intent statement left intentionally)

### Memory updates

#### R13: Affected memory files reflect the wired behavior
The four Affected Memory files SHALL be updated per the intake's § Affected Memory: `pipeline/execution-skills.md` (sequencer/block dispatch contract + CLI branch + `spawn=` surfacing + refined block-contract line), `_shared/context-loading.md` (§ Per-Stage Model Resolution — replace the stale "3c/3d's job … do not read `spawn=`" claim), `runtime/dispatch.md` (resolve the "content is 3d's business" reference — record the `{stage}-result.yaml` schema + five-state consumption), `pipeline/hooks-may-enhance-never-own.md` (the worker-side `fab status refresh` epilogue is prompt-owned, not a hook). Memory↔memory links use bundle-relative `/...` form. Indexes regenerated via `fab memory-index`.

- **GIVEN** the four Affected Memory files
- **WHEN** apply completes
- **THEN** each reflects the wired behavior with bundle-relative memory↔memory links, and `fab memory-index` has regenerated the indexes

### Non-Goals

- No Go changes, no migrations, no template changes — markdown only. If a Go change appears required, STOP and report.
- No semantic amendment to `docs/specs/harness-adapters.md` (the fixed contract) — wiring-only conformance.
- No `--timeout` on `fab dispatch start` in v1; no poll backoff; no automatic GC.

### Design Decisions

1. **Canonical CLI-adapter home is `_preamble.md` § Subagent Dispatch**: sites reference it, do not restate the five-state machine — *Why*: mirrors the existing canonical-contract pattern the seam already uses (single-source discipline) — *Rejected*: restating the machine at each site (drift risk, the project's top rework cause).
2. **`harness-adapters.md` left untouched**: its § Skill wiring uses design-intent framing ("3d implements") — *Why*: Constitution VI makes specs pre-implementation design intent, not post-hoc status; a cosmetic "implements"→"implemented" tense edit carries no semantic value and risks reading as a contract touch under the strict wiring-only mandate — *Rejected*: editing it to mark 3d "landed" (the intake permits this only "if the tense warrants it"; it does not).
3. **`fab-adopt.md` is in the block-contract sweep class**: it reuses `_pipeline.md`/`_review.md` dispatch and carries the literal "do NOT run `fab status`; return results only" line at two sites — *Why*: the intake mandates sweeping *every* occurrence of the old line, and code-quality.md § Sibling & Mirror Sweeps treats a missed class member as must-fix — *Rejected*: leaving it (would strand a stale block-contract line at a live dispatch site and fail review's cross_references check).

## Tasks

### Phase 1: Canonical contract (`_preamble.md`)

- [x] T001 In `src/kit/skills/_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution: replace the stale line ~313 ("`spawn=` is for the cross-harness dispatch follow-ups (3c/3d), not read here") with the branch-on-`spawn=` rule and add the canonical **CLI-adapter dispatch procedure** subsection (start-on-stdin → `sleep 30` poll → five-state handling), the CLI-path model/effort seam note (spawn command embeds full ID + effort, Agent-tool seams do not apply), the `spawn=` compliance-visibility extension, the no-cleanup-after-`done` note, and the `{stage}-result.yaml` minimal schema (status vs verdict split). <!-- R1 R2 R3 R4 R5 R9 R12 -->
- [x] T002 In the same `_preamble.md` section: add the dispatch-prompt obligations that bind BOTH adapters (result obligation with the CLI file path `.fab-dispatch/{id}/{stage}-result.yaml` vs native structural return; standard subagent context files; terminal `fab status refresh` epilogue). <!-- R6 R7 -->

### Phase 2: Block-contract carve-out sweep

- [x] T003 Sweep the "do NOT run `fab status` commands; return results only" line to the transition-prohibition + required-`fab status refresh` carve-out at EVERY occurrence: `src/kit/skills/_pipeline.md` (Behavior dispatch note line ~60 + Steps 1/2/3 + Auto-Rework item 3), `src/kit/skills/fab-continue.md` (Normal Flow Step 1 dispatch contract line ~67), `src/kit/skills/fab-adopt.md` (lines ~100, ~108). Preserve "orchestrator owns all transitions." <!-- R8 -->

### Phase 3: Dispatch-site wiring (reference the canonical procedure)

- [x] T004 In `src/kit/skills/_pipeline.md`: branch every dispatch site on `spawn=` presence by reference to `_preamble.md` (do NOT restate the five-state machine) — the Behavior § Per-stage model resolution note (surface `spawn=` alongside `model=`/`effort=`; branch on presence), Step 1 (apply), Step 2 (review), Step 3 (hydrate), and Auto-Rework Loop items 3–4 (re-dispatch apply, fresh re-review). <!-- R1 R3 R4 -->
- [x] T005 In `src/kit/skills/fab-continue.md`: branch on `spawn=` by reference in Normal Flow Step 1 sub-agent dispatch contract (surface `spawn=`, five-state handling by reference), the stage-table rows (`intake`/`ready` apply sequencer, `apply`, `review`, `hydrate`), and the Review Behavior nested-reviewer resolution note (on a CLI-dispatched review worker the nested resolution happens inside the worker where sub-agent support may be absent → nesting degradation, R10). <!-- R1 R3 R4 R10 -->
- [x] T006 In `src/kit/skills/_review.md` § Shared Review Dispatch: add the canonical nesting-degradation note (parallel sub-agents on a harness with support; sequential-inline on one without; only concurrency degrades, outcome contract identical) and state that the CLI-path review dispatch prompt carries the same degradation instruction. <!-- R10 -->

### Phase 4: SPEC mirrors, aggregate specs, stale pointers

- [x] T007 [P] Update `docs/specs/skills/SPEC-_preamble.md` — mirror the CLI-adapter dispatch procedure, `spawn=` branch, five-state machine, dispatch-prompt obligations, `{stage}-result.yaml` schema, and refreshed `spawn=` line description into its Summary + Flow. <!-- R11 -->
- [x] T008 [P] Update `docs/specs/skills/SPEC-_pipeline.md` — mirror the `spawn=` branch + block-contract carve-out into its Summary + per-cycle choreography + Flow. <!-- R11 -->
- [x] T009 [P] Update `docs/specs/skills/SPEC-fab-continue.md` — mirror the Step 1 `spawn=` branch, block-contract carve-out, and Review nested-reviewer CLI degradation into its Summary + Flow. <!-- R11 -->
- [x] T010 [P] Update `docs/specs/skills/SPEC-_review.md` — mirror the nesting-degradation note into its Summary + Flow. <!-- R11 -->
- [x] T011 [P] Update `docs/specs/skills/SPEC-fab-adopt.md` (only if `fab-adopt.md` was edited in T003) — mirror the block-contract carve-out at its three dispatch references. <!-- R11 -->
- [x] T012 Repoint the stale forward pointers in `docs/specs/stage-models.md` lines ~145 and ~284 ("the dispatch that RUNS the command (`fab dispatch`) and the skill dispatch-seam wiring are separate follow-up changes (3c/3d)") to name `harness-adapters.md` + this change as landed. Confirm `docs/specs/architecture.md` config block and `docs/specs/index.md` need no change (accurate as-is). <!-- R12 -->
- [x] T013 Repo-wide grep sweep for `3c/3d`, `follow-ups (3c`, `not read here`, `3d's job`, `content is 3d`, `against which the 3d` across `src/` and `docs/` (excluding `/archive/`); update every surviving stale occurrence in the sweep class. <!-- R12 --> <!-- rework: review cycle 1 — the block-contract-line sweep missed two docs/ class members: docs/specs/skills/SPEC-fab-ff.md:5 (old "do NOT run fab status" line + model=/effort=-only surfacing, no spawn= branch — refine to the carve-out matching SPEC-_pipeline.md) — add the old block-contract line's literal phrasing to the grep patterns -->

### Phase 5: Memory + index regen

- [x] T014 Update `docs/memory/pipeline/execution-skills.md` — the sequencer/block dispatch contract: add the CLI-adapter branch, the `spawn=` surfacing, and the refined block-contract line (transition prohibition + required `fab status refresh` epilogue). <!-- R13 --> <!-- rework: review cycle 1 — ~line 95 (§ Shared Pipeline Bracket) still restates the OLD un-refined "do NOT run fab status; return results only" prompt contract, contradicting the refined line ~21 in the same file; refine it (or point at _preamble.md § Dispatch-Prompt Obligations) -->
- [x] T015 Update `docs/memory/_shared/context-loading.md` § Per-Stage Model Resolution — replace the stale "Consuming the `spawn=` line … is 3c/3d's job … do not read `spawn=`" claim (line ~121) with the wired behavior (branch on `spawn=`; CLI path bypasses the alias/effort-prompt seams). <!-- R13 -->
- [x] T016 Update `docs/memory/runtime/dispatch.md` — resolve the "content is 3d's business" / "against which the 3d skill wiring conforms" forward references: record the `{stage}-result.yaml` minimal schema and that the skill wiring now consumes the five states. <!-- R13 -->
- [x] T017 Update `docs/memory/pipeline/hooks-may-enhance-never-own.md` — note the dispatch protocol's worker-side `fab status refresh` epilogue is now written into the dispatch prompts (a prompt epilogue, not a hook) as the protocol-owned step, per the spec's hooks-enhance-never-own rule. <!-- R13 -->
- [x] T018 Run `fab memory-index --check` (must be exit 0/1; exit 2 ⇒ STOP + surface), then `fab memory-index` to regenerate the root/domain/sub-domain indexes. Do not hand-edit index rows. <!-- R13 -->

## Execution Order

- T001 → T002 (both edit the same `_preamble.md` section; sequential to avoid conflicting edits).
- Phase 2 (T003) before Phase 3 (T004–T006) — the carve-out reword and the `spawn=` branch touch overlapping regions of the same dispatch-site paragraphs; do the carve-out first, then layer the branch reference.
- Phase 3 after Phase 1 (sites reference the canonical `_preamble.md` procedure T001 establishes).
- Phase 4 SPEC mirrors (T007–T011) are `[P]` — distinct files; each after its source skill edit (T007 after T001–T002; T008 after T003–T004; T009 after T003+T005; T010 after T006; T011 after T003).
- T013 (grep sweep) after all skill + spec + memory edits so it catches any residue.
- T018 (index regen) last, after all memory-file edits (T014–T017).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution is the sole canonical home for the CLI-adapter procedure; it branches on `spawn=` presence with no `agent.spawn_command` fallback; dispatch sites reference it rather than restating the five-state machine.
- [x] A-002 R2: The canonical procedure documents `fab dispatch start` on stdin (no `--timeout` v1), `sleep 30` polling of `fab dispatch status`, and all five states with the exact `done`/`failed`/`failed (no-result)`/`orphaned` handling from the intake.
- [x] A-003 R3: The CLI path applies neither the Agent `model` param nor the effort-prompt instruction (the `spawn=` command carries the full ID + effort); sites keep the single `--alias` call and make no second resolve call.
- [x] A-004 R4: Every dispatch site surfaces the resolved `spawn=` line alongside `model=`/`effort=`.
- [x] A-005 R5: No cleanup call is emitted by the wiring after a `done` dispatch.
- [x] A-006 R6: Both adapters' dispatch prompts carry the result obligation (CLI file path `.fab-dispatch/{id}/{stage}-result.yaml`; native structural return).
- [x] A-007 R7: Both adapters' dispatch prompts carry the standard subagent context files instruction and end with a `fab status refresh` epilogue.
- [x] A-008 R8: The block-contract carve-out (prohibit the six transition commands; require terminal `fab status refresh`) is applied at every occurrence of the old line, preserving "orchestrator owns all transitions." <!-- MET (rework cycle 1, 2026-07-02): the two un-refined occurrences were refined — docs/memory/pipeline/execution-skills.md § Shared Pipeline Bracket now carries the carve-out (matching line ~21), and docs/specs/skills/SPEC-fab-ff.md line 5 now carries the carve-out + spawn= branch (matching SPEC-_pipeline.md). Remaining literal-line matches are legitimately historical (log.md/log.seed.md changelog entries) or definitional (_preamble.md § Block-contract carve-out quotes the old line to explain the refinement). -->
- [x] A-009 R9: The `{stage}-result.yaml` minimal schema is documented with the load-bearing `status` vs `verdict` split (a `done` result with `verdict: fail` routes to the normal review-fail path).
- [x] A-010 R10: The nesting-degradation note lives canonically in `_review.md` § Shared Review Dispatch and is carried in the CLI-path review dispatch prompt; only concurrency degrades, outcome contract identical.
- [x] A-011 R13: The four Affected Memory files reflect the wired behavior with bundle-relative memory↔memory links; `fab memory-index` regenerated the indexes.

### Behavioral Correctness

- [x] A-012 R1: When `spawn=` is absent the native dispatch path is functionally identical to pre-change behavior (no new steps, no behavioral regression).
- [x] A-013 R9: A review worker that found a must-fix records `status: success` + `verdict: fail`, read as dispatch-state `done`, and the orchestrator takes the normal review-fail transition (not a dispatch-failure path).

### Scenario Coverage

- [x] A-014 R2: `failed (no-result)` is never treated as `done` at any CLI-path dispatch site — logs surfaced and stop.
- [x] A-015 R10: A review dispatched to a harness without sub-agent support runs inward+outward+merge sequentially inline and returns the same merged findings + verdict.

### Removal Verification

- [x] A-016 R12: A repo-wide grep for `3c/3d`, `follow-ups (3c`, `not read here`, `3d's job`, `content is 3d`, `against which the 3d` (excluding `/archive/`) finds no surviving stale claim in the sweep class; `stage-models.md` lines ~145/~284 repoint to `harness-adapters.md` + this change.

### Code Quality

- [x] A-017 SPEC-mirror sync: every edited `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in this same change (Constitution Additional Constraints; R11).
- [x] A-018 Canonical source only: all skill edits are in `src/kit/skills/*.md`; no edit under `.claude/skills/` (gitignored deployed copies).
- [x] A-019 Markdown-only artifacts: no Go/template/migration changes; standard CommonMark; no binary formats.
- [x] A-020 Pattern consistency: new prose follows the existing dispatch-seam framing (canonical-contract-plus-references pattern) and the bundle-relative link convention.
- [x] A-021 documentation_accuracy + cross_references: dispatch facts stated once in `_preamble.md` and referenced (not duplicated) at sites; all cross-references resolve; the whole sweep class (SPEC mirrors, aggregate specs, memory) is consistent.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality (the CLI-adapter wiring prose) without making existing code redundant. All stale forward-pointers ("not read here", "3c/3d's job", "content is 3d's business", "against which the 3d skill wiring conforms") were repointed in place, not left orphaned; the five-state machine remains enumerated only in the canonical `_preamble.md` plus the pre-existing `_cli-fab.md` runtime reference (untouched).

## Assumptions

<!-- These carry forward the intake's fully-graded Assumptions table (rows 1–13),
     plus apply-time wiring decisions. The intake rows are authoritative; the
     apply-added rows (14–16) grade decisions made while planning the wiring. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Implement strictly against `docs/specs/harness-adapters.md`; any contract flaw found during wiring → report as a spec-amendment need, never a silent redefinition in skill files | Intake #1 + spec § Skill wiring | S:95 R:90 A:95 D:95 |
| 2 | Certain | Scope is wiring-only markdown: skills + SPEC mirrors + aggregate specs + memory; no Go, no migrations, no template changes | Intake #2; 3c shipped the full runtime | S:90 R:85 A:95 D:90 |
| 3 | Certain | Branch on `spawn=` presence from the single existing `fab resolve-agent <stage> --alias` call; absence ⇒ native unchanged; NO fallback to `agent.spawn_command`; per-stage native/CLI mixing allowed | Intake #3 (decided in 3b/3c) | S:95 R:85 A:95 D:95 |
| 4 | Certain | On the CLI path the Agent-tool model/effort seams do not apply — the `spawn=` command embeds the full model ID + substituted effort even under `--alias` | Intake #4; `_cli-fab.md` § fab resolve-agent | S:90 R:85 A:95 D:90 |
| 5 | Certain | Five-state handling exactly per intake #5 (`done` read result + normal transition; `failed`/`failed (no-result)`/`orphaned` surface + stop; verdict-fail-in-done is a review outcome) | Intake #5 + spec five-state machine | S:95 R:85 A:90 D:90 |
| 6 | Certain | No cleanup calls after a `done` dispatch — `.fab-dispatch/` cleanup stays archive-time + explicit `clean` | Intake #6 (fixed in 3c) | S:95 R:90 A:95 D:95 |
| 7 | Confident | Poll cadence `sleep 30` between `fab dispatch status` polls; no backoff in v1 | Intake #7; stages run minutes-long, 30s responsive without spam; trivially tunable prose | S:60 R:90 A:80 D:65 |
| 8 | Confident | No `--timeout` passed to `fab dispatch start` in v1 | Intake #8; orphan detection + `kill` cover failure modes | S:65 R:90 A:75 D:75 |
| 9 | Confident | `{stage}-result.yaml` schema as proposed in intake § What Changes 3 — common `stage`/`status`/`summary` + apply `failed_task`/`reason` + review `verdict`/`findings{must_fix,should_fix,nice_to_have}`; `status` (worker) distinct from `verdict` (review) | Intake #9; mirrors native blocks' documented return contracts | S:70 R:70 A:80 D:70 |
| 10 | Confident | Nesting-degradation placement: BOTH a canonical `_review.md` note AND the instruction carried in the CLI-path review dispatch prompt | Intake #10; a cross-harness worker may never read fab skill files | S:60 R:90 A:75 D:60 |
| 11 | Confident | Block-contract carve-out semantics: prohibit `fab status` transition commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`), REQUIRE the terminal `fab status refresh`; exact phrasing decided at apply | Intake #11; semantics fixed by description + spec obligation 3 | S:70 R:80 A:85 D:75 |
| 12 | Certain | Canonical CLI-adapter procedure lives in `_preamble.md` § Subagent Dispatch; sites reference it | Intake #12; mirrors the existing canonical-contract pattern | S:75 R:85 A:90 D:85 |
| 13 | Certain | Dispatch-prompt obligations (result instruction, context files, refresh epilogue) written into BOTH adapters' prompts, native included | Intake #13; spec § Dispatch-prompt obligations | S:90 R:80 A:90 D:90 |
| 14 | Confident | `fab-adopt.md` is a member of the block-contract sweep class (its 2 dispatch references carry the literal old line) and is edited in T003 + mirrored in SPEC-fab-adopt.md (T011), even though the intake's file list omitted it | Intake mandates sweeping *every* occurrence; code-quality.md § Sibling & Mirror Sweeps makes a missed class member must-fix; per-file Affected lists under-cover cross-cutting prose | S:80 R:85 A:85 D:80 |
| 15 | Certain | `docs/specs/harness-adapters.md` is left untouched — its § Skill wiring "3d implements" is design-intent framing (Constitution VI), and a cosmetic tense edit carries no semantic value while risking a contract-touch reading | Intake #1 + § What Changes 7 permit the edit only "if the tense warrants it"; it does not | S:85 R:90 A:85 D:85 |
| 16 | Confident | `docs/specs/architecture.md` config block (lines ~216–245) and `docs/specs/index.md` line 28 need NO change — both describe the `spawn=`/adapter split accurately as config/landscape prose, carrying no "wiring pending" pointer | Read both in full; neither asserts the wiring is unshipped | S:70 R:85 A:80 D:75 |

16 assumptions (10 certain, 6 confident, 0 tentative).
