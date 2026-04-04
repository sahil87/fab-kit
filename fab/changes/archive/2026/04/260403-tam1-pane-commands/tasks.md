# Tasks: Fab Pane Command Group

**Change**: 260403-tam1-pane-commands
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/pane/` package with `pane.go` — define `PaneContext` struct and export `ValidatePane()`, `ResolvePaneContext()`, `GetPanePID()`, `FindMainWorktreeRoot()` functions. Extract resolution logic from `src/go/fab/cmd/fab/panemap.go` (functions: `resolvePane`, `gitWorktreeRoot`, `worktreeDisplayPath`, `readFabCurrent`, `resolveAgentState`, `loadPaneMapRuntimeFile`, `formatIdleDuration`, `findMainWorktreeRoot`)
- [x] T002 Create `src/go/fab/internal/pane/pane_test.go` — unit tests for `ValidatePane` (mock tmux output), `FormatIdleDuration`, `WorktreeDisplayPath`, and other pure functions extracted from panemap_test.go
- [x] T003 Create `src/go/fab/cmd/fab/pane.go` — `paneCmd()` returning `*cobra.Command` with `Use: "pane"`, `Short: "Tmux pane operations"`, registering four subcommands. Update `src/go/fab/cmd/fab/main.go` to replace `paneMapCmd()` with `paneCmd()`

## Phase 2: Core Implementation

- [x] T004 Refactor `src/go/fab/cmd/fab/panemap.go` — change `paneMapCmd()` `Use` field from `"pane-map"` to `"map"`. Replace inline resolution logic with calls to `internal/pane` package functions. Remove duplicated helper functions (now in `internal/pane`)
- [x] T005 Update `src/go/fab/cmd/fab/panemap_test.go` — adjust tests for any function signatures that changed during extraction. Ensure all existing tests still pass
- [x] T006 [P] Create `src/go/fab/cmd/fab/pane_capture.go` — `paneCaptureCmd()` with positional `<pane>` arg, `-l` (int, default 50), `--json`, `--raw` (mutually exclusive). Implements: validate pane, capture via `tmux capture-pane -t <pane> -p -l <N>`, resolve context via `pane.ResolvePaneContext()`, format output (default header+content, JSON, raw)
- [x] T007 [P] Create `src/go/fab/cmd/fab/pane_send.go` — `paneSendCmd()` with positional `<pane>` and `<text>` args, `--no-enter`, `--force`. Implements: validate pane, resolve context, check agent idle (reject if active/unknown unless --force), execute `tmux send-keys -t <pane> "<text>" Enter`
- [x] T008 [P] Create `src/go/fab/cmd/fab/pane_process.go` — `paneProcessCmd()` with positional `<pane>` arg, `--json`. Implements: validate pane, get PID via `pane.GetPanePID()`, discover process tree (platform-specific), classify processes, format output
- [x] T009 Create `src/go/fab/cmd/fab/pane_process_linux.go` (`//go:build linux`) — Linux process tree discovery via `/proc/<pid>/task/<tid>/children` recursive walk, reading `/proc/<pid>/comm` and `/proc/<pid>/cmdline`
- [x] T010 Create `src/go/fab/cmd/fab/pane_process_darwin.go` (`//go:build darwin`) — macOS process tree discovery via `ps -o pid,ppid,comm -ax` with PPID traversal, `ps -o args= -p <pid>` for cmdline

## Phase 3: Integration & Tests

- [x] T011 [P] Create `src/go/fab/cmd/fab/pane_capture_test.go` — test JSON output structure, raw output passthrough, mutually exclusive flags, invalid pane error
- [x] T012 [P] Create `src/go/fab/cmd/fab/pane_send_test.go` — test idle validation rejection, force flag bypass, no-enter flag, non-existent pane error
- [x] T013 [P] Create `src/go/fab/cmd/fab/pane_process_test.go` — test process classification logic, JSON output structure, has_agent detection, tree nesting

## Phase 4: Skill File Updates

- [x] T014 [P] Update `src/kit/skills/_cli-fab.md` — replace `## fab pane-map` section with `## fab pane` section documenting the parent command and all four subcommands (map, capture, send, process). Update Command Reference table entry
- [x] T015 [P] Update `src/kit/skills/_cli-external.md` — remove `capture-pane` and `send-keys` rows from tmux Commands table. Update Usage Notes to reference `fab pane capture` and `fab pane send`. Keep `new-window` row

---

## Execution Order

- T001 blocks T002, T004, T006, T007, T008 (all depend on `internal/pane` package)
- T003 blocks T004 (pane parent must exist before map subcommand registration)
- T004 blocks T005 (test updates follow refactor)
- T008 blocks T009, T010 (platform files implement interface used by T008)
- T006, T007, T008 are independent of each other ([P] within phase)
- T011, T012, T013 are independent of each other ([P])
- T014, T015 are independent of each other ([P])
