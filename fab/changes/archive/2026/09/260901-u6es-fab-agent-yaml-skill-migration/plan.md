# Plan: fab agent YAML Skill Migration

**Change**: 260901-u6es-fab-agent-yaml-skill-migration
**Intake**: `intake.md`

## Requirements

### Skills: Dispatch-site migration

#### R1: Dispatch sites run `fab agent <stage> -o yaml`
Every skill-prose dispatch site that today instructs running `fab resolve-agent <stage> --alias` SHALL instead instruct running `fab agent <stage> -o yaml`, 1:1 in semantics: branch on `dispatch:` **key presence** (absent ⇔ native rung), Agent-tool `model` parameter ← `model_alias` (empty for non-Claude IDs — then no native seam exists, exactly as today), effort instruction ← `effort`, and the `dispatch.command` value is never executed by the skill. The surfacing rule SHALL reword to surfacing the resolved YAML (at minimum `provider`/`model`/`model_alias`/`effort` and `dispatch:` presence); an all-empty resolution remains a flag-don't-dispatch signal. Files: `src/kit/skills/_preamble.md` (§ Per-Stage Model Resolution seam+override tables, § Worker Continuation profile-fixity ×2, § CLI-Adapter Dispatch branch table), `_pipeline.md` (§ Stage Dispatch Procedure items 1+5, § Light Lane no-dispatch mentions), `fab-continue.md`, `fab-fff.md`, `fab-proceed.md`, `_cli-agents.md`.

- **GIVEN** any dispatch site in `src/kit/skills/`
- **WHEN** it instructs per-stage resolution
- **THEN** the command it names is `fab agent <stage> -o yaml` and the branch/model/effort reads are the YAML keys above
- **AND** `grep -rn 'resolve-agent.*--alias' src/kit/skills/` finds no remaining *instruction* to run it (the `_cli-fab.md` reference section documenting the deprecated command itself is exempt)

#### R2: Choreography unchanged; the "unlabelled" rationale re-anchored
The migration SHALL NOT change any dispatch choreography: the "attempt `start` first and let its refusal discriminate" discovery, pane readiness gate, wait/recovery machine, stage-aware reap timing, worker continuation, and profile fixity stay verbatim. Prose that today justifies the start-probe by "the `dispatch=` line is unlabelled" SHALL be re-anchored truthfully — the YAML's `dispatch.rung` is labelled but deliberately NOT consumed until Change 4 (`0i4x`) — never deleted as rationale and never converted into rung-based branching.

- **GIVEN** `_preamble.md` § CLI-Adapter Dispatch after the migration
- **WHEN** step 1's mode-discovery prose is read
- **THEN** it still teaches the `start`-probe as the discriminator, notes the labelled `rung:` exists but is unconsumed (Change 4's job), and no consumer branches on `rung:`

### Docs: resolve-agent deprecation

#### R3: `_cli-fab.md` deprecation banner, reference content survives
`_cli-fab.md` § fab resolve-agent SHALL gain a deprecation banner pointing at § fab agent (`-o yaml`) as the surface skills consume; the command's own reference documentation (ladder, fill precedence, flags, output lines) STAYS — it documents a working command until a later post-window deletion. Facts already owned by § fab agent are pointed at, not restated (owner-or-pointer). Instructional mentions elsewhere in the file (e.g., § fab dispatch's "re-run `fab resolve-agent` … `dispatch=` line" guidance, the operator-launcher exception "resolves without `--alias`") SHALL reword against the YAML surface. `fab resolve-agent`'s flags, arguments, and byte-stable output are NOT modified.

- **GIVEN** `_cli-fab.md` after the change
- **WHEN** § fab resolve-agent is read
- **THEN** a deprecation banner names § fab agent / `-o yaml` as the skill-consumed surface, and the section still fully documents the working command

### Specs: design-owner and aggregate updates

#### R4: Specs present the YAML surface as the skill-consumed one
`docs/specs/stage-models.md` (§ Skill wiring — the design owner) SHALL rewrite the skill-wiring prose onto `fab agent <stage> -o yaml`, presenting `fab resolve-agent` as the deprecated alias surface with an unchanged contract. `docs/specs/harness-adapters.md` (branch-discriminator + resolve-step references), and the aggregates `skills.md`, `glossary.md`, `architecture.md`, `config.md`, `index.md` SHALL be swept in the same pass (the known sibling-sweep class).

- **GIVEN** the specs after the change
- **WHEN** any spec describes how a pipeline stage resolves its worker profile
- **THEN** it names `fab agent <stage> -o yaml` and the `dispatch:`-key branch, with resolve-agent mentioned only as the deprecated alias or in its own contract description

### Memory: present-truth updates

#### R5: Memory content files migrated; logs untouched; indexes regenerated
The memory content files SHALL be updated to present truth: `runtime/providers-and-profiles.md` (consumer-facing lines migrate; the § resolve-agent requirement block stays as the deprecated command's contract with a deprecation note), `_shared/context-loading.md` (§ Per-Stage Model Resolution wiring), `_shared/configuration.md`, `pipeline/execution-skills.md`, `runtime/dispatch.md`, `distribution/kit-architecture.md`, `runtime/agent-primitives.md`, `pipeline/issue-linking.md`, `distribution/migrations.md`, `runtime/index.md`, `runtime/operator.md`. `log.md`/`log.seed.md` rows, `fab/changes/archive/`, `docs/findings/`, and `fab/plans/` are history and SHALL NOT be edited. `fab memory-index` SHALL be re-run after edits (byte-stable output committed).

- **GIVEN** the memory tree after the change
- **WHEN** `grep -rn resolve-agent docs/memory --include='*.md'` runs excluding `log.md`/`log.seed.md`
- **THEN** every remaining occurrence either documents the deprecated command's own contract (with the deprecation framing) or is deliberate history — none instructs a consumer to run it for stage dispatch

### Go: user-facing string literals (bounded)

#### R6: The two instructional Go strings migrate, with their pinned tests
`cmd/fab/dispatch_start.go:532`'s native-mode error SHALL reword its remedy to "re-run `fab agent <stage> -o yaml` and dispatch natively when the `dispatch:` key is absent"; `internal/configref/configref.go:770`'s generated config-fence line SHALL name `fab agent <stage|role> -o yaml`. Tests pinning either string SHALL be updated (`dispatch_start_test.go`, `dispatch_restart_test.go`, `config_show_init_test.go`, `noproject_config_test.go` — verify which actually pin). No other Go changes; `fab resolve-agent` golden/parity tests stay untouched and green.

- **GIVEN** the Go tree after the change
- **WHEN** `go test ./...` runs in `src/go/fab`
- **THEN** all tests pass, the two literals name the new surface, and `git diff` shows no change to `cmd/fab/resolve_agent.go`, `internal/agent/` behavior, or any golden expectation for resolve-agent output

### Sweep: contrastive-phrase completeness

#### R7: Phrase-class sweep clean before review
Before apply finishes, a repo-wide sweep SHALL cover the token `resolve-agent` (and `resolve_agent`), plus the phrase classes `--alias`, `dispatch=`, "`model=` line", "model=/effort=", "ordered lines"/"byte-stable lines" (where claimed as the skill-consumed surface) — across `src/kit/skills/`, `docs/specs/`, `docs/memory/` content files, user-facing Go string literals, and `*_test.go` comments that describe skills consuming resolve-agent output. Historical artifacts are excluded. `cmd/fab/skill.md`'s CLI enumeration keeps its resolve-agent row (it enumerates existing commands); add `fab agent` alongside only if the surrounding text presents resolve-agent as the resolution surface.

- **GIVEN** apply is about to finish
- **WHEN** the sweep greps run
- **THEN** every hit is classified (migrated / deprecated-command-contract / history / enumeration) with zero unclassified instructional leftovers

### Non-Goals

- No mutation of `fab resolve-agent`'s surface or output (frozen; parity via the shared struct)
- No rung-based choreography (Change 4, backlog `0i4x`)
- No change to `fab agent`'s Go surface (Change 2 shipped everything cited)
- No operator-launcher resolution-path change (`WithProfile` direct stays)
- No migration file: the config-fence comment regenerates via `fab config upgrade` by design

### Design Decisions

#### Re-anchor, don't delete, the "unlabelled" rationale
**Decision**: Prose justifying the start-probe discovery keeps its rationale, restated as "the YAML labels the rung, but consumers deliberately don't read it until Change 4".
**Why**: Deleting the rationale would leave the start-probe choreography unexplained; branching on the label here would smuggle Change 4 into a sweep change.
**Rejected**: Consuming `rung:` now — it would make the mechanical migration and the semantic simplification un-separable for review/revert.
*Introduced by*: 260901-u6es-fab-agent-yaml-skill-migration

#### User-facing Go string literals ride the sweep
**Decision**: The two Go strings that instruct callers to run the old surface migrate in this change, with their pinned tests.
**Why**: Standing repo lesson — behavior-claim sweeps must include user-facing STRING LITERALS; leaving them would ship contradictory guidance.
**Rejected**: Deferring to a follow-up (ships a version whose error text contradicts its own skills).
*Introduced by*: 260901-u6es-fab-agent-yaml-skill-migration

## Tasks

### Phase 1: Setup

- [x] T001 Verify the shipped schema and footprint: run `go run ./cmd/fab agent apply -o yaml` from `src/go/fab` (installed 2.23.9 bottle predates #636 — never verify against it); re-grep the token + phrase classes (`resolve-agent`, `resolve_agent`, `--alias`, `dispatch=`, "`model=` line", "model=/effort=") across `src/kit/skills/ docs/specs/ docs/memory/ src/go/`, and record the classified occurrence list (migrate / deprecated-contract / history / enumeration) in the change folder as `sweep-list.md` <!-- R7 -->

### Phase 2: Skill dispatch sites

- [x] T002 Migrate `src/kit/skills/_preamble.md`: § Per-Stage Model Resolution (seam table, override table, surfacing rule → YAML keys incl. `model_alias`), § Worker Continuation (profile-fixity mentions ×2), § CLI-Adapter Dispatch (branch table → `dispatch:` key presence; re-anchor the "unlabelled" rationale per R2; pane-mode bullets' resolve-agent error guidance) <!-- R1 -->
- [x] T003 [P] Migrate `src/kit/skills/_pipeline.md`: § Stage Dispatch Procedure items 1 and 5, § Light Lane "no `fab resolve-agent`" mentions, Step 3 framing <!-- R1 -->
- [x] T004 [P] Migrate `src/kit/skills/fab-continue.md` (8 occurrences: per-stage resolve + surface instructions) <!-- R1 -->
- [x] T005 [P] Migrate `src/kit/skills/fab-fff.md` (Steps 4–5 + Behavior note) and `src/kit/skills/fab-proceed.md` (3) <!-- R1 -->
- [x] T006 [P] Migrate `src/kit/skills/_cli-agents.md` (2 operator-facing references) <!-- R1 -->
- [x] T007 Update `src/kit/skills/_cli-fab.md`: deprecation banner on § fab resolve-agent pointing at § fab agent; reword instructional mentions elsewhere (§ fab dispatch remedy text, operator-launcher exception vs the YAML surface); keep the reference content per owner-or-pointer <!-- R3 -->

### Phase 3: Specs and memory

- [x] T008 <!-- rework cycle 2: operator-launcher exception at stage-models.md:757 still contrasts against --alias; reword vs the YAML model_alias seam --><!-- rework: model_alias claim wrong for non-Claude IDs (must-fix) + unmatched paren in Skill wiring sentence --> Rewrite `docs/specs/stage-models.md` § Skill wiring onto `fab agent <stage> -o yaml` (+ its 28 occurrences classified per R4) and sweep `docs/specs/harness-adapters.md` (7) <!-- R4 -->
- [x] T009 <!-- rework cycle 2: config.md:406 LoadPath consumer list has duplicate `agent` entry after inserting deprecated resolve-agent --> [P] Sweep aggregate specs: `docs/specs/skills.md` (5), `glossary.md` (4), `architecture.md` (2), `config.md` (2), `index.md` (1) <!-- R4 -->
- [x] T010 Migrate memory content files per R5 (11 files; providers-and-profiles.md is the heavy one — consumer lines migrate, the resolve-agent requirement block gains deprecation framing; logs untouched) <!-- R5 -->
- [x] T011 Run `fab memory-index` and commit regenerated indexes <!-- R5 -->

### Phase 4: Go literals, tests, final sweep

- [x] T012 <!-- rework: RequiresResolveAgent test identifiers/comments stale after remedy migration --> Update `src/go/fab/cmd/fab/dispatch_start.go:532` error remedy and `src/go/fab/internal/configref/configref.go:770` fence line; update whichever tests pin those strings; run `go test ./cmd/... ./internal/configref/...` then `go test ./...` in `src/go/fab` <!-- R6 -->
- [x] T013 Sweep `*_test.go` comments describing skills consuming resolve-agent output forms (comments only — assertions/goldens untouched); decide `cmd/fab/skill.md` enumeration in context <!-- R7 -->
- [x] T014 Final contrastive-phrase sweep per R7 against `sweep-list.md`; classify every residual hit; fix unclassified leftovers <!-- R7 -->

## Execution Order

- T001 first (the classified sweep list drives everything).
- T002 before T003–T006 (the canon file settles the replacement wording the others echo).
- T012–T014 last; T014 is the gate before finishing apply.

## Acceptance

### Functional Completeness

- [x] A-001 R1: Every dispatch site in `src/kit/skills/` instructs `fab agent <stage> -o yaml` with the `dispatch:`-key branch, `model_alias` model seam, and `effort` key; no skill instructs running `fab resolve-agent` for stage dispatch
- [x] A-002 R3: `_cli-fab.md` § fab resolve-agent carries a deprecation banner naming § fab agent, with its reference content intact
- [x] A-003 R4: `stage-models.md` § Skill wiring and `harness-adapters.md` present the YAML surface; aggregate specs swept
- [x] A-004 R5: All 11 memory content files migrated; `fab memory-index --check` clean; log files byte-untouched
- [x] A-005 R6: Both Go string literals name the new surface; pinned tests updated; `go test ./...` green in `src/go/fab`

### Behavioral Correctness

- [x] A-006 R2: No choreography change — start-probe discovery, gate, reap timing, continuation, profile fixity textually preserved; the "unlabelled" rationale re-anchored (rung labelled but unconsumed until Change 4), and no prose branches on `rung:`
- [x] A-007 R6: `cmd/fab/resolve_agent.go` and resolve-agent golden/parity expectations are diff-clean (frozen contract untouched)

### Scenario Coverage

- [x] A-008 R1: The R1 grep scenario holds: no remaining `resolve-agent --alias` *instruction* in `src/kit/skills/` outside the `_cli-fab.md` deprecated-command reference
- [x] A-009 R5: The R5 grep scenario holds: every remaining `docs/memory/` occurrence (logs excluded) is deprecated-contract or history, never a consumer instruction

### Edge Cases & Error Handling

- [x] A-010 R7: Phrase-class residuals (`--alias`, `dispatch=`, "`model=` line") classified with zero unclassified instructional leftovers; historical artifacts (`log.md`/`log.seed.md`, `archive/`, `docs/findings/`, `fab/plans/`) untouched
- [x] A-011 R6: Non-Claude `model_alias: ""` case documented at the native seam exactly as today's empty-`model=` inherit signal (no invented fallback)

### Code Quality

- [x] A-012 Pattern consistency: edits match each file's existing voice and structure (skills' imperative canon style, specs' RFC-2119 style, memory's FKF present-truth style)
- [x] A-013 No unnecessary duplication: owner-or-pointer respected — no file both states the migrated rule and points at its owner; `_preamble.md` stays the canon the others point at
- [x] A-014 Canonical source only: all skill edits in `src/kit/skills/` (never `.claude/skills/`); CLI-instruction changes reflected in `_cli-fab.md`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/cmd/fab/resolve_agent.go` and its frozen line-projection tests — pipeline skills no longer consume this surface; retain it through the documented compatibility window, then reassess it in the planned post-window retirement change.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Surfacing rule's minimum field list = `provider`/`model`/`model_alias`/`effort` + `dispatch:` presence | Transliterates the existing lines rule; exact list is editorial and cheap to adjust | S:65 R:85 A:80 D:70 |
| 2 | Confident | `sweep-list.md` lives in the change folder as an apply artifact (not committed to docs) | Matches prior sweep-heavy runs' working style; keeps the classified list reviewable | S:60 R:90 A:80 D:75 |
| 3 | Confident | Deprecation banner wording follows the plan doc: "kept working for out-of-band users and muscle memory; deletion is a later post-window change" | Plan doc § resolve-agent disposition states it verbatim | S:75 R:85 A:85 D:80 |

3 assumptions (0 certain, 3 confident, 0 tentative).
