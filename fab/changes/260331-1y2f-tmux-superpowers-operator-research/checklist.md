# Quality Checklist: Tmux Superpowers — Operator Research

**Change**: 260331-1y2f-tmux-superpowers-operator-research
**Generated**: 2026-04-01
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [ ] CHK-001 Parent command registration: `fab pane` with no subcommand prints help listing `map`, `capture`, `send`, `process`
- [ ] CHK-002 Pane map renamed: `fab pane map` produces identical output to former `fab pane-map` with all flags (`--json`, `--session`, `--all-sessions`)
- [ ] CHK-003 Pane capture raw: `fab pane capture <pane> -l N` outputs last N lines as plain text
- [ ] CHK-004 Pane capture JSON: `--json` output includes `pane`, `lines`, `content`, `change`, `stage`, `agent_state` fields with correct types
- [ ] CHK-005 Pane capture no-fab context: JSON output has `null` for `change`, `stage`, `agent_state` when pane is not in a fab worktree
- [ ] CHK-006 Pane send safe mode: default send validates pane existence and agent idle state before sending
- [ ] CHK-007 Pane send force mode: `--force` skips idle check but still validates pane existence
- [ ] CHK-008 Pane send non-fab panes: non-fab panes treated as idle (send succeeds without `--force`)
- [ ] CHK-009 Pane process states: command reports one of `running`, `waiting-for-input`, `sleeping`, `stopped`, `exited`
- [ ] CHK-010 Pane process JSON: `--json` output includes `pane`, `pid`, `state`, `process_name`, `change`
- [ ] CHK-011 Cross-platform process detection: Linux uses `/proc/{pid}/stat` + `/proc/{pid}/wchan`; macOS uses `ps` + `lsof`
- [ ] CHK-012 Graceful degradation: ambiguous process states fall back to `sleeping` (no error)
- [ ] CHK-013 CLI docs updated: `_cli-fab.md` documents all four `fab pane` subcommands with full signatures
- [ ] CHK-014 External CLI docs updated: `_cli-external.md` no longer documents `tmux capture-pane` or `tmux send-keys`; only `tmux new-window` remains
- [ ] CHK-015 Operator skill updated: `fab-operator7.md` Section 3 uses `fab pane send`; Section 5 uses `fab pane process` pre-filter

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [ ] CHK-016 Pane map flag preservation: `--session` and `--all-sessions` remain mutually exclusive under `fab pane map`
- [ ] CHK-017 Send error messages: pane-not-found prints `"pane <pane> not found"` to stderr; active-agent prints `"agent in <pane> is active, use --force to override"` to stderr
- [ ] CHK-018 Send executes tmux correctly: sends `tmux send-keys -t <pane> "<text>" Enter` (text followed by Enter)
- [ ] CHK-019 Capture pane-not-found: exits code 1 with stderr message when pane does not exist
- [ ] CHK-020 Process pane-not-found: exits code 1 with stderr message when pane does not exist

## Removal Verification
<!-- Every deprecated requirement is actually gone -->
- [ ] CHK-021 Old pane-map removed: `fab pane-map` returns unknown command error (no alias, no backward compatibility)
- [ ] CHK-022 No dead pane-map registration: `main.go` no longer registers `paneMapCmd()` at root level

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [ ] CHK-023 Scenario — parent help: `fab pane` lists all four subcommands
- [ ] CHK-024 Scenario — JSON capture with active change: enriched JSON includes correct change, stage, agent_state
- [ ] CHK-025 Scenario — send blocked by active agent: command exits 1, no tmux send-keys executed
- [ ] CHK-026 Scenario — force send to active agent: command succeeds (exit 0)
- [ ] CHK-027 Scenario — detect waiting-for-input: `fab pane process` returns `waiting-for-input` for tty-blocked process
- [ ] CHK-028 Scenario — detect exited: `fab pane process` returns `exited` when only shell is running
- [ ] CHK-029 Scenario — operator pre-send simplified: Section 3 uses single `fab pane send` for steps 1-2
- [ ] CHK-030 Scenario — operator question detection: Section 5 runs `fab pane process` before capture-and-regex

## Edge Cases & Error Handling
<!-- Error states, boundary conditions, failure modes -->
- [ ] CHK-031 Capture with no `-l` flag: captures all visible lines (mirrors `tmux capture-pane` default)
- [ ] CHK-032 Send to non-existent pane with `--force`: still fails (pane existence always checked)
- [ ] CHK-033 Process with ambiguous wchan: returns `sleeping` not error
- [ ] CHK-034 Process on wrong platform: Linux code does not shell out to `ps`/`lsof`; macOS code does not read `/proc`
- [ ] CHK-035 Runtime file missing: `pane send` to pane with no `.fab-runtime.yaml` treats as idle (non-fab pane behavior)

## Code Quality
<!-- Baseline items plus project-specific principles and anti-patterns -->
- [ ] CHK-036 Pattern consistency: new `pane_*.go` files follow naming and structural patterns of `panemap.go` and other commands in `src/go/fab/cmd/fab/`
- [ ] CHK-037 No unnecessary duplication: reuses existing helpers (`gitWorktreeRoot`, `readFabCurrent`, `resolveAgentState`, `resolvePaneChange`, runtime loading) from `panemap.go`
- [ ] CHK-038 Readability over cleverness: process state detection logic is straightforward with clear state-to-condition mapping
- [ ] CHK-039 Follow existing patterns: Cobra command registration follows `statusCmd()`, `runtimeCmd()` parent-with-subcommands pattern
- [ ] CHK-040 No god functions: each command's `RunE` delegates to focused helpers (< 50 lines each)
- [ ] CHK-041 No magic strings: process states (`running`, `waiting-for-input`, etc.) defined as named constants
- [ ] CHK-042 No duplicated utilities: tmux interaction helpers (pane existence check, CWD resolution) shared across subcommands

## Security
<!-- Only include if the change has security surface -->
- [ ] CHK-043 Send-keys injection: `fab pane send` properly quotes/escapes the `<text>` argument to prevent tmux command injection
- [ ] CHK-044 Process introspection scope: `/proc` reads and `ps`/`lsof` calls are limited to the resolved foreground PID (no arbitrary PID access)

## Documentation Accuracy
<!-- Project-specific: extra_categories from config.yaml -->
- [ ] CHK-045 `_cli-fab.md` signatures match implementation: all flag names, positional args, defaults, and output formats match the actual Go code
- [ ] CHK-046 `_cli-external.md` accuracy: tmux section only lists `tmux new-window`, no stale references to `tmux capture-pane` or `tmux send-keys`
- [ ] CHK-047 `fab-operator7.md` accuracy: Section 3 and Section 5 reference correct `fab pane` subcommand names and flags

## Cross References
<!-- Project-specific: extra_categories from config.yaml -->
- [ ] CHK-048 `_cli-fab.md` to `_cli-external.md`: cross-references are consistent (external doc points to fab pane commands where appropriate)
- [ ] CHK-049 `fab-operator7.md` to `_cli-fab.md`: operator skill references match documented command signatures
- [ ] CHK-050 Test coverage cross-ref: every `fab pane` subcommand has a corresponding `_test.go` file

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
