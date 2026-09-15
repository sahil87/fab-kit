# Plan: Shared agent.Resolution Struct + Parity Tests

**Change**: 260901-mp8d-agent-resolution-struct-parity
**Intake**: `intake.md`

## Requirements

### Resolution Engine: The Shared Struct

#### R1: One resolution-result struct
`internal/agent` SHALL define a `Resolution` struct holding the complete resolution result: `Selector`, `Kind` (`role`|`stage`|`provider`), `Role`, `Provider`, `Model` (full ID), `ModelAlias` (Agent-tool short alias via `ModelAlias()`, empty for non-Claude IDs), `Effort`, `Template` (raw command slot, placeholders intact), `FillMode` (`template`|`append`), `Command` (substituted line), `Dispatch` (nil pointer, or `{Rung: pane|headless, Command: <substituted>}`), and `Source` (per-field provenance). Composition of this struct MUST be the only place resolution results are assembled — no second composition site survives in `cmd/fab/agent.go` or `cmd/fab/resolve_agent.go`.

- **GIVEN** the refactor is complete
- **WHEN** grepping `cmd/fab` for profile→output assembly (`formatAgentProfile` semantics, `agentResolutionYAML`, `dispatchLineFor` call sites)
- **THEN** both commands obtain a composed `agent.Resolution` from one shared composer and only *render* it

#### R2: resolve-agent output is a byte-identical line projection
`fab resolve-agent`'s flags, arguments, and stdout bytes SHALL NOT change. Its ordered `model=`/`effort=`/`provider=`/`dispatch=` lines (omit-when-empty; empty model still emits `model=`; `--alias` maps only the `model=` line via `ModelAlias`; `dispatch=` always embeds the full model ID) become a line-projection method of `agent.Resolution`.

- **GIVEN** the golden tests written before the refactor (R6)
- **WHEN** the same invocations run after the refactor
- **THEN** stdout is byte-identical for the full matrix, and the line renderer is a projection of the struct

#### R3: `fab agent -o yaml` carries the full schema (additive)
The `-o yaml` document SHALL extend Change 1's seven keys (`selector`, `kind`, `role`, `provider`, `model`, `effort`, `command` — names, values, and relative order preserved) with `model_alias`, `template`, `fill_mode`, `source`, and `dispatch`. `dispatch:` key absent ⇔ native rung; when present it carries a **labelled** `rung: pane|headless` plus the substituted `command`. `model_alias` is always emitted for Claude IDs and empty for non-Claude. Only the `-o yaml` sink derives dispatch; `--print`, exec, and `-t` outputs stay byte-identical and derive nothing new.

- **GIVEN** a config resolving the doing role to claude (native capability) under default `dispatch.mode: native`
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** the document has all non-dispatch keys, `model_alias: opus` (for a claude-opus-* model), and NO `dispatch:` key
- **GIVEN** a provider whose ladder lands on a CLI rung (e.g. headless under native preference)
- **WHEN** `fab agent <stage> -o yaml` runs
- **THEN** `dispatch:` is present with `rung: headless` and the full-ID-substituted command

#### R4: Per-field provenance (`source`)
The struct SHALL record which config tier / fill rung supplied `provider`, `model`, and `effort` — vocabulary drawn from the actual precedence rungs: `flag`, `agent.profiles.<role>`, `agent.session` / `agent.workers`, `providers.<name>.profiles.<role>`, `providers.<name>.profiles.default`, `built-in`, and an empty-value signal for the inherit case. Provenance MUST come from the same single implementation of the fill precedence (`ResolveRoleWith` refactors to delegate to a traced variant — the precedence is implemented once).

- **GIVEN** a project config with `agent.profiles.review.model` pinned and provider fills for effort
- **WHEN** the review role resolves
- **THEN** `source.model` names `agent.profiles.review` and `source.effort` names the provider fill rung that supplied it

#### R5: Shared dispatch derivation
The dispatch-rung derivation currently local to `cmd/fab/resolve_agent.go` (`dispatchLineFor` — `dispatch.SelectMode` over provider capabilities + `$TMUX`, preference from `cfg.GetDispatchMode()`) SHALL be composed once in the shared composer and consumed by both surfaces. `$TMUX` stays read at the cobra layer and passed in (SelectMode purity precedent). On the `-o yaml` sink a no-reachable-rung condition surfaces the same actionable error resolve-agent emits (`noDispatchCapabilityError`) — parity includes the error posture.

- **GIVEN** identical config and environment
- **WHEN** `fab resolve-agent apply` and `fab agent apply -o yaml` both run
- **THEN** the `dispatch=` line's presence and command equal the `dispatch:` key's presence and `command` field

#### R6: Golden + parity test matrix
Golden tests SHALL pin `fab resolve-agent`'s exact bytes BEFORE the refactor, and a parity matrix SHALL assert resolve-agent's lines ≡ the struct's line projection, across: templated vs plain (append-mode) provider commands; empty fills (empty `model=` inherit signal); `--alias` on/off; non-Claude ID pass-through (`gpt-5` verbatim); dispatch present (pane rung, headless rung) and absent (native); dated Claude variants (`claude-haiku-4-5-20251001` → `haiku`). `fab agent`'s existing sinks (`--print`, `-t`, Change 1's minimal `-o yaml` keys) keep golden coverage.

- **GIVEN** the matrix cases as table-driven tests
- **WHEN** `go test ./src/go/fab/cmd/fab/... ./src/go/fab/internal/agent/...` runs
- **THEN** all golden and parity cases pass

#### R7: Documentation
`src/kit/skills/_cli-fab.md` § fab agent SHALL document the full `-o yaml` schema (all keys, absence ⇔ native, labelled rung, `model_alias` and `source` semantics); § fab resolve-agent SHALL gain exactly one line noting the shared resolution engine — no deprecation banner (that is Change 3), no contract change. `docs/specs/stage-models.md` gains a surgical mention that the structured YAML surface carries the full resolution. Stale forward-pointing prose from Change 1 ("minimal in this change; the schema extends additively later" in `cmd/fab/agent.go`, the minimal-key list in `_cli-fab.md`) SHALL be updated in place.

- **GIVEN** the docs sweep is done
- **WHEN** grepping `-o yaml`, "minimal projection", "schema extends additively", `agentResolutionYAML` repo-wide
- **THEN** no stale claim about the minimal schema survives outside archived changes and the plans folder

### Non-Goals

- No mutation of `fab resolve-agent`'s flags/args/output — parity by construction, never by editing it (plan non-goal; deprecation banner is Change 3, deletion post-window)
- No change to the operator launcher path (`roleSessionCommand`, `operator.go` `WithProfile` direct), `fab pane open` / `fab batch` / `fab dispatch start` composition
- No change to `spawn.WithProfile` semantics (template/append modes, empty-value token-drop); the fill-mode seam only *exposes* the existing discriminator
- No skill-prose dispatch-site migration (Change 3) and no choreography change (Change 4)
- No provider grammar validation on any new surface (verbatim pass-through)
- Bare `--provider` fill-bypass semantics untouched

### Design Decisions

#### Composer in cmd/fab, struct + projections in internal/agent
**Decision**: the `Resolution` struct, its line projection, and its YAML serialization live in `internal/agent`; the one *composer* function that fills it (calling `agent.ResolveRoleWith`/traced variant, `spawn.WithProfile`, `dispatch.SelectMode`) lives in `cmd/fab` (package main, shared by both commands — e.g. a new `cmd/fab/resolution.go`).
**Why**: `internal/spawn` imports `internal/agent` (spawn.go:14), so `internal/agent` cannot call `spawn.WithProfile` without an import cycle; both commands share package main, so one composer there is a real single site with zero graph surgery.
**Rejected**: hoisting `WithProfile` into `internal/agent` (touches 7 template-substitution consumer sites the plan freezes); a new `internal/resolution` package importing agent+spawn+dispatch (viable, but a whole package for one composer both consumers of which are in cmd/fab is structure without payoff — revisit at Change 3 if a third consumer appears).
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

#### Provenance via a traced resolver, single precedence implementation
**Decision**: add a traced variant of the role resolution in `internal/agent` returning `(Profile, Source, error)`; `ResolveRoleWith` delegates to it and discards the trace. The `firstNonEmpty` chains become provenance-carrying picks inside the traced implementation only.
**Why**: the fill precedence must stay implemented exactly once (R4); tracing at the point each rung is consulted is the only honest provenance.
**Rejected**: post-hoc inference of provenance by comparing the resolved value against each rung's candidate (ambiguous when two rungs hold the same string).
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

#### `-o yaml` adopts resolve-agent's dispatch-error posture
**Decision**: the `-o yaml` sink derives dispatch for every addressing kind (stage, role, and bare-provider) and surfaces `noDispatchCapabilityError` when no rung at/below `dispatch.mode` is reachable — even though Change 1's `-o yaml` never errored this way.
**Why**: parity is the change's whole point; Change 3's migrating consumers need identical semantics including the actionable no-rung error. The affected invocations are exotic (a provider with no reachable rung), and `--print`/exec/`-t` gain no new error paths.
**Rejected**: omitting `dispatch:` on derivation failure (lies under the absence ⇔ native rule); deriving only for stage kind (breaks the parity matrix, which spans resolve-agent's `<stage|role>` surface).
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

## Tasks

### Phase 1: Golden Baseline (before any refactor)

- [x] T001 Golden tests pinning `fab resolve-agent`'s exact stdout bytes across the R6 matrix (templated vs plain commands, empty fills / empty `model=`, `--alias` on/off, `gpt-5` pass-through, dispatch pane/headless/native via config + `$TMUX` control, `claude-haiku-4-5-20251001` → `haiku`) in `src/go/fab/cmd/fab/resolve_agent_test.go`; MUST pass against the current code before T006+ <!-- R6 -->
- [x] T002 [P] Confirm/extend golden coverage for `fab agent --print`, `-t`, and Change 1's seven-key `-o yaml` doc in `src/go/fab/cmd/fab/agent_test.go` (representative invocations incl. bare-provider and selector+--provider forms); MUST pass before T006+ <!-- R6 -->

### Phase 2: Struct, Provenance, Seams (internal packages)

- [x] T003 Define `Resolution` (+ `DispatchResolution`, `Source`) in `src/go/fab/internal/agent/agent.go` (or a new `resolution.go` in that package) with YAML tags ordering Change 1's seven keys first; implement the resolve-agent line projection (absorbing `formatAgentProfile`'s omit-when-empty semantics + `--alias` mapping) with unit tests in `internal/agent` <!-- R1, R2 -->
- [x] T004 Traced role resolution in `src/go/fab/internal/agent/agent.go`: `(Profile, Source, error)` variant implementing the existing precedence once, `ResolveRoleWith` delegating; unit tests covering every provenance rung (flag override, `agent.profiles.<role>`, depth knobs, `providers.<p>.profiles.<role>`, `.default`, built-in, empty/inherit) <!-- R4 -->
- [x] T005 [P] Export the fill-mode discriminator from `src/go/fab/internal/spawn/spawn.go` (e.g. `IsTemplate` wrapping the existing `isTemplate`) + test; no semantic change to `WithProfile` <!-- R1 -->

### Phase 3: Composer + Renderers

- [x] T006 Single composer in a new `src/go/fab/cmd/fab/resolution.go` (package main): compose `agent.Resolution` from (cfg, addressing kind/selector, overrides, headless flag, `$TMUX`, dispatch preference) via the traced resolver + `spawn.WithProfile` + `dispatch.SelectMode`; hoist `dispatchLineFor` semantics into it (delete the resolve_agent.go local once unused) <!-- R1, R5 -->
- [x] T007 `src/go/fab/cmd/fab/resolve_agent.go` renders via the struct's line projection; flags/args untouched; T001 goldens pass unchanged <!-- R2 -->
- [x] T008 `src/go/fab/cmd/fab/agent.go`: `-o yaml` serializes the full `Resolution` (delete `agentResolutionYAML`); `--print`/exec/`-t` ride the same composed struct without new derivations or error paths; update the stale "minimal in this change" comment block (:60-62) and the package-doc key list <!-- R3 -->
- [x] T009 Parity test matrix asserting `fab resolve-agent` output ≡ the struct line projection across the full R6 matrix (drive both through the composer in `cmd/fab` tests); plus `-o yaml` full-schema goldens (dispatch present pane/headless, absent native, `model_alias` claude/non-claude, `source` shape, no-rung error case) <!-- R6, R3, R5 -->

### Phase 4: Docs + Sweep

- [x] T010 `src/kit/skills/_cli-fab.md`: § fab agent full `-o yaml` schema table (all keys + semantics); § fab resolve-agent one shared-engine line (no deprecation banner) <!-- R7 -->
- [x] T011 [P] `docs/specs/stage-models.md`: surgical mention of the full structured resolution on the YAML surface <!-- R7 -->
- [x] T012 Sibling sweep: grep `-o yaml`, `agentResolutionYAML`, "minimal projection", "schema extends additively", "minimal in this change", "Change 2" phrase classes repo-wide (skills, aggregate specs `skills.md`/`glossary.md`/`architecture.md`, memory files, `*_test.go` comments); update every stale forward-pointing claim in place (exclude `fab/changes/` archives and `fab/plans/`) <!-- R7 -->
- [x] T013 Run `gofmt` and the affected test packages (`go test ./src/go/fab/cmd/fab/... ./src/go/fab/internal/agent/... ./src/go/fab/internal/spawn/...`), then the full `./src/go/...` suite once green <!-- R6 -->

## Execution Order

- T001/T002 MUST land (and pass) before T006–T008 touch any renderer — they are the byte-neutrality oracle
- T003 → T006 → T007/T008 → T009; T004 and T005 block T006; T010–T012 after code settles; T013 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `agent.Resolution` exists with all thirteen fields (incl. nested dispatch + source) and both commands obtain it from one shared composer — no residual assembly in either command file
- [x] A-002 R3: `fab agent <stage> -o yaml` emits the full schema; Change 1's seven keys keep names, values, and relative order; new keys are additive
- [x] A-003 R4: `source` reports the supplying rung for provider/model/effort using the precedence-rung vocabulary
- [x] A-004 R5: dispatch presence/command on the YAML surface equals resolve-agent's `dispatch=` line for identical config+env, and the rung is labelled

### Behavioral Correctness

- [x] A-005 R2: `fab resolve-agent` stdout is byte-identical before/after across the full R6 matrix (golden tests written pre-refactor, passing post-refactor unmodified)
- [x] A-006 R3: `fab agent --print`, exec composition, and `-t` output are byte-identical for every invocation legal today; only the `-o yaml` sink gained keys/derivations

### Scenario Coverage

- [x] A-007 R6: parity matrix covers templated/plain, empty fills, alias on/off, non-Claude pass-through, pane/headless/native dispatch, dated Claude variants — all table-driven and green
- [x] A-008 R5: the no-reachable-rung case errors identically (same `noDispatchCapabilityError` text) on both surfaces

### Edge Cases & Error Handling

- [x] A-009 R3: bare-provider kind (`--provider X -o yaml`) emits dispatch derived from X's capabilities; empty model still yields `model: ""` + `model_alias: ""` (inherit signal preserved, non-Claude alias empty rule holds)

### Code Quality

- [x] A-010 Pattern consistency: new code matches surrounding comment density/idiom (package-doc contracts, purity notes); YAML tags follow existing snake_case
- [x] A-011 No unnecessary duplication: fill precedence implemented once (traced resolver), template detection reused from `internal/spawn`, dispatch ladder reused from `internal/dispatch` — no re-derivations
- [x] A-012 CLI ⇒ docs + tests: `_cli-fab.md` updated and tests shipped in the same change (constitution constraint); no edits under `.claude/skills/`
- [x] A-013 Owner-or-pointer: § fab resolve-agent's new line points at the shared engine without restating the schema `_cli-fab.md` § fab agent owns

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — the refactor already removed the superseded per-command assembly helpers and no additional production code became redundant or unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Composer lives in `cmd/fab` (package main, shared file); struct + projections in `internal/agent` | Forced by the spawn→agent import direction verified at plan time; recorded as a Design Decision | S:70 R:80 A:90 D:75 |
| 2 | Confident | `-o yaml` adopts resolve-agent's no-rung error posture on every addressing kind | Parity-by-construction goal + Change 3 consumer needs; exotic-config edge accepted and tested | S:55 R:75 A:80 D:65 |
| 3 | Confident | `source` vocabulary = the literal precedence-rung names (`flag`, `agent.profiles.<role>`, `agent.session`/`agent.workers`, `providers.<p>.profiles.<role>`, `providers.<p>.profiles.default`, `built-in`) with empty-value inherit signal | Intake assumption 3 carried forward; additive key, cheap to extend | S:55 R:75 A:75 D:60 |
| 4 | Confident | Change 1's seven YAML keys keep their relative order; new keys append after them | Conservative read of "additive" from 77vz intake; struct field order controls marshal order | S:70 R:80 A:80 D:75 |

4 assumptions (0 certain, 4 confident, 0 tentative).
