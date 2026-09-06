# Plan: Pane Dispatch Geometry Floor

**Change**: 260906-cxe0-pane-dispatch-geometry-floor
**Intake**: `intake.md`

## Requirements

### Dispatch: Geometry floor on the split shape

#### R1: Planned-size computation and floor verdict are a pure decision
`internal/dispatch` SHALL gain a pure function (no tmux probe, no I/O — the `SelectMode`/`SelectPaneShape`/`splitPlacement` pattern) that, given a resolved `pane.SplitPlacement`, the measured geometry, the configured column width, and the floor (`min_cols`/`min_rows`), computes the worker pane the split would yield and returns whether it is below the floor:

- Carving split (`Direction == SplitRight`): worker width = `window_width * columnWidth / 100`, worker height = `window_height`.
- Stacking split (`Direction == SplitBelow`): worker width = the sibling pane's width, worker height = the sibling pane's height / 2 (tmux even-splits an unsized `-v`).
- Below floor ⇔ computed width < `min_cols` OR computed height < `min_rows`. A dimension exactly at the floor passes.

- **GIVEN** a 127x16 window, `column_width` 35, floor 80x20
- **WHEN** the carve decision is computed
- **THEN** the planned worker is 44x16, both below floor, and the verdict is "demote to window"

- **GIVEN** a 240x60 window, `column_width` 35, floor 80x20
- **WHEN** the carve decision is computed
- **THEN** the planned worker is 84x60 and the split proceeds unchanged

#### R2: Geometry probe is fail-open
The geometry read (`#{window_width}`/`#{window_height}` of the split target's window, plus `#{pane_width}`/`#{pane_height}` of the target pane for the stacking case) SHALL ride a single `tmux display-message -p -t <target>` query through `internal/pane`'s shared `RunCmd`/`WithServer` helpers. A failed or unparseable probe SHALL NOT fail or demote the dispatch: the launch proceeds with today's split behavior byte-for-byte (warn at most, per the package's placement-is-cosmetic posture).

- **GIVEN** a geometry probe that errors (tmux hiccup, unparseable output)
- **WHEN** a pane worker is dispatched from a split-shaped launch
- **THEN** the sized split runs exactly as before this change, and the dispatch launches

#### R3: Floor demotion opens a manually-sized window and warns
When the floor verdict is "below floor", the launch SHALL NOT split. It SHALL open the worker in its own tmux window with a manual size (R5) and emit a stderr warning naming the measured window, the requested column, the computed dimension vs. the floor, and the action, in the shape:

```
warning: window 127x16 too narrow for a 35% column (44 cols < 80): opening worker in its own window
```

(the height-limited case names rows analogously, e.g. `window 127x16 too short for a stacked split (8 rows < 20): opening worker in its own window`).

- **GIVEN** a dispatcher in a 127x16 viewer-shrunk window, floor 80x20
- **WHEN** `fab dispatch open` launches a pane worker
- **THEN** no split occurs, the worker opens in a new manually-sized window, the warning above appears on stderr, and the record carries the new pane's ID exactly as the window shape always has

#### R4: Fine geometry keeps today's behavior byte-for-byte
When the planned worker meets the floor (or the probe fails, R2), the split path — including `SplitTarget`'s sibling stacking, the sized carve, and `OpenSplitPane`'s `tmux rejected the sized split` unsized retry — SHALL be byte-for-byte unchanged. The pre-existing `ShapeWindow` reasons (`--server`, no `$TMUX_PANE`) SHALL keep today's unsized `OpenWindow` behavior unchanged.

- **GIVEN** any geometry at or above the floor
- **WHEN** a split-shaped dispatch launches
- **THEN** the tmux argv sequence is identical to before this change

### Pane: Manually-sized window creator

#### R5: Sized-window creation escapes viewer sizing
`internal/pane` SHALL gain a sized-window creator that opens the worker window **detached** and pins its size manually — the escape from `window-size latest` viewer shrinking:

```
tmux new-window -d -P -F '#{pane_id} #{window_id}' -n <title> -c <dir> <cmd>
tmux set-option -w -t <window_id> window-size manual
tmux resize-window -t <window_id> -x 200 -y 50
```

The 200x50 size SHALL be a named Go constant, not config. Failures of the two sizing follow-ups after a successful `new-window` are NON-FATAL warnings (the `OpenSplitPane` title-set posture): the worker is already running and identified by pane ID. Existing `OpenWindow` callers keep their current signature and behavior.

- **GIVEN** the floor demoted a split
- **WHEN** the sized window opens
- **THEN** the rendered argv is the three-command sequence above with the constant size, the new window does not steal the user's focus (`-d`), and a failed `resize-window` warns without failing the dispatch

### Config: `dispatch.min_cols` / `dispatch.min_rows`

#### R6: Floor keys plumb end-to-end like the existing dispatch keys
The two keys SHALL follow the `dispatch.column_width` pattern at every seam:

- `src/go/fab/defaults.yaml` `dispatch:` block gains `min_cols: 80` / `min_rows: 20` — the single value source, init-pushed by `internal/agent` into new `config.DefaultDispatchMinCols`/`DefaultDispatchMinRows` symbols (no Go literal copies).
- `internal/config.DispatchConfig` gains `MinCols`/`MinRows int` yaml fields; nil-safe accessors `GetDispatchMinCols()`/`GetDispatchMinRows()` treat absent/zero/negative as unset → default (positive values pass; no upper-bound clamp — a floor larger than any window simply always demotes).
- `internal/configscope` dotted-key registry gains both keys (scope inherits top-level `dispatch` = both), which makes `FAB_DISPATCH_MIN_COLS`/`FAB_DISPATCH_MIN_ROWS` env overrides work mechanically.
- `internal/configref` registry gains both rows (Default sourced from the config symbols, KindInt, ScopeBoth, Advertise true, rendered inline in the `dispatch.mode` Segment like `column_width`/`reap_done` — the shared dispatch fence block and its header line gain the new keys).
- The `dispatch.column_width` Description and the dispatch Segment prose SHALL mention the floor (a carve that would land below `dispatch.min_cols`/`min_rows` opens a manually-sized window instead).

- **GIVEN** a project with no dispatch overrides
- **WHEN** `fab config explain dispatch.min_cols` runs
- **THEN** it reports default 80, scope both, and the floor semantics; `fab config explain dispatch.column_width` mentions the floor

### Docs: memory and skill prose

#### R7: Placement docs gain the floor
`docs/memory/runtime/dispatch.md`'s split-placement requirement SHALL gain the geometry floor (pre-check formulas, floor keys + defaults, the manually-sized detached window fallback + its 200x50 constant, the warning shape, fail-open probe posture). `docs/memory/_shared/configuration.md`'s `dispatch` section SHALL gain the two keys. `docs/memory/runtime/pane-commands.md`'s shared-pane-package prose SHALL note the sized-window creator if it enumerates `create.go`'s creators. Sibling prose restating placement/key enumerations SHALL be swept: `src/kit/skills/_preamble.md` (always-load config enumeration), `src/kit/skills/_cli-fab.md` (§ fab config full-schema enumeration; § fab dispatch placement paragraph). Memory indexes regenerate via `fab memory-index`.

- **GIVEN** the shipped change
- **WHEN** `grep -rn "min_cols" docs/memory src/kit/skills` runs
- **THEN** the dispatch memory, configuration memory, `_preamble.md`, and `_cli-fab.md` all describe the floor consistently, and no doc still claims the carve is unconditional

### Non-Goals

- The readiness probe (rk side — `narrow %N (WxH)` verdict on `rk mux await --ready`, box-drawing-tolerant sentinel) — a parallel run-kit session owns it; fab does not branch on `narrow` yet.
- Adding the floor keys to the `fab setup` wizard's advanced section — the wizard's curated set stays as-is; `fab config set` covers the keys.
- Re-checking or repairing geometry after launch — the floor is a creation-time decision, matching the column invariant's creation-time-only rule.
- No migration: both keys are additive with built-in defaults (absent = default, the existing dispatch-block pattern).

### Design Decisions

#### Floor check runs after SplitTarget, inside the launch path
**Decision**: The floor is evaluated in `launchPane` (cobra layer) after `SplitTarget` resolves the placement: probe geometry for the resolved target, feed the pure decision in `internal/dispatch`, and demote to the sized window on a below-floor verdict.
**Why**: The verdict depends on WHICH split was planned (carve vs. stack target and direction), which only exists after `SplitTarget`. Keeping the decision pure in `internal/dispatch` and the probe/demotion in the cobra layer preserves the package's pure-policy/mechanics split (`SelectMode` precedent: env and probe I/O live in the cobra layer).
**Rejected**: Folding the floor into `SelectPaneShape` — it runs before the sibling probe, so it cannot price the stacking case, and it would need geometry inputs threaded through `resolveMode` where no tmux probe belongs.
*Introduced by*: 260906-cxe0-pane-dispatch-geometry-floor

#### Manual window size is a constant, not config
**Decision**: 200x50, named Go constants in `internal/pane`.
**Why**: The floor pair is the policy knob; the escape window's size has no demonstrated tuning need, and the task spec proposes exactly this.
**Rejected**: `dispatch.window_cols`/`window_rows` config keys — speculative surface; can be added later without migration if a need appears.
*Introduced by*: 260906-cxe0-pane-dispatch-geometry-floor

#### Geometry probe failure is fail-open to the split
**Decision**: A failed probe proceeds with today's split byte-for-byte (warning at most), never demotes and never fails the dispatch.
**Why**: Placement is cosmetic and must never fail an otherwise-launchable dispatch (the package's standing posture); demoting on an unmeasured window would change layout on flaky evidence.
**Rejected**: Demote-on-unknown — safer against tiny windows but changes layout for every transient tmux hiccup, violating R4's byte-for-byte guarantee.
*Introduced by*: 260906-cxe0-pane-dispatch-geometry-floor

## Tasks

### Phase 1: Setup (config plumbing)

- [x] T001 Add `min_cols: 80` / `min_rows: 20` to `src/go/fab/defaults.yaml` `dispatch:` block; add `config.DefaultDispatchMinCols`/`DefaultDispatchMinRows` symbols and the `internal/agent` init() push (`src/go/fab/internal/agent/agent.go`); extend `TestDefaultsFileDispatchBlockIsPinned` and `TestConfigDispatchDefaultsMatchDefaultsFile` (`src/go/fab/internal/agent/defaults_test.go`) to pin both <!-- R6 -->
- [x] T002 Add `MinCols`/`MinRows int` yaml fields to `DispatchConfig` and nil-safe validated accessors `GetDispatchMinCols()`/`GetDispatchMinRows()` in `src/go/fab/internal/config/config.go` (absent/zero/negative → default); add accessor table tests and the env-var mapping rows (`FAB_DISPATCH_MIN_COLS`/`FAB_DISPATCH_MIN_ROWS`) in `src/go/fab/internal/config/config_test.go` <!-- R6 -->
- [x] T003 Register `dispatch.min_cols`/`dispatch.min_rows` in `src/go/fab/internal/configscope/configscope.go` dotted-key list (+ its test); add both registry rows in `src/go/fab/internal/configref/configref.go` (Default from config symbols, KindInt, ScopeBoth, Advertise true, no own Segment — rendered in the dispatch Segment); update `dispatchSegment()`/`dispatchShortSegment()` and the `dispatch.column_width` Description to mention the floor; extend the configref/configupgrade fence tests (`src/go/fab/internal/configupgrade/configupgrade_test.go`, `src/go/fab/internal/configref/configref_test.go`) <!-- R6 -->

### Phase 2: Core Implementation

- [x] T004 [P] Add the geometry probe to `src/go/fab/internal/pane` (e.g. `create.go` or a new `geometry.go`): one `display-message -p -t <target> -F '#{window_width} #{window_height} #{pane_width} #{pane_height}'` query via `RunCmd`/`WithServer`, returning a small geometry struct; extract the pure output-parsing half and unit-test it (malformed output, short fields) <!-- R2 -->
- [x] T005 [P] Add the pure floor decision to `src/go/fab/internal/dispatch/pane_mode.go`: planned worker size from (`SplitPlacement`, geometry, columnWidth) per R1's two formulas, below-floor verdict, and the warning-string composer (R3's exact wording, cols and rows variants); table tests covering carve below/at/above floor (80 exactly passes), stack height-limited, stack width-limited, and both-fine <!-- R1, R3 -->
- [x] T006 [P] Add the sized-window creator to `src/go/fab/internal/pane/create.go`: `new-window -d -P -F '#{pane_id} #{window_id}' -n <title> -c <dir> <cmd>` then `set-option -w -t <win> window-size manual` + `resize-window -t <win> -x 200 -y 50`; `ManualWindowCols`/`ManualWindowRows` constants (200/50); sizing follow-up failures return as warnings (title-set precedent); extract the argv composers and unit-test all three argv renderings without a tmux server <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T007 Wire the floor into `launchPane` (`src/go/fab/cmd/fab/dispatch_start.go`): thread `minCols`/`minRows` through `paneTarget` (filled in `runDispatchLaunch` beside `columnWidth`); in the `ShapeSplit` branch after `SplitTarget`, probe geometry (probe error ⇒ warn-free fail-open to today's split), evaluate the floor, and on demotion emit the R3 warning to stderr and open via the T006 creator (report string keeps the window form, e.g. `pane %s, window %s`); record fields (`Pane`/`Window`/`Server`/PanePID) identical across shapes <!-- R1, R2, R3, R4 -->
- [x] T008 Run the affected package tests: `go test ./internal/dispatch/... ./internal/pane/... ./internal/config/... ./internal/configref/... ./internal/configscope/... ./internal/configupgrade/... ./internal/agent/... ./cmd/fab/...` from `src/go/fab`; fix regressions (existing placement/argv tests MUST pass unchanged per R4) <!-- R4 -->

### Phase 4: Polish (docs)

- [x] T009 Update memory: `docs/memory/runtime/dispatch.md` split-placement requirement (floor pre-check, formulas, keys+defaults, sized-window fallback + constants, warning shape, fail-open posture); `docs/memory/_shared/configuration.md` `dispatch` key section (both keys, defaults 80/20, scope both, env vars); `docs/memory/runtime/pane-commands.md` shared-pane-package creator prose if it enumerates creators; regenerate indexes via `fab memory-index` <!-- R7 -->
- [x] T010 Sweep sibling skill prose: `src/kit/skills/_preamble.md` always-load `dispatch.*` enumeration; `src/kit/skills/_cli-fab.md` § fab config full-schema key enumeration and § fab dispatch placement paragraph (floor sentence); repo-wide `grep -n "column_width"` to catch any other enumeration restating the carve as unconditional <!-- R7 -->

## Execution Order

- T001 → T002 → T003 (each builds on the previous symbols); T004/T005/T006 are independent of each other and of Phase 1
- T007 needs T002 (accessors), T004, T005, T006
- T008 after T007; T009/T010 after T008

## Acceptance

### Functional Completeness

- [x] A-001 R1: A pure `internal/dispatch` function computes planned worker size (carve: `winW*cw/100` x `winH`; stack: sibling width x sibling height/2) and a below-floor verdict, with table tests and no I/O
- [x] A-002 R2: The geometry probe is a single shared-helper tmux query with a unit-tested parse half, and its failure leaves the dispatch on today's split path
- [x] A-003 R3: A below-floor verdict opens a manually-sized detached window and emits the reason-naming warning in the specified shape
- [x] A-004 R5: The sized-window creator renders the `new-window -d`/`window-size manual`/`resize-window -x 200 -y 50` sequence with named constants, and sizing failures are non-fatal warnings
- [x] A-005 R6: `dispatch.min_cols`/`min_rows` resolve through defaults.yaml → agent init push → config accessors → configscope keys → configref rows, with env overrides and fence rendering

### Behavioral Correctness

- [x] A-006 R4: With geometry at/above the floor, the split argv sequence (sibling stack, sized carve, unsized retry) is byte-for-byte unchanged — pre-existing dispatch/pane tests pass unmodified
- [x] A-007 R4: The pre-existing `ShapeWindow` reasons (`--server`, empty `$TMUX_PANE`) still use today's unsized `OpenWindow` with no manual sizing

### Scenario Coverage

- [x] A-008 R1: Boundary tests: computed 80 cols at floor 80 passes; 79 fails; 20 rows passes; 19 fails
- [x] A-009 R3: The 127x16 / 35% motivating case yields the demotion warning quoting `44 cols < 80` and a window launch

### Edge Cases & Error Handling

- [x] A-010 R2: Probe error and unparseable geometry both fall open to the split with no demotion and no dispatch failure
- [x] A-011 R5: `resize-window`/`set-option` failure after a successful `new-window` warns and still records the pane ID

### Code Quality

- [x] A-012 Pattern consistency: pure decisions in `internal/dispatch`, tmux mechanics in `internal/pane`, env/config reads in the cobra layer — matching the package headers' stated split
- [x] A-013 No unnecessary duplication: geometry query rides `RunCmd`/`WithServer`; the floor pair reuses the `column_width` accessor/registry/pin patterns rather than new mechanisms
- [x] A-014 No magic numbers: 80/20 live only in defaults.yaml; 200x50 are named constants
- [x] A-015 Go changes ship tests; CLI-adjacent docs (`_cli-fab.md`) and memory updated in the same change (constitution Additional Constraints)
- [x] A-016 Owner-or-pointer: skill-prose edits update owners or pointers, never restate an owned rule alongside its pointer

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality (geometry floor, sized-window fallback, two config keys) without making existing code redundant. All pre-existing creators (`OpenWindow`/`OpenSplitPane`/`OpenPlainPane`) and the sized-split retry path retain their callers and behavior.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Floor evaluated in `launchPane` after `SplitTarget` (pure verdict in `internal/dispatch`, probe in `internal/pane`, demotion at cobra layer) | The verdict needs the resolved placement; matches the package's stated pure-policy/mechanics split | S:70 R:80 A:90 D:80 |
| 2 | Confident | One combined `display-message` query returns window + target-pane geometry for both carve and stack cases | Single round-trip, single parse; the carve target (dispatcher) and stack target (sibling) each carry the needed fields | S:60 R:85 A:85 D:75 |
| 3 | Confident | No upper-bound clamp on min_cols/min_rows (only ≤0 → default) | A floor larger than any window degrades gracefully (always window shape); clamping would invent a policy the spec doesn't state | S:55 R:85 A:80 D:70 |
| 4 | Confident | Demotion report string keeps the existing `pane %s, window %s` form (the warning carries the why) | Downstream is shape-blind and pane-ID keyed; a third report form would ripple into docs/tests for no consumer | S:60 R:80 A:85 D:75 |
| 5 | Certain | Probe failure fail-open produces NO demotion warning (silent or minor warn), preserving R4 | Spec requirement 4 byte-for-byte + placement-is-cosmetic posture | S:80 R:85 A:90 D:85 |

5 assumptions (1 certain, 4 confident, 0 tentative).
