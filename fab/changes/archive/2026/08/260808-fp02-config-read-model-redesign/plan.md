# Plan: Config Read-Model Redesign + UX Follow-ups

**Change**: 260808-fp02-config-read-model-redesign
**Intake**: `intake.md`

## Requirements

### Config loader: the read model

#### R1: Per-leaf deep merge with empty-skip
The config read model SHALL resolve each leaf by walking the cascade tiers from highest precedence to lowest and taking the first tier that defines a **non-empty** value for that leaf. A leaf whose value at some tier is `null`, `""`, `[]`, or `{}` SHALL neither win nor block — it falls through to the next tier down. `false` and `0` are real values and MUST NOT be skipped. Maps merge per-key recursively; lists and scalars are leaves and replace wholesale. The rule SHALL have exactly one implementation (`internal/config`), consumed by the loader and by every provenance surface.

- **GIVEN** a system config with `agent: {workers: codex}` and a project config with `agent: {workers: null}`
- **WHEN** any consumer loads config through `internal/config.LoadPath`
- **THEN** `agent.workers` resolves to `codex` — the explicit null falls through instead of shadowing
- **AND** a project `dispatch: {reap_done: false}` still resolves `false` (a bool is a real value)

#### R2: Precedence inverts to Env > System > Project > Defaults
The system layer (`~/.fab-kit/config.yaml`) SHALL outrank the project layer (`fab/project/config.yaml`). Environment overrides stay highest; built-in defaults stay lowest. Scope enforcement SHALL be retained unchanged, so the inversion is observable only for preference-class (`scope: both`/`system`) keys.

- **GIVEN** a system config setting `agent.workers: codex` and a project config setting `agent.workers: gemini`
- **WHEN** config is loaded
- **THEN** `agent.workers` resolves to `codex`
- **AND** a project-scoped key (e.g. `test_paths`) placed in the system file is still pruned with a `fab: warning:`, so no semantics-class key can be affected by the inversion

#### R3: Materialized defaults tier for the read model
The built-in defaults SHALL be projected to a YAML-shaped map (`configref.DefaultsMap` / `DefaultsMapFor`) and consumed as the real bottom tier by every read-model surface (`fab config show`, `--origin`, the `set` shadow check, the `unset` live-tier notice). The explicit-presence machinery (`originValue{value, set}`, `highestOriginValue`'s presence bit) SHALL be removed: "this tier defines the leaf" is the same non-empty test R1 defines.

- **GIVEN** a repo with no project or system config
- **WHEN** `fab config show agent.workers --origin` runs
- **THEN** exactly one line is printed, attributing the value to the `default` tier
- **AND** no code path distinguishes "present but null" from "absent"

#### R4: Keyed `show <key> --origin` lists the full stack
Keyed `fab config show <key> --origin` SHALL print one line per tier that defines the key (non-empty), highest precedence first, with the winner marked `(effective)` and the rest `(shadowed)`. No new flag is added. Bare all-keys `show --origin` SHALL remain winner-only, one line per leaf, with its current origin labels. Map-valued keys SHALL drill down per leaf, each leaf listing its own defining tiers. Bare all-keys `show` (no `--origin`) SHALL continue NOT to materialize defaults.

- **GIVEN** `FAB_AGENT_WORKERS=codex`, a system config with `agent.workers: kimi3`, and the built-in default `claude`
- **WHEN** `fab config show agent.workers --origin` runs
- **THEN** three lines are printed — env (effective), system (shadowed), default (shadowed) — each naming its tier and label
- **AND** `fab config show --origin` still prints a single winner line for `agent.workers`

#### R5: `show --origin` drill-down rows are knob-aware
The composed `agent.profiles.<role>` rows in the origin projection SHALL be resolved against the **live** config rather than against a nil config, so a depth knob naming a non-claude provider is reported honestly. Resolution SHALL stay provider-neutral: a knob naming a provider fab ships nothing for is reported verbatim, never replaced by the built-in.

- **GIVEN** a project config with `agent.workers: codex`
- **WHEN** `fab config show --origin` runs
- **THEN** `agent.profiles.doing.provider` reads `codex`, not `claude`
- **AND** the row's origin label remains `default` (it is derived, not overridden)

### `fab config` mutation UX

#### R6: `set` warns when a higher tier shadows the written value
`fab config set <key> <value>` (either target) SHALL exit 0 and print a `fab: warning:` on stderr when a higher-precedence tier defines the same key, naming that tier (the environment variable, or the file path). Writing `--system` while only the project file defines the key SHALL NOT warn (the system layer now wins).

- **GIVEN** `FAB_AGENT_WORKERS=codex` is exported
- **WHEN** `fab config set agent.workers gemini` runs
- **THEN** the write succeeds, exit code is 0, and stderr names `$FAB_AGENT_WORKERS` as the winning tier
- **AND** with the project file defining `agent.workers`, `fab config set agent.workers gemini --system` prints no shadow warning

#### R7: `set <key> ""` is refused
`fab config set` SHALL refuse an empty (or whitespace-only) scalar value with a non-zero exit and guidance pointing at `fab config unset`, because an empty leaf falls through under R1 and can never be effective.

- **GIVEN** any known scalar key
- **WHEN** `fab config set agent.workers ""` runs
- **THEN** the command exits non-zero, writes nothing, and the message names `fab config unset agent.workers` as the intended verb

#### R8: `unset`'s no-op notice names the live tier
Unsetting a key absent from the target file SHALL stay an exit-0 no-op, and SHALL additionally report where the key IS live (system/project path, or the environment variable) with the command that would remove it. When only the built-in default supplies the key, the notice stays as today.

- **GIVEN** `agent.workers` set only in `~/.fab-kit/config.yaml`
- **WHEN** `fab config unset agent.workers` runs (project target)
- **THEN** exit is 0 and the output names the system path plus `fab config unset agent.workers --system`
- **AND** when the environment defines it, the notice names the variable and states that `unset` cannot remove it

### Documentation & sweep class

#### R9: The cascade-order claim is swept repo-wide
Every statement of the cascade order — in specs, kit skills, Go doc comments, and **user-facing string literals** (cobra help/Long text, the system-scaffold header, registry descriptions and rendered segments) — SHALL state `environment > system > project > built-in defaults` and the empty-skip semantics. Each `src/kit/skills/*.md` edit SHALL carry its `docs/specs/skills/SPEC-*.md` mirror update in this change.

- **GIVEN** the repo after this change
- **WHEN** `grep -rE "project > system|environment > project|project-over-system"` runs over `src/` and `docs/`
- **THEN** no occurrence remains that asserts the old order as current behavior
- **AND** `src/kit/skills/_cli-fab.md` and `docs/specs/skills/SPEC-_cli-fab.md` both describe the new order, the full-stack keyed `--origin`, the `set` shadow warning, the empty-value refusal, and the `unset` live-tier notice

#### R10: Fail-open and scope enforcement are unchanged
An absent system file SHALL remain byte-identical to the pre-cascade single-file behavior; a malformed/unreadable system file SHALL warn and be skipped; a malformed **project** file SHALL still error; a project-scoped key in the system file or in an environment variable SHALL still warn and be ignored. `fab config set --system` SHALL stay gated on `scope ∈ {system, both}`.

- **GIVEN** a system config containing `test_paths: [...]` (project-scoped)
- **WHEN** config is loaded
- **THEN** the key is pruned with a `fab: warning:` and the project file's value stands
- **AND** a malformed system file leaves the project file's values effective with a warning and no error

### Non-Goals

- No migration file — nothing on disk changes shape (intake assumption 7).
- The point-of-use default fallback seams stay in place (intake assumption 6); collapsing them wholesale remains C5 item 1.
- `dispatch.reap_done` keeps its `*bool` modeling — see Design Decisions.
- The remaining C5 items (dispatch constants into `defaults.yaml`, retiring the fab-kit stub config, `upgrade --check`, fence scope annotations) stay out.
- `docs/memory/` updates are hydrate's stage, not apply's.

### Design Decisions

#### The defaults tier materializes at the read-model boundary, not inside the loader
**Decision**: `configref.DefaultsMap`/`DefaultsMapFor` project the registry's canonical defaults into a YAML-shaped map that every read-model surface (show, `--origin`, the `set` shadow check, the `unset` notice) merges as tier 0 through the single shared merge rule. `internal/config.LoadPath` keeps merging environment + system + project and leaves built-in values to the retained point-of-use seams.
**Why**: Two independent constraints. (1) `internal/config` cannot import `internal/configref` — `configref → agent → config` would close an import cycle, and the intake names the registry as the projection's source. (2) More decisively, the registry's `agent.profiles` default is a *derived* value (resolved from the provider fills); merging it into the loader's map would land a full `{provider, model, effort}` in `Config.Agent.Profiles`, which the resolver reads as a **user override** outranking the provider's own fills — inverting the documented fill precedence and breaking `--provider` swaps. A registry-defaults tier is sound for display and provenance, not for the resolver.
**Rejected**: Moving `defaults.yaml` into a new leaf package so the loader could merge tier 0 itself (buys the same effective values the retained point-of-use seams already produce, at the cost of a structural refactor the intake's Impact list does not name — and does not fix the `agent.profiles` poisoning above); injecting a defaults provider into `internal/config` via a package var (spooky action, and `internal/config`'s own tests would see a different tier stack than the binary).
*Introduced by*: 260808-fp02-config-read-model-redesign

#### `dispatch.reap_done` stays a `*bool`
**Decision**: `DispatchConfig.ReapDone` keeps its `*bool` modeling and `GetDispatchReapDone` keeps the nil-means-default rung.
**Why**: The collapse to a plain `bool` is only safe once the **loader's** merged map always carries the leaf, which the decision above deliberately does not do. Under empty-skip an explicit `reap_done: false` survives the merge either way, so the pointer is not a correctness gap — it is the mechanism that keeps an absent key distinguishable from an explicit `false` at unmarshal. Intake assumption 12 anticipated this deferral.
**Rejected**: Collapsing to `bool` anyway (every project that never sets the key would silently stop reaping); injecting only the `dispatch` constants as a partial tier 0 in the loader (an arbitrary half-layer — the defaults tier would then mean different things in the loader and in `show`).
*Introduced by*: 260808-fp02-config-read-model-redesign

#### Keyed `--origin` prints `key = value  # <tier> <label>  (effective|shadowed)`
**Decision**: Keyed `--origin` output is uniform — one `key = value  # <tier> <label>  (marker)` line per defining tier, even when only one tier defines the key. Bare all-keys `--origin` keeps its current winner-only shape and label vocabulary.
**Why**: A stack listing needs the tier word (`env`/`system`/`project`/`default`) to be readable — two file paths are otherwise distinguishable only by inspection — and a shape that changes with the number of defining tiers would be unparseable and surprising. Bare `--origin` is a different question ("who set this?") and its `git config --show-origin` label vocabulary is established; churning it buys nothing.
**Rejected**: Keeping the current bare-suffix form for the single-tier case (output shape would depend on the data); adopting the tier-word vocabulary in bare `--origin` too (a gratuitous break of an established byte-level surface).
*Introduced by*: 260808-fp02-config-read-model-redesign

#### Empty-skip governs the environment layer too
**Decision**: An environment variable whose parsed value is empty (`null`, `""`, `[]`, `{}`) is treated as unset — it contributes no overlay leaf and no provenance entry.
**Why**: The read model has exactly one emptiness rule; an env layer that kept explicit-null shadowing would be the last surviving presence semantic, which is what this change exists to delete. It also generalizes the shipped "set-but-empty behaves as unset" rule instead of contradicting it.
**Rejected**: Keeping `FAB_X=null` as a real shadowing override (reintroduces the footgun one tier up).
*Introduced by*: 260808-fp02-config-read-model-redesign

## Tasks

### Phase 1: Core read model (internal/config)

- [x] T001 Add the emptiness predicate and the layered merge to `src/go/fab/internal/config/config.go`: exported `IsEmptyValue(v any) bool` (nil, `""`, empty list, empty map — never `false`/`0`) and exported `MergeLayers(layers ...map[string]any) map[string]any` folding lowest-precedence-first with empty-skip; replace `deepMerge` with the empty-skip pairwise merge it wraps <!-- R1 -->
- [x] T002 Rewire `LoadPath` and `LoadLayers` to `MergeLayers(projectMap, systemMap, envMap)` (system now above project) and update both doc comments plus `Layers.Effective`'s comment to the new order and empty-skip semantics <!-- R2 -->
- [x] T003 Treat an empty parsed environment value as unset in `loadEnvLayer` (skip before overlay/origin recording) and update its doc comment <!-- R1 -->
- [x] T004 Update the precedence prose in `src/go/fab/internal/config/config.go` Go docs: `DispatchConfig.ReapDone` (why the pointer stays), `GetAgentProfile`'s cross-scope legacy-alias LIMITATION (the inversion direction flips — a project-layer `profiles` now beats a system-layer `tiers`) <!-- R9 -->
- [x] T005 Update `src/go/fab/internal/config/config_test.go`: new-order cases (rename/retarget `TestCascade_ScalarReplaceProjectWins`, `TestCascade_DispatchWatchableProjectWins`), empty-skip table tests (null/empty-string/empty-list/empty-map fall through; `false`/`0` survive), and the env explicit-null case (`TestEnvCascade_ExplicitNullRemainsPresent` → falls through) <!-- R1 R2 R10 -->

### Phase 2: Defaults tier + provenance rewrite

- [x] T006 <!-- rework: DefaultsMapFor resolves roles via agent.ResolveRole(cfg, name), which layers the user's own agent.profiles override back into the DEFAULTS tier — derive from knob + provider fills only (resolve against a cfg copy with Agent.Profiles/Agent.Tiers cleared) --> Add `DefaultsMap()` and `DefaultsMapFor(cfg *config.Config)` in `src/go/fab/internal/configref/defaultsmap.go` — nest each non-nil row `Default` under its dotted key, JSON-normalize to the generic shape, merge through `config.MergeLayers`; `DefaultsMapFor` recomputes the `agent.profiles` rows via `agent.ResolveRole(cfg, role)` <!-- R3 R5 -->
- [x] T007 <!-- rework: add regression tests for the override-echo bug: a project agent.profiles.review.model override must NOT appear as the default tier's value; knob+fills only --> Add `src/go/fab/internal/configref/defaultsmap_test.go` coverage for `DefaultsMap`/`DefaultsMapFor`: every default-bearing row projected, the empty-default convention respected, and the knob-aware `agent.profiles.<role>.provider` projection (including provider pass-through) <!-- R3 R5 -->
- [x] T008 Rewrite the provenance engine in `src/go/fab/cmd/fab/config.go`: delete `originValue`/`highestOriginValue`/`defaultSubtree`/`normalizeToGeneric`, replace with an ordered tier list (env, system, project, default) whose "defines" test is `!config.IsEmptyValue`, and re-express `flattenOrigin`/`originAtPath`/`mergeableOriginMaps` over it <!-- R3 R2 -->
- [x] T009 Rebuild `renderShowKey` on `config.MergeLayers` over the four tier maps, and implement the full-stack keyed `--origin` listing (one line per defining tier, winner first, `(effective)`/`(shadowed)` markers, per-leaf drill-down for map-valued keys) <!-- R4 -->
- [x] T010 <!-- rework: read model is loaded TWICE per invocation (LoadLayers + readModelDefaults→LoadPath), duplicating every fail-open fab: warning: and paying a defaults projection bare show never uses — build the defaults tier from the already-loaded layers and skip the projection when neither --origin nor a key is given (A-019) --> Load the live `*config.Config` in `configShowCmd` and pass it to `configref.DefaultsMapFor` so the drill-down is knob-aware <!-- R5 -->
- [x] T011 Update the `show` cobra `Long`/`Example`/flag help in `src/go/fab/cmd/fab/config.go` for the new order and the full-stack keyed `--origin` <!-- R9 R4 -->

### Phase 3: Mutation UX

- [x] T012 <!-- rework: effectiveTierFor re-loads the cascade (second load, duplicated warnings on set/unset where baseline printed none); also extract the duplicated dotted-path descent (keyedOriginLines vs effectiveTierFor) into one shared helper (A-023) --> Add the shared shadow-resolution helper in `src/go/fab/cmd/fab/config.go` (given a key and a write target, return the winning tier and its label, fail-open to "no warning" when the repo/layers cannot be resolved) <!-- R6 R8 -->
- [x] T013 <!-- rework: refusal tests the RAW argv (strings.TrimSpace(rawValue)), so quoted-empty forms ('""', "''") are accepted and write an empty leaf empty-skip can never honor — test the PARSED scalar (config.IsEmptyValue on parseSetScalar's result) (R7/A-007) --> Refuse an empty/whitespace-only value in `configSetCmd` before any write, pointing at `fab config unset` <!-- R7 -->
- [x] T014 Emit the `fab: warning:` shadow notice on stderr after a successful `set`/`set --system`, naming the winning higher tier; exit stays 0 <!-- R6 -->
- [x] T015 Extend the `unset` no-op path to append the live-tier notice (system/project path with the suggested command, or the environment variable with the "cannot be unset" note); default-only stays as today <!-- R8 -->
- [x] T016 Update the `set`/`unset` cobra `Long` text for the empty-value refusal, the shadow warning, and the live-tier notice <!-- R9 -->
- [x] T017 <!-- rework: extend tests for the reworked behaviors: quoted-empty set refusal ('""'/"''"), no duplicated fab: warning: on show/--origin/set/unset (A-019 single-load), default-tier shows built-in (not the user's override) under an agent.profiles override --> Extend `src/go/fab/cmd/fab/config_show_init_test.go` for the new order and UX: full-stack keyed `--origin` golden output, empty-collection/null fall-through (replacing `TestConfigShowKey_EnvironmentNullWins` / `TestConfigShowKey_EmptyCollectionKeepsWinningOrigin` semantics), knob-aware drill-down, `set` shadow warning (project and `--system` targets, plus the no-warning case), empty-value refusal, and the `unset` live-tier notice <!-- R2 R4 R5 R6 R7 R8 -->
- [x] T018 Update `src/go/fab/cmd/fab/resolve_agent_test.go`'s `TestResolveCrossScopeLegacyAliasPrecedence` and any cascade-order assertions to the new order <!-- R2 -->

### Phase 4: Sweep — string literals, kit skills, specs

- [x] T019 [P] Sweep the user-facing string literals: `SystemScaffoldHeader` in `src/go/fab/internal/configupgrade/mutation.go` ("Resolves BELOW any project's…"), the `scope both` clauses in `src/go/fab/internal/configref/configref.go` row `Description`s and rendered segments, and the cascade-order comments in `src/go/fab/cmd/fab/dispatch_reap.go` and `src/go/fab/internal/agent/defaults.yaml` <!-- R9 -->
- [x] T020 [P] Update `src/kit/skills/_cli-fab.md`: the § heading and body of "The config cascade", the `fab config show` mode table (keyed `--origin` full stack), the `set`/`unset` paragraphs, and the `dispatch.reap_done` cascade note (~line 702) <!-- R9 -->
- [x] T021 <!-- rework: SPEC-_cli-fab.md:33 (fab dispatch row) still says "four-layer environment/project/system/default cascade" — slash-spelled order evaded the R9 grep; sweep the WHOLE mirror file for order claims, not just the fab config row --> Mirror T020 into `docs/specs/skills/SPEC-_cli-fab.md` (the `fab config` row) <!-- R9 -->
- [x] T022 [P] Rewrite the affected sections of `docs/specs/config.md`: § Override cascade (order, empty-skip, defaults-as-read-model-tier), § Six intent-grouped verbs (keyed `--origin`, `set` warning + refusal, `unset` notice), § Default semantics (the `*bool` rationale under the new read model), § Scope taxonomy precedence sentence <!-- R9 -->
- [x] T023 [P] Sweep the remaining doc claims: `docs/specs/index.md`, `docs/specs/skills/SPEC-_preamble.md`, `docs/specs/stage-models.md`, `docs/specs/architecture.md` (the mirrored fence sample), and the cross-scope caveat in `src/kit/migrations/2.16.19-to-2.17.0.md` <!-- R9 -->
- [x] T025 Collapse `internal/configupgrade`'s duplicated `defaultSubtreeFor`/`mergeGeneric` (explicitly a mirror of the now-deleted `cmd/fab` helpers) onto `configref.DefaultsMap()` — the B-hygiene comparison now reads the same materialized defaults tier the read model does <!-- R3 -->
- [x] T024 Run the affected Go packages' tests (`internal/config`, `internal/configref`, `internal/configupgrade`, `cmd/fab`) and then the full `src/go/fab` suite; fix fallout <!-- R1 R2 R10 -->

## Execution Order

- T001 blocks T002/T003; T002 blocks T008/T009.
- T006 blocks T007/T008/T010.
- T012 blocks T014/T015.
- T019–T023 are independent of each other but depend on the behavior being settled (Phases 1–3).
- T024 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `internal/config` exposes one emptiness predicate and one layered merge, and every read-model surface resolves leaves through them
- [x] A-002 R2: the loader merges environment over system over project, with built-in values beneath
- [x] A-003 R3: a materialized defaults map is projected from the registry and consumed as tier 0 by show/`--origin`/set/unset; `originValue` and its presence bit no longer exist
- [x] A-004 R4: keyed `fab config show <key> --origin` prints one line per defining tier with the winner marked; bare `--origin` is unchanged in shape *(verified byte-identical against the pre-change binary on a clean repo)*
- [x] A-005 R5: `agent.profiles.<role>` drill-down rows resolve against the live config *(knob-aware, and derived from the DEPTH KNOB + provider fills only — `liveRoleProfiles` strips `Agent.Profiles`/`Agent.Tiers` first, so a leaf the user set is never echoed back as its own built-in)*
- [x] A-006 R6: a successful `set` shadowed by a higher tier prints a `fab: warning:` naming that tier and exits 0
- [x] A-007 R7: `fab config set <key> ""` is refused with `unset` guidance and writes nothing *(the guard tests the PARSED value via `config.IsEmptyValue`, so the quoted-empty spellings and an explicit `null` are refused too)*
- [x] A-008 R8: an `unset` no-op names the tier where the key is live and the command that would remove it
- [x] A-009 R9: the cascade order and empty-skip semantics are stated consistently across specs, kit skills, Go docs, and user-facing strings *(the whole SPEC mirror re-swept for slash-spelled order claims, not just the `fab config` row)*

### Behavioral Correctness

- [x] A-010 R1: an explicit `null`/`""`/`[]`/`{}` at any tier falls through instead of shadowing, while `false` and `0` survive the merge
- [x] A-011 R2: with the same preference key set in both files, the system value is effective (the shipped project-wins behavior is gone)
- [x] A-012 R2: a project-scoped key remains impossible to set from the system layer, so no semantics-class key is affected by the inversion
- [x] A-013 R6: `set --system` shadowed only by the project file prints no warning

### Removal Verification

- [x] A-014 R3: no code path distinguishes "present but null" from "absent" — `originValue`, `highestOriginValue`'s presence bit, and `defaultSubtree` are gone, with no dead helpers left behind *(verified: zero remaining references to `originValue`, `highestOriginValue`, `defaultSubtree`, `mapGet`, `subtreeAt`, `mergeGeneric`; `go vet ./...` clean)*

### Scenario Coverage

- [x] A-015 R1: table tests cover the empty-skip matrix (null/empty string/empty list/empty map fall through; `false`/`0`/non-empty win) across tiers
- [x] A-016 R4: a test pins the three-tier full-stack listing (env effective, system shadowed, default shadowed) for one key
- [x] A-017 R6: tests cover the shadow warning for both write targets, including the negative case
- [x] A-018 R8: a test pins the live-tier notice for a system-live key and for an environment-live key

### Edge Cases & Error Handling

- [x] A-019 R10: absent/malformed system file, malformed project file, and project-scoped system/environment keys all behave exactly as before (warn-and-skip, error, warn-and-ignore) *(each invocation now loads the read model ONCE — `readModelDefaults` takes the already-loaded layers through `config.FromMap` — so a fail-open warning prints exactly once, and bare `show` skips the defaults projection entirely; the mutation notices resolve the cascade once by design, being a shadow check over it)*
- [x] A-020 R6: the shadow check is fail-open — an unresolvable repo or unreadable layer suppresses the warning rather than failing the write
- [x] A-021 R5: a depth knob naming an unknown provider is reported verbatim by the drill-down (provider neutrality) and does not fail `show --origin`

### Code Quality

- [x] A-022 Pattern consistency: new code follows the surrounding naming, nil-safety, and fail-open warning conventions of `internal/config` and `cmd/fab`
- [x] A-023 No unnecessary duplication: the emptiness/merge rule and the tier-resolution helper each exist once and are reused by the loader, the origin projection, and the mutation notices *(the dotted-path descent is now the one shared `descendPath`, consumed by `keyedOriginLines` and `effectiveTierFor`)*
- [x] A-024 Canonical source only: skill edits land in `src/kit/skills/`, never in `.claude/skills/`
- [x] A-025 SPEC-mirror sync: every `src/kit/skills/*.md` edit carries its `docs/specs/skills/SPEC-*.md` update in this change
- [x] A-026 CLI ⇒ docs + tests: the `fab config` behavior changes are reflected in `src/kit/skills/_cli-fab.md` and covered by Go tests in the same change
- [x] A-027 Sibling & mirror sweep: the cascade-order phrase class was swept up front, including user-facing Go string literals *(re-swept for the slash-spelled variant that evaded the `>` grep; `dispatch_reap.go`'s "four-layer" wording aligned to "four-tier" too)*
- [x] A-028 No migration needed: nothing on disk changes shape, so no `src/kit/migrations/` file is added

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

Verified after rework 2 — every Go symbol the change orphaned is gone (`originValue`, `highestOriginValue`, `defaultSubtree`, `mapGet`, `subtreeAt`, `mergeGeneric`, `mergeableOriginMaps`, `deepMerge`, `configupgrade.defaultSubtreeFor`/`normalizeToGeneric`: zero remaining references; `go vet ./...` clean). What is left is prose:

- `src/go/fab/cmd/fab/config_show_init_test.go:716` — the `TestConfigShowOrigin_HigherLayerScalarReplacesSubtree` doc comment still says "must honor **deepMerge** precedence"; `deepMerge` no longer exists (it is `config.MergeLayers`/`mergeOver`). Last live mention of a deleted symbol in the Go tree.
- `docs/memory/_shared/configuration.md` — the knob-blind `show --origin` under-reporting paragraph and the explicit-null/presence-tracking sentence in § the six verbs are now false; delete rather than reword. The same file's § Override Cascade also names `deepMerge` and the old layer order (lines 3, 38, 47, 49, 67, 468, 470) — hydrate's stage, on the intake's Affected Memory list.
- `docs/memory/runtime/providers-and-profiles.md` § Design Decisions, "`DefaultProfile` Is Resolution Against a Nil Config" — the `show --origin` half of that entry is obsolete (the drill-down composes against the live config now); the `resolve-agent` half stands. Lines 358/360 also carry the old inversion direction and the `deepMerge` name (hydrate's stage).
- `docs/memory/distribution/kit-architecture.md:283,287` and `docs/memory/runtime/dispatch.md:476,574` — the same stale "four-layer environment > project > system" claim, in two files the intake's Affected Memory list did NOT name. Hydrate must sweep the whole phrase class, not just the two listed files (`fab/project/code-quality.md` § Sibling & Mirror Sweeps). `docs/memory/_shared/index.md:10` follows automatically from the description via `fab memory-index`.
- Not candidates, deliberately retained: `DispatchConfig.ReapDone *bool` + `GetDispatchReapDone`'s nil rung, and the point-of-use default fallback seams — both kept by explicit Design Decisions above (the loader's merged tree still carries no defaults tier; the wholesale collapse is C5 item 1). Likewise `configref.roleProfileDefault`'s non-`omitempty` json tags: no shipped role default carries an empty leaf, so the projection loses nothing.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The materialized defaults tier lands at the read-model boundary (`configref` projection consumed by `cmd/fab`), not inside `internal/config.LoadPath` | The intake names `internal/configref` as the projection source, and configref cannot be imported by the loader (`configref → agent → config` cycle); merging the derived `agent.profiles` defaults into `Config` would also outrank provider fills and break `--provider` swaps | S:70 R:55 A:80 D:60 |
| 2 | Confident | `dispatch.reap_done` keeps its `*bool` (collapse deferred) | Follows directly from assumption 1 — the loader's merged map still does not carry the leaf; intake assumption 12 explicitly permits deferral | S:75 R:70 A:85 D:70 |
| 3 | Confident | Keyed `--origin` renders `key = value  # <tier> <label>  (effective\|shadowed)` uniformly; bare `--origin` keeps its current shape and labels | Intake assumption 8 makes the format plan latitude and fixes only the contract (one line per defining tier, winner marked, no new flag) | S:60 R:85 A:70 D:55 |
| 4 | Confident | Empty-skip applies to the environment layer too — `FAB_X=null` becomes unset, contributing no overlay leaf and no provenance entry | The read model has one emptiness rule; keeping env-null shadowing would preserve the exact presence semantic the change deletes, and it generalizes the shipped set-but-empty rule | S:70 R:75 A:80 D:65 |
| 5 | Confident | The `set` shadow warning fires whenever a higher tier defines the key, even if that tier holds the same value; the message states which tier wins rather than asserting the values differ | A same-valued higher tier still owns the effective value, so the write is inert either way; one always-accurate message beats a value-comparison special case | S:60 R:85 A:70 D:60 |
| 6 | Confident | "settable once machine-wide" prose in registry descriptions/segments is clarified in place (it now also outranks the project file) rather than reworded wholesale | The claim stays true after the flip; a minimal clarification keeps the byte-stable reference churn proportionate while the intake's sweep item is still satisfied | S:65 R:85 A:70 D:60 |
| 7 | Certain | No migration file ships | Intake assumption 7 — read-semantics change only, nothing on disk restructures | S:90 R:70 A:90 D:90 |
| 8 | Confident | The shipped `2.16.19-to-2.17.0` migration's cross-scope caveat is corrected in place (the inversion direction flips) rather than left stale | It is instructional text a user reads while applying the migration; a precedence claim that is now backwards would misdirect | S:65 R:80 A:75 D:65 |
| 9 | Confident | `internal/configupgrade`'s copy of the default-subtree helpers is collapsed onto `configref.DefaultsMap()` rather than left with a stale "mirrors cmd/fab" comment | Its documented twin is deleted by this change, so the copy is now unowned duplication; the two projections agree on every shipped row (no registry default carries an empty leaf), and B-hygiene is advisory-only | S:60 R:80 A:75 D:60 |

9 assumptions (1 certain, 8 confident, 0 tentative).
