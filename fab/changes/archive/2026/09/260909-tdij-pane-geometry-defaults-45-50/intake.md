# Intake: Pane-Dispatch Geometry Defaults 45/50

**Change**: 260909-tdij-pane-geometry-defaults-45-50
**Created**: 2026-09-09

## Origin

Conversational — dispatched by `/fab-proceed` (promptless-defer) from a user discussion that had already settled the decision. The user's raw request, synthesized:

> Lower the pane-dispatch geometry defaults so 120-column tmux windows enter pane mode. With the shipped defaults (`column_width: 40`, `min_cols: 60`, set yesterday in PR #654, commit `614cb822`), a 120-col window yields a 48-col worker column, which is below the 60-col floor, so the launch demotes to a manually-sized detached window. I want 120-col windows to split. Change the defaults to `column_width: 45` and `min_cols: 50`. Third requirement: very little change in the rest of the source code.

Key decisions reached in the conversation (all user-confirmed):

1. **Pure defaults bump** — `column_width: 40 → 45`, `min_cols: 60 → 50` in `src/go/fab/defaults.yaml`. `min_rows: 20`, `mode: native`, `reap_done: true` unchanged.
2. **Accepted consequences** — at 120 cols the worker gets 54 cols (≥ 50, passes) and the dispatcher keeps 66 cols (55%). The smallest window that still splits is **112 cols** (112 × 45 / 100 = 50, exactly at the floor, which passes); 111 cols and below (111 × 45 / 100 = 49 < 50) demote to the manual window as today. The user explicitly accepted 112 as the new minimum and 50 cols as the acceptable worker floor.
3. **Alternative rejected** — a code change making the carve fall back to an absolute `-l <min_cols>` column (`max(pct, min_cols)`, demoting only when `window < 2 × min_cols`). Rejected because it changes the formula and the split argv; the user wants "very little change in the rest of the source code". No new config keys, no formula change, no migration.
4. **Footprint** — follow PR #654 (`git show 614cb822 --stat`) exactly: it did the previous bump (35 → 40, 80 → 60) across 17 files, and this change touches the same class of sites with the new values.

## Why

**The pain point.** `fab dispatch open` carves the pane-mode worker column as `window_width × dispatch.column_width / 100` and, when that lands below `dispatch.min_cols`, demotes to a manually-sized detached window (`pane.OpenManualWindow`, 200×50). The floor exists for viewer-shrunk windows (phone attachments), but at the shipped 40/60 it also catches the ordinary **120-column** tmux window — a very common terminal width — because 120 × 40 / 100 = 48 < 60. So on a 120-col window every stage worker opens in a separate window instead of beside the dispatching agent, which defeats the two-tier "workers as panes beside the agent you are watching" design.

**If we don't fix it.** Users on 120-col windows never see pane mode work out of the box; they must discover and override two `dispatch.*` knobs machine-wide to get the documented default behaviour. #654 already moved the numbers once (35/80 → 40/60) for the same reason and stopped one step short of the 120-col case.

**Why this approach.** Two defaults already govern the outcome, so the fix is entirely within the existing policy knobs — no Go logic, no new keys, no migration (the values are advertised in the config fence as commented-out defaults; a user who overrode them is untouched). The alternative (`max(pct, min_cols)` fallback) would change the carve formula that `SplitBelowFloor`, `splitArgs`, the docs, and the tests all describe, for a marginal gain over simply choosing better defaults. 45/50 is the smallest move that admits 120-col windows with a worker column (54 cols) still comfortably above the 50-col floor the user judged acceptable.

## What Changes

### 1. The single value source — `src/go/fab/defaults.yaml`

```yaml
dispatch:
  mode: native
  column_width: 45   # was 40
  min_cols: 50       # was 60
  min_rows: 20
  reap_done: true
```

This is the **only** place the values are defined (`config.DefaultDispatch*` are init-injected vars carrying no literals — see `docs/memory/_shared/configuration.md` § "Dispatch Defaults Are Init-Injected from `defaults.yaml`"). Everything below is a pin, a rendering, a comment, or a doc restatement of these two numbers.

### 2. ZERO Go logic change

`src/go/fab/internal/dispatch/pane_mode.go` (`SplitBelowFloor`, `plannedWorkerSize`, `splitPlacement`) and `src/go/fab/internal/pane/create.go`'s behaviour (`splitArgs`, `OpenManualWindow`, the `ManualWindowCols/ManualWindowRows` 200×50 constants) stay **byte-for-byte** identical except for the one comment noted in §4. The formula `cols = window_width × columnWidth / 100` (integer division), "a dimension exactly at the floor passes", "width is named when both dimensions fail", and the fail-open probe are untouched.

Resulting behaviour with the new defaults (all computed with Go integer division):

| Window | Worker cols (× 45 / 100) | vs floor 50 | Outcome |
|--------|--------------------------|-------------|---------|
| 120×40 | 54 | ≥ 50 | **splits**; dispatcher keeps 66 cols (55%) |
| 112×40 | 50 | = 50 | splits (exactly at the floor passes) — the new minimum |
| 111×40 | 49 | < 50 | demotes: `window 111x40 too narrow for a 45% column (49 cols < 50): opening worker in its own window` |
| 100×16 | 45 | < 50 | demotes (width named even though rows 16 < 20 also fail) |
| 80×24 | 36 | < 50 | demotes: `window 80x24 too narrow for a 45% column (36 cols < 50): opening worker in its own window` |

### 3. Pinned drift-guard tests (values only, `go test ./...` under `src/go/fab` must pass)

- `src/go/fab/internal/agent/defaults_test.go` — `TestDefaultsFileDispatchBlockIsPinned`: the two pins `cfg.Dispatch.ColumnWidth != 40` → `45` and `cfg.Dispatch.MinCols != 60` → `50` (each appears twice per line: the comparison and the `%d` pinned argument). `TestConfigDispatchDefaultsMatchDefaultsFile` compares the injected symbols to the parsed file with no literals — it passes unchanged and is the wiring guard.
- `src/go/fab/cmd/fab/dispatch_open_test.go` — `TestDispatchOpen_BelowFloorDemotesToSizedWindow`: the doc comment ("would carve a 32-column worker at the default 40% — below the 60x20 floor") becomes "a 36-column worker at the default 45% — below the 50x20 floor", and `wantWarning` becomes exactly `window 80x24 too narrow for a 45% column (36 cols < 50): opening worker in its own window` (80 × 45 / 100 = 36).
- `src/go/fab/cmd/fab/setup_test.go` — `TestSetupWizard_AdvancedOptInAsksAllKeys`: the expected prompt `dispatch.column_width [40]:` → `dispatch.column_width [45]:`.
- `src/go/fab/internal/config/config_test.go` — `TestCascade_DispatchColumnWidthFromSystemLayer` writes a **project** `column_width: 45` to prove the system layer's 25 outranks it. #654 moved that literal 40 → 45 precisely because 40 had become the default; now 45 is the default, so move it again to a non-default value (e.g. `35`) and update the paired message `want the system's 25 to beat the project's 35`. `TestCascade_DispatchMinColsFromSystemLayer`'s project `min_cols: 60` stays distinct from the new default 50 — leave it.
- **Not in the class** (leave untouched): `src/go/fab/internal/dispatch/dispatch_test.go`'s `SplitBelowFloor` table tests pass explicit `columnWidth`/`minCols` arguments (`35% … 44 cols < 80`, `40% … 80 cols < 81`, `carve of 127x16 at 35%`) — they exercise the pure function, not the defaults, and #654 correctly did not touch them.

### 4. Go source comments and descriptions (no behaviour)

- `src/go/fab/internal/configref/configref.go` — the `dispatch.column_width` row: comment "the cascade bottoms out at 40" → `45`; `Description` trailing "Default 40." → "Default 45.". The `dispatch.min_cols` row: `Description` trailing "Default 60." → "Default 50.". (These strings are rendered by `fab config explain`/`show`; they are prose, not the value source.)
- `src/go/fab/internal/pane/create.go` — the `splitArgs` doc comment example `` (`-l 40%`) `` → `` (`-l 45%`) ``.

### 5. New fence-advert digest — `src/go/fab/internal/configupgrade/configupgrade.go`

`knownGeneratedSystemParagraphDigests` is the append-only, byte-exact catalog of every paragraph `fab config init --system` has ever rendered. The `dispatch` advert paragraph embeds the default values, so changing them produces a new paragraph identity; `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` fails until it is appended, printing the exact missing digest:

```
current generated paragraph "# dispatch.mode / dispatch.column_width / …" is missing digest <sha256> from the historical catalog
```

Procedure (same pattern #654 used):
1. After editing `defaults.yaml`, run `go test ./internal/configupgrade -run TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` under `src/go/fab`; copy the reported digest.
2. Demote the existing comment `// Current dispatch advert with the raised column_width 40 / lowered min_cols 60 defaults.` to historical wording (drop "Current", e.g. `// Dispatch advert with the column_width 40 / min_cols 60 defaults (PR #654).`) — #654 demoted the prior `260906-cxe0` comment the same way.
3. Append the new entry directly after it: `// Current dispatch advert with the column_width 45 / min_cols 50 defaults (260909-tdij).` plus the digest line. **Never remove an older digest** — older system files must still be recognised as generated.

### 6. Canonical skill sources (`src/kit/skills/`, never the deployed copies)

- `src/kit/skills/_cli-fab.md` § fab dispatch open (the two-tier tmux hierarchy paragraph, ~line 722): "`dispatch.column_width` (default 40)" → `(default 45)`; "`dispatch.min_cols`/`dispatch.min_rows` (defaults 60/20)" → `(defaults 50/20)`.
- `src/kit/skills/_preamble.md` § Always Load, the `fab/project/config.yaml` bullet (~line 54): "the split geometry floor; defaults `60`/`20`" → "defaults `50`/`20`".

`.agents/skills/` and `.claude/skills/` are regenerated by `fab sync` and are not edited (Constitution V, code-quality.md anti-pattern).

### 7. Memory restatements (`docs/memory/`)

`docs/memory/runtime/dispatch.md` § Split placement:
- "carving the Left/Right column at `dispatch.column_width` percent (default 40, …)" → `(default 45, …)`.
- "defaults **60 cols × 20 rows**" → "**50 cols × 20 rows**".
- The example warning string in that paragraph must be **recomputed**, not just re-numbered: the current example `window 127x16 too narrow for a 40% column (50 cols < 60)` would be arithmetically false at 45/50 (127 × 45 / 100 = 57 ≥ 50 passes the width check). Replace with a window that does demote on width: `warning: window 100x16 too narrow for a 45% column (45 cols < 50): opening worker in its own window` (100 × 45 / 100 = 45). Keeping the height at 16 preserves the viewer-shrunk narrative and the "both dimensions fail, width is named" property of the original example.
- Scenario "the carving split is sized and the stacking split is not": "`dispatch.column_width` resolving to 40" → `45`; "the split argv carries `-h -l 40%` and the dispatcher keeps ~60%" → "`-h -l 45%` … keeps ~55%".
- Scenario "a below-floor split demotes to a manually-sized window": GIVEN "a viewer-shrunk 127x16 window, `dispatch.column_width` 40, floor 60x20" → "a viewer-shrunk 100x16 window, `dispatch.column_width` 45, floor 50x20"; THEN warning → `warning: window 100x16 too narrow for a 45% column (45 cols < 50): opening worker in its own window`.

`docs/memory/_shared/configuration.md` § `dispatch`:
- `column_width` bullet "default `40`" → "`45`"; `min_cols`/`min_rows` bullet "defaults `60` / `20`" → "`50` / `20`".
- § Design Decisions → "Dispatch Scalars Carry Their Typed Defaults" and "Dispatch Defaults Are Init-Injected from `defaults.yaml`": the value lists `column_width: 40`, `min_cols: 60` → `45`, `50`. Hydrate MAY append `260909-tdij-pane-geometry-defaults-45-50` to their *Introduced by* lines and MAY record the 120-col rationale (decision 2 in Origin) as a one-line note; #654 itself only substituted values.

### 8. Spec restatements (`docs/specs/`, value-only — the design intent is unchanged)

- `docs/specs/architecture.md` config walkthrough: "`dispatch.column_width` (optional, default 40)" → `45`; "`dispatch.min_cols` / `dispatch.min_rows` (optional, defaults 60/20)" → `50/20`; the YAML block `column_width: 40` / `min_cols: 60` → `45` / `50`.
- `docs/specs/config.md`: the two enumerations of the five dispatch defaults (`column_width: 40`, `min_cols: 60` and "`dispatch.column_width`'s `40`, `dispatch.min_cols`' `60`") → `45` / `50`.
- `docs/specs/glossary.md`: the `dispatch.min_cols` / `dispatch.min_rows` row "defaults `60`/`20`" → "`50`/`20`".
- `docs/specs/harness-adapters.md` § pane adapter: "carving the column at `dispatch.column_width` (default 40)" → `45`; "geometry floor (defaults 60/20)" → `50/20`.

### 9. The regenerated project fence — `fab/project/config.yaml`

The commented dispatch advert inside the `>>> fab reference (kit 2.24.2) >>>` fence must read `#   column_width: 45` / `#   min_cols: 50`. #654 changed exactly those two lines and nothing else in the file. Regenerate with `fab config upgrade` run from a **locally built** binary (`go build ./cmd/fab` under `src/go/fab`, then that binary's `config upgrade`) — the installed `fab` (2.24.3 on this machine) renders its own embedded defaults and would not reflect the edit. If the locally built binary's run changes anything beyond those two value lines (e.g. the fence header version), revert the extra noise and keep only the two value lines — release commits own the header.

### 10. Sibling-sweep definition (code-quality.md § Sibling Sweeps)

The class was defined **up front** by a repo-wide grep (excluding `.git`, `.agents`, `.claude`, `fab/changes`, `docs/**/archive`) for `column_width: 40`, `40% column`, `-l 40%`, `(default 40)`, `Default 40.`, `min_cols: 60`, `60 cols`, `< 60)`, `Default 60.`, `defaults 60/20`, `` defaults `60` ``, `60x20`, `~60%`. Every hit is enumerated in §§1–9 (17 files, matching #654's footprint one-for-one). No hits in `README.md`, `docs/site/`, testdata/golden files, or `fab/backlog.md`. Before finishing apply, **re-run the same grep** and confirm the only remaining hits are the deliberately excluded explicit-argument `dispatch_test.go` cases and the historical `configupgrade.go` digest comment.

### Out of scope

- Any change to the floor formula, `SplitBelowFloor`/`plannedWorkerSize`/`splitArgs`, or the manual-window constants (200×50).
- `min_rows`, `mode`, `reap_done`, claude model defaults, or any other `defaults.yaml` value.
- The user's `~/.fab-kit/config.yaml` system file (its fence is regenerated by the user's own `fab config upgrade` after release).
- A migration — nothing restructures user data; the advert is commented-out reference text, and user overrides keep winning.
- A release/version bump — owned by the release process.

## Affected Memory

- `runtime/dispatch`: (modify) § Split placement — default column width 45, floor 50 × 20, recomputed `100x16 … (45 cols < 50)` example warning, the two GIVEN/WHEN/THEN scenarios (`-l 45%` / ~55%; 100x16 demotion)
- `_shared/configuration`: (modify) § `dispatch` bullets (`column_width` default `45`, `min_cols`/`min_rows` defaults `50` / `20`) and the value lists in the two Design Decisions entries ("Dispatch Scalars Carry Their Typed Defaults", "Dispatch Defaults Are Init-Injected from `defaults.yaml`")

## Impact

**Code (value/prose only, no logic):** `src/go/fab/defaults.yaml`; `src/go/fab/internal/configref/configref.go`; `src/go/fab/internal/configupgrade/configupgrade.go` (one appended digest + comment demotion); `src/go/fab/internal/pane/create.go` (comment).

**Tests:** `src/go/fab/internal/agent/defaults_test.go`, `src/go/fab/internal/config/config_test.go`, `src/go/fab/cmd/fab/dispatch_open_test.go`, `src/go/fab/cmd/fab/setup_test.go`. Verification: from `src/go/fab`, scope first to `go test ./internal/agent ./internal/config ./internal/configupgrade ./internal/dispatch ./internal/pane ./cmd/fab`, then `go test ./...`. Constitution VII: the tests are updated to the new spec values, not the implementation bent to the tests.

**Kit skills:** `src/kit/skills/_cli-fab.md`, `src/kit/skills/_preamble.md` (prose-only value restatements; no flow/tool/sub-agent change).

**Docs:** `docs/memory/runtime/dispatch.md`, `docs/memory/_shared/configuration.md`, `docs/specs/architecture.md`, `docs/specs/config.md`, `docs/specs/glossary.md`, `docs/specs/harness-adapters.md`.

**Generated:** `fab/project/config.yaml` fence (two commented value lines).

**Runtime effect:** only pane-mode dispatches on windows between 112 and 149 columns wide change outcome (they now split instead of demoting); 150+ cols already split at 40/60 (150 × 40 / 100 = 60) and still split at 45/50 (67). Windows below 112 cols demote as before. Users with explicit `dispatch.column_width`/`dispatch.min_cols` overrides are unaffected (presence = intent). Dispatchers on wide windows keep 55% instead of 60% of the width.

**Systems:** none beyond tmux geometry; no CLI signature change (so no `_cli-fab.md` command-surface update is required beyond the two default restatements); no migration.

## Open Questions

- None. The values, the rejected alternative, the accepted new minimum (112 cols), and the footprint were all confirmed by the user before dispatch.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New defaults are exactly `column_width: 45`, `min_cols: 50`; `min_rows`/`mode`/`reap_done` unchanged | User-confirmed values in discussion; single value source is `defaults.yaml` | S:95 R:90 A:95 D:95 |
| 2 | Certain | Zero Go logic change — value substitution and comments only; the `max(pct, min_cols)` fallback is rejected | Discussed — user rejected the code-change alternative in favour of a pure defaults bump ("very little change in the rest of the source code") | S:95 R:90 A:95 D:95 |
| 3 | Certain | New minimum splitting window is 112 cols; 111 and below demote; dispatcher keeps 55% | Arithmetic consequence of 45/50 with integer division, explicitly accepted by the user | S:90 R:85 A:100 D:95 |
| 4 | Certain | Touch the same 17-file class PR #654 (`614cb822`) touched, defined up front by repo-wide grep | User named the footprint; grep confirmed the class matches one-for-one with no extra hits | S:90 R:85 A:95 D:90 |
| 5 | Certain | `dispatch_test.go` explicit-argument `SplitBelowFloor` cases (35%/80, 40%/81, 127x16) stay untouched | They test the pure function with literal arguments, not the defaults; #654 precedent left them alone | S:85 R:90 A:95 D:90 |
| 6 | Certain | Recomputed doc example is a `100x16` window → `45 cols < 50` (replacing `127x16 … 50 cols < 60`, which would be false at 45/50) | Width-limited demotion needs `w × 45 / 100 < 50`; 100 is the round choice the description itself suggests and keeps the viewer-shrunk, both-dimensions-fail narrative | S:80 R:90 A:90 D:75 |
| 7 | Certain | Move `TestCascade_DispatchColumnWidthFromSystemLayer`'s project literal off the new default (45 → e.g. 35) so the test still discriminates project from built-in | Follows #654, which moved 40 → 45 when 40 became the default; low stakes, single line | S:70 R:95 A:90 D:80 |
| 8 | Certain | New configupgrade digest obtained by running `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` after the `defaults.yaml` edit; old "Current …" comment demoted to historical, no digest removed | Append-only catalog contract stated in the source; identical to #654's mechanics | S:85 R:90 A:90 D:90 |
| 9 | Certain | Regenerate the project fence with a locally built `fab` (or edit the two value lines to the identical renderer output); the fence header stays `kit 2.24.2` | #654's fence diff was exactly the two value lines; installed `fab` is 2.24.3 and renders its own embedded defaults | S:75 R:90 A:85 D:80 |
| 10 | Certain | Change type `chore` (defaults/config bump, no new capability) | Description says config/chore-class; flat 3.0 gate is type-independent so low stakes | S:70 R:95 A:85 D:80 |
| 11 | Certain | Hydrate updates the two memory files by value substitution; appending `260909-tdij` to the two Design Decisions *Introduced by* lines and a one-line 120-col rationale note is optional | #654 substituted values only; a traceability line is cheap and reversible | S:65 R:95 A:85 D:75 |

11 assumptions (11 certain, 0 confident, 0 tentative, 0 unresolved).
