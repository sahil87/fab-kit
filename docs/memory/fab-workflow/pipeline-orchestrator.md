# Pipeline Orchestrator

**Domain**: fab-workflow

## Overview

The pipeline orchestrator automates multi-change execution in dependency order. It reads a YAML manifest (`fab/pipelines/*.yaml`), resolves the dependency chain, and dispatches each change into an isolated worktree where `fab-ff` runs the full pipeline (tasks → apply → review → hydrate).

V1 is serial (one change at a time). By default, the orchestrator exits when all changes reach a terminal state (`done`, `failed`, `invalid`). Set `watch: true` in the manifest for infinite-loop mode (live editing — the human adds entries while the orchestrator processes earlier ones).

## Requirements

### Manifest Format

Pipeline manifests are YAML files in `fab/pipelines/` with:
- `base` — branch that root nodes branch from. Optional — defaults to the current branch if omitted, with `main` as the last-resort fallback (e.g., detached HEAD). When resolved, the value is written back to the manifest for downstream consistency.
- `watch` — optional boolean. `true` for infinite-loop mode (live editing), `false` or absent for finite mode (exit when all terminal). Default: `false`
- `changes[]` — list of entries, each with `id`, `depends_on`, and optional `stage`

The `stage` field is written by the orchestrator. Valid values: `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate` (intermediate), `done`, `failed`, `invalid` (terminal).

`depends_on` supports at most one entry in v1. Multi-parent dependencies require manual merge of parent branches first.

### Orchestrator Core (run.sh)

`run.sh` accepts a manifest path and runs a dispatch loop (finite by default, infinite with `watch: true`):

1. Re-reads the manifest from disk (live editing support)
2. Validates: schema, circular deps, single-dep constraint, reference integrity
3. Identifies dispatchable changes (deps all `done`, self not terminal)
4. Dispatches first dispatchable in list order (serial, deterministic)
5. After dispatch returns pane ID, enters a **unified polling loop** per change (5-second interval)
6. If nothing dispatchable: checks for finite exit (if `watch` is not `true` and all changes are terminal, prints summary and exits 0). Otherwise sleeps 30 seconds (configurable via `PIPELINE_POLL_INTERVAL`), re-reads. Uses `\r` in-place line update (no scrolling).

**Unified polling loop** (`poll_change()`): Monitors the interactive Claude session via `.status.yaml` polling and pane-alive checks. State machine:
- `polling_fab_ff` → detects `hydrate:done` (triggers ship)
- `shipping` → pushes `/git-pr` via `tmux send-keys`, polls `gh pr view` for PR creation
- Terminal states: `done` (PR detected) or `failed` (timeout, pane death)

**Progress rendering**: Each poll iteration calls `statusman.sh progress-line` and renders in-place: `<id>: <progress> (<elapsed>)`.

**Configurable timeouts**: `PIPELINE_FF_TIMEOUT` (default 1800s/30min), `PIPELINE_SHIP_TIMEOUT` (default 300s/5min).

**Agent idle signal**: Claude Code hook scripts (`fab/.kit/hooks/on-stop.sh`, `fab/.kit/hooks/on-session-start.sh`) write/clear an `agent.idle_since` timestamp in `.fab-runtime.yaml` at the repo root (gitignored). The file is keyed by full change folder name (`YYMMDD-XXXX-slug`), so each change's agent state is independent. This provides an explicit filesystem-based idle signal that the orchestrator (or future coordination tools) can poll instead of relying on fixed-delay heuristics (`CLAUDE_STARTUP_DELAY`, `POST_SWITCH_DELAY`, `PIPELINE_SHIP_DELAY`). The hooks are registered by `fab/.kit/sync/5-sync-hooks.sh`. The orchestrator does not yet consume this signal — it continues to use fixed delays. Replacing delays with idle-signal polling is a future enhancement.

**Stage classification for resumability**:
- Terminal (`done`, `failed`, `invalid`) — skip permanently
- Intermediate (pipeline stage names) — re-dispatch into fresh worktree
- Absent — dispatch normally

On SIGINT: kills all tracked interactive panes, prints structured summary (Completed/Failed/Blocked/Skipped/Pending with worktree paths), exits 130.

Output: in-place progress updates per change.

### Change Dispatch (dispatch.sh)

`dispatch.sh` handles a single change — creates pane, sends commands, confirms switch, then returns:

1. **Worktree creation** — branch name is `{branch_prefix}{change-id}`. Dependent nodes: `git branch <change-branch> <parent-branch>` from local branch ref first, then `wt-create --non-interactive --reuse --worktree-open skip --worktree-name <change-id> <change-branch>`. The `--reuse` flag makes worktree creation idempotent — if a worktree with that name already exists (from a previous interrupted run), it returns the existing path instead of erroring.
2. **Artifact provisioning** — copies `fab/changes/<id>/` from source repo to worktree if not present
3. **Prerequisite validation** — intake.md, spec.md, confidence gate. Writes `invalid` on failure.
4. **Interactive pane creation** — starts a bare `claude --dangerously-skip-permissions` session in a tmux split pane (no initial command). First dispatch: horizontal split (`-h`); subsequent: vertical split stacked below previous (`-v -t $LAST_PANE_ID`). The pane appears immediately, giving the user visual feedback.
5. **fab-switch via send-keys** — after a startup delay (`CLAUDE_STARTUP_DELAY`, default 3s), sends `/fab-switch $CHANGE_ID` to the pane via `tmux send-keys` (text, 0.5s gap, Enter). `/fab-switch` is git-free — branch setup is handled by `create_worktree()` before the pane is created.
6. **fab/current polling** — polls `$wt_path/fab/current` until it matches `$CHANGE_ID` (interval: `SWITCH_POLL_INTERVAL` 2s, timeout: `SWITCH_POLL_TIMEOUT` 60s). Checks pane alive each iteration. Timeout or pane death marks the change `failed` in the manifest.
7. **fab-ff via send-keys** — after a post-switch delay (`POST_SWITCH_DELAY`, default 5s), sends `/fab-ff` to the pane via `tmux send-keys` (text, 0.5s gap, Enter).
8. **Output** — two stdout lines: worktree path + pane ID. `run.sh` captures both.

`dispatch.sh` polls `fab/current` to confirm switch completion but does NOT poll for fab-ff completion or shipping — `run.sh`'s polling loop handles that.

Infrastructure failures (wt-create, claude, git) abort the orchestrator entirely.

Worktrees and interactive panes are left in place after dispatch for manual inspection.

### Example Scaffold

`fab/pipelines/example.yaml` — fully commented-out annotated example covering manifest format, dependency syntax, stage values, prerequisites, and the live-editing contract.

## Scripts

| Script | Location | Purpose |
|--------|----------|---------|
| `batch-pipeline.sh` | `fab/.kit/scripts/batch-pipeline.sh` | User-facing entry point (listing, matching, delegation) |
| `batch-pipeline-series.sh` | `fab/.kit/scripts/batch-pipeline-series.sh` | Sequential chain shorthand (generates manifest, delegates) |
| `run.sh` | `fab/.kit/scripts/pipeline/run.sh` | Main orchestrator loop |
| `dispatch.sh` | `fab/.kit/scripts/pipeline/dispatch.sh` | Per-change dispatch |

### batch-pipeline.sh

User-facing entry point on PATH. Owns all UX: no-args/`--list` lists available pipelines from `fab/pipelines/*.yaml` (excluding `example.yaml`), `-h`/`--help` prints usage, positional arguments use case-insensitive substring matching against manifest basenames. Arguments with `/` or ending `.yaml` bypass matching. Delegates to `pipeline/run.sh` via `exec` with arg passthrough.

### batch-pipeline-series.sh

Shorthand for running a sequential chain of changes. Accepts one or more change IDs as positional arguments and an optional `--base <branch>` flag (defaults to current branch). Generates a temporary manifest at `fab/pipelines/.series-{epoch}.yaml` with a linear dependency chain (first change depends on `[]`, each subsequent depends on its predecessor). No `watch` field (finite mode by default). Delegates to `pipeline/run.sh` via `exec`. The generated manifest is not cleaned up — left for debugging.

### Change ID Resolution

`run.sh` resolves each manifest entry's `id` field through `changeman resolve` before dispatching. This allows manifests to use short IDs (e.g., `a7k2`) or partial names. The manifest's internal consistency is preserved — `id` and `depends_on` values match each other as written. Resolution maps to the actual `fab/changes/` folder only at dispatch time. Resolution failure marks the change as `invalid` in the manifest.

### Worktree Naming

`dispatch.sh` uses `--worktree-name "$CHANGE_ID"` in its `wt-create` invocation, producing readable worktree directory names matching the change ID. This follows the pattern established by `batch-fab-switch-change.sh`.

### Progress Rendering

`statusman.sh progress-line` produces a single-line visual pipeline progress string for left-pane rendering. Done stages joined by ` → `, active stage + ` ⏳`, failed + ` ✗`, pending omitted. All-done appends ` ✓`. Examples: `spec → tasks → apply ⏳`, `intake → spec → tasks → apply → review ✗`.

### Stage Detection

`run.sh`'s polling loop uses `statusman progress-map` to detect `hydrate:done`, which triggers shipping. Intermediate states like `review:failed` are not treated as terminal — fab-ff manages its own rework lifecycle internally. The orchestrator relies on `hydrate:done` (success), pane death (failure), and timeout (failure) as the only terminal conditions.

### Shipping

After `hydrate:done` is detected, `run.sh` waits for Claude to finish its turn output before sending the ship command. The delay (`PIPELINE_SHIP_DELAY`, default 8s) prevents the Enter keystroke from being swallowed while Claude is still outputting its summary. The ship command (`/git-pr`) is sent as two separate `tmux send-keys` calls — text first, 0.5s gap, then Enter — to prevent keystroke buffering issues. Both calls include `2>/dev/null` with error handling so that if the pane has died by the time a `send-keys` call runs, it fails gracefully; if the pane dies during the fixed delay, that failure is only discovered when sending begins. The `/git-pr` skill autonomously commits, pushes, and creates a GitHub PR — it is a native fab-kit skill (defined in `fab/.kit/skills/git-pr.md`).

Ship completion is detected by polling for a `.pr-done` sentinel file at `$wt_path/fab/changes/$resolved_id/.pr-done`. This file is written by `/git-pr` as its very last action, after all git operations (including a second commit+push of `.status.yaml` with the PR URL) are complete. The sentinel provides a race-free filesystem signal — its existence guarantees the branch tip is clean and the PR is fully created. The sentinel file is gitignored.

### Pane Lifecycle

Each dispatched change gets its own tmux split pane (stacked vertically in the right panel). Sessions remain open for inspection — they are NOT auto-killed on completion. SIGINT kills all tracked panes.

## Design Decisions

### Serial Execution in V1
**Decision**: Process one change at a time in topological order.
**Why**: Avoids concurrent worktree management, manifest race conditions, and interleaved output. Parallel is a documented stretch goal.
**Rejected**: Background processes with PID tracking.

### Finite Exit as Default
**Decision**: run.sh exits when all changes are terminal by default. `watch: true` in the manifest opts into infinite-loop mode for live editing.
**Why**: Most pipeline runs have a known, finite set of changes. The infinite loop is only needed for the live-editing workflow. Making finite the default reduces ceremony for the common case and enables `batch-pipeline-series.sh` to work naturally without special flags.
**Rejected**: `--finite` CLI flag (control belongs in the manifest, not the CLI). Previous default was infinite loop, which required Ctrl+C for normal completion.

### All-Interactive Model: Bare Pane + Send-Keys
**Decision**: A bare interactive Claude session is created first, then fab-switch and fab-ff are both sent via `tmux send-keys`. Shipping is pushed to the same session.
**Why**: The previous hybrid model (`claude -p` for fab-switch, interactive for fab-ff) ran fab-switch invisibly before the pane existed, leaving the user with no feedback for 5-15 seconds. Moving everything into the interactive pane provides immediate visual feedback. The send-keys approach gives sequencing control while keeping both commands visible. `fab/current` polling confirms switch completion before sending fab-ff.
**Rejected**: `claude -p` for fab-switch (invisible, no user feedback), passing fab-switch as the initial tmux command (can't sequence fab-ff after switch completion).

### Polling in run.sh, Not dispatch.sh
**Decision**: `dispatch.sh` spawns the interactive pane and exits immediately. `run.sh` owns all polling, progress rendering, and completion detection.
**Why**: Keeps dispatch.sh simple and stateless. run.sh already owns the main loop and SIGINT handling. Centralizing all waiting avoids duplicated SIGINT handling.
**Rejected**: dispatch.sh waits for completion (blocks serial dispatch, duplicates SIGINT handling).

### Stacked Vertical Splits
**Decision**: Each dispatched change gets its own vertical split within the right panel. Sessions never auto-killed.
**Why**: Users need to inspect failed or completed sessions. Killing panes loses diagnostic context. Vertical stacking provides natural visual grouping.
**Rejected**: Single reusable right pane (can't inspect previous sessions), horizontal splits (poor screen width use).

### Single-Dependency Restriction
**Decision**: `depends_on` limited to one entry in v1.
**Why**: Multi-parent dependencies create a branch topology problem — D depending on B and C can't branch from both. Resolution requires manual merge.
**Rejected**: Branching from last-completed parent (loses other parent's code).

### Local Branch Refs for Dependent Nodes
**Decision**: `dispatch.sh` branches dependent nodes from local `refs/heads/` instead of `origin/`.
**Why**: Git branches are shared across all worktrees of the same repo. The parent branch exists locally after its worktree was created. Using `origin/` assumed a push had completed, which was an implementation coincidence — the push happens during `/git-pr` but branch resolution happens earlier.
**Rejected**: `origin/` with fetch-first — adds network dependency and latency for no benefit when the branch is already local.

### Infrastructure Failures Abort
**Decision**: wt-create/claude/git failures abort the orchestrator entirely.
**Why**: Infrastructure failures indicate a broken environment. Continuing would likely fail on subsequent changes.
**Rejected**: Mark individual change failed and continue (masks environment issues).

## Testing

BATS test suite at `src/scripts/pipeline/test.bats` covers pure-logic functions from `run.sh` and `dispatch.sh`. Both scripts have source guards (`if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then main; fi`) enabling function-level testing without triggering `main()`.

### Coverage

**run.sh functions**: `validate_manifest` (8 scenarios: valid, missing base, empty changes, missing id, missing depends_on, dangling reference, multi-dependency, watch field), `detect_cycles` (4 scenarios: linear chain, direct cycle, indirect cycle, independent nodes), `is_terminal` (all stage values), `is_dispatchable` (4 scenarios), `find_next_dispatchable` (5 scenarios), `get_parent_branch` (root and dependent nodes), `all_terminal` (5 scenarios: all done, all failed, mixed terminal, one pending, one intermediate).

**dispatch.sh functions**: `provision_artifacts` (3 scenarios: first provision, re-provision stale, missing source), `validate_prerequisites` (3 scenarios: missing intake, missing spec, passing gate).

### Not Covered (deferred)

`poll_change()` state machine, `main()` loops in both scripts, `run_pipeline()`, `create_worktree()`, `batch-pipeline.sh`, `batch-pipeline-series.sh`. These require complex infrastructure mocking (tmux, Claude CLI, wt-create, sleep loops).

### Test Patterns

- External commands (`tmux`, `claude`, `gh`, `changeman.sh`, `statusman.sh`, `calc-score.sh`) are stubbed via executables in `$TEST_DIR/bin/` prepended to `$PATH`
- YAML manifest fixtures created inline per test via `make_manifest` helper
- Each test sources the script under test, then overrides computed globals (`CHANGEMAN`, `CONFIG_FILE`, `FAB_DIR`, etc.) before calling functions
- `setup()` creates isolated `TEST_DIR`; `teardown()` removes it

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260306-1lwf-extract-agent-runtime-file | 2026-03-06 | Agent idle signal now targets `.fab-runtime.yaml` (repo root, gitignored) keyed by change folder name, instead of `.status.yaml`. Orchestrator still does not consume the signal. |
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added agent idle signal documentation: `on-stop.sh` and `on-session-start.sh` hooks write/clear `agent.idle_since` in `.status.yaml`, registered by `5-sync-hooks.sh`. Signal-only — orchestrator does not yet consume it (fixed delays remain). Replacing delays with idle-signal polling is a future enhancement. |
| 260223-xiuk-batch-pipeline-single-change-and-base-branch | 2026-02-23 | `batch-pipeline-series.sh` now accepts a single change argument (minimum lowered from 2 to 1). `run.sh` `validate_manifest()` treats `base` as optional — resolves to current branch via `git branch --show-current` with `main` fallback, writes resolved value back to manifest. Fixed detached HEAD fallback in both scripts (explicit empty-check replaces `\|\|` pattern). Scaffold `example.yaml` synced with main copy. Test suite: 44→46 tests. |
| 260227-gasp-consolidate-status-field-naming | 2026-02-27 | `.shipped` sentinel renamed to `.pr-done`. Pipeline runner updated accordingly. |
| 260222-trdc-git-pr-shipped-sentinel-and-status-commit | 2026-02-22 | Ship completion detection now uses `.shipped` sentinel file (gitignored, written by `/git-pr` after all git ops) instead of `statusman is-shipped` polling. Eliminates TOCTOU race where `.status.yaml` update was visible before commit+push completed. `/git-pr` now performs a second commit+push of `.status.yaml` before writing the sentinel. |
| 260222-6ldg-wt-create-reuse-flag | 2026-02-22 | Added `--reuse` flag to `wt-create` — returns existing worktree path on name collision instead of erroring. `dispatch.sh` now passes `--reuse` (removed bespoke `wt_get_worktree_path_by_name` pre-check and `wt-common.sh` source import). `batch-fab-switch-change.sh` also passes `--reuse` for idempotent re-runs. |
| 260222-bcfy-batch-pipeline-series-rename | 2026-02-22 | Renamed `batch-fab-pipeline.sh` → `batch-pipeline.sh`. Added `watch` manifest field and finite-exit default to `run.sh` (exits when all terminal; `watch: true` for infinite loop). Fixed `dispatch.sh` to use local branch refs instead of `origin/`. Added `batch-pipeline-series.sh` (sequential chain shorthand — generates temp manifest, delegates to run.sh). Added `.gitignore` pattern for generated series manifests. Added `all_terminal` function to run.sh. Test suite: 38→44 tests. |
| 260222-n811-absorb-ship-command | 2026-02-22 | Replaced external `/changes:ship pr` (prompt-pantry) with native `/git-pr` skill. Updated `run.sh` ship command and log message. Added `git-pr` to `fab-help.sh` Completion group. |
| 260221-8bs9-add-pipeline-orchestrator-tests | 2026-02-21 | Added BATS test suite (38 tests) for run.sh and dispatch.sh pure-logic functions. Added source guards to both scripts for testability. Moved `trap on_sigint INT` inside `main()` to prevent side effects when sourced. |
| 260221-6ljc-fix-pipeline-ship-timing | 2026-02-21 | Added `PIPELINE_SHIP_DELAY` (default 8s) wait after `hydrate:done` before sending ship command. Split `tmux send-keys` into text + Enter with 0.5s gap. Added `2>/dev/null` error handling on send-keys calls for graceful pane-death during delay. |
| 260221-h1l8-fix-orchestrator-false-fail-on-review | 2026-02-21 | Removed `:failed` catch-all from `poll_change()` — `review:failed` is a normal intermediate state in fab-ff's rework loop, not a terminal condition. Removed stale `[pipeline]` prefix from progress printf. |
| 260221-2spf-fix-pipeline-dispatch-timing | 2026-02-21 | Replaced `claude -p` fab-switch with visible interactive execution. dispatch.sh now creates a bare Claude pane first, sends fab-switch via send-keys, polls `fab/current` for switch confirmation, then sends fab-ff via send-keys. Added configurable delays (`CLAUDE_STARTUP_DELAY`, `POST_SWITCH_DELAY`) and polling (`SWITCH_POLL_INTERVAL`, `SWITCH_POLL_TIMEOUT`). Updated "Hybrid Model" design decision to "All-Interactive Model". |
| 260221-ay66-interactive-pipeline-pane | 2026-02-21 | Replaced passive `tail -f` log pane with interactive Claude sessions per dispatch. fab-ff now runs in interactive mode (not `claude -p`), shipping via `tmux send-keys` to the same session. Added unified polling loop in `run.sh` with progress-line rendering and state machine (polling_fab_ff → shipping → done/failed). Added `statusman.sh progress-line` command. Stacked vertical pane layout. Removed `ship()` from dispatch.sh. |
| 260221-i0z6-move-env-packages-add-fab-pipeline | 2026-02-21 | Added `batch-fab-pipeline.sh` user-facing entry point with listing, partial name matching, help, and `exec` delegation. Added changeman resolve for manifest change IDs in `run.sh`. Added `--worktree-name` to `wt-create` call in `dispatch.sh`. Replaced raw yq hydrate check with statusman `display-stage` in `dispatch.sh`. |
| 260221-wy0e-pipeline-orchestrator | 2026-02-21 | Initial implementation — serial orchestrator with run.sh + dispatch.sh, YAML manifest format, wt-create integration, Claude CLI execution, stacked PRs, SIGINT handling, example scaffold |
