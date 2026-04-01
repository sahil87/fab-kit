# Tasks: Tmux Superpowers — Operator Research

**Change**: 260331-1y2f-tmux-superpowers-operator-research
**Spec**: `spec.md`
**Intake**: `intake.md`

<!--
  TASK FORMAT: - [ ] {ID} [{markers}] {Description with file paths}

  Markers (optional, combine as needed):
    [P]   — Parallelizable (different files, no dependencies on other [P] tasks in same group)

  IDs are sequential: T001, T002, ...
  Include exact file paths in descriptions.
  Each task should be completable in one focused session.

  Tasks are grouped by phase. Phases execute sequentially.
  Within a phase, [P] tasks can execute in parallel.
-->

## Phase 1: Setup

<!-- Create pane parent command group, re-register pane-map under it, update main.go -->

- [x] T001 Create `pane` parent command group in `src/go/fab/cmd/fab/pane.go` — Cobra command with `Use: "pane"`, no `RunE` (prints help), short description. Register `map`, `capture`, `send`, `process` subcommands (stubs for now). In `src/go/fab/cmd/fab/main.go`, replace `paneMapCmd()` registration with `paneCmd()`.
- [x] T002 Move existing `paneMapCmd()` under `pane` parent: in `src/go/fab/cmd/fab/panemap.go`, change `Use` from `"pane-map"` to `"map"` and update `Short` to reflect subcommand context. Verify `pane map --json`, `--session`, `--all-sessions` flags still work.
- [x] T003 Rename `src/go/fab/cmd/fab/panemap_test.go` to `src/go/fab/cmd/fab/pane_map_test.go` (or keep and adjust) — update any test references from `pane-map` to `pane map`. Ensure existing tests pass under the new command path.

## Phase 2: Core Implementation

<!-- Primary functionality for capture, send, and process subcommands -->

- [x] T004 [P] Implement `fab pane capture` in `src/go/fab/cmd/fab/pane_capture.go` — positional `<pane>` arg, `-l N` flag (default: all visible lines), raw text output via `tmux capture-pane -t <pane> -p`. Validate pane existence (exit 1 with stderr message if not found). Reuse `gitWorktreeRoot`, `readFabCurrent`, `resolveAgentState` from `src/go/fab/cmd/fab/panemap.go` for `--json` mode enrichment (pane, lines, content, change, stage, agent_state).
- [x] T005 [P] Implement `fab pane send` in `src/go/fab/cmd/fab/pane_send.go` — two positional args `<pane> <text>`, `--force` flag. Default mode: (1) verify pane exists via `tmux display-message -t <pane> -p '#{pane_id}'`, (2) resolve worktree root from pane CWD, read `.fab-runtime.yaml` idle state (reuse `resolveAgentState` logic from `panemap.go`), (3) if not idle and not `--force`, exit 1 with `"agent in <pane> is active, use --force to override"`. Non-fab panes treated as idle. Send via `tmux send-keys -t <pane> "<text>" Enter`.
- [x] T006 [P] Implement process state detection helpers in `src/go/fab/cmd/fab/pane_process.go` — platform abstraction using `runtime.GOOS`: Linux reads `/proc/{pid}/stat` and `/proc/{pid}/wchan`; macOS uses `ps -o stat= -p {pid}` and `lsof`. Detect five states: `running`, `waiting-for-input`, `sleeping`, `stopped`, `exited`. Ambiguous states fall back to `sleeping`.
- [x] T007 Wire `fab pane process` command in `src/go/fab/cmd/fab/pane_process.go` — positional `<pane>` arg, `--json` flag. Resolve foreground PID via `tmux display-message -t <pane> -p '#{pane_pid}'`, walk process tree to foreground group leader. Default output: single state word. JSON output: `{pane, pid, state, process_name, change}`. Reuse `resolvePaneChange` from `panemap.go` for change context.

## Phase 3: Integration & Edge Cases

<!-- Tests for new commands, edge case handling -->

- [x] T008 [P] Add tests for `fab pane capture` in `src/go/fab/cmd/fab/pane_capture_test.go` — test raw output mode, JSON output mode with fab context, JSON with no fab context (null fields), pane-not-found error case, `-l` flag line limiting.
- [x] T009 [P] Add tests for `fab pane send` in `src/go/fab/cmd/fab/pane_send_test.go` — test successful send to idle agent, blocked by active agent, `--force` overrides idle check, pane-not-found error, non-fab pane treated as idle, force still fails on non-existent pane.
- [x] T010 [P] Add tests for `fab pane process` in `src/go/fab/cmd/fab/pane_process_test.go` — test each of the five states (mock `/proc` reads or `ps`/`lsof` output), JSON output format, pane-not-found error, ambiguous wchan falls back to `sleeping`.
- [x] T011 Add integration test for `pane` parent command in `src/go/fab/cmd/fab/pane_test.go` — verify `fab pane` with no subcommand prints help listing all four subcommands, verify `fab pane-map` is no longer recognized.

## Phase 4: Polish

<!-- Documentation updates per constitution requirement and spec -->

- [x] T012 [P] Update `fab/.kit/skills/_cli-fab.md` — add documentation for `fab pane map`, `fab pane capture`, `fab pane send`, `fab pane process` with full signatures, flags, and output formats. Remove `fab pane-map` entry.
- [x] T013 [P] Update `fab/.kit/skills/_cli-external.md` — remove `tmux capture-pane` and `tmux send-keys` entries from the tmux section; keep only `tmux new-window`. Add cross-references to `fab pane capture` and `fab pane send`.
- [x] T014 [P] Update `fab/.kit/skills/fab-operator7.md` — Section 3 (Pre-Send Validation): replace steps 1-2 with `fab pane send` call. Section 5 (Question Detection): add `fab pane process` pre-filter step before capture-and-regex.

---

## Execution Order

- T001 blocks T002, T003, T004, T005, T006, T007 (parent command must exist first)
- T002 and T003 should complete before T004-T007 (map subcommand stable before adding siblings)
- T006 blocks T007 (process helpers needed before command wiring)
- T004, T005, T007 are independent of each other after T002
- T008, T009, T010, T011 are independent of each other
- T012, T013, T014 are independent of each other
