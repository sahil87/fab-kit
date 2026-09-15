# Intake: Pane Map on rk Enumeration

**Change**: 260820-89vn-pane-map-rk-enumeration
**Created**: 2026-08-20

## Origin

One-shot `/fab-new` invocation (no prior discussion in the conversation):

> Part 8 of docs/specs/cli-layering.md (fab-kit half): enrich rk mux panes with change/stage — consume the newly-native rk mux panes enumeration (run-kit v3.17.18, PR #677) and enrich its output with fab change/stage data, replacing the old fab pane map cached-join role. Dep: run-kit half merged and released (v3.17.18, confirmed).

The cli-layering spec lives in the **run-kit repo** (`~/code/sahil87/run-kit/docs/specs/cli-layering.md`). Its Part 8 row delivers, across both repos: "`rk mux panes` (native enumeration + agent state); rk's server drops the cached `fab pane map` join in `sessions.go` (kills the StatusDot 5s-lag class); fab enriches enumeration with change/stage". The run-kit half is **merged and released** (v3.17.18, PR #677, change `260820-hol4-mux-panes-native-pane-map`) — verified on this machine: `rk --version` reports v3.17.18 and `rk mux panes --json` works. Run-kit's server now derives fab state natively from each pane's cwd at request time; it no longer subprocess-joins `fab pane map`, so fab's `--json` output has lost its one machine consumer. This change is the fab-kit half: `fab pane map` consumes rk's native enumeration and keeps only the choreography-enrichment role.

## Why

1. **The pain point**: cli-layering's two-layer model gives rk the tmux substrate and fab the choreography, with delegation rule 1: "each tool delegates to the other for facts the other layer owns; neither reimplements the other's layer." Today `fab pane map` does both halves itself — its own `tmux list-panes -F` enumeration, its own raw `@rk_agent_state` option parse, then the change/stage join. Now that `rk mux panes` exists as the canonical substrate enumeration (reconciled agent state, internal-session exclusion, pinned-window dedup), fab's copy is exactly the duplicate-implementation liability the layering plan exists to remove — the same class as the retired `fab pane send`/`await` gate duplicate (Part 5).

2. **If we don't fix it**: fab's enumeration diverges from rk's user-facing truth — `fab pane map` shows rk-internal noise rows (`_rk-pin-*` pin-sessions, the `_rk-ctl` anchor) and duplicates pinned windows, and its raw agent-state read can disagree with rk's reconciled view (the dashboard and the operator's per-tick snapshot disagree about the same pane). The Part 8 row stays half-done and the pane-map row of the fab command-surface split table keeps its "today" caveat indefinitely.

3. **Why this approach**: capability-probed, fail-open in-binary delegation (delegation rule 2's posture, the same `command -v rk` gate every skill-level rk use carries). fab-kit stays installable and fully functional without rk — the existing internal enumeration remains as the verbatim fallback — while an rk-present machine gets the strictly better substrate view for free. Rejected: a big-bang replacement that requires rk (violates rule 2); skill-level composition (`rk mux panes --json` piped into a new fab stdin-filter verb) — the change/stage join is Go logic inside fab and the operator's hot path wants one command, matching how rk's server consumed one command before.

## What Changes

### 1. `fab pane map` enumeration delegates to `rk mux panes --json` (rk-gated, fail-open)

`runPaneMap` gains a delegation seam ahead of `discoverPanes`:

- **Probe = the attempt.** If `rk` resolves on PATH (`exec.LookPath`, the `batch new` wt-guard precedent), run `rk mux panes --json`, appending `-L <server>` when `--server`/`-L` was passed (rk's mux family takes the same flag with the same resolution order). **Any** failure — rk absent, an older rk without the subcommand (cobra unknown-command, exit ≠ 0), non-zero exit, or unparseable JSON — falls back **silently** to the existing internal `tmux list-panes` enumeration, byte-identical to today. No warning, no error: absence/failure degrades, never breaks (delegation rule 2).
- **Row mapping** (rk JSON → fab's pane entry): `pane`←`pane`, `tab`←`window_name`, `cwd`←`cwd`, `session`←`session`, `index`←`window_index`, `windowID`←`window_id`, agent state ← rk's reconciled `agent_state` + `agent_state_duration` (carried structurally — no re-read of the `@rk_agent_state` pane option on the delegated path). rk's contract (run-kit memory `agent-messaging.md` § `rk mux panes`) pins the exact `--json` field set: `session`, `session_id`, `window_index`, `window_id`, `window_name`, `window_active`, `pane`, `pane_index`, `pane_active`, `command`, `cwd`, `agent_state`, `agent_state_duration`.
- **Session scoping is fab's job.** `rk mux panes` always enumerates the whole server; fab filters the rows to preserve its flag contract: default mode → rows whose `session` equals the current session (resolved via `tmux display-message -p '#{session_name}'`; the `$TMUX`-required guard stays), `--session <name>` → equality filter on that name, `--all-sessions` → no filter.
- **Enrichment is unchanged.** The per-cwd resolution pipeline (git worktree root → `fab/` → `.fab-status.yaml` → `.status.yaml` → change/stage/display_state/PR) runs identically on both paths, with the existing per-cwd and per-repo caches.

### 2. Row-set semantics differ deliberately between the two paths

When delegated, `fab pane map` adopts rk's **filtered** view: `_rk-pin-*` pin-sessions and the `_rk-ctl` anchor contribute no rows, and a pinned window lists exactly once via its home session — matching the dashboard's user-facing truth ("an enrichment consumer wants one row per real pane", the hol4 design decision). The rk-absent fallback keeps today's raw enumeration verbatim — fab does **not** reimplement rk's session-filtering conventions (rule 1 cuts both ways).

### 3. Output contracts stay stable

- **Table**: same columns (`Session`/`Pane`/`WinIdx`/`Tab`/`Worktree`/`Change`/`Stage`/`Agent`), same rendering. The Agent column keeps `active` / `waiting` / `idle (<dur>)` / `—`.
- **JSON**: same field set (`session`, `window_index`, `window_id`, `pane`, `tab`, `worktree`, `repo`, `change`, `stage`, `display_state`, `agent_state`, `agent_idle_duration`, `pr_url`, `pr_number`), same nullability contract. `agent_idle_duration` keeps fab's idle-only semantics: rk reports a duration for `waiting` too (`agent_state_duration`), but fab's field stays populated only for `idle` — the delegated path drops a waiting-row duration rather than changing the published contract. rk-only fields (`command`, `window_name`, `window_active`, `pane_index`, `session_id`) are **not** adopted.
- The observable delta on an rk-present machine is therefore: reconciled (rather than raw) agent state, and rk's filtered row set. Everything else is byte-compatible.

### 4. Docs, tests, and scope boundary

- **`src/kit/skills/_cli-fab.md` § fab pane map**: document the delegation (enumeration source, fail-open fallback, filtered-row-set delta) — constitution requirement for any fab CLI behavior change.
- **Tests** (`panemap_test.go` + any new helper's tests): the rk-JSON → entry mapping (including `agent_state`/`agent_state_duration` → display/JSON fields), session filtering of the whole-server enumeration, and fallback triggers (rk missing, non-zero exit, malformed JSON) via an injectable exec seam (the `setupcheck` LookPath-seam precedent).
- **Sibling sweep at apply**: grep `pane map` across `src/kit/skills/`, `docs/specs/`, `docs/memory/` for enumeration-source claims (e.g. `_cli-fab.md`'s "read from the SAME `list-panes` call — zero extra subprocesses" phrasing, `pane-commands.md`'s subprocess-economy section) — these become fallback-path-only claims.
- **Out of scope**: `fab resolve --pane` (resolve.go) keeps the internal enumeration — it needs only cwd-per-pane and is not part of the pane-map role Part 8 splits. Skill-facing guidance (`fab-operator.md`, `_cli-agents.md`) keeps invoking `fab pane map` unchanged — the delegation is inside the binary, so no guidance re-point is needed (unlike Part 7). The cli-layering spec itself lives in run-kit and is not edited by this change (its execution-plan row is marked done from the run-kit side / by the operator).

## Affected Memory

- `runtime/pane-commands.md`: (modify) § `fab pane map` — enumeration delegation to `rk mux panes --json`, the fail-open fallback, session-scoping-as-filter, the filtered-row-set and reconciled-agent-state deltas; the subprocess-economy prose becomes fallback-path-scoped
- `runtime/runtime-agents.md`: (modify) narrow the "fab reads the pane option via plain tmux commands" claim — the delegated `map` path consumes rk's reconciled state via `rk mux panes --json`; the plain-tmux read remains the fallback and the `capture` path

## Impact

- `src/go/fab/cmd/fab/panemap.go` — the delegation seam, rk-JSON parsing/mapping, session filtering (possibly a small helper in `internal/pane` if shared machinery fits better there)
- `src/go/fab/cmd/fab/panemap_test.go` — mapping/filter/fallback tests
- `src/kit/skills/_cli-fab.md` — § fab pane map behavior update
- Hydrate: `docs/memory/runtime/pane-commands.md`, `docs/memory/runtime/runtime-agents.md`
- No migration (no user-data restructuring); no new flags; no exit-code changes (`map` keeps its plain exit-1 scheme)
- Release: MINOR (feat)
- Dependency: run-kit ≥ v3.17.18 **when present** (older rk fails the probe and falls back — no version pin, no hard requirement)

## Open Questions

None — the spec row, the shipped run-kit half, and the delegation rules determine the design; remaining choices are graded below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The surface stays `fab pane map`; delegation replaces fab's own enumeration when rk is present, keeping fab as the change/stage enrichment layer | The spec's pane-map row says exactly this ("rk owns enumeration, fab enriches with change/stage"); the description restates it | S:90 R:75 A:90 D:90 |
| 2 | Certain | Fail-open fallback to the existing internal tmux enumeration whenever rk is absent or the call fails (old rk, non-zero exit, bad JSON) — silent, never an error | Delegation rule 2 verbatim; the attempt-is-the-probe pattern is established (rk skill version-skew fallback) | S:85 R:85 A:95 D:90 |
| 3 | Confident | Delegation is in-binary (`fab pane map` execs `rk mux panes --json`), not a new stdin-filter verb or skill-level piping | "Consume the enumeration and enrich its output" reads as direct consumption; the join is Go logic; the operator hot path wants one command; wt-exec precedent in `batch new` | S:65 R:55 A:70 D:65 |
| 4 | Confident | The delegated path adopts rk's filtered row set (internal sessions excluded, pinned windows once); the fallback keeps today's raw view | hol4's design decision ("one row per real pane"); reimplementing rk's filter in the fallback would violate rule 1 | S:60 R:70 A:75 D:70 |
| 5 | Confident | Output contracts stay stable: same table columns, same JSON field set; rk-only fields not adopted; `agent_idle_duration` stays idle-only (a waiting-row duration from rk is dropped, not surfaced) | Contract-stability precedent is strong in this repo (byte-identical table rules across prior pane-map changes); additive fields are cheap to add later if a consumer appears | S:55 R:75 A:70 D:60 |
| 6 | Confident | Agent state on the delegated path is rk's reconciled value taken verbatim (no re-read of the `@rk_agent_state` option); the fallback keeps the raw-option parse | rk's reconciler is the strictly-better source and the point of delegating; double-reading would reintroduce the disagreement being removed | S:65 R:75 A:80 D:75 |
| 7 | Confident | `fab resolve --pane` keeps the internal enumeration — out of this change's scope | It consumes only cwd-per-pane (no agent state, no row-set semantics); Part 8 names only the pane-map role | S:55 R:80 A:75 D:70 |

7 assumptions (2 certain, 5 confident, 0 tentative, 0 unresolved).
