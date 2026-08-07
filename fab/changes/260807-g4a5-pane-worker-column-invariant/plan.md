# Plan: Pane-Mode Worker Column Invariant — Record-Based Sibling Detection + Sized First Split

**Change**: 260807-g4a5-pane-worker-column-invariant
**Intake**: `intake.md`

## Requirements

### Dispatch: pane-worker placement keys on dispatch records, not pane titles

#### R1: Sibling detection SHALL enumerate dispatch records, never pane titles
`fab dispatch start`'s split-placement probe SHALL identify existing worker panes by intersecting
the **pane IDs recorded in the checkout's dispatch records** (`.fab-dispatch/*/{stage}.yaml`
`pane:` fields) with the dispatcher window's live `tmux list-panes` output. The title-keyed scan
(`DispatchTitlePrefix` matching over `#{pane_title}`) SHALL be removed: a harness running inside a
worker pane rewrites the pane title within seconds, so a title probe finds nothing and every
worker re-splits the dispatcher. Enumeration scope is **every** `.fab-dispatch/*/` record dir in
the checkout (not only the active change's), bounded by the intersection.

Records whose `pane:` is empty (headless dispatches) SHALL be ignored, and `{stage}-result.yaml`
SHALL NOT be read as a record. The intersection with the window's live pane list IS the liveness
**and** same-window filter — a dead pane, or a live pane in another window, does not appear in
`list-panes -t <dispatcherPane>` output.

- **GIVEN** a live worker pane recorded in `.fab-dispatch/abcd/apply.yaml` whose pane **title has
  been clobbered** by the harness running in it
- **WHEN** a second pane dispatch starts from the same dispatcher pane
- **THEN** the probe still finds that worker pane and the new worker splits it
- **AND** GIVEN no live recorded worker pane in the dispatcher's window, the probe reports none

#### R2: The column invariant is a creation-time rule
The first worker SHALL split the dispatcher's own pane `-h` (carving the Left/Right column);
every later worker SHALL split the **last** live recorded worker pane `-v` (stacking inside the
right column). "Last" SHALL be the bottom-most such pane in `list-panes` index order. fab SHALL
NOT run `select-layout`, SHALL NOT rearrange user-made panes, and SHALL NOT re-touch the
vertical Left/Right separator once carved. There is no layout-repair pass: an already-mangled
window stays as-is until those panes die.

- **GIVEN** a dispatcher pane and two live recorded worker panes in its window
- **WHEN** a third pane dispatch starts
- **THEN** it splits the second worker pane `-v` (not the dispatcher, not the first worker)
- **AND** no `select-layout` or resize command is issued against the dispatcher's window

#### R3: The column-carving split SHALL be sized; stacking splits SHALL NOT be
Every column-carving `-h` split SHALL pass `-l {n}%` (tmux ≥ 3.1 syntax) so the dispatcher keeps
`100 − n`% of the window width. `-v` stacking splits SHALL stay unsized (tmux even-splits within
the column). The size SHALL apply on the **degraded** path too — a failed record probe still
carves a *sized* column off the dispatcher.

- **GIVEN** `dispatch.column_width` resolving to 35
- **WHEN** the first pane worker of a window is dispatched
- **THEN** the split argv carries `-h -l 35%` and the worker pane is narrower than the dispatcher's
- **AND** GIVEN a second worker stacking under it, that split argv carries `-v` and no `-l`

#### R4: Placement stays cosmetic — every failure degrades, warn-only
A failing record read, a failing `list-panes` probe, or a tmux that rejects `-l N%` (pre-3.1)
SHALL degrade rather than fail a dispatch that would otherwise launch: the probe failure falls
back to the sized `-h` split off the dispatcher with a stderr warning, and a rejected size
retries the same split **unsized** with a stderr warning. A failed pane-title set stays
non-fatal, as today.

- **GIVEN** a tmux too old to accept `-l N%`
- **WHEN** a pane worker is dispatched
- **THEN** the sized split is retried unsized, the dispatch succeeds, and stderr carries a
  one-line warning
- **AND** GIVEN an unreadable `.fab-dispatch/` tree, the dispatch still launches with a warning
  and the dispatcher-`-h` fallback

#### R5: `fab dispatch restart` uses the identical placement rule
`restart` SHALL reach the same record-based placement and the same sizing through the shared
launch path — no restart-specific placement branch, no new flag, no record schema change.

- **GIVEN** an orphaned pane dispatch and a live recorded sibling worker in the dispatcher's window
- **WHEN** `fab dispatch restart <change> <stage>` runs from that dispatcher pane
- **THEN** the relaunched worker stacks in the right column exactly as a fresh `start` would

### Config: `dispatch.column_width`

#### R6: A new `both`-scoped integer field with a built-in default of 35
`dispatch.column_width` SHALL be modeled on `internal/config`'s `DispatchConfig`, read through a
nil-safe accessor defaulting to the named constant `config.DefaultDispatchColumnWidth` (35), and
registered as a `configref` `Field` row beside `dispatch.watchable` (scope `both`, advertised,
`Default: 35`) so `fab config reference`, `--json`, and `fab config upgrade`'s managed fence all
carry it. Because an absent YAML int is indistinguishable from `0`, a resolved value outside
`1..99` SHALL fall back to the default (a 0% column is unsized and a 100% column leaves the
dispatcher nothing). No migration is required — the field is additive.

- **GIVEN** `dispatch:\n  column_width: 20` in the project config
- **WHEN** a first pane worker is dispatched
- **THEN** the carving split uses `-l 20%`
- **AND** GIVEN the key absent, or set to `0`/`100`/a negative, the resolved width is 35
- **AND** GIVEN the key set only in `~/.fab-kit/config.yaml`, it is honored (scope `both`, not pruned)

### Docs: the constitution-bound sweep class

#### R7: Every surface restating the title-keyed rule or the unsized split SHALL be updated
The placement-rule prose SHALL be updated in the same change: `src/kit/skills/_cli-fab.md` and
`src/kit/skills/_preamble.md` plus their `docs/specs/skills/SPEC-*.md` mirrors,
`docs/specs/harness-adapters.md` § 3, and `docs/specs/config.md` (the new field row across its
default-semantics / scope / advertise lists). No `docs/memory/` file is edited at apply —
`runtime/dispatch` is hydrate's Affected-Memory target (see Assumptions #4).

- **GIVEN** the shipped implementation
- **WHEN** the repo is grepped for the old claims (`fab-`-titled sibling, title-keyed placement,
  even/unsized first split)
- **THEN** no `src/kit/` or `docs/specs/` file still asserts them

### Non-Goals

- `select-layout` / layout-normalization enforcement, and repair of an already-mangled window —
  fab only stops creating new mess.
- run-kit-owned layout enforcement (tmux hooks / server poll).
- A separate `fab-{id}-workers` window, done-worker reaping, richer pane names/colors — deferred
  to backlog idea `[zfl7]`.
- Any change to headless dispatch, the `ShapeWindow` new-window fallback, or non-tmux paths.
- A migration file (the config field is additive).

### Design Decisions

#### Placement keys on the record's pane ID, and the live `list-panes` intersection is the filter
**Decision**: `SiblingDispatchPane` collects the `pane:` field of every `.fab-dispatch/*/{stage}.yaml`
record in the checkout, then keeps the **last** pane in `tmux list-panes -t <dispatcherPane> -F
'#{pane_id}'` that appears in that set.
**Why**: Pane IDs are server-global and stable for the pane's lifetime — the same reason
`status`/`kill`/`capture` already key on them — whereas pane titles are rewritten by the harness
running inside the worker, which is the defect. Intersecting with the window's own live pane list
collapses three filters into one probe: liveness (a dead pane is absent), same-window scoping (a
`-t <pane>` target resolves to that pane's window), and the geometric "last" ordering the stacked
column is built in. It also makes the all-record-dirs enumeration scope safe: a pane recorded by
another change in another window simply never matches.
**Rejected**: Per-record `PaneAlive` probes plus a separate window lookup (N tmux calls for the
same answer one `list-panes` already gives, and it would still need the list order to define
"last"). Scanning only the active change's record dir (a window can host two changes' workers).
Keeping the title scan as a fallback (it is the broken signal — a fallback would silently
resurrect the bug).
*Introduced by*: 260807-g4a5-pane-worker-column-invariant

#### The size rides the resolved placement, so one rule covers the fallback path
**Decision**: `SplitTarget` returns a `SplitPlacement{Target, Direction, SizePercent}` and sets
`SizePercent` only on the column-carving `SplitRight` decision; `SplitArgs` renders `-l {n}%`
from it, and `OpenSplitPane` merely executes the placement.
**Why**: "Size the carving split, never a stacking split" then exists once, in the decision, not
at each call site — and the degraded branch (probe failed ⇒ carve off the dispatcher) is the same
`SplitRight` decision, so it inherits the size for free rather than needing its own copy of the
rule. Bundling also keeps `OpenSplitPane`'s parameter list from growing a seventh argument.
**Rejected**: A `sizePercent` parameter threaded separately through `SplitTarget` and
`OpenSplitPane` (two places to remember the direction condition). Sizing inside `OpenSplitPane`
by re-deriving the direction (a second copy of the rule, able to drift from `SplitTarget`).
*Introduced by*: 260807-g4a5-pane-worker-column-invariant

#### A rejected size retries unsized rather than probing the tmux version
**Decision**: `OpenSplitPane` runs the sized split and, if tmux rejects it while a size was
requested, retries the identical split with the size dropped, returning the first failure as a
non-fatal warning.
**Why**: It covers the whole class of "this tmux will not take this size" — pre-3.1 with no
`-l N%` syntax, a window too narrow for the requested percentage — with no version string to
parse (`3.1a`, `next-3.4`, distro forks) and no extra tmux round-trip on the happy path. It is
also exactly the existing degradation contract: placement is cosmetic, so it warns and proceeds.
**Rejected**: A `tmux -V` version probe (a second call on every dispatch, and version-string
parsing that rots). Failing the dispatch (contradicts placement-is-cosmetic). Silently dropping
the size (the user set a knob; a silent no-op is unexplainable from output).
*Introduced by*: 260807-g4a5-pane-worker-column-invariant

#### `DispatchTitlePrefix` and `lastDispatchPane` are deleted, not retained
**Decision**: Both are removed with the title probe they exist for; pane titles are still SET at
spawn for identification (`select-pane -T`), and `WindowName` keeps composing `fab-{id}-{stage}`.
**Why**: `DispatchTitlePrefix`'s only consumer was the title scan (`WindowName` composes its
prefix inline), so retaining it would leave an exported constant documenting a rule the code no
longer follows — the most misleading kind of dead code.
**Rejected**: Keeping the constant and pointing `WindowName` at it (invents a consumer to justify
a symbol whose stated purpose is gone). Keeping the title probe as a secondary signal (see the
first decision).
*Introduced by*: 260807-g4a5-pane-worker-column-invariant

## Tasks

### Phase 1: Config field

- [x] T001 Add `ColumnWidth int \`yaml:"column_width"\`` to `DispatchConfig`, the named constant `DefaultDispatchColumnWidth = 35`, and the nil-safe `GetDispatchColumnWidth()` accessor (out-of-range ⇒ default) in `src/go/fab/internal/config/config.go` <!-- R6 -->
- [x] T002 Register the `dispatch.column_width` row in `src/go/fab/internal/configref/configref.go` (Default from `config.DefaultDispatchColumnWidth`, scope `both`, advertised, empty Segment) and extend `dispatchSegment()` to document + scaffold `column_width` inside the shared commented `dispatch:` block <!-- R6 -->
- [x] T003 [P] Tests for the field in `src/go/fab/internal/config/config_test.go` — parses from the project layer, honored from the system layer (scope `both`), project-wins, absent/0/100/negative ⇒ 35, nil-safe accessor — mirroring the `dispatch.watchable` cases <!-- R6 -->
- [x] T004 [P] Registry tests in `src/go/fab/cmd/fab/config_test.go` — add `dispatch.column_width` to the `hasDefault` set and the scope-assignment want-map, and add a row test (Default 35, scope both, advertised, rendered commented in the reference, scaffold parses inert) <!-- R6 -->
- [x] T005 [P] Fence test in `src/go/fab/internal/configupgrade/configupgrade_test.go` — the un-overridden field is advertised in the managed fence via the shared `dispatch:` segment <!-- R6 -->

### Phase 2: Record-based placement + sized split

- [x] T006 Add the record-enumeration helper `recordedPanes(repoRoot)` to `src/go/fab/internal/dispatch/dispatch.go` — walk `.fab-dispatch/*/`, load each `{stage}.yaml` (skipping `-result.yaml`), collect non-empty `Pane` values; absent tree ⇒ empty set, no error <!-- R1 --> <!-- rework: (1) signature must be `(map[string]bool, error)` — distinguish os.IsNotExist on the tree root (normal first-dispatch, no warning) from real read/parse failures, which currently degrade SILENTLY via continue/early-return against R4/A-004; (2) must take the probe's `server` and skip records whose `Server` differs (pane IDs are per-socket; a --server record's %N false-matches a default-socket pane against R1/A-017 — default-socket records carry Server:"", equality is the exact test) -->
- [x] T007 Rewrite the placement half of `src/go/fab/internal/dispatch/pane_mode.go`: replace `SiblingDispatchPane`'s title scan with the record ∩ `list-panes` intersection (pure `lastRecordedPane` parser), delete `DispatchTitlePrefix` + `lastDispatchPane`, introduce `SplitPlacement` + the pure `splitPlacement` decision, and re-signature `SplitTarget(server, dispatcherPane, repoRoot, columnWidth)` <!-- R1 R2 R3 --> <!-- rework: join recordedPanes' new error into SiblingDispatchPane's returned error (and pass `server` through to it) so a real record-read failure reaches the caller's warn path instead of returning nil — degraded placement decision unchanged -->
- [x] T008 Add the pure `SplitArgs(place, dir, cmd)` argv composer (sized `-l {n}%` only for `SplitRight`) and rework `OpenSplitPane(server, place, title, dir, cmd)` to execute a placement, retry unsized on a rejected size, and return non-fatal `warnings []error` <!-- R3 R4 -->
- [x] T009 Update the `pane_mode.go` file-header and split-direction/`OpenSplitPane` doc comments to describe the record-keyed, sized-column rule <!-- R1 R2 R3 -->

### Phase 3: Cmd wiring

- [x] T010 Wire the change context and column width through `src/go/fab/cmd/fab/dispatch_start.go`: carry `columnWidth` on `paneTarget` (filled by `runDispatchLaunch` from `cfg.GetDispatchColumnWidth()`), pass `repoRoot`/width into `dispatch.SplitTarget`, and surface `OpenSplitPane`'s warnings <!-- R1 R3 R4 --> <!-- rework: verify launchPane's existing warn now fires for record-read failures surfaced through SplitTarget's probeErr (was: only a failing tmux list-panes reached it) — wording should name what failed -->
- [x] T011 Confirm `dispatch_restart.go` needs no placement branch (shared `resolveMode` + `runDispatchLaunch`) and update its doc comment where it enumerates what is re-derived <!-- R5 -->

### Phase 4: Tests

- [x] T012 [P] `src/go/fab/internal/dispatch/dispatch_test.go` — table tests for `recordedPanes` (headless records and result files skipped, absent tree), the pure `lastRecordedPane` intersection, the pure `splitPlacement` decision (sibling ⇒ `-v` unsized; none ⇒ `-h` sized), and `SplitArgs` argv composition; delete `TestLastDispatchPane` + `TestDispatchTitlePrefixMatchesWindowName` <!-- R1 R2 R3 --> <!-- rework: add cases — (1) unreadable record/corrupt YAML returns a non-nil error (and absent tree root still returns empty set + nil error); (2) server filter: a record with Server:"other" is skipped when probing the default socket (and vice versa), Server:"" matches default-socket probes; close A-004 + A-017 -->
- [x] T013 `src/go/fab/cmd/fab/dispatch_start_test.go` — the regression test: with the first worker's pane **title clobbered**, a second dispatch still stacks in the column; plus the sized-column assertion (default 35 and a configured width) and window hermeticity (isolate `HOME` in the shared fixture) <!-- R1 R2 R3 -->
- [x] T014 [P] `src/go/fab/cmd/fab/dispatch_restart_test.go` — a restart from the dispatcher's pane with a live recorded sibling stacks in the right column <!-- R5 -->

### Phase 5: Docs & spec sweep

- [x] T015 [P] Update `src/kit/skills/_cli-fab.md` § fab dispatch (split-placement bullet: record-keyed detection, sized carving split, `dispatch.column_width`, degradation) <!-- R7 -->
- [x] T016 [P] Update `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch pane-mode bullets (the two-tier hierarchy bullet's `fab-`-titled sibling phrasing) <!-- R7 -->
- [x] T017 [P] Update the SPEC mirrors `docs/specs/skills/SPEC-_cli-fab.md` and `docs/specs/skills/SPEC-_preamble.md` <!-- R7 -->
- [x] T018 [P] Update `docs/specs/harness-adapters.md` § 3 (interactive-pane adapter split/placement bullets) <!-- R7 -->
- [x] T019 [P] Update `docs/specs/config.md` (`dispatch.column_width` in the default-semantics, scope, and advertise lists) <!-- R7 -->
- [x] T020 Repo-wide grep for the old claims (`fab-`-titled sibling, title-keyed placement, unsized/even first split) across `src/kit/` and `docs/specs/`; fix any aggregate restatement found <!-- R7 -->

## Execution Order

- T001 blocks T002 (the registry row sources its Default from the config constant) and T003/T004/T005
- T006 blocks T007; T007 blocks T008 → T009 → T010 → T011
- T012–T014 follow Phase 3; T013 depends on T010
- Phase 5 is independent of Phases 1–4 and may run alongside them

## Acceptance

### Functional Completeness

- [x] A-001 R1: Sibling detection reads dispatch-record pane IDs intersected with the dispatcher window's live `list-panes` output; no code path matches on `#{pane_title}` for placement
- [x] A-002 R2: The first worker splits the dispatcher `-h`, later workers split the last live recorded worker `-v`; no `select-layout`/resize call exists anywhere in the dispatch package or its cmd wiring
- [x] A-003 R3: The column-carving split argv carries `-l {n}%` and stacking splits carry no `-l`
- [x] A-004 R4: Record-read, probe, and rejected-size failures all degrade with a stderr warning and still launch the worker
- [x] A-005 R5: `restart` reaches the same placement through the shared launch path with no restart-specific branch
- [x] A-006 R6: `dispatch.column_width` parses from either config layer, defaults to 35, and appears in `fab config reference`, `--json`, and the managed fence
- [x] A-007 R7: `_cli-fab.md`, `_preamble.md`, both SPEC mirrors, `harness-adapters.md`, and `config.md` describe the record-keyed sized-column rule

### Behavioral Correctness

- [x] A-008 R1: A worker whose pane title has been clobbered is still found as a sibling — the defect's regression test fails against the pre-change implementation
- [x] A-009 R3: With the default width the dispatcher keeps ~65% of the window; the worker column is measurably narrower than an even split
- [x] A-010 R6: An out-of-range `column_width` (0, 100, negative) resolves to 35 rather than producing a degenerate split

### Removal Verification

- [x] A-011 R1: `DispatchTitlePrefix` and `lastDispatchPane` are gone from the tree, along with their tests, with no dangling references in Go code, skills, or specs

### Scenario Coverage

- [x] A-012 R1: Table tests cover `recordedPanes` (headless record skipped, `-result.yaml` skipped, absent tree) and the `lastRecordedPane` intersection (no match, last-wins, unrecorded panes ignored)
- [x] A-013 R2: An integration test proves three panes in one window with the second worker sharing the first worker's left edge and a distinct top edge
- [x] A-014 R3: A test pins the sized argv composition and an integration test measures the resulting pane widths for both the default and a configured width
- [x] A-015 R5: A restart integration test proves column stacking on the restart path

### Edge Cases & Error Handling

- [x] A-016 R4: A tmux that rejects the size still launches the worker (retried unsized) and the warning names the fallback
- [x] A-017 R1: Enumerating all record dirs cannot misplace a worker — a recorded pane in another window or a dead recorded pane never matches

### Code Quality

- [x] A-018 Pattern consistency: New symbols follow the package's pure-function-plus-thin-tmux-wrapper convention (`SelectMode`/`SelectPaneShape`/`DeriveState` precedent), with env/config reads staying in the cobra layer
- [x] A-019 No unnecessary duplication: Record loading reuses `dispatch.Load`; tmux invocation reuses `internal/pane`'s `RunCmd`/`WithServer`/`StderrError`; the sizing rule exists in exactly one place
- [x] A-020 Named constants: the column-width default and the `-l` size flag are named constants, not magic values (code-quality § Anti-Patterns)
- [x] A-021 Canonical source only: no file under `.claude/skills/` is edited; kit changes land in `src/kit/skills/`
- [x] A-022 CLI ⇒ docs + tests: the Go change ships tests and updates `src/kit/skills/_cli-fab.md`
- [x] A-023 SPEC-mirror sync: every edited `src/kit/skills/*.md` carries its `docs/specs/skills/SPEC-*.md` update
- [x] A-024 No migration needed: the additive config field restructures no user data (context.md § Migrations)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- A-004 closed (rework 1): `recordedPanes` returns `(map[string]bool, error)` — an absent tree is the
  benign empty-set/nil-error case, every real read/parse failure is joined into `SiblingDispatchPane`'s
  error and surfaced by `launchPane`'s warn, which now names the failure AND the resolved placement.
  A partial read keeps the readable records (an unread record can only fail to find a sibling, never
  invent one), so the degraded placement is either the clean answer or the sized carve.
- A-017 closed (rework 1): `recordedPanes` takes the probe's `server` and skips records whose `Server`
  differs — pane IDs are per-socket, and default-socket records carry `Server: ""`, so equality is the
  exact test in both directions.

## Deletion Candidates

- `dispatch.SplitArgs` / `dispatch.SizeFlag` (`internal/dispatch/pane_mode.go:471`, `:328`) — newly **exported** with no cross-package caller; both are used only inside `pane_mode.go` and its in-package tests, so the export widens the package surface for nothing.
- `dispatch.SplitBelow` (`internal/dispatch/pane_mode.go:325`) — no cross-package caller left: `cmd/fab` used to pass a bare `direction` into `OpenSplitPane` and now passes a `SplitPlacement` that carries it. Its twin `SplitRight` is **still** referenced cross-package (`cmd/fab/dispatch_start.go:506`, in `describePlacement`), so the pair can drop to package scope only if that comparison moves into the dispatch package (see the nice-to-have below).
- `describePlacement` (`cmd/fab/dispatch_start.go:506`) — candidate for relocation rather than deletion: it is the only cross-package reader of a `SplitPlacement`'s `Direction`, so as a method on `SplitPlacement` it would keep the placement vocabulary beside the type and free both direction constants to become package-scope.
- `dispatch.SiblingDispatchPane` (`internal/dispatch/pane_mode.go:394`) — exported but reached only through `SplitTarget`, which is the sole cmd-layer entry point; the export predates this change but is now unambiguously internal-only.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The record ∩ live-`list-panes` intersection replaces per-record `PaneAlive` calls — the intersection *is* the liveness + same-window filter | Intake assumption #8 already defines "last" as the list-panes intersection; deriving liveness from the same output is strictly cheaper and cannot disagree with the ordering | S:75 R:85 A:85 D:80 |
| 2 | Confident | A rejected `-l N%` is handled by retrying the split unsized, not by probing `tmux -V` | Covers pre-3.1 syntax and too-narrow windows in one branch, with no version-string parsing; intake assumption #9 mandates only the degrade-with-warning outcome, not the detection mechanism | S:70 R:85 A:80 D:75 |
| 3 | Confident | `dispatch.column_width` outside `1..99` falls back to 35 (an absent YAML int is indistinguishable from 0, so 0 cannot mean "unsized") | Mirrors the `dispatch.watchable` bool boundary-case reasoning already recorded in `docs/specs/config.md` § Default semantics | S:70 R:85 A:85 D:70 |
| 4 | Certain | `docs/memory/runtime/dispatch.md` is NOT edited during apply — memory is hydrate's stage (`/fab-continue` Key Properties: "Modifies docs/memory/? Yes — during hydrate"), and the intake lists it as Affected Memory | Editing it at apply would double-write with hydrate's present-truth merge; the intake's sweep bullet names the file as in-scope for the change, which hydrate satisfies | S:90 R:90 A:95 D:90 |
| 5 | Confident | `dispatch.column_width` shares `dispatch.watchable`'s rendered Segment (one commented `dispatch:` block) and carries an empty `Segment` of its own | The `project.name`/`project.description`/`project.linear_workspace` rows establish exactly this pattern for multiple keys under one YAML block, and the fence's override detection is already top-level-key scoped | S:80 R:85 A:85 D:80 |
| 6 | Confident | `config.DefaultDispatchColumnWidth` is the canonical symbol the configref row interpolates (configref → config is a new, cycle-free import edge) | configref's stated discipline is that every default comes from a canonical Go symbol; `internal/config` imports only `internal/configscope`, so the edge cannot cycle | S:70 R:80 A:85 D:75 |
| 7 | Confident | `HOME` is isolated in the shared `cmd/fab` dispatch test fixture | Config now influences pane placement, so a developer's `~/.fab-kit/config.yaml` could otherwise change a placement assertion's expected geometry; the reference tests already use this discipline | S:65 R:90 A:85 D:75 |

7 assumptions (1 certain, 6 confident, 0 tentative).
