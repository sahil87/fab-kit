# Intake: Pane-Mode Worker Column Invariant — Record-Based Sibling Detection + Sized First Split

**Change**: 260807-g4a5-pane-worker-column-invariant
**Created**: 2026-08-07

## Origin

Created by `/fab-proceed`'s promptless create-intake dispatch (`{questioning-mode} = promptless-defer` — no questions asked; would-be-asked decisions deferred per SRAD). The design was synthesized from a `/fab-discuss` conversation with the project owner on 2026-08-07 and is user-directed on all five core decisions (quoted in § What Changes).

> `fab dispatch` pane-mode stage workers are meant to form a stacked right-hand column beside the dispatching session agent. In practice the window degrades into N equal-width columns with the session agent squeezed (observed 2026-08-07: 7 equal columns, agent at ~14% width). Fix the placement rule: record-based sibling detection (dispatch records already persist pane IDs) instead of title probing (harnesses clobber pane titles), plus a sized first split (`-l 35%`, configurable) so the session agent keeps most of the window.

## Why

**Problem.** Pane-mode dispatch (`fab dispatch start`, auto-pane inside tmux) is designed to place stage workers in a stacked right-hand column: the first worker splits the dispatcher's pane `-h` (carving the column), later workers stack `-v` under the last live worker sibling. The placement decision lives in `SplitTarget` / `SiblingDispatchPane` in `src/go/fab/internal/dispatch/pane_mode.go`. In practice the layout degrades into N equal-width columns with the session agent squeezed to a sliver (observed 2026-08-07: 7 equal columns, dispatching agent at ~14% width) — the window becomes unreadable exactly where the user is supposed to be watching.

**Root cause.** `SiblingDispatchPane` keys sibling detection on pane TITLES: it lists the window's panes and keeps the last one whose `#{pane_title}` carries `DispatchTitlePrefix` (`"fab-"`). But harnesses running inside the worker pane (e.g. Claude Code) rewrite the pane title via terminal escape sequences, so the `fab-{id}-{stage}` title set at spawn is clobbered within seconds. Sibling detection then finds nothing, and every subsequent worker takes the "no sibling" branch — re-splitting the dispatcher `-h` and creating another full-height column.

**Secondary problem.** Even when stacking works, the first `-h` split is an even halving — the dispatcher keeps only ~50%. The user generally wants maximum area reserved for the session agent (the pane they actually watch); the worker column can be much narrower.

**Consequence of not fixing.** Every multi-stage pipeline run inside tmux progressively destroys the operator's working layout; the flagship "watchable dispatch" experience (`dispatch.watchable`, auto-pane in tmux) actively degrades the session it is supposed to enhance.

**Why this approach.** Every pane dispatch already persists its tmux pane ID in its dispatch record (`Dispatch.Pane` in `.fab-dispatch/{4-char-change-id}/{stage}.yaml`, written by `launchPane` in `src/go/fab/cmd/fab/dispatch_start.go`). Pane IDs are server-global, stable for the pane's lifetime, and immune to title clobbering — the same reason the record already keys `status`/`kill`/`capture` on pane ID rather than name. Placement should key on the same identity. Alternatives (layout enforcement via `select-layout`, run-kit-owned tmux hooks, a separate workers window) were considered and rejected in discussion — see § Non-Goals / Rejected Alternatives.

## What Changes

The five decisions below are user-directed (from the 2026-08-07 discussion); capture is faithful.

### 1. Column invariant

After the first worker's `-h` split creates **Left** (dispatcher) / **Right** (worker column), every later worker splits ONLY within the right column (`-v`). fab never touches the vertical Left/Right separator again:

- NO `select-layout`, no layout normalization.
- Never rearranges user-made panes; never fights the user's manual resizes.
- Only horizontal separators *inside* the right column are ever added or moved (by `-v` splits stacking workers).

The invariant is a *creation-time* rule, not an enforcement loop: fab only stops creating new mess; it never repairs an already-mangled layout (accepted limitation — existing multi-column windows stay as-is until those panes die).

### 2. Record-based sibling detection replaces title probing

The new-worker placement rule becomes:

1. Enumerate the checkout's dispatch records (`.fab-dispatch/{4-char-change-id}/{stage}.yaml` — the package already has `DirFor`/`Load`; a record-enumeration helper is added to `src/go/fab/internal/dispatch/`).
2. Filter to records with a recorded `Pane` whose pane is live (`PaneAlive`) — and located in the dispatcher's own window, since the column invariant is per-window (see Assumptions #8).
3. Split the **last** live worker pane `-v` (stacking the column). No live recorded sibling → first-worker case: split the dispatcher `-h` (sized, per #3 below).

Title clobbering becomes irrelevant to placement. **Titles are still set** at spawn for identification (`OpenSplitPane`'s `select-pane -T` call stays); only the placement probe changes. `SiblingDispatchPane`'s title-scan (`DispatchTitlePrefix` matching, `lastDispatchPane` parsing) is superseded by record enumeration; the split-direction constants' doc text and surrounding comments in `pane_mode.go` (which currently document the title-keyed rule) are updated to describe the record-keyed rule.

Cmd wiring: `SplitTarget` is currently called from `launchPane` (`src/go/fab/cmd/fab/dispatch_start.go:447`) with only `(server, dispatcherPane)` — the change context (repo root / change ID needed to find record dirs) must be passed into split-target resolution.

### 3. Size the column once, at the first split

- The first (column-carving) `-h` split runs `split-window -h -l 35%` (tmux ≥ 3.1 syntax), so the dispatcher keeps ~65% from the moment the column is carved.
- New config knob for the column width: **`dispatch.column_width`**, scope `both` (settable machine-wide in `~/.fab-kit/config.yaml`, like `dispatch.watchable`), default **35** (percent). Registered via the config field registry (`src/go/fab/internal/configref/configref.go` — a new Field row next to `dispatch.watchable` in the `dispatch:` segment) per `docs/specs/config.md`, so `fab config reference`/`--json`/`fab config upgrade`'s managed fence all pick it up. Accessor in `src/go/fab/internal/config/config.go` beside `GetDispatchWatchable`.
- Sizing applies wherever a column-carving `-h` split happens — including the degraded fallback path (a failed record probe still carves a sized column off the dispatcher).
- `-v` stacking splits stay unsized (tmux even-splits within the column; the Left/Right separator is untouched).

### 4. Placement stays cosmetic (graceful degradation)

Record-read or liveness-probe failures degrade gracefully to the current fallback — split the dispatcher `-h` — with a warn-only stderr notice, exactly the existing `SplitTarget`/`launchPane` degradation contract ("placement is cosmetic, so a failing probe must never fail a dispatch that would otherwise launch"). This includes a tmux too old for `-l N%` (pre-3.1): degrade rather than fail the dispatch.

### 5. `fab dispatch restart` uses the same rule

`restart` re-derives its mode/shape from the current environment (`resolveMode` in `dispatch_restart.go` already shares `dispatch_start.go`'s resolution). The restart path must use the same record-based placement rule — a restarted pane worker lands in the right column exactly as a fresh dispatch does. No restart-specific placement logic.

### 6. Tests + docs/spec/skill sweep (constitution-bound)

- **Go tests in the same change** (constitution Additional Constraints; code-quality § Test Strategy): table tests for the new record-based split-target resolution (pure parts stay pure — mirror `SelectMode`/`SelectPaneShape`/`lastDispatchPane`'s testable shape), the sized-split argv composition, config knob parsing/cascade (pattern: `dispatch.watchable` tests in `config_test.go`), and degradation paths.
- **`src/kit/skills/_cli-fab.md`** — update the `fab dispatch` section for changed placement behavior (and any changed flag/knob surface), plus its mirror `docs/specs/skills/SPEC-_cli-fab.md`.
- **Spec/skill mirrors restating the stacked-column placement rule** (code-quality § Sibling & Mirror Sweeps — sweep the whole class up front):
  - `docs/specs/harness-adapters.md` § 3 (interactive-pane adapter — the split/placement bullets, currently describing the `fab-`-title sibling rule and unsized `-h` split)
  - `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch pane-mode bullets (two-tier hierarchy / "splits the last existing `fab-`-titled sibling pane (`-v`) or ... the agent's own pane (`-h`)" text) + its mirror `docs/specs/skills/SPEC-_preamble.md`
  - `docs/specs/config.md` (new field row for `dispatch.column_width`)
  - `docs/memory/runtime/dispatch.md` (the runtime memory file documenting fab dispatch — placement rule + new knob), with `fab memory-index` regeneration
  - Grep repo-wide for the old claim phrases (`fab-`-titled sibling, title-keyed placement, even split) to catch aggregate restatements (`skills.md`, `glossary.md`, `architecture.md`).
- **No migration file**: the new config field is additive (absent ⇒ default 35); no user data is restructured. `fab config upgrade` regenerates the reference fence from the registry.

## Non-Goals / Rejected Alternatives

Rejected in discussion (do not resurrect during apply):

- **Enforcing `select-layout main-vertical` (deck-v) after every pane event** — rejected: rearranges the user's hand-made panes and resets manual resizes on every dispatch event.
- **run-kit-owned layout enforcement (tmux hooks / server poll)** — noted as a possible long-term direction, out of scope; does nothing on machines without run-kit.
- **Workers in a separate `fab-{id}-workers` window plus done-worker reaping** — rejected: loses side-by-side glanceability.
- **Done-worker pane hygiene + richer pane names/colors** — explicitly deferred to backlog idea `[zfl7]` (main worktree backlog). OUT of scope here.
- **Layout repair** — fab never repairs an already-mangled layout (existing multi-column windows stay as-is until those panes die); it only stops creating new mess.

## Affected Memory

- `runtime/dispatch`: (modify) pane-mode placement rule (record-based sibling detection, column invariant, sized first split), the `dispatch.column_width` knob, and the degradation contract

## Impact

- `src/go/fab/internal/dispatch/pane_mode.go` — `SiblingDispatchPane` / `lastDispatchPane` superseded by record-based enumeration; `SplitTarget` re-signatured to take change/record context; `SplitRight`/`SplitBelow` doc text and file-header placement comments updated; sized `-h` split composition
- `src/go/fab/internal/dispatch/` — new record-enumeration helper (list record dirs / load records / filter live panes)
- `src/go/fab/cmd/fab/dispatch_start.go` — `launchPane` wiring passes change context into split-target resolution; sized split; warn texts
- `src/go/fab/cmd/fab/dispatch_restart.go` — same placement path (shared resolution)
- `src/go/fab/internal/config/config.go` + `src/go/fab/internal/configref/configref.go` — `dispatch.column_width` field (struct, accessor, registry row, segment)
- Tests: `dispatch` package tests + `cmd/fab/dispatch_*_test.go` + `config_test.go`
- Docs/skills: `src/kit/skills/_cli-fab.md`, `src/kit/skills/_preamble.md`, `docs/specs/skills/SPEC-_cli-fab.md`, `docs/specs/skills/SPEC-_preamble.md`, `docs/specs/harness-adapters.md`, `docs/specs/config.md`, `docs/memory/runtime/dispatch.md` (+ index regen)
- No behavior change for headless dispatch, `ShapeWindow` fallback (new-window shape), or any non-tmux path

## Open Questions

- Should record enumeration scan only the active change's `.fab-dispatch/{id}/` dir, or all record dirs in the checkout? A window typically hosts one change's workers, but nothing enforces it. The discussion explicitly deferred this ("defer, don't ask"). Assumed below (Assumptions #10): scan all record dirs in the checkout, bounded by the liveness + same-window filter. <!-- assumed: enumeration scope = all .fab-dispatch/*/ record dirs in the checkout, filtered to live panes in the dispatcher's window — discussion deferred the scope; all-dirs is strictly more robust when one window hosts two changes' workers and costs one extra directory listing -->

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Column invariant: first worker splits dispatcher `-h` (Left/Right), later workers split only inside the right column (`-v`); fab never touches the vertical separator again — no `select-layout`, no rearranging user panes, no fighting manual resizes | Discussed — user-directed decision 1, verbatim | S:95 R:70 A:90 D:95 |
| 2 | Certain | Sibling detection keys on dispatch records (`Dispatch.Pane` + `PaneAlive`), not pane titles; titles still set for identification, only the placement probe changes | Discussed — user-directed decision 2; root cause (harness title clobbering) confirmed in code (`SiblingDispatchPane` title scan) | S:90 R:75 A:90 D:90 |
| 3 | Certain | First split sized: `split-window -h -l 35%`; config knob `dispatch.column_width`, scope `both`, default 35%, registered via the config field registry per `docs/specs/config.md` | Discussed — user-directed decision 3 with specific values | S:90 R:80 A:90 D:85 |
| 4 | Certain | Placement stays cosmetic: record-read/liveness failures degrade to the current fallback (split dispatcher `-h`), warn-only — never fail an otherwise-launchable dispatch | Discussed — user-directed decision 4; matches existing `SplitTarget` degradation contract | S:90 R:80 A:95 D:90 |
| 5 | Certain | `fab dispatch restart` re-derives mode/shape from the environment and uses the same record-based placement rule | Discussed — user-directed decision 5; restart already shares `resolveMode` | S:85 R:80 A:90 D:90 |
| 6 | Certain | Out of scope: layout repair, `select-layout` enforcement, run-kit layout ownership, separate workers window, done-worker reaping / pane names+colors (backlog `[zfl7]`) | Discussed — alternatives explicitly rejected or deferred by user | S:90 R:80 A:90 D:90 |
| 7 | Confident | Exact knob spelling `dispatch.column_width`, integer percent value rendered as `-l {n}%` | User gave the name as "e.g." — one obvious reading, mirrors `dispatch.watchable` naming/segment; trivially renameable pre-release | S:70 R:85 A:80 D:70 |
| 8 | Confident | Sibling filter is liveness AND same-window (dispatcher's window); "last" = bottom-most by intersecting record pane IDs with the window's `list-panes` order | Column invariant is per-window; splitting `-v` against a live pane in another window would misplace the worker; list-panes intersection preserves current geometric "last" semantics | S:50 R:80 A:80 D:65 |
| 9 | Confident | tmux < 3.1 (no `-l N%`): degrade to the unsized `-h` split with a warn-only notice rather than failing the dispatch | Follows directly from decision 4's placement-is-cosmetic principle | S:55 R:85 A:75 D:60 |
| 10 | Tentative | Record-enumeration scope: scan ALL `.fab-dispatch/*/` record dirs in the checkout (not just the active change's), bounded by the liveness + same-window filter | Deferred — promptless dispatch; discussion explicitly left scope unsettled ("defer, don't ask"). All-dirs assumed as the robust default; one-line filter change either way, revisable via /fab-clarify | S:30 R:75 A:40 D:35 |
| 11 | Confident | Sizing applies to every column-carving `-h` split, including the degraded fallback path; `-v` stacking splits stay unsized | "Maximum area reserved for the session agent" is the stated goal of the sizing; a fallback column squeezing the agent to 50% would reintroduce the secondary problem | S:55 R:85 A:80 D:70 |
| 12 | Confident | `change_type` = fix (root cause is a defect: title-keyed sibling detection broken by harness title rewrites; the knob is subordinate to the repair) | Primary deliverable repairs degraded designed behavior; flat 3.0 gate regardless | S:60 R:90 A:70 D:60 |
| 13 | Certain | Constitution sweep class: Go tests in-change; `_cli-fab.md` + SPEC mirror; `_preamble.md` pane bullets + SPEC mirror; `harness-adapters.md` § 3; `config.md` field row; `runtime/dispatch` memory + index regen | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps give a deterministic answer | S:85 R:80 A:95 D:90 |
| 14 | Certain | No migration file: additive config field with a default; no user-data restructuring; `fab config upgrade` regenerates the reference fence from the registry | context.md § Migrations applies only to restructuring existing user data; absent field ⇒ default is the existing cascade contract | S:75 R:85 A:90 D:85 |

14 assumptions (8 certain, 5 confident, 1 tentative, 0 unresolved).
