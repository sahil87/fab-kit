# Plan: Config Source Consolidation (Config-Overhaul C5)

**Change**: 260809-wll4-config-source-consolidation
**Intake**: `intake.md`

## Requirements

### Built-in Defaults: Single Value Source

#### R1: Dispatch defaults live only in `defaults.yaml`
The three `dispatch.*` built-in default values (`mode: native`, `column_width: 35`, `reap_done: true`) SHALL have exactly one value source: `src/go/fab/internal/agent/defaults.yaml`. The Go constants `DefaultDispatchMode`/`DefaultDispatchColumnWidth`/`DefaultDispatchReapDone` in `src/go/fab/internal/config/config.go` MUST be removed as independent value literals. The fail-open accessor behavior on `config.Config` (`GetDispatchMode`/`GetDispatchColumnWidth`/`GetDispatchReapDone`) MUST be preserved byte-for-byte: nil/absent/invalid/out-of-range inputs resolve to the same default values, and the invalid-mode warning text is unchanged.

- **GIVEN** the built-in tier of the config cascade
- **WHEN** any consumer reads a dispatch default (absent, invalid, or out-of-range user value)
- **THEN** the value it receives is sourced from `defaults.yaml`'s `dispatch:` block
- **AND** no Go source file carries an independent copy of the values `native`/`35`/`true` as dispatch defaults

#### R2: Registry and consumers interpolate from the same source
`internal/configref` registry rows and rendered segments, and `cmd/fab` consumers (e.g. `dispatch_reap.go`), MUST obtain dispatch default values from the single `defaults.yaml` source (via the Go seam chosen per the Design Decision below), never from a second literal.

- **GIVEN** a registry row or rendered fence segment referencing a dispatch default
- **WHEN** `fab config explain`, `fab config init --project`, or `fab config upgrade` renders output
- **THEN** every dispatch default in the output traces back to `defaults.yaml`

#### R3: Drift guards name themselves on failure
Tests guarding the single-source arrangement MUST follow the established `defaults.yaml` pin-test pattern: a test that fails when the `dispatch:` block drifts, and a test that fails when the Go exposure seam (config↔agent wiring) breaks, each naming itself and the fix in its failure output.

- **GIVEN** a maintainer edits `defaults.yaml`'s `dispatch:` block or the seam wiring
- **WHEN** the test suite runs
- **THEN** a drift or wiring break fails a test whose output names the test, the expectation, and where the canonical value lives

### fab-kit Binary: Stub Retirement

#### R4: The embedded stub config.yaml becomes a hard error
`src/go/fab-kit/internal/init.go`'s fallback to an embedded minimal stub config.yaml (when the installed fab-go cannot generate one) MUST be replaced by a clear, non-zero-exit error telling the user to upgrade fab-go. Zero second copies of config content remain in any binary: `stubConfigHeader`, `renderStubConfig`, and `writeStubConfig` are deleted, and `init_test.go`'s stub-fallback cases become error-path cases.

- **GIVEN** a machine whose installed fab-go predates `fab config init --project` (2.15.x)
- **WHEN** the user runs `fab init` and the shell-out fails to produce a config.yaml
- **THEN** `fab init` exits non-zero with a message naming the cause and instructing the user to upgrade fab-go
- **AND** no stub config.yaml content is written or embedded anywhere

### Config Drift Probe

#### R5: `fab config upgrade --check`
The existing `upgrade` verb SHALL gain a `--check` flag: it computes exactly what `upgrade` would do and reports it, but writes nothing. It exits non-zero when a run would change the file (stale fence kit-version stamp, unparked unknown keys, missing fence, any rendered-content delta — including a missing config.yaml, which a real run would create) and exit 0 when the file is clean. Output names what would change (the same report lines the applying run prints).

- **GIVEN** a repo whose fab/project/config.yaml has drifted from the registry (stale stamp, unparked key, or missing fence)
- **WHEN** a human or CI runs `fab config upgrade --check`
- **THEN** the command exits non-zero, names what would change, and the file on disk is byte-identical before and after
- **AND** on a clean file it exits 0 with an up-to-date message and writes nothing

### Generated-File Text: Scope Annotations + Prose Diet

#### R6: Fence adverts carry scope annotations and machine-wide pointers
Each field advert rendered into the managed fence (and the `--system` scaffold) SHALL carry its scope as a `[project|system|both]` annotation sourced from the `internal/configscope` taxonomy on the registry row. Adverts for preference-class fields (`scope: both`) SHALL carry a "settable machine-wide: `fab config set --system <key> <value>`" pointer instead of inviting an uncomment-in-repo.

- **GIVEN** a freshly rendered managed fence or system scaffold
- **WHEN** the reader scans any field advert
- **THEN** the advert states the field's scope tag, and a `scope: both` advert points at `fab config set --system` for the machine-wide home

#### R7: Generated files carry short descriptions, not essays
The configref rendered segments consumed by generated files (the project `config.yaml` managed fence and `init --system`'s scaffold) SHALL be tightened so each field carries a short description plus a `fab config explain <key>` pointer for the full prose. The registry keeps the essays — `fab config explain` output is unchanged in depth. Rendering remains byte-stable and idempotent; golden/byte-stability tests move once for items 4+7 together.

- **GIVEN** the current paragraph-length per-field fence essays
- **WHEN** the fence or system scaffold is regenerated
- **THEN** each field carries a short description line plus a `fab config explain <key>` pointer, and the full prose remains available via `fab config explain`

### Init Verb: Preview and Overwrite

#### R8: `fab config init --print`
The `init` verb SHALL gain a `--print` flag that renders the exact file `init` would write to stdout with zero writes. It composes with both `--project` (default/bare mode) and `--system`. An existing target file does not block `--print` (it is a preview, not a write).

- **GIVEN** any repo state (config present or absent)
- **WHEN** the user runs `fab config init --print` (optionally with `--system`, or with the project seed flags)
- **THEN** stdout carries exactly the bytes the corresponding applying run would write, and no file is created or modified

#### R9: `fab config init --force`
The `init` verb SHALL gain a `--force` flag that replaces the existing-file refusal with an explicit overwrite. Refusal remains the default when `--force` is absent.

- **GIVEN** an existing fab/project/config.yaml (or ~/.fab-kit/config.yaml with --system)
- **WHEN** the user runs `fab config init --force`
- **THEN** the file is overwritten with freshly rendered content
- **AND** without `--force` the command refuses exactly as before

### Docs & Surface Obligations

#### R10: `_cli-fab.md` and its SPEC mirror document the new surface
`src/kit/skills/_cli-fab.md` § fab config MUST document the new `--check`, `--print`, and `--force` flags and the new fence-text shape, and `docs/specs/skills/SPEC-_cli-fab.md` MUST be updated in the same change (constitution + code-review must-fix). The sibling sweep includes repo-wide restatements of the fab config surface (e.g. `src/go/fab/cmd/fab/skill.md`, `docs/site/skill.md`) where they describe these verbs.

- **GIVEN** the merged CLI changes (R5, R8, R9, and the R6/R7 text changes)
- **WHEN** a reader consults `_cli-fab.md` or `SPEC-_cli-fab.md`
- **THEN** the documented `fab config upgrade`/`fab config init` signatures and fence description match the shipped behavior

#### R11: `docs/specs/config.md` reflects the consolidated sources
`docs/specs/config.md` MUST be updated: the managed-fence section (scope annotations + short-line format), the `upgrade --check` and `init --print`/`--force` verb descriptions, and the embed census (`defaults.yaml` values + configref registry + configscope taxonomy + `src/kit/scaffold/`, zero stub copies, stub retirement in fab-kit).

- **GIVEN** the shipped consolidation
- **WHEN** a reader consults `docs/specs/config.md`
- **THEN** every claim about value sources, the fence format, and the two verbs matches the implementation

#### R12: shll standards check on the final CLI text
Before finalizing flag naming and help text, the change MUST be checked against the shll toolkit standards (`shll standards`, constitution § Toolkit Standards) governing CLI surface/help output.

- **GIVEN** the final help text for `--check`, `--print`, `--force`
- **WHEN** `shll standards` entries governing CLI surface are consulted
- **THEN** the shipped names and help text conform, or deviations are recorded

### Non-Goals

- Merging `defaults.yaml` as a true layer 0 inside `config.LoadPath` — plan-resolved deferral ("do NOT fold this into the same change"); fp02 settled the materialized-defaults tier at the read-model boundary.
- Plan item 5 (`show --origin` knob-blind fix) — shipped by fp02 (#557); dropped at intake.
- A migration file — no user-data restructure; the managed fence self-heals via `fab config upgrade` and the stub retirement changes only binary fallback behavior.
- Regenerating this repo's own `fab/project/config.yaml` fence — happens on the next `fab config upgrade` after release, not in this diff.
- Memory updates (`docs/memory/`) — owned by the hydrate stage per the intake's Affected Memory list.

### Design Decisions

#### Init-injected vars as the config↔agent seam for dispatch defaults
**Decision**: `internal/config`'s three dispatch default constants become package-level vars (same exported names, no literals). `internal/agent` — which owns and parses the embedded `defaults.yaml` — assigns the parsed `dispatch:` values into them from an `init()` function. Accessor signatures, names, and fail-open semantics are unchanged, so consumers (configref, cmd/fab, existing tests) keep compiling against the same symbols.
**Why**: `internal/agent` imports `internal/config`, so config cannot read the values back from agent (the configref→agent→config cycle fp02 documented). Push-from-agent at init is the only direction the import graph allows, and keeping the exported names avoids a repo-wide call-site churn. Go guarantees all package inits complete before any runtime use, so every real binary (the fab module always links agent) sees the injected values.
**Rejected**: (a) Moving the accessors off `config.Config` into agent — churns ~6 call sites plus tests and loses the nil-safe method idiom for zero gain. (b) A second embedded file in `internal/config` — violates the one-value-source requirement outright. (c) Keeping fallback literals in config "just in case" — recreates the duplication this change exists to kill. The zero-value hazard (a test binary that never links agent sees empty defaults) is closed by a self-naming blank-import test file in `internal/config` and the R3 wiring guard test.
*Introduced by*: 260809-wll4-config-source-consolidation

#### Items 4 and 7 land as one renderer rework
**Decision**: The scope annotations (R6) and the prose diet (R7) are implemented together as a single rewrite of the configref segment renderer, with byte-stability/golden tests moved once.
**Why**: Both rework the same rendered fence text; moving goldens twice would be pure waste (plan-resolved, user-confirmed).
**Rejected**: Shipping scope annotations first and the diet later — doubles the golden churn for no behavioral benefit.
*Introduced by*: 260809-wll4-config-source-consolidation

#### `--check` as a compute-without-write path in `configupgrade`
**Decision**: `internal/configupgrade` gains a `Check(path, kitVersion)` entry point that shares `Upgrade`'s render/validate computation but returns the would-change verdict and report without writing; the cobra verb maps would-change to a non-zero exit.
**Why**: `Upgrade` already isolates computation (`render` + `validateYAML`) from the atomic write; reusing it guarantees `--check` can never disagree with a real run about what would change.
**Rejected**: A separate diff/preview implementation — a second opinion on drift is a second source of truth.
*Introduced by*: 260809-wll4-config-source-consolidation

## Tasks

### Phase 1: Core Implementation

- [x] T001 Fold dispatch defaults into `src/go/fab/internal/agent/defaults.yaml` (add the `dispatch: {mode: native, column_width: 35, reap_done: true}` block with a comment noting it is the single value source); in `src/go/fab/internal/config/config.go` convert the three `DefaultDispatch*` constants to package-level vars (no literals) with a comment documenting the injection contract; in `src/go/fab/internal/agent/agent.go` add the `init()` that assigns the parsed values into them; add the self-naming blank-import test file in `src/go/fab/internal/config/` linking agent into the config test binary; add pin tests in `src/go/fab/internal/agent/defaults_test.go` (or agent_test.go) guarding the dispatch block values and the injection wiring, named to name themselves on failure <!-- R1, R3 -->
- [x] T002 Sweep `src/go/fab/internal/configref/configref.go` (registry rows ~535–565, dispatchSegment ~875–920, header comment ~27), `src/go/fab/cmd/fab/dispatch_reap.go`, and all test files referencing `DefaultDispatch*` (`src/go/fab/internal/config/config_test.go`, `src/go/fab/cmd/fab/config_test.go`, `src/go/fab/cmd/fab/dispatch_open_test.go`, `src/go/fab/internal/configupgrade/configupgrade_test.go`) so every dispatch-default read traces to the defaults.yaml source and stale comments describing the constants are rewritten; grep repo-wide for remaining independent literals of the three values <!-- R2 -->

### Phase 2: fab-kit Stub Retirement

- [x] T003 [P] In `src/go/fab-kit/internal/init.go` replace the stub fallback in `generateProjectConfig` with a clear non-zero error instructing the user to upgrade fab-go; delete `stubConfigHeader`, `renderStubConfig`, and `writeStubConfig` and any now-unused imports; update `src/go/fab-kit/internal/init_test.go` stub-fallback cases to error-path cases <!-- R4 -->

### Phase 3: Config Drift Probe

- [x] T004 Add `Check(path, kitVersion) (Result, error)` to `src/go/fab/internal/configupgrade/configupgrade.go` sharing `Upgrade`'s compute path (render + validateYAML) with zero writes — would-change (including missing file) reported via `Result.Changed` plus the report lines; add focused tests in `src/go/fab/internal/configupgrade/configupgrade_test.go` (drifted stamp / unparked key / missing fence / missing file / clean) <!-- R5 -->
- [x] T005 Wire `--check` onto `configUpgradeCmd` in `src/go/fab/cmd/fab/config.go`: zero writes, non-zero exit naming what would change when drifted, exit 0 "already up to date" when clean; update the verb's Long help; add tests in `src/go/fab/cmd/fab/config_test.go` including a file-hash-unchanged assertion <!-- R5 -->

### Phase 4: Fence Renderer Rework (items 4+7)

- [x] T006 Rewrite the configref segment renderers in `src/go/fab/internal/configref/configref.go` (`providersSegment`/`agentSegment`/`dispatchSegment`/`stageHooksSegment` + per-field segments) so each field advert carries a short description, a `fab config explain <key>` pointer, and a `[project|system|both]` scope tag from the row's configscope value, with `scope: both` adverts carrying the "settable machine-wide: `fab config set --system <key> <value>`" pointer; registry essays stay for `fab config explain`; move all golden/byte-stability tests once (`configref_test.go`, `configupgrade` `golden_test.go`/`freeze_test.go`/`configupgrade_test.go`, `cmd/fab/config_test.go` reference-text assertions) and verify `init --system`'s scaffold and the fence remain byte-stable and idempotent <!-- R6, R7 -->

### Phase 5: Init Verb Flags

- [x] T007 Add `--print` and `--force` to `configInitCmd` in `src/go/fab/cmd/fab/config.go`: `--print` renders exactly what would be written to stdout with zero writes and composes with `--project`/bare and `--system` (existing file does not block it); `--force` replaces the overwrite refusal in both `runConfigInitProject` and `runConfigInitSystem` (refusal stays default); update help text (the "(no --force)" note) and add tests in `src/go/fab/cmd/fab/config_test.go` <!-- R8, R9 -->

### Phase 6: Polish — Docs & Standards

- [x] T008 Update `src/kit/skills/_cli-fab.md` § fab config for `--check`, `--print`, `--force` and the new fence-text shape; update `docs/specs/skills/SPEC-_cli-fab.md` in lockstep; sweep repo-wide restatements of the fab config surface (`src/go/fab/cmd/fab/skill.md`, `docs/site/skill.md`, and any grep hits for the old fence description) <!-- R10 -->
- [x] T009 [P] Update `docs/specs/config.md`: managed-fence section (scope annotations + short-line format + explain pointers), `upgrade --check`, `init --print`/`--force`, the single-values-file claim for the built-in tier, the embed census (defaults.yaml + configref + configscope + `src/kit/scaffold/`, zero stub copies), and the fab-kit stub retirement <!-- R11 -->
- [x] T010 Run the shll standards check (`shll standards`, reading each entry governing CLI surface/help output) against the final `--check`/`--print`/`--force` naming and help text; fix or record any deviation <!-- R12 -->

## Execution Order

- T001 blocks T002 (the seam T002 sweeps consumers against)
- T004 blocks T005 (engine before verb wiring)
- T003 is independent (separate Go module `src/go/fab-kit`) — can run alongside Phase 1
- T006 is independent of T001–T005 in code, but shares golden test files with T004/T005's expectations — run after T005 to keep test edits in one pass
- T007 follows T005 (same cobra file, adjacent verbs)
- T008–T010 run last, against the final shipped text

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` carries the `dispatch:` block; `config.go` carries no dispatch-default literals; `GetDispatchMode`/`GetDispatchColumnWidth`/`GetDispatchReapDone` return the defaults.yaml-sourced values for nil/absent/invalid/out-of-range inputs with unchanged warning text
- [x] A-002 R2: configref registry rows and rendered segments, and `cmd/fab` consumers, all read dispatch defaults from the single source; a repo-wide grep finds no second literal
- [x] A-003 R3: pin/wiring tests exist and fail with self-naming output when the dispatch block or the seam wiring is broken (demonstrated by a temporary deliberate break during apply)
- [x] A-004 R4: fab-kit's init fallback is a non-zero error naming the fab-go upgrade path; `stubConfigHeader`/`renderStubConfig`/`writeStubConfig` are gone; no config content is embedded in the fab-kit binary
- [x] A-005 R5: `fab config upgrade --check` exits non-zero with named drift and zero writes on a drifted file; exits 0 on a clean file
- [x] A-006 R6: every fence/scaffold field advert carries its `[project|system|both]` scope tag, and `scope: both` adverts carry the `fab config set --system <key> <value>` pointer
- [x] A-007 R7: fence and system-scaffold field adverts are short description + `fab config explain <key>` pointer; `fab config explain` retains the full prose
- [x] A-008 R8: `fab config init --print` emits exactly the bytes the applying run would write, composing with bare/`--project` and `--system`, writing nothing
- [x] A-009 R9: `fab config init --force` overwrites an existing target in both modes; without it the refusal is unchanged
- [x] A-010 R10: `_cli-fab.md` and `SPEC-_cli-fab.md` document the new flags and fence shape in lockstep with the shipped CLI
- [x] A-011 R11: `docs/specs/config.md` matches the shipped fence format, verbs, single-source claim, and embed census
- [x] A-012 R12: the shll standards check was run against the final flag names/help text and conforms (or deviations are recorded in ## Notes)

### Behavioral Correctness

- [x] A-013 R1: `go test ./src/go/fab/...` and `go test ./src/go/fab-kit/...` pass with accessor semantics byte-for-byte preserved (all pre-existing dispatch accessor tests green against the injected source)
- [x] A-014 R5: `fab config upgrade` without `--check` is behaviorally unchanged (writes, stamps, idempotent)
- [x] A-015 R7: regenerated fence/scaffold output is byte-stable across runs and `fab config upgrade` on an already-regenerated file is a no-op

### Scenario Coverage

- [x] A-016 R5: tests cover stale fence stamp, unparked unknown key, missing fence, and clean file under `--check`, asserting both exit code and no-write (content/hash unchanged)
- [x] A-017 R8: tests assert `--print` output equals the corresponding written file byte-for-byte for both modes, and that an existing file does not block `--print`
- [x] A-018 R9: tests assert `--force` overwrites in both modes and default refusal is preserved
- [x] A-019 R4: init_test.go error-path case asserts the non-zero exit and the upgrade-fab-go message when the shell-out cannot produce a config

### Edge Cases & Error Handling

- [x] A-020 R5: `--check` on a repo with no config.yaml exits non-zero (a real run would create the file) and creates nothing
- [x] A-021 R1: a Go test binary that does not link agent is prevented from silently reading zero defaults — the blank-import link test in `internal/config` plus the agent wiring guard cover this
- [x] A-022 R8/R9: `--print` combined with `--force` is documented and tested as a pure preview (print wins, zero writes)

### Code Quality

- [x] A-023 Pattern consistency: new code follows the surrounding Go naming/comment idioms (nil-safe accessors, doc-comment style, self-naming pin tests)
- [x] A-024 No unnecessary duplication: the single-value-source goal is met without introducing a second copy of any default or prose; existing helpers (`CommentOutSegment`, render path) are reused
- [x] A-025 Readability over cleverness: functions stay focused (no >50-line god functions), no magic strings/numbers without named constants
- [x] A-026 Go changes ship tests: every touched `.go` file has accompanying test updates, scoped to the affected packages first (per `fab/project/code-quality.md` test strategy)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- `docs/memory/` updates are intentionally absent from ## Tasks — they are the hydrate stage's job (intake § Affected Memory).
- T010 standards record (A-012): checked against `shll standards` — `principles` (№1 non-interactive flag-based consent: `--force` is explicit consent, refusal names the remedy; №4 exit-code convention: drift = exit 1 operational failure, not 2; №5 mutation boundaries: `--print` is an accurate preview sharing the render path) and `help-dump` (verified from a source build: exit 0, stdout-only valid JSON, `tool: fab`/`schema_version: 1`, hidden nodes filtered, and the `fab config upgrade`/`fab config init` nodes carry the new flags in their `text`). No deviations.

## Deletion Candidates

- `src/kit/skills/fab-setup.md:146` — the Create-mode condition "The file is a placeholder generation containing `My Project` (the embedded-stub fallback name)" names the retired stub: post-retirement nothing produces a live `My Project` placeholder (`renderInitPreamble` omits the `project:` block when the seed name is empty), so the condition is dead or mis-attributed — re-ground or drop it (surfaced as a should-fix finding; not deleted by review).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Go seam for R1: constants become package-level vars in `internal/config` (same exported names, no literals), assigned by an `internal/agent` `init()` from the parsed `defaults.yaml`; zero-link hazard closed by a blank-import test file + wiring guard test | Cycle (agent→config) forces push-from-agent; keeping exported names avoids repo-wide call-site churn; Go init-order guarantees values before runtime use; graded against the intake's Tentative #4 which left the seam to apply | S:60 R:70 A:65 D:55 |
| 2 | Confident | Exact fence-annotation byte layout (scope-tag placement, pointer wording) is tuned against golden tests during T006 — semantics fixed by R6/R7, byte layout free | Intake assumption #5: plan fixes the semantic, not the layout; single renderer, easily tuned | S:65 R:70 A:70 D:60 |
| 3 | Confident | `--check` maps would-change to exit code 1 (drift found), not a usage-error code; output reuses the upgrade report lines prefixed as advisory | Matches CI-probe conventions (lint-style non-zero on findings); the engine already produces the report lines | S:60 R:75 A:70 D:60 |
| 4 | Confident | `--print` is never blocked by an existing target file, and `--print` + `--force` is a pure preview (print wins, zero writes) | A preview probe is useless if the common "file already exists" state errors it out; zero-write is the flag's defining property | S:60 R:70 A:65 D:55 |
| 5 | Confident | The stub-retirement error message names the cause (installed fab-go cannot generate config.yaml) and the remedy (upgrade fab-go), and the shell-out failure no longer writes anything | Plan/intake fix the semantic ("clear error telling the user to upgrade fab-go"); wording tuned at implementation | S:70 R:75 A:75 D:65 |

5 assumptions (0 certain, 5 confident, 0 tentative).
