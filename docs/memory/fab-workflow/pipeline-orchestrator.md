# Pipeline Orchestrator

**Domain**: fab-workflow

## Overview

The pipeline orchestrator automates multi-change execution in dependency order. It reads a YAML manifest (`fab/pipelines/*.yaml`), resolves the dependency chain, and dispatches each change into an isolated worktree where `fab-ff` runs the full pipeline (tasks → apply → review → hydrate). The manifest is a live contract — the human adds entries while the orchestrator processes earlier ones.

V1 is serial (one change at a time). The orchestrator runs indefinitely until killed with Ctrl+C.

## Requirements

### Manifest Format

Pipeline manifests are YAML files in `fab/pipelines/` with:
- `base` — branch that root nodes branch from (typically `main`)
- `changes[]` — list of entries, each with `id`, `depends_on`, and optional `stage`

The `stage` field is written by the orchestrator. Valid values: `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate` (intermediate), `done`, `failed`, `invalid` (terminal).

`depends_on` supports at most one entry in v1. Multi-parent dependencies require manual merge of parent branches first.

### Orchestrator Core (run.sh)

`run.sh` accepts a manifest path and runs an infinite dispatch loop:

1. Re-reads the manifest from disk (live editing support)
2. Validates: schema, circular deps, single-dep constraint, reference integrity
3. Identifies dispatchable changes (deps all `done`, self not terminal)
4. Dispatches first dispatchable in list order (serial, deterministic)
5. After dispatch returns pane ID, enters a **unified polling loop** per change (5-second interval)
6. If nothing dispatchable: sleeps 30 seconds (configurable via `PIPELINE_POLL_INTERVAL`), re-reads. Uses `\r` in-place line update (no scrolling).

**Unified polling loop** (`poll_change()`): Monitors the interactive Claude session via `.status.yaml` polling and pane-alive checks. State machine:
- `polling_fab_ff` → detects `hydrate:done` (triggers ship)
- `shipping` → pushes `/changes:ship pr` via `tmux send-keys`, polls `gh pr view` for PR creation
- Terminal states: `done` (PR detected) or `failed` (timeout, pane death)

**Progress rendering**: Each poll iteration calls `stageman.sh progress-line` and renders in-place: `<id>: <progress> (<elapsed>)`.

**Configurable timeouts**: `PIPELINE_FF_TIMEOUT` (default 1800s/30min), `PIPELINE_SHIP_TIMEOUT` (default 300s/5min).

**Stage classification for resumability**:
- Terminal (`done`, `failed`, `invalid`) — skip permanently
- Intermediate (pipeline stage names) — re-dispatch into fresh worktree
- Absent — dispatch normally

On SIGINT: kills all tracked interactive panes, prints structured summary (Completed/Failed/Blocked/Skipped/Pending with worktree paths), exits 130.

Output: in-place progress updates per change.

### Change Dispatch (dispatch.sh)

`dispatch.sh` handles a single change — creates pane, sends commands, confirms switch, then returns:

1. **Worktree creation** — branch name is `{branch_prefix}{change-id}`. Root nodes: `wt-create --non-interactive --worktree-open skip <change-branch>`. Dependent nodes: `git branch <change-branch> origin/<parent-branch>` first, then `wt-create`.
2. **Artifact provisioning** — copies `fab/changes/<id>/` from source repo to worktree if not present
3. **Prerequisite validation** — intake.md, spec.md, confidence gate. Writes `invalid` on failure.
4. **Interactive pane creation** — starts a bare `claude --dangerously-skip-permissions` session in a tmux split pane (no initial command). First dispatch: horizontal split (`-h`); subsequent: vertical split stacked below previous (`-v -t $LAST_PANE_ID`). The pane appears immediately, giving the user visual feedback.
5. **fab-switch via send-keys** — after a startup delay (`CLAUDE_STARTUP_DELAY`, default 3s), sends `/fab-switch $CHANGE_ID --no-branch-change` to the pane via `tmux send-keys` (text, 0.5s gap, Enter).
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
| `batch-fab-pipeline.sh` | `fab/.kit/scripts/batch-fab-pipeline.sh` | User-facing entry point (listing, matching, delegation) |
| `run.sh` | `fab/.kit/scripts/pipeline/run.sh` | Main orchestrator loop |
| `dispatch.sh` | `fab/.kit/scripts/pipeline/dispatch.sh` | Per-change dispatch |

### batch-fab-pipeline.sh

User-facing entry point on PATH. Owns all UX: no-args/`--list` lists available pipelines from `fab/pipelines/*.yaml` (excluding `example.yaml`), `-h`/`--help` prints usage, positional arguments use case-insensitive substring matching against manifest basenames. Arguments with `/` or ending `.yaml` bypass matching. Delegates to `pipeline/run.sh` via `exec` with arg passthrough.

### Change ID Resolution

`run.sh` resolves each manifest entry's `id` field through `changeman resolve` before dispatching. This allows manifests to use short IDs (e.g., `a7k2`) or partial names. The manifest's internal consistency is preserved — `id` and `depends_on` values match each other as written. Resolution maps to the actual `fab/changes/` folder only at dispatch time. Resolution failure marks the change as `invalid` in the manifest.

### Worktree Naming

`dispatch.sh` uses `--worktree-name "$CHANGE_ID"` in its `wt-create` invocation, producing readable worktree directory names matching the change ID. This follows the pattern established by `batch-fab-switch-change.sh`.

### Progress Rendering

`stageman.sh progress-line` produces a single-line visual pipeline progress string for left-pane rendering. Done stages joined by ` → `, active stage + ` ⏳`, failed + ` ✗`, pending omitted. All-done appends ` ✓`. Examples: `spec → tasks → apply ⏳`, `intake → spec → tasks → apply → review ✗`.

### Stage Detection

`run.sh`'s polling loop uses `stageman progress-map` to detect `hydrate:done`, which triggers shipping. Intermediate states like `review:failed` are not treated as terminal — fab-ff manages its own rework lifecycle internally. The orchestrator relies on `hydrate:done` (success), pane death (failure), and timeout (failure) as the only terminal conditions.

### Shipping

After `hydrate:done` is detected, `run.sh` waits for Claude to finish its turn output before sending the ship command. The delay (`PIPELINE_SHIP_DELAY`, default 8s) prevents the Enter keystroke from being swallowed while Claude is still outputting its summary. The ship command is sent as two separate `tmux send-keys` calls — text first, 0.5s gap, then Enter — to prevent keystroke buffering issues. Both calls include `2>/dev/null` with error handling so that if the pane has died by the time a `send-keys` call runs, it fails gracefully; if the pane dies during the fixed delay, that failure is only discovered when sending begins. The session retains full fab-ff conversation context, producing contextual commit messages and PR descriptions. Ship completion is detected by polling `gh pr view` from the worktree.

### Pane Lifecycle

Each dispatched change gets its own tmux split pane (stacked vertically in the right panel). Sessions remain open for inspection — they are NOT auto-killed on completion. SIGINT kills all tracked panes.

## Design Decisions

### Serial Execution in V1
**Decision**: Process one change at a time in topological order.
**Why**: Avoids concurrent worktree management, manifest race conditions, and interleaved output. Parallel is a documented stretch goal.
**Rejected**: Background processes with PID tracking.

### Infinite Loop with SIGINT Exit
**Decision**: run.sh runs indefinitely, polling for new manifest entries.
**Why**: Supports the live-contract model — the human stays ahead, adding entries at their pace. The orchestrator is a daemon-like process, not a batch job.
**Rejected**: Exit when all changes done (prevents the human from adding more entries).

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

### Infrastructure Failures Abort
**Decision**: wt-create/claude/git failures abort the orchestrator entirely.
**Why**: Infrastructure failures indicate a broken environment. Continuing would likely fail on subsequent changes.
**Rejected**: Mark individual change failed and continue (masks environment issues).

## Testing

BATS test suite at `src/scripts/pipeline/test.bats` covers pure-logic functions from `run.sh` and `dispatch.sh`. Both scripts have source guards (`if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then main; fi`) enabling function-level testing without triggering `main()`.

### Coverage

**run.sh functions**: `validate_manifest` (7 scenarios: valid, missing base, empty changes, missing id, missing depends_on, dangling reference, multi-dependency), `detect_cycles` (4 scenarios: linear chain, direct cycle, indirect cycle, independent nodes), `is_terminal` (all stage values), `is_dispatchable` (4 scenarios), `find_next_dispatchable` (5 scenarios), `get_parent_branch` (root and dependent nodes).

**dispatch.sh functions**: `provision_artifacts` (3 scenarios: first provision, re-provision stale, missing source), `validate_prerequisites` (3 scenarios: missing intake, missing spec, passing gate).

### Not Covered (deferred)

`poll_change()` state machine, `main()` loops in both scripts, `run_pipeline()`, `create_worktree()`, `batch-fab-pipeline.sh`. These require complex infrastructure mocking (tmux, Claude CLI, wt-create, sleep loops).

### Test Patterns

- External commands (`tmux`, `claude`, `gh`, `changeman.sh`, `stageman.sh`, `calc-score.sh`) are stubbed via executables in `$TEST_DIR/bin/` prepended to `$PATH`
- YAML manifest fixtures created inline per test via `make_manifest` helper
- Each test sources the script under test, then overrides computed globals (`CHANGEMAN`, `CONFIG_FILE`, `FAB_DIR`, etc.) before calling functions
- `setup()` creates isolated `TEST_DIR`; `teardown()` removes it

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260221-8bs9-add-pipeline-orchestrator-tests | 2026-02-21 | Added BATS test suite (38 tests) for run.sh and dispatch.sh pure-logic functions. Added source guards to both scripts for testability. Moved `trap on_sigint INT` inside `main()` to prevent side effects when sourced. |
| 260221-6ljc-fix-pipeline-ship-timing | 2026-02-21 | Added `PIPELINE_SHIP_DELAY` (default 8s) wait after `hydrate:done` before sending ship command. Split `tmux send-keys` into text + Enter with 0.5s gap. Added `2>/dev/null` error handling on send-keys calls for graceful pane-death during delay. |
| 260221-h1l8-fix-orchestrator-false-fail-on-review | 2026-02-21 | Removed `:failed` catch-all from `poll_change()` — `review:failed` is a normal intermediate state in fab-ff's rework loop, not a terminal condition. Removed stale `[pipeline]` prefix from progress printf. |
| 260221-2spf-fix-pipeline-dispatch-timing | 2026-02-21 | Replaced `claude -p` fab-switch with visible interactive execution. dispatch.sh now creates a bare Claude pane first, sends fab-switch via send-keys, polls `fab/current` for switch confirmation, then sends fab-ff via send-keys. Added configurable delays (`CLAUDE_STARTUP_DELAY`, `POST_SWITCH_DELAY`) and polling (`SWITCH_POLL_INTERVAL`, `SWITCH_POLL_TIMEOUT`). Updated "Hybrid Model" design decision to "All-Interactive Model". |
| 260221-ay66-interactive-pipeline-pane | 2026-02-21 | Replaced passive `tail -f` log pane with interactive Claude sessions per dispatch. fab-ff now runs in interactive mode (not `claude -p`), shipping via `tmux send-keys` to the same session. Added unified polling loop in `run.sh` with progress-line rendering and state machine (polling_fab_ff → shipping → done/failed). Added `stageman.sh progress-line` command. Stacked vertical pane layout. Removed `ship()` from dispatch.sh. |
| 260221-i0z6-move-env-packages-add-fab-pipeline | 2026-02-21 | Added `batch-fab-pipeline.sh` user-facing entry point with listing, partial name matching, help, and `exec` delegation. Added changeman resolve for manifest change IDs in `run.sh`. Added `--worktree-name` to `wt-create` call in `dispatch.sh`. Replaced raw yq hydrate check with stageman `display-stage` in `dispatch.sh`. |
| 260221-wy0e-pipeline-orchestrator | 2026-02-21 | Initial implementation — serial orchestrator with run.sh + dispatch.sh, YAML manifest format, wt-create integration, Claude CLI execution, stacked PRs, SIGINT handling, example scaffold |
