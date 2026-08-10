# Plan: config-show-compose-defaults

**Change**: 260810-rvza-config-show-compose-defaults
**Intake**: `intake.md`

## Requirements

### Config CLI: `fab config show` Composed Output

#### R1: Bare `show` materializes the built-in defaults tier

Bare `fab config show` (no key, no `--origin`) MUST print the fully composed
config — environment > system > project > built-in defaults — as YAML, merging
the registry defaults projection beneath the file and environment layers. A
field no file/env tier defines MUST surface at its built-in value rather than
being omitted.

- **GIVEN** a repo whose project and system configs do not set `dispatch.mode`
- **WHEN** the user runs `fab config show`
- **THEN** the output contains `dispatch.mode: native` (the built-in default)
- **AND** any live override of a field still wins over its built-in default

#### R2: The defaults tier reuses the shared knob-aware projection

The composed tier MUST be produced by the same `readModelDefaults` →
`configref.DefaultsMap`/`DefaultsMapFor` projection the keyed and `--origin`
paths already use, so the derived `agent.profiles` rows compose against the
live depth knobs (`agent.session`/`agent.workers`). A second or divergent
projection MUST NOT be introduced.

- **GIVEN** a project config setting `agent.workers: codex`
- **WHEN** the user runs `fab config show`
- **THEN** the composed `agent.profiles.doing.provider` row reports `codex`
  (the provider the depth knob selects), not the nil-config `claude`

#### R3: `--origin` semantics are unchanged

`--origin` MUST remain purely the provenance annotator: winner-only listing
without a key, full-stack listing with a key, per-key drill-down for
map-valued fields. No output, exit-code, or flag change ships for `--origin`.

- **GIVEN** any repo state the existing `--origin` tests cover
- **WHEN** the user runs `fab config show --origin` or `fab config show <key> --origin`
- **THEN** the output is byte-identical to the pre-change behavior (all
  existing `--origin` tests stay green unmodified)

#### R4: No sparse-view companion flag

No `--set-only`/`--sparse` flag ships in this change. The sparse
"only what I set" view stays reachable by reading the config files or
filtering `--origin` output for non-`default` tiers.

- **GIVEN** the changed `fab config show` command surface
- **WHEN** its flags are enumerated
- **THEN** only `--origin` exists — no new flag is added

#### R5: Help text and docs state the composed behavior

Every user-facing statement of the old behavior MUST be rewritten: the
`configShowCmd` `Long`/`Example` text in `src/go/fab/cmd/fab/config.go`
(the "built-in defaults are NOT materialized here" carve-out is deleted),
`src/kit/skills/_cli-fab.md` (mode-table row and the "Bare output remains
unchanged" line), `docs/specs/config.md` (show-verb bullet),
`docs/specs/skills/SPEC-_cli-fab.md` (the same claim), and
`docs/memory/_shared/configuration.md` (the show-verb description).

- **GIVEN** the implementation change has landed
- **WHEN** the repo is grepped for the old claim ("not materialized",
  "never consults the defaults tier", "skips the defaults projection",
  "Bare output remains unchanged")
- **THEN** no live surface carries the stale claim (historical artifacts
  under `fab/changes/` and `fab/plans/` excepted — history is not rewritten)

#### R6: Tests cover the composed bare-show output

`src/go/fab/cmd/fab/config_show_init_test.go` MUST gain a case asserting a
pure-default field (e.g. `dispatch.mode: native`) appears in bare `show`,
that a live override still wins, and that the derived profiles are
knob-aware; stale comments pinning the old skip behavior MUST be refreshed.

- **GIVEN** a bare project repo in the test harness
- **WHEN** `go test ./cmd/fab` runs
- **THEN** the composed bare-show behavior is asserted by a dedicated test
  and all pre-existing show/origin tests pass

### Non-Goals

- No `--set-only`/`--sparse` flag — YAGNI until asked for; additive later.
- No loader change — `internal/config` keeps its point-of-use defaults
  seams; the defaults tier is composed in the read model only.
- No change to the keyed `fab config show <key>` form — it already composes
  defaults and stays as-is.
- No compatibility carve-out for output consumers — bare `show` is a
  human-facing pure query; the change is additive (new keys appear, existing
  keys keep shape).

### Design Decisions

#### Reuse the `DefaultsMapFor` projection for bare `show`

**Decision**: Bare `show` merges the output of the existing
`readModelDefaults` helper (→ `configref.DefaultsMap`/`DefaultsMapFor`)
beneath `layers.Effective` via `config.MergeLayers`, inside `renderShow`.
**Why**: The keyed and `--origin` paths already compose defaults through
this exact projection, including knob-aware derived `agent.profiles` rows —
one projection cannot drift from itself.
**Rejected**: A second bare-show-specific projection — it would duplicate
the knob-awareness logic and silently diverge from the keyed/`--origin`
views it must agree with.
*Introduced by*: 260810-rvza-config-show-compose-defaults

#### Empty-state notice evaluated against the composed map

**Decision**: `renderShow`'s "# (no effective config …)" guard tests the
composed map (defaults merged in), so a repo with no file/env config prints
the defaults tier instead of the notice.
**Why**: The composed output IS the answer on an empty repo — printing the
notice would hide exactly the built-ins this change exists to surface. The
guard remains only as a defensive no-output case.
**Rejected**: Keeping the guard on `layers.Effective` — it would make bare
`show` on a fresh repo print "no effective config" while defaults resolve
at point-of-use, reproducing the original under-reporting in miniature.
*Introduced by*: 260810-rvza-config-show-compose-defaults

### Deprecated Requirements

#### Bare `show` prints the env/system/project merge only, defaults not materialized

**Reason**: It silently under-reported the effective config and contradicted
the command's own `Long` help text; the keyed and `--origin` forms already
composed defaults, so bare `show` was inconsistent with its own command.
**Migration**: Bare `show` composes the defaults tier (R1/R2); the sparse
view remains reachable via the config files or `--origin` filtering.

## Tasks

### Phase 2: Core Implementation

- [x] T001 In `src/go/fab/cmd/fab/config.go`, compose the defaults tier into bare `show`: always run `readModelDefaults` in `configShowCmd`'s `RunE` (drop the `withOrigin || len(args) == 1` condition), merge it beneath `layers.Effective` in `renderShow` via `config.MergeLayers(defaults, layers.Effective)` with the empty-state guard on the composed map, and refresh the stale RunE comment ("Bare `show` … never consults the defaults tier") <!-- R1, R2 -->
- [x] T002 Rewrite the `configShowCmd` `Long` text in `src/go/fab/cmd/fab/config.go` — delete the "built-in defaults are NOT materialized here" carve-out and state that bare `show` prints the fully composed config with defaults merged beneath env/system/project <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T003 In `src/go/fab/cmd/fab/config_show_init_test.go`, add a bare-show composition test (pure-default `dispatch.mode: native` appears, live `agent.workers: codex` override wins, `agent.profiles.doing.provider` is knob-aware `codex`) and refresh the stale "Bare `show` skips the defaults projection entirely" comment on `TestConfigLoaderWarningsAreNotDuplicated` <!-- R1, R2, R6 -->

### Phase 4: Polish

- [x] T004 [P] Update `src/kit/skills/_cli-fab.md` § fab config show — rewrite the bare-show row of the mode table (fully composed, defaults materialized) and delete the "Bare output remains unchanged" sentence <!-- R5 -->
- [x] T005 [P] Update `docs/specs/config.md` show-verb bullet — replace the "built-in defaults are NOT materialized here … surfaced explicitly only by `--origin`" claim with the composed-output description; `--origin` described as provenance-only <!-- R5 -->
- [x] T006 [P] Update `docs/specs/skills/SPEC-_cli-fab.md` line 31 — replace the "built-in defaults are NOT materialized here (they apply at point-of-use)" claim in the `fab config` row <!-- R5 -->
- [x] T007 [P] Update `docs/memory/_shared/configuration.md` show-verb bullet (present-truth rewrite, no transition narration) — bare `show` composes the defaults tier via `readModelDefaults`; `--origin` is provenance-only <!-- R5 -->

## Execution Order

- T001 blocks T003 (the test asserts T001's behavior)
- T002–T007 are independent of each other and of T001/T003 (docs/prose only)

## Acceptance

### Functional Completeness

- [x] A-001 R1: On a bare project, `fab config show` output includes a pure-default field at its built-in value (`dispatch.mode: native`) and every live override still wins over its default
- [x] A-002 R2: With `agent.workers: codex` set, bare `show` reports `agent.profiles.doing.provider: codex` — the knob-aware shared projection, not a nil-config copy
- [x] A-003 R3: `fab config show --origin` and keyed `--origin` outputs are byte-identical to pre-change behavior (all existing `--origin` tests pass unmodified)
- [x] A-004 R4: `fab config show` has no new flag — only `--origin` exists
- [x] A-005 R5: `Long`/`Example` text and all five docs surfaces state the composed behavior; the "NOT materialized" carve-out is gone everywhere

### Behavioral Correctness

- [x] A-006 R1: Previously omitted default-valued keys now appear in bare `show`; previously present keys keep their values and YAML shape (additive change)

### Removal Verification

- [x] A-007 R5: Grep for "not materialized", "never consults the defaults tier", "skips the defaults projection", "Bare output remains unchanged" across `src/kit/skills/`, `docs/specs/`, `docs/memory/`, and Go help strings finds no stale occurrence (historical `fab/changes/` and `fab/plans/` artifacts excepted)

### Scenario Coverage

- [x] A-008 R6: A dedicated test asserts composed bare-show output (pure-default field materialized, override wins, knob-aware profiles); `TestConfigShow_EnvironmentEffectiveAndOrigins` still proves env values win in composed output

### Edge Cases & Error Handling

- [x] A-009 R1: A repo with no project/system config prints the defaults tier rather than the "# (no effective config …)" notice; the guard survives only as the composed-map-is-empty defensive case
- [x] A-010 R2: The knob-blind fallback (`configref.DefaultsMap()` when the merged tree will not unmarshal) still degrades gracefully without failing the query
- [x] A-011 R1: Exactly one read-model load per invocation — `TestConfigLoaderWarningsAreNotDuplicated` stays green with bare `show` in its case list

### Code Quality

- [x] A-012 Pattern consistency: New code follows the naming/structure of the surrounding read-model code in `config.go` (tier-composition via `config.MergeLayers`, intent-explaining comments)
- [x] A-013 No unnecessary duplication: `readModelDefaults`/`DefaultsMapFor`/`MergeLayers` reused — no second defaults projection introduced
- [x] A-014 Mirror sweeps: `_cli-fab.md` change ships with `docs/specs/skills/SPEC-_cli-fab.md` and `docs/specs/config.md` updates in the same change; the memory file documenting the behavior is updated

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. The removed `withOrigin || len(args) == 1` gate leaves no orphaned symbols: `readModelDefaults`, `renderShow`, and `renderShowKey` all retain call sites, and the empty-state guard in `renderShow` is deliberately retained as the defensive no-output case (Design Decision: "Empty-state notice evaluated against the composed map").

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Bare `fab config show` prints the fully composed config, defaults tier included | User accepted this explicitly in the originating discussion (intake) | S:90 R:85 A:90 D:85 |
| 2 | Confident | The empty-state notice is evaluated against the composed map, so a config-less repo prints the defaults tier | Composed output is the answer this change exists to give; keeping the notice would re-hide defaults | S:70 R:80 A:75 D:70 |
| 3 | Confident | Memory + SPEC mirrors are updated during apply as part of the sweep class, ahead of hydrate | The dispatch's sweep discipline and code-quality.md § Sibling & Mirror Sweeps name memory/SPEC files as in-class for a behavior change | S:75 R:80 A:75 D:65 |

3 assumptions (1 certain, 2 confident, 0 tentative).
