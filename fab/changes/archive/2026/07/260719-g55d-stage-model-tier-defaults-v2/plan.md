# Plan: Stage-Model Tier Defaults v2

**Change**: 260719-g55d-stage-model-tier-defaults-v2
**Intake**: `intake.md`

> **Reworked 2026-07-20** for the user's mid-pipeline corrections: the `fast`→`ship` rename is cancelled (no `TierShip`, no migration), and two dispatch seams are added to scope — `/fab-proceed` prefix-step tier resolution (R8) and `/fab-continue` ship/review-pr tier resolution (R9).

## Requirements

### Agent tiers: taxonomy

#### R1: Six-tier set — new `hydrate` tier, NO renames
The `internal/agent` package SHALL define six role tiers — `default`, `operator`, `doing`, `review`, `hydrate`, `fast` — by ADDING a new constant `TierHydrate` (value `"hydrate"`) to the existing five. `TierFast` (value `"fast"`) SHALL be kept unchanged — there is no `TierShip` and no `fast`→`ship` rename. `IsTierName`, `TierNames`, `StageNames`, `ModelAlias`, `ResolveTier`, `Resolve`, and alias handling SHALL be mechanically unchanged (they read the maps; only map contents/constants change). A tier is stage-named only where it maps 1:1 to a single referent (`review`, `hydrate`); `default`, `doing`, and `fast` keep role names because each is multi-referent (`fast` governs the ship stage AND the `/fab-proceed` prefix-step dispatches — see R8).

- **GIVEN** the `defaultTiers` map
- **WHEN** its keys are enumerated via `TierNames()`
- **THEN** they are exactly `default`, `doing`, `fast`, `hydrate`, `operator`, `review` (sorted)
- **AND** a `hydrate` tier is present and the `fast` tier is retained (no `ship` tier exists)

#### R2: hydrate splits out of `doing` — the only mapping change
The FIXED, fab-owned `stageTiers` map SHALL map `hydrate` → `hydrate` (its own tier); `ship` → `fast` (unchanged), `intake`→`default`, `apply`→`doing`, `review`→`review`, `review-pr`→`doing` (all unchanged). The hydrate row is the ONLY mapping change. No `stage_tiers:` config and no per-stage escape hatch SHALL be added — the mapping stays fab-owned and non-overridable.

- **GIVEN** the `stageTiers` map
- **WHEN** `TierForStage("hydrate")` is called
- **THEN** it returns `hydrate` (not `doing`)
- **AND** `TierForStage("ship")` returns `fast` (unchanged from HEAD)

### Agent tiers: default profiles

#### R3: New default tier profiles (the shipped kit defaults)
The `defaultTiers` map SHALL carry these `{provider, model, effort}` profiles: `default` = claude/`claude-fable-5`/`high`; `operator` = claude/`claude-sonnet-5`/`medium`; `doing` = claude/`claude-fable-5`/`xhigh`; `review` = claude/`claude-opus-4-8`/`xhigh`; `hydrate` = claude/`claude-opus-4-8`/`high`; `fast` = claude/`claude-sonnet-5`/`medium` (the `low`→`medium` effort raise stands).

- **GIVEN** no project override
- **WHEN** `Resolve(nil, "apply")` is called (apply ∈ doing)
- **THEN** it returns `{claude, claude-fable-5, xhigh}`
- **AND** `Resolve(nil, "review")` returns `{claude, claude-opus-4-8, xhigh}`
- **AND** `Resolve(nil, "hydrate")` returns `{claude, claude-opus-4-8, high}`
- **AND** `Resolve(nil, "ship")` (via tier `fast`) returns `{claude, claude-sonnet-5, medium}`
- **AND** `Resolve(nil, "intake")` returns `{claude, claude-fable-5, high}`

### Agent tiers: stage/tier name-collision rule

#### R4: Replace the "disjoint" claim with the fixed-point collision rule + drift-guard test
Every code comment, test comment, skill, and spec that asserts the stage-name and tier-name sets are "disjoint" SHALL be replaced with the actual rule: *a tier may share a stage's name only when that stage maps to that same-named tier* (every stage-name/tier-name collision is a fixed point: `stageTiers[name] == name`). The collision set is `{review, hydrate}` only — `ship` is a stage but NOT a tier (it maps to `fast`), so it is not a collision. A new drift-guard test in the agent package SHALL assert that for every name present in both the stage set and the tier set, `stageTiers[name] == name`. `resolveStageOrTier`'s tier-first check order SHALL be unchanged.

- **GIVEN** the stage set and the tier set
- **WHEN** their intersection is computed (`review`, `hydrate`)
- **THEN** for each shared name, `stageTiers[name] == name`
- **AND** the new test fails if a future edit breaks any fixed point
- **AND** `IsTierName("ship")` is false; `IsTierName("hydrate")` is true

### Config reference generation

#### R5: configref reflects the six-tier taxonomy
`internal/configref`'s `tierStages` reference-prose map SHALL gain a `hydrate` entry (referent `hydrate`), drop `hydrate` from `doing`'s stage list (`doing` → `apply, review-pr`), and extend the `fast` entry to list its non-stage referent (`ship, /fab-proceed prefix steps`); `default` likewise lists its `/fab-proceed create-intake` referent. The `fast` key is KEPT (no rename). The rendered reference's FIXED stage→tier mapping block and built-in default profiles block SHALL reflect the new six-tier curve (both generated from the Go constants via `tierRows()`). No `renamed_from` registry entry SHALL be added (`renamed_from` stays `""` on every row).

- **GIVEN** `fab config reference`
- **WHEN** its output is rendered
- **THEN** the agent.tiers block lists six tiers (incl. `fast`, not `ship`) with the new profiles
- **AND** the FIXED mapping shows `hydrate` on its own line, `fast` → `ship, /fab-proceed prefix steps`, and `doing` as `apply, review-pr`

### Existing-override carry-forward

#### R6: No migration — the hydrate split ships as an upgrade note only
No migration file and no `renamed_from` use SHALL ship with this change — no config key changes meaning or goes inert (the `fast` tier keeps its name; `agent.tiers.doing` still governs apply/review-pr; `agent.tiers.hydrate` is a newly-recognized key that was simply ignored before). The residual — a project with an `agent.tiers.doing` override previously governed its hydrate stage through it, and after the split hydrate resolves the new `hydrate` kit default (opus-4-8/high) unless the project adds a `hydrate:` override — SHALL be documented as an **upgrade note in `docs/specs/stage-models.md`**, not a migration.

- **GIVEN** a project with an `agent.tiers.doing` override but no `hydrate` override
- **WHEN** it upgrades to this kit
- **THEN** its hydrate stage resolves the `hydrate` kit default (opus-4-8/high), not its `doing` value
- **AND** no `src/kit/migrations/` file is added and no `renamed_from` is set

### Docs / specs / skills sweep

#### R7: Sweep the full doc/spec/skill mirror class for the taxonomy change
Every doc/spec/skill site that enumerates the five tiers, states old default profiles, asserts "disjoint", or names `ship` as a tier SHALL be updated to the six-tier taxonomy (`fast` kept), new profiles, and the fixed-point rule (collision set `{review, hydrate}`). This includes `docs/specs/stage-models.md` (both drift-guarded tables, role-tiers table, "Why these defaults", § Resolution, § Haiku excluded, § apply↔review coupling, § Fable upgrade path + the new upgrade note, § Config schema example, § Skill wiring), `src/kit/skills/_preamble.md` + `docs/specs/skills/SPEC-_preamble.md`, `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md`, `docs/specs/config.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md`, and `cmd/fab/agent.go` docs.

- **GIVEN** a repo-wide grep of `ship`-as-tier, "disjoint", "five role tiers", and the old profile strings (excluding `docs/findings/`, `docs/memory/`, change folders)
- **WHEN** apply finishes
- **THEN** no spec/skill/comment site outside `docs/memory/` still asserts the five-tier taxonomy, names `ship` as a tier, states the old profiles, or asserts "disjoint"
- **AND** `docs/specs/stage-models.md`'s two drift-guarded tables match the Go maps (the doc test passes)

### Dispatch-seam tiering (new in scope)

#### R8: `/fab-proceed` prefix steps resolve tiers per-step (skill wiring only)
`src/kit/skills/fab-proceed.md` SHALL replace its prefix-step model-resolution exemption with per-step tier resolution: the `/fab-switch` and `/git-branch` prefix-step dispatches resolve `fab resolve-agent fast --alias`; the `_intake` create-intake dispatch resolves `fab resolve-agent default --alias`. Each SHALL surface the resolved `model=/effort=` (compliance visibility) and dispatch through the two seams (model on the Agent tool `model` param, effort as an imperative prompt line; empty ⇒ omit either). This is **tier-NAME** resolution — the resolver already accepts tier names positionally — so NO Go change is required. The SPEC mirror `docs/specs/skills/SPEC-fab-proceed.md` SHALL be updated to match.

- **GIVEN** `/fab-proceed` runs a prefix step
- **WHEN** it dispatches `/fab-switch` or `/git-branch`
- **THEN** it first runs `fab resolve-agent fast --alias` and applies the profile through the two seams
- **AND** the `_intake` create-intake dispatch runs `fab resolve-agent default --alias`
- **AND** no Go source changes (skill + SPEC mirror only)

#### R9: `/fab-continue`'s ship and review-pr rows resolve tiers (caller-invariance)
`src/kit/skills/fab-continue.md` SHALL resolve `fab resolve-agent ship --alias` (ship row) and `fab resolve-agent review-pr --alias` (review-pr `active` and `failed` rows) before delegating to `/git-pr` / `/git-pr-review`, surfacing `model=/effort=` and applying the two seams — mirroring `/fab-fff` Steps 4–5 exactly. `/git-pr` and `/git-pr-review` SHALL continue to self-manage their own `fab status` transitions (only the model/effort seam is added; no `dispatch=` block-adapter branch applies). The Dispatch-shorthand note SHALL be updated, plus the SPEC mirror `docs/specs/skills/SPEC-fab-continue.md`. The target invariant — **a stage resolves the same tier regardless of which caller drives it** (`/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-proceed`) — SHALL be stated in the skill/spec.

- **GIVEN** plain `/fab-continue` reaches the ship (or review-pr) row
- **WHEN** it delegates to `/git-pr` (or `/git-pr-review`)
- **THEN** it first runs `fab resolve-agent ship --alias` (or `review-pr`) and applies the two seams
- **AND** `/git-pr` / `/git-pr-review` still run their own `finish`/`fail` transitions
- **AND** the caller-invariance invariant is documented

### Non-Goals

- **`fast`→`ship` rename** — cancelled by user correction; `fast` stays multi-referent (ship stage + prefix steps), so a stage name would misname it.
- **Migration file / `renamed_from`** — no key restructuring, so neither ships (see R6).
- **Sticky-apply** — reusing the named apply agent across apply↔review rework cycles; a separate follow-up change.
- **User-overridable stage→tier mapping** (`stage_tiers:` or per-stage `agent.stages:`) — rejected; taxonomy stays fab-owned.
- **`docs/memory/` edits** — memory updates are hydrate's job. The grep sweep's memory hits are handled at hydrate.

### Design Decisions

#### The hydrate split ships as an upgrade note, not a migration
**Decision**: Ship no migration file and set no `renamed_from`; document the `agent.tiers.doing`→hydrate residual as an upgrade note in `docs/specs/stage-models.md`.
**Why**: With the `fast`→`ship` rename cancelled, no config KEY changes meaning or goes inert. `agent.tiers.doing` still governs apply and review-pr exactly as before; `agent.tiers.hydrate` is a newly-recognized key that was previously ignored. The only behavioral shift is that a project overriding `doing` no longer governs hydrate through it — a defaults change a project opts back into by adding a `hydrate:` override. Since nothing restructures user data, the migration convention (context.md § Migrations) does not trigger; an upgrade note is the right documentation surface.
**Rejected**: A migration file (nothing to restructure — a no-op migration would be dead code); `renamed_from` (no key was renamed).
*Introduced by*: 260719-g55d-stage-model-tier-defaults-v2

#### `fast` keeps its role name because it is multi-referent
**Decision**: Do not rename `fast`; a tier is stage-named only where it maps 1:1 to a single referent (`review`, `hydrate`).
**Why**: After R8, `fast` governs the ship stage AND the `/fab-proceed` prefix-step dispatches (`/fab-switch`, `/git-branch`) — two referents. A stage name (`ship`) would misname it. `default` and `doing` are likewise multi-referent and keep role names. Keeping `fast` also eliminates the carry-forward migration and the `renamed_from` sub-key question entirely.
**Rejected**: Renaming `fast`→`ship` (initially planned; cancelled by user correction — misnames a multi-referent tier and forces an unnecessary migration).
*Introduced by*: 260719-g55d-stage-model-tier-defaults-v2

#### Two untiered dispatch seams are closed via tier-name resolution (no Go change)
**Decision**: Tier the `/fab-proceed` prefix steps (R8) and `/fab-continue`'s ship/review-pr rows (R9) by having the skills call `fab resolve-agent <tier|stage> --alias` and apply the two seams; make no Go change.
**Why**: The resolver already accepts a tier name positionally (the `fab agent <tier>` path), so prefix steps can resolve `fast`/`default` by name with zero Go work. Ship/review-pr already have a resolution contract in `/fab-fff` Steps 4–5; `/fab-continue` simply mirrors it. Both close the caller-asymmetry gap where a stage/step resolved a different (inherited) model depending on which command drove it.
**Rejected**: Adding a Go knob or a per-step config (unnecessary — the pure-query resolver already covers it); leaving the seams untiered (breaks the caller-invariance invariant the intake names).
*Introduced by*: 260719-g55d-stage-model-tier-defaults-v2

## Tasks

### Phase 1: Core Go maps + constants

- [x] T001 In `src/go/fab/internal/agent/agent.go`: add `TierHydrate` (value `"hydrate"`), KEEP `TierFast` (value `"fast"`) — no `TierShip`; refresh the `Role-tier names` doc comment (six tiers; `fast` multi-referent — ship + prefix steps) and per-constant comments <!-- R1 -->
- [x] T002 Update `defaultTiers` in `agent.go` to the six new profiles (default fable/high, operator sonnet/medium, doing fable/xhigh, review opus/xhigh, hydrate opus/high, fast sonnet/medium) <!-- R3 -->
- [x] T003 Update `stageTiers` in `agent.go`: `hydrate`→`TierHydrate` (only change); `ship`→`TierFast` unchanged; refresh the map doc comment (collision set `{review, hydrate}`; ship maps to fast) <!-- R2 -->
- [x] T004 Rewrite the `IsTierName` doc comment in `agent.go` and the `resolveStageOrTier`/command doc comments in `src/go/fab/cmd/fab/resolve_agent.go` to state the fixed-point rule with collision set `{review, hydrate}` (drop "disjoint"; note ship→fast); behavior unchanged <!-- R4 -->

### Phase 2: configref reference generation

- [x] T005 Update `tierStages` in `src/go/fab/internal/configref/configref.go`: add `agent.TierHydrate` → `"hydrate"`, KEEP `agent.TierFast` (extend its referents to `ship, /fab-proceed prefix steps`), extend `default`'s referents to include `/fab-proceed create-intake`, change `agent.TierDoing` to `"apply, review-pr"`; leave `RenamedFrom` empty on all rows <!-- R5 -->

### Phase 3: No migration

- [x] T006 [reworked: rename cancelled] DELETE `src/kit/migrations/2.16.4-to-2.17.0.md` — no key rename means no carry-forward and no migration; `renamed_from` stays `""`. The doing-override/hydrate residual is documented as an upgrade note in stage-models.md (T010) instead <!-- R6 -->

### Phase 4: Go tests

- [x] T007 Update `src/go/fab/internal/agent/agent_test.go`: `TestResolveDefaults` (ship via fast = sonnet/medium; hydrate own tier opus/high; doing fable/xhigh; review opus/xhigh; default fable/high), `TestResolvePerFieldMerge`/`TestResolveVerbatimNoValidation` (use `fast` tier name, not `ship`), `TestIsTierName` (`"hydrate"` a tier; `"ship"` stays in the not-a-tier list), `TestTablesExhaustive` (tier set `default,doing,fast,hydrate,operator,review`), add `TestStageTierCollisionsAreFixedPoints` (collision set `{review, hydrate}`) <!-- R1 R2 R3 R4 -->
- [x] T008 <!-- rework: review cycle 1 — line 55 comment still says "ship tier"; restore "fast tier" --> Update `src/go/fab/cmd/fab/resolve_agent_test.go`: byte-exact defaults (intake fable/high, ship→fast sonnet/medium, apply fable/xhigh, review opus/xhigh, hydrate opus/high), `fast` (not `ship`) tier-key fixtures, disjoint→fixed-point comment <!-- R3 R4 -->
- [x] T009 Update `src/go/fab/cmd/fab/config_test.go` comments ("five"→"six" tiers); confirm reference-render coverage assertions still pass (they key on `doing`). Revert the `internal/config/config_test.go` fixture key back to `fast` (ship is not a tier) <!-- R5 R7 -->

### Phase 5: Docs / specs / skills sweep

- [x] T010 Update `docs/specs/stage-models.md`: both drift-guarded tables (default profiles → six rows incl. `fast`; stage→tier → hydrate own tier, ship→fast), role-tiers table (six, `fast` multi-referent, naming rationale), "Why these defaults", § Resolution fixed-point rule (collision `{review, hydrate}`, ship→fast), § Haiku excluded (ship stage governed by `fast`), § apply↔review coupling (doing fable/xhigh), § Fable upgrade path, § Config schema example, the new **doing-override upgrade note** (R6), and § Skill wiring (caller-invariant resolution: R8 prefix steps + R9 ship/review-pr) <!-- R7 R6 R8 R9 -->
- [x] T011 Update `src/kit/skills/_preamble.md` § Always Load config.yaml description ("five role tiers"→"six") <!-- R7 -->
- [x] T012 Verify `docs/specs/skills/SPEC-_preamble.md` — no tier-count edit needed (its "five" mentions are dispatch states, not tiers) <!-- R7 -->
- [x] T013 Update `src/kit/skills/_cli-fab.md`: § fab resolve-agent (six tier-name list with `fast`, fixed-point rule collision `{review, hydrate}`, FIXED-mapping + default-profiles prose for the curve), § fab agent (six tier list); + `cmd/fab/agent.go` doc <!-- R7 -->
- [x] T014 Update `docs/specs/skills/SPEC-_cli-fab.md` mirror: fab resolve-agent row fixed-point rule (collision `{review, hydrate}`, ship→fast) <!-- R7 -->
- [x] T015 <!-- rework: review cycle 1 — docs/specs/index.md:27 still enumerates the five-tier set; sweep class includes the specs index --> Update `docs/specs/config.md` (six tier list, `fast`), `docs/specs/glossary.md` (Role tier entry — six, `fast` multi-referent), `docs/specs/architecture.md` (config example — six tiers, `fast`, new profiles), and `docs/specs/index.md` (stage-models row — six-tier set incl. `hydrate`) <!-- R7 -->

### Phase 6: New dispatch-seam wiring (corrections 2 & 3)

- [x] T016 Update `src/kit/skills/fab-proceed.md`: replace the prefix-step model-resolution exemption note with per-step tier resolution, and add a resolution step to each of the three prefix-step dispatch procedures (`_intake` → `default`; `/fab-switch` + `/git-branch` → `fast`) <!-- R8 -->
- [x] T017 Update `docs/specs/skills/SPEC-fab-proceed.md`: the per-stage-model paragraph — prefix steps now tiered (fast for switch/branch, default for create-intake), skill wiring only <!-- R8 -->
- [x] T018 Update `src/kit/skills/fab-continue.md`: add `fab resolve-agent ship --alias` / `fab resolve-agent review-pr --alias` to the ship, review-pr `active`, and review-pr `failed` rows; rewrite the Dispatch-shorthand note (ship/review-pr tiered, mirror fab-fff Steps 4–5, self-manage transitions, no dispatch= branch); state the caller-invariance invariant <!-- R9 -->
- [x] T019 Update `docs/specs/skills/SPEC-fab-continue.md`: ship/review-pr are tiered (mirror fab-fff Steps 4–5), still self-manage transitions; state the caller-invariance invariant <!-- R9 -->

### Phase 7: Verify

- [x] T020 Run affected Go packages (`internal/agent`, `internal/configref`, `internal/config`, `cmd/fab`); fix failures; then `go build ./...`, `go vet`, and the full module suite; render `fab config reference` to confirm the six-tier taxonomy (fast, not ship) <!-- R1 R2 R3 R4 R5 R7 -->

## Execution Order

- T001–T003 precede T005 (configref reads `agent.TierHydrate`/`TierFast`) and T007/T008 (tests assert the new maps).
- T004 precedes T007 (fixed-point test lives in the agent package).
- T010 must land before T020 (`stagemodels_doc_test.go` parses `stage-models.md`'s tables — they must match the Go maps).
- T006 (delete migration) is independent.
- Phase 6 (T016–T019, skill wiring) is independent of the Go changes.

## Acceptance

### Functional Completeness

- [x] A-001 R1: The six-tier set `default`/`operator`/`doing`/`review`/`hydrate`/`fast` exists; `TierHydrate` (`"hydrate"`) is added and `TierFast` (`"fast"`) is retained (no `TierShip`); `TierNames()` returns the six names.
- [x] A-002 R2: `stageTiers` maps `hydrate`→`hydrate` (only change) and `ship`→`fast` (unchanged); `apply`/`review-pr`→`doing`, `review`→`review`, `intake`→`default`.
- [x] A-003 R3: `Resolve` returns the tabled profiles per stage (apply fable/xhigh, review opus/xhigh, hydrate opus/high, ship via fast sonnet/medium, intake fable/high, operator sonnet/medium).
- [x] A-004 R4: A drift-guard test asserts every stage/tier name collision is a fixed point (collision set `{review, hydrate}`); `IsTierName("ship")` is false, `IsTierName("hydrate")` is true; no "disjoint" claim remains in Go comments/tests.
- [x] A-005 R5: `fab config reference` renders six tiers (incl. `fast`, not `ship`) with the new profiles; the FIXED mapping shows hydrate own tier, `fast` → `ship, /fab-proceed prefix steps`, and doing as `apply, review-pr`; no `renamed_from` added.
- [x] A-006 R6: No `src/kit/migrations/` file ships; no `renamed_from` is set; the doing-override→hydrate residual is documented as an upgrade note in stage-models.md.
- [x] A-007 R7: All doc/spec/skill/comment sites (outside `docs/memory/`, `docs/findings/`, change folders) reflect the six-tier taxonomy (`fast` kept), new profiles, and the fixed-point rule (collision `{review, hydrate}`); no site names `ship` as a tier. *(Review cycle 2: both cycle-1 residues fixed — `docs/specs/index.md:27` now lists the six-tier set incl. `hydrate`; `resolve_agent_test.go:55` says "fast tier". Repo-wide greps for "disjoint"/five-tier/ship-as-tier are clean; the only five-tier text left is the version-pinned historical migration `src/kit/migrations/2.12.1-to-2.13.0.md`, which correctly describes the 2.13.0 shape.)*
- [x] A-008 R8: `/fab-proceed`'s three prefix-step dispatch procedures resolve tiers by name (`_intake`→`default`, `/fab-switch`+`/git-branch`→`fast`) and apply the two seams; the exemption note is replaced; SPEC mirror updated; no Go change.
- [x] A-009 R9: `/fab-continue`'s ship and review-pr (active + failed) rows resolve `fab resolve-agent ship`/`review-pr --alias` before delegating, mirroring `/fab-fff` Steps 4–5; `/git-pr`/`/git-pr-review` keep self-managing transitions; SPEC mirror updated; caller-invariance stated.

### Behavioral Correctness

- [x] A-010 R2: `hydrate` resolves to its own tier (opus/high), not `doing` (fable/xhigh) — verified by `TestResolveDefaults`.
- [x] A-011 R2: `ship` still resolves via the `fast` tier (sonnet/medium) — unchanged mapping; a `fast:` override still governs the ship stage.
- [x] A-012 R7: `docs/specs/stage-models.md`'s two drift-guarded tables match the Go maps (`TestDocTablesMatchAgentMaps` passes).

### Removal Verification

- [x] A-013 R1/R6: No `TierShip` constant, no `ship`-as-tier reference in Go/specs/skills (outside `docs/memory`/`docs/findings`), and no `src/kit/migrations/2.16.4-to-2.17.0.md` remain. *(Review cycle 2: the cycle-1 residue is fixed — `resolve_agent_test.go:55` now reads "ship resolves to the fast tier default."; grep for `TierShip` and ship-as-tier is clean; `src/kit/migrations/` carries no 2.16.4-to-2.17.0 file.)*

### Scenario Coverage

- [x] A-014 R3: A per-field override (e.g. `fast: { effort: low }`) merges over the new `fast` default (sonnet/medium) — verified in `resolve_agent_test.go` / `agent_test.go`.
- [x] A-015 R8/R9: The caller-invariance invariant (a stage/step resolves the same tier regardless of caller) holds across `/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-proceed` and is documented.

### Edge Cases & Error Handling

- [x] A-016 R4: `resolveStageOrTier` behavior is unchanged (tier-first order retained); resolving `review`/`hydrate` by name and by stage yields identical profiles (fixed-point collisions); `ship` resolves only as a stage.

### Code Quality

- [x] A-017 Pattern consistency: New tier/constant naming and map style follow the existing `agent.go` conventions (provider written explicitly on every line).
- [x] A-018 No unnecessary duplication: configref reference stays generated from the Go constants (no literal copy of tier values); skill wiring reuses the existing two-seam resolution contract.
- [x] A-019 Canonical source only: no edits under `.claude/skills/`; all skill edits are in `src/kit/skills/` with SPEC mirrors updated.
- [x] A-020 SPEC-mirror sync: every `src/kit/skills/*.md` edit (`_preamble`, `_cli-fab`, `fab-proceed`, `fab-continue`) carries its `docs/specs/skills/SPEC-*.md` update. (SPEC-_preamble verified per T012: it restates no tier count — no edit site exists.)
- [x] A-021 CLI ⇒ docs + tests: the tier-name/default changes visible via `fab resolve-agent`/`fab agent`/`fab config reference` update `_cli-fab.md` and ship test updates.
- [x] A-022 Go changes ship tests: every touched `.go` file has corresponding test updates; tests conform to the spec.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds a tier and rewires existing dispatch/doc surfaces without making existing code redundant: `TierFast` and every prior tier/stage constant remain live referents (verified: `stageTiers`, `tierStages`, and the operator launcher all still consume them), the configref registry rows are all still consumed, no function or branch lost its callers in the diff, and the one artifact the mid-pipeline rework made redundant (`src/kit/migrations/2.16.4-to-2.17.0.md`) was already deleted by task T006 (verified absent).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New `hydrate` tier; NO renames — `fast` keeps its name (the `ship` rename was cancelled by user correction); six-tier set `default`/`operator`/`doing`/`review`/`hydrate`/`fast` | User-issued mid-pipeline correction — explicit and final | S:95 R:75 A:95 D:95 |
| 2 | Certain | Stage→tier split: hydrate→`hydrate` is the ONLY mapping change; ship stays on `fast`; mapping stays fab-owned and non-overridable | Discussed — upstream taxonomy evolution the spec reserves to fab-kit | S:95 R:75 A:90 D:95 |
| 3 | Certain | New default profiles exactly as tabled (default fable/high, doing fable/xhigh, review opus/xhigh, hydrate opus/high, fast sonnet/medium, operator sonnet/medium) | Discussed — specific values agreed; the fast low→medium raise stands; one-map change | S:95 R:85 A:90 D:90 |
| 4 | Certain | Fixed-point collision rule + drift-guard test; collision set is `{review, hydrate}` only; `"ship"` stays in TestIsTierName's not-a-tier list | User correction pins the collision set; the claim is already false today (`review` collides) | S:90 R:85 A:95 D:90 |
| 5 | Certain | No migration file and no `renamed_from` use — nothing restructures user data; the doing-override/hydrate residual ships as an upgrade note in stage-models.md | User correction: no key changes meaning or goes inert, so the migration convention does not trigger | S:90 R:80 A:90 D:90 |
| 6 | Confident | `docs/specs/skills/SPEC-_preamble.md` needs no tier-count edit (its "five" mentions are the five dispatch STATES, not tiers) | Grep confirms SPEC-_preamble's "five" hits are `five-state`/`five states`; no tier-count phrase | S:80 R:85 A:80 D:75 |
| 7 | Confident | `config_test.go` reference-coverage assertions keep passing with only comment updates (they key on the `doing` tier; no assertion embeds a tier literal or old profile string) | Reviewed config_test.go: assertions check `GetAgentTier("doing")` + JSON `default` presence, not specific tier values | S:80 R:80 A:80 D:75 |
| 8 | Certain | `resolveStageOrTier` tier-first order, `IsTierName`/`ModelAlias`/`ResolveTier` mechanics, and alias handling stay untouched; only comments and map contents change | Intake states mechanics untouched; identity collisions make order immaterial | S:95 R:85 A:90 D:90 |
| 9 | Certain | `/fab-proceed` prefix steps resolve tiers per-step by NAME (`fast` for `/fab-switch`+`/git-branch`, `default` for `_intake` create-intake); skill wiring only, no Go change | User-issued correction — explicit values per step; resolver already accepts tier names positionally | S:95 R:85 A:90 D:95 |
| 10 | Certain | `/fab-continue`'s ship and review-pr (active + failed) rows resolve `ship`/`review-pr` before delegating, mirroring fab-fff Steps 4–5; those skills keep self-managing transitions; caller-invariance is the target invariant | User-issued correction — explicit contract named | S:95 R:85 A:90 D:95 |

10 assumptions (8 certain, 2 confident, 0 tentative).
