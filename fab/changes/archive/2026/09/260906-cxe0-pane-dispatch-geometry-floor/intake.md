# Intake: Pane Dispatch Geometry Floor

**Change**: 260906-cxe0-pane-dispatch-geometry-floor
**Created**: 2026-09-06

## Origin

One-shot operator task spec (`OPERATOR_TASK_SPEC.md` in this worktree), driven autonomously via `/fab-new` → `/fab-fff`.

> Pane dispatch geometry floor — never carve a worker column too narrow for an agent TUI.
>
> Observed 2026-09-06, window `23:riff-racing-saola` on `runKit`: pane-mode dispatch split the orchestrator's own pane with `split-window -h -l 35%`. The window was 127x16 because tmux sizes a window to its most recent viewing client (window-size latest) and the user was viewing from a phone. The kimi worker landed in a 54x14 pane, its composer reflowed to ~17 columns, and `rk mux await --ready` reported `parked` (the sentinel wrapped inside the bordered composer, so no echo). The orchestrator burned three round trips diagnosing, then improvised: moved the worker to its own window and `resize-window` to 190x44, which worked.

The spec enumerates the wanted behavior in five numbered points (geometry read before ShapeSplit, floor-gated fall to ShapeWindow with manual sizing, a reason-naming warning, byte-for-byte preservation when geometry is fine, tests + docs), reproduced in full under What Changes.

## Why

1. **The pain point**: `SplitTarget` carves the worker column at `dispatch.column_width` percent with no knowledge of the window's absolute size. tmux sizes a window to its most recent viewing client (`window-size latest`), so a phone-attached viewer shrinks the orchestrator's window to e.g. 127x16 — and a 35% carve of that yields a 44-column pane no agent TUI can run in. The worker boots, its composer reflows to ~17 columns, the readiness sentinel wraps inside the bordered composer, and `rk mux await --ready` reports `parked`. Nothing is actually wrong with the worker; the geometry alone dead-ends the dispatch.
2. **The consequence if unfixed**: every pane dispatch from a viewer-shrunk window burns orchestrator round trips diagnosing a phantom `parked`, and the recovery (move to own window, `resize-window`) is manual improvisation. The existing `tmux rejected the sized split` retry path never fires — tmux happily grants a 44-column split; it is the *agent TUI* that cannot live in one.
3. **Why this approach**: the split-vs-window decision already exists (`SelectPaneShape` → `ShapeSplit`/`ShapeWindow`), and the fallback shape already exists (`OpenWindow`). The fix is to feed the existing decision the one input it is missing — the dispatcher's window geometry — and to make the fallback window immune to viewer sizing via `window-size manual` + `resize-window`. A detached (`-d`) manually-sized window is exactly the escape the operator improvised by hand; automating it inside the existing shape ladder keeps every other behavior byte-for-byte.

## What Changes

### 1. Geometry-floor decision (internal/dispatch/pane_mode.go)

Before the split shape is used, read the dispatching pane's window geometry — `#{window_width}` / `#{window_height}` of the dispatcher's window (a `tmux display-message -p -t <dispatcherPane>` format query, going through `internal/pane`'s shared `RunCmd`/`WithServer` helpers like every other tmux invocation). Compute the pane the planned split would yield:

- **Carving split** (`-h` off the dispatcher, the no-sibling case): worker width = `window_width * column_width / 100`, worker height = `window_height`.
- **Stacking split** (`-v` under an existing sibling, inside the column): worker width = the existing column's width (the sibling pane's `#{pane_width}`), worker height = the sibling's `#{pane_height}` halved (tmux even-splits an unsized `-v`).

If the resulting worker pane would be below the floor (`dispatch.min_cols` / `dispatch.min_rows`, default **80 cols x 20 rows**), do NOT split: fall to `ShapeWindow` with manual sizing (change area 2). The decision itself (geometry + planned placement + floor → shape) is a **pure function** — table-testable with no tmux probe, matching `SelectMode`/`SelectPaneShape`/`splitPlacement`'s shape in the package; the cobra layer (or a thin probe helper) supplies the measured numbers.

Geometry-probe failure is **fail-open**: placement machinery must never fail a dispatch that would otherwise launch (the package's existing warn-only posture), so an unreadable geometry proceeds with today's split behavior byte-for-byte, at most warning.

### 2. Manually-sized window fallback (internal/pane/create.go)

When the floor rejects the split, the worker opens as a window with a **manual size**, the escape from viewer-sized windows:

```
tmux new-window -d ... -P -F '#{pane_id}' -n <title> -c <dir> <cmd>
tmux set-option -w -t <win> window-size manual
tmux resize-window -t <win> -x 200 -y 50
```

- `-d` (detached) — do not switch the user's view to the new window.
- The size is a Go **constant** (proposed 200x50), not config — no demonstrated need to tune it, and the floor pair already gives users the policy knob. Constant named per code-quality's no-magic-numbers rule.
- Sizing failures after a successful `new-window` are **non-fatal warnings** (same posture as the title-set failure in `OpenSplitPane`): the worker is already running and identified.
- The existing `OpenWindow` callers (ShapeWindow via `--server`, via missing `$TMUX_PANE`, operator spawns) keep today's behavior **byte-for-byte** — manual sizing applies only to the floor-triggered fallback. Implementation shape: a new sized-window creator (or an options parameter) in `internal/pane`, leaving `OpenWindow`'s existing signature/behavior untouched for existing callers.

### 3. Warning naming the reason (cmd/fab dispatch launch path)

When the floor demotes a split to a window, emit a warning to stderr naming the measured geometry, the computed column, the floor, and the action:

```
window 127x16 too narrow for a 35% column (44 cols < 80): opening worker in its own window
```

(The stacked-split variant names the failing dimension analogously, e.g. rows.) This rides the same `warning:` stderr convention as the existing placement warnings, so the orchestrator sees why the layout differs from the requested shape.

### 4. Config plumbing (config.go, defaults.yaml, configref)

New keys `dispatch.min_cols` / `dispatch.min_rows`:

- `src/go/fab/defaults.yaml` `dispatch:` block gains `min_cols: 80` / `min_rows: 20` — the single value source, init-pushed into `internal/config`'s `DefaultDispatchMinCols`/`DefaultDispatchMinRows` exported symbols exactly like `DefaultDispatchColumnWidth` (the existing pins `TestDefaultsFileDispatchBlockIsPinned` / `TestConfigDispatchDefaultsMatchDefaultsFile` extend to the new pair).
- `internal/config` `DispatchConfig` gains `MinCols`/`MinRows int` yaml fields with nil-safe validated accessors `GetDispatchMinCols()`/`GetDispatchMinRows()` (absent/zero/negative → default, matching `GetDispatchColumnWidth`'s posture).
- **Scope `both`** (machine-wide settable via `fab config set --system`), matching the other three dispatch keys.
- `internal/configref` registry rows for both keys under the `dispatch.mode` owner segment, so `fab config explain`, the fenced reference block in config.yaml, `--json`, and env-var override (`FAB_DISPATCH_MIN_COLS`/`FAB_DISPATCH_MIN_ROWS`, per the existing `dispatch.column_width` → `FAB_DISPATCH_COLUMN_WIDTH` convention) all pick them up.
- The `fab config explain dispatch.column_width` prose is updated to mention the floor (a too-narrow computed column falls back to a manually-sized window).

### 5. Tests

- Pure decision: geometry (window WxH, planned placement, floor) → shape, table-driven in `internal/dispatch` — including at-floor boundary (80 exactly passes), just-below, height-limited stacking, and the fail-open probe-error case.
- Config plumbing: accessor validation rows (absent, zero, negative, valid), defaults.yaml pin extension, configref registry/fence/JSON guards, env-var mapping row.
- Argv rendering of the manual-size window: the `new-window -d` argv, the `set-option -w window-size manual` and `resize-window -x -y` follow-ups, unit-tested without a tmux server (the `plainPaneArgs`/`splitArgs` precedent).

### 6. Docs

- `docs/memory/runtime/dispatch.md` — the "Split placement is a record-keyed stacked column" requirement gains the geometry floor: the carve/stack pre-check, the floor keys and defaults, the manually-sized detached window fallback and its constant, the warning format, and the fail-open probe posture.
- `docs/memory/_shared/configuration.md` — the `dispatch.*` config docs gain `min_cols`/`min_rows` (defaults, scope `both`, env vars).
- `docs/memory/runtime/pane-commands.md` — the shared-pane-package section notes the sized-window creation mechanics if its prose enumerates `create.go`'s creators.
- `fab config explain` prose per change area 4.

## Affected Memory

- `runtime/dispatch`: (modify) split-placement requirement gains the geometry floor, the manually-sized window fallback, and the demotion warning
- `_shared/configuration`: (modify) `dispatch.min_cols` / `dispatch.min_rows` keys — defaults 80/20, scope both, env overrides
- `runtime/pane-commands`: (modify) shared pane package creator list gains the manual-size window mechanics (only if its prose enumerates the creators)

## Impact

- `src/go/fab/internal/dispatch/pane_mode.go` — geometry floor decision (new pure function), wiring into the shape/placement path
- `src/go/fab/internal/pane/create.go` — manually-sized detached window creator; existing `OpenWindow`/`OpenSplitPane` untouched for existing callers
- `src/go/fab/cmd/fab/dispatch_start.go` — geometry probe at the cobra layer (where `$TMUX_PANE` and config already live), floor demotion warning, threading `min_cols`/`min_rows` through `paneTarget`
- `src/go/fab/internal/config/config.go` — `DispatchConfig` fields, default symbols, accessors
- `src/go/fab/defaults.yaml` — `dispatch.min_cols: 80` / `min_rows: 20`
- `src/go/fab/internal/configref/configref.go` — registry rows + explain/fence prose
- Tests across `internal/dispatch`, `internal/pane`, `internal/config`, `internal/configref`, `internal/agent` (defaults pins)
- Docs: `docs/memory/runtime/dispatch.md`, `docs/memory/_shared/configuration.md`, `docs/memory/runtime/pane-commands.md`
- No migration needed: both keys are additive with built-in defaults (absent key = default, the existing dispatch-block pattern)

Out of scope (per the task spec): the readiness probe itself — a parallel run-kit session is adding a `narrow %N (WxH)` verdict to `rk mux await --ready` and box-drawing-tolerant sentinel matching. fab may later branch on `narrow`, but this change does not wait for it.

## Open Questions

- None — the task spec is unusually concrete (exact files, exact defaults, exact warning wording, explicit out-of-scope).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Floor defaults 80 cols x 20 rows via `dispatch.min_cols`/`min_rows`, scope both, in defaults.yaml + config.go + fenced reference | Spec states values, key names, and "scope both" explicitly | S:95 R:90 A:95 D:95 |
| 2 | Confident | Manual window size is a Go constant 200x50, not config | Spec proposes exactly this ("make it a constant, not config, unless you see a reason") — no reason found; the floor pair is the policy knob | S:85 R:85 A:80 D:75 |
| 3 | Certain | Geometry read = `#{window_width}`/`#{window_height}` of the dispatching pane's window; carve column = `window_width * column_width / 100`; stack = existing column width + halved sibling height | Spec dictates the formats and both formulas | S:95 R:85 A:90 D:90 |
| 4 | Confident | Floor decision is a pure function in `internal/dispatch` with the tmux geometry read at the cobra layer / probe helper | Not spec-dictated, but `SelectMode`/`SelectPaneShape`/`splitPlacement` purity is the package's stated, tested pattern | S:70 R:80 A:90 D:80 |
| 5 | Confident | Geometry-probe failure is fail-open to today's split (warn at most) | Package posture is explicit: placement is cosmetic and must never fail an otherwise-launchable dispatch | S:60 R:80 A:85 D:75 |
| 6 | Certain | Existing `ShapeWindow` paths and fine-geometry splits stay byte-for-byte; the sized-split retry path stays | Spec requirement 4 verbatim | S:95 R:90 A:95 D:95 |
| 7 | Confident | Fallback window is created detached (`-d`) with `set-option -w window-size manual` + `resize-window -x 200 -y 50`, sizing failures non-fatal | Spec gives the command sequence incl. `-d`; non-fatal posture matches `OpenSplitPane`'s title-set precedent | S:85 R:80 A:85 D:80 |
| 8 | Confident | Memory targets are `runtime/dispatch.md` + `_shared/configuration.md` (+ `pane-commands.md` if it enumerates creators) — the spec's "pane-commands.md (or wherever pane dispatch lives)" resolves to dispatch.md | dispatch.md § split-placement requirement is where the column invariant actually lives; configuration.md owns the dispatch.* key docs | S:70 R:90 A:85 D:80 |
| 9 | Confident | Env overrides `FAB_DISPATCH_MIN_COLS`/`FAB_DISPATCH_MIN_ROWS` come with the configref rows | The registry mechanically derives env vars for dispatch.* keys (`FAB_DISPATCH_COLUMN_WIDTH` precedent); omitting them would be a gap, not a choice | S:55 R:85 A:85 D:80 |

9 assumptions (3 certain, 6 confident, 0 tentative, 0 unresolved).
