# Plan: Pane-Dispatch Geometry Defaults 45/50

**Change**: 260909-tdij-pane-geometry-defaults-45-50
**Intake**: `intake.md`

## Requirements

### Dispatch: Geometry Defaults

#### R1: The shipped dispatch geometry defaults are 45 / 50
`src/go/fab/defaults.yaml` MUST ship `dispatch.column_width: 45` and `dispatch.min_cols: 50`. `mode: native`, `min_rows: 20`, and `reap_done: true` MUST be unchanged. These two lines are the sole value source; `config.DefaultDispatchColumnWidth` / `config.DefaultDispatchMinCols` MUST continue to carry no literal of their own.

- **GIVEN** a project and system config with no `dispatch.column_width` / `dispatch.min_cols` override
- **WHEN** `fab config show dispatch` (or any dispatch-time accessor) resolves the two fields
- **THEN** `column_width` resolves to `45` and `min_cols` to `50`, and `min_rows` still resolves to `20`

#### R2: Zero Go logic change
The carve formula (`window_width × column_width / 100`, integer division), the "exactly at the floor passes" rule, the width-before-height ordering of `SplitBelowFloor`, the fail-open geometry probe, the `-l <n>%` split argv, and the `ManualWindowCols/ManualWindowRows` 200×50 constants MUST be byte-for-byte unchanged. Only comments and `Description` strings in Go files change.

- **GIVEN** a dispatcher pane in a 120-column, 40-row tmux window with default config
- **WHEN** `fab dispatch open` prices the carving split
- **THEN** the worker is planned at 54 cols (≥ 50) and the launch splits with `-h -l 45%`; the dispatcher keeps 66 cols

- **GIVEN** the same dispatcher in a 111-column window
- **WHEN** the carving split is priced
- **THEN** 49 cols < 50 demotes to the manual window with `warning: window 111x40 too narrow for a 45% column (49 cols < 50): opening worker in its own window`

- **GIVEN** the same dispatcher in a 112-column window
- **WHEN** the carving split is priced
- **THEN** 50 cols is exactly at the floor and the launch splits (the new minimum splitting width)

#### R3: Drift guards pin the new values and the suite is green
Every test that pins or exercises the dispatch defaults MUST be updated to the new values and MUST pass: `TestDefaultsFileDispatchBlockIsPinned` (45/50), `TestDispatchOpen_BelowFloorDemotesToSizedWindow` (`window 80x24 too narrow for a 45% column (36 cols < 50)`), `TestSetupWizard_AdvancedOptInAsksAllKeys` (`dispatch.column_width [45]:`), and `TestCascade_DispatchColumnWidthFromSystemLayer`'s project literal MUST move off the new default (45 → 35) so it still discriminates project from built-in. `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` MUST pass by appending the new dispatch-advert digest to `knownGeneratedSystemParagraphDigests` (never removing an old one). `dispatch_test.go`'s explicit-argument `SplitBelowFloor` table cases MUST NOT be touched.

- **GIVEN** the edited `defaults.yaml`
- **WHEN** `go test ./...` runs under `src/go/fab`
- **THEN** every package passes, and the configupgrade catalog carries a new digest whose comment names it as the current 45/50 advert while the prior entry's comment is demoted to historical

#### R4: Every prose restatement of the old values is swept
All non-test restatements of `40` (column width) and `60` (min_cols) in the sibling class MUST read `45` / `50`: `configref.go` (the `dispatch.column_width` row comment "bottoms out at 40" and its `Description` "Default 40."; the `dispatch.min_cols` `Description` "Default 60."), `pane/create.go` (`-l 40%` comment example), `src/kit/skills/_cli-fab.md` (`default 40`, `defaults 60/20`), `src/kit/skills/_preamble.md` (`` defaults `60`/`20` ``), `docs/specs/architecture.md`, `docs/specs/config.md`, `docs/specs/glossary.md`, `docs/specs/harness-adapters.md`, `docs/memory/runtime/dispatch.md`, and `docs/memory/_shared/configuration.md` (including the pre-#654 stale `column_width: 35` / `min_cols: 80` at line 34 of the `--json` paragraph). Worked examples MUST be recomputed so their arithmetic is true at 45/50: the `127x16 … (50 cols < 60)` example becomes `100x16 … (45 cols < 50)`; `-h -l 40%` / `~60%` becomes `-h -l 45%` / `~55%`. Deployed copies under `.agents/skills/` and `.claude/skills/` MUST NOT be edited.

- **GIVEN** the finished apply
- **WHEN** the repo is grepped (excluding `.git`, `.agents`, `.claude`, `.opencode`, `fab/changes`, `docs/**/archive` — every deployed-copy target is excluded because `fab sync` regenerates them) for `column_width: 40`, `40% column`, `-l 40%`, `(default 40)`, `default 40` (bare, word-bounded), `` default `40` `` (backticked), `Default 40.`, `bottoms out at 40`, `min_cols: 60`, `60 cols`, `< 60)`, `default 60`, `` default `60` ``, `Default 60.`, `defaults 60/20`, `` defaults `60` ``, `60x20`, `~60%`, `column_width \[40\]`, `column_width`: `35`, `min_cols`: `80`
- **THEN** the only hits are `dispatch_test.go`'s explicit-argument table cases, `config_test.go`'s deliberately non-default project literals (`min_cols: 60`, `column_width: 35`), and the historical digest comment in `configupgrade.go`

#### R5: The project fence advert reflects the new defaults
The commented dispatch advert inside `fab/project/config.yaml`'s `>>> fab reference (kit 2.24.2) >>>` fence MUST read `#   column_width: 45` and `#   min_cols: 50`. Nothing else in the file — including the fence header's kit version — changes.

- **GIVEN** the edited `defaults.yaml` and a locally built `fab`
- **WHEN** `fab config upgrade` regenerates the fence (or the two lines are edited to the identical renderer output)
- **THEN** `git diff fab/project/config.yaml` shows exactly the two value lines changed

### Non-Goals

- Any change to `SplitBelowFloor` / `plannedWorkerSize` / `splitArgs` / `OpenManualWindow`, or to the 200×50 manual-window constants — the `max(pct, min_cols)` fallback was rejected in favour of this defaults bump
- `min_rows`, `mode`, `reap_done`, or any claude model default
- A migration (commented advert text and user overrides are untouched) or a release/version bump
- The user's `~/.fab-kit/config.yaml`

### Design Decisions

#### Lower the geometry floor by value, not by formula
**Decision**: Admit 120-column windows into pane mode by shipping `column_width: 45` / `min_cols: 50` — a pure defaults bump.
**Why**: Two existing policy knobs already govern the outcome; at 120 cols the worker gets 54 cols and the smallest splitting window becomes 112 cols, both accepted by the user. The third requirement was minimal change to the rest of the source.
**Rejected**: An absolute-column fallback (`-l max(pct%, min_cols)`, demoting only when `window < 2 × min_cols`) — it changes the carve formula, the split argv, the warning grammar, and every doc/test that describes them, for a marginal gain over better defaults.
*Introduced by*: 260909-tdij-pane-geometry-defaults-45-50

## Tasks

### Phase 1: Setup

- [x] T001 Edit `src/go/fab/defaults.yaml` (`column_width: 45`, `min_cols: 50`); update the pinned tests `src/go/fab/internal/agent/defaults_test.go` (45/50), `src/go/fab/cmd/fab/dispatch_open_test.go` (comment + `wantWarning` → `window 80x24 too narrow for a 45% column (36 cols < 50): …`), `src/go/fab/cmd/fab/setup_test.go` (`dispatch.column_width [45]:`), `src/go/fab/internal/config/config_test.go` (`TestCascade_DispatchColumnWidthFromSystemLayer` project literal 45 → 35 and its message); run `go test ./internal/configupgrade -run TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer`, append the reported digest to `knownGeneratedSystemParagraphDigests` in `src/go/fab/internal/configupgrade/configupgrade.go` with a "Current … 45 / 50 (260909-tdij)" comment and demote the prior "Current …" comment to historical; then `go test ./...` under `src/go/fab` green <!-- R1, R3 -->

### Phase 2: Core Implementation

- [x] T002 [P] Sweep Go prose: `src/go/fab/internal/configref/configref.go` (`bottoms out at 40` → 45, `Default 40.` → `Default 45.`, `Default 60.` → `Default 50.`) and `src/go/fab/internal/pane/create.go` (`-l 40%` → `-l 45%`); no logic change <!-- R2, R4 -->
- [x] T003 [P] Sweep kit skill sources `src/kit/skills/_cli-fab.md` (`default 40` → 45, `defaults 60/20` → 50/20) and `src/kit/skills/_preamble.md` (`` defaults `60`/`20` `` → `` `50`/`20` ``); regenerate the `fab/project/config.yaml` fence with a locally built `fab` (`go build -o /tmp/... ./cmd/fab` under `src/go/fab`, then `config upgrade`) and keep only the two value-line changes <!-- R4, R5 -->
- [x] T004 [P] Sweep specs: `docs/specs/architecture.md` (defaults 60/20 → 50/20; YAML block 40/60 → 45/50; walkthrough comment `(optional, default 40)` → 45), `docs/specs/config.md` (`column_width: 40`, `min_cols: 60` → 45/50), `docs/specs/glossary.md` (`` defaults `60`/`20` `` → `50`/`20`; `dispatch.column_width` row `` default `40` `` → `45`), `docs/specs/harness-adapters.md` (default 40 → 45, defaults 60/20 → 50/20) <!-- R4 --> <!-- rework cycle 1: review found architecture.md:326 and glossary.md:30 still at `default 40` — the sweep grep lacked the bare/backticked spellings; fixed and grep widened ->
- [x] T005 [P] Sweep memory restatements as current truth: `docs/memory/runtime/dispatch.md` (default 40 → 45; `60 cols × 20 rows` → `50 cols × 20 rows`; recompute the `127x16 … (50 cols < 60)` example to `100x16 … (45 cols < 50)` in the prose and the GIVEN/THEN scenario; `-h -l 40%`/`~60%` → `-h -l 45%`/`~55%`; `resolving to 40` → 45; `floor 60x20` → `50x20`) and `docs/memory/_shared/configuration.md` (line 34 stale `35`/`80` → 45/50; `column_width` bullet `` default `40` `` → `45`; `min_cols`/`min_rows` bullet defaults `60`/`20` → `50`/`20`; the two Design Decisions value lists `column_width: 40`, `min_cols: 60` → 45/50); then re-run the R4 sweep grep and confirm only the allowed hits remain <!-- R4 --> <!-- rework cycle 1: review found configuration.md:245 `column_width` bullet still at `default 40`; fixed -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` ships `column_width: 45`, `min_cols: 50`; `mode`, `min_rows`, `reap_done` unchanged; `fab config show dispatch` from a locally built binary reports 45/50
- [x] A-002 R3: `go test ./...` under `src/go/fab` passes; the four pinned tests assert the new values; the configupgrade digest catalog has one new entry and no removals
- [x] A-003 R5: `git diff fab/project/config.yaml` is exactly the two commented value lines

### Behavioral Correctness

- [x] A-004 R2: `git diff` of `internal/dispatch/pane_mode.go`, `internal/pane/create.go`, and `cmd/fab/dispatch_start.go` contains no non-comment change (create.go: one comment line only; the other two: untouched)
- [x] A-005 R4: every restated value in the sibling class reads 45/50, including the pre-#654 stale `35`/`80` in `configuration.md` (rework cycle 1 fixed the three stale `default 40` sites at `docs/specs/architecture.md:326`, `docs/memory/_shared/configuration.md:245`, `docs/specs/glossary.md:30`; re-verified by the re-reviewer's independent grep)

### Scenario Coverage

- [x] A-006 R2: `TestDispatchOpen_BelowFloorDemotesToSizedWindow` exercises the 80×24 demotion with the recomputed `36 cols < 50` warning and passes
- [x] A-007 R4: the recomputed doc example (`100x16 → 45 cols < 50`) is arithmetically true under integer division and still demotes on width

### Edge Cases & Error Handling

- [x] A-008 R3: `TestCascade_DispatchColumnWidthFromSystemLayer`'s project literal is not equal to the new default (so the test still proves the system layer beats the project layer)
- [x] A-009 R4: `dispatch_test.go`'s explicit-argument `SplitBelowFloor` cases are untouched

### Code Quality

- [x] A-010 Pattern consistency: the configupgrade digest append and comment demotion follow #654's exact pattern
- [x] A-011 No unnecessary duplication: no new Go literal for either default is introduced anywhere (values flow from `defaults.yaml` via the init-injected symbols)
- [x] A-012 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`
- [x] A-013 Sibling sweep: the R4 grep was run before finishing apply and its residue matches the allowed list (rework cycle 1 widened the sweep patterns to the bare/backticked spellings; the re-reviewer's independent re-run of the widened grep confirms the residue is only `dispatch_test.go`'s explicit-argument cases, `config_test.go`'s deliberate non-default literals, and the historical `configupgrade.go` digest comment)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Memory value restatements (`dispatch.md`, `configuration.md`) are swept during apply (T005), with hydrate adding provenance/summary and regenerating indexes | code-quality.md § Sibling Sweeps puts the behaviour-documenting memory file in the sweep class; review treats stale copies as must-fix, so sweeping before review avoids a guaranteed rework cycle | S:85 R:95 A:90 D:85 |
| 2 | Certain | The stale `dispatch.column_width: 35` / `min_cols: 80` in `configuration.md`'s `--json` paragraph is corrected in the same sweep | Same class, missed by #654; leaving a third value pair in the docs would be worse than the change it rides on | S:80 R:95 A:95 D:90 |
| 3 | Certain | Five tasks → light lane; all work runs inline in the orchestrator | Task count ≤ 5 by the bracket's rule; the work is value substitution across known sites | S:90 R:90 A:90 D:90 |

3 assumptions (3 certain, 0 confident, 0 tentative).
