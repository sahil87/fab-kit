# Quality Checklist: Agents Runtime Unified

**Change**: 260419-o5ej-agents-runtime-unified
**Generated**: 2026-04-19
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Runtime schema `_agents[session_id]` structure: `.fab-runtime.yaml` uses top-level `_agents` map + `last_run_gc` field; entries have `idle_since` when idle, plus optional `change`/`pid`/`tmux_server`/`tmux_pane`/`transcript_path`
- [x] CHK-002 Schema location invariants: `.fab-runtime.yaml` lives at each worktree repo root; missing file is equivalent to empty `_agents` and does not error on read
- [x] CHK-003 Hook stdin JSON parsing: `fab hook stop|session-start|user-prompt` extract `session_id`; malformed JSON or missing field exits 0 silently
- [x] CHK-004 Stop hook write semantics: writes `_agents[session_id]` with `idle_since = now()` and all currently-available optional fields
- [x] CHK-005 Session-start hook semantics: deletes `_agents[session_id]` entirely
- [x] CHK-006 User-prompt hook semantics: removes only `idle_since`; preserves `change`, `pid`, `tmux_server`, `tmux_pane`, `transcript_path`
- [x] CHK-007 Hook env read: `$TMUX_PANE` → `tmux_pane`, basename of `$TMUX` first component → `tmux_server`, omitted when unset
- [x] CHK-008 Grandparent PID resolution: Linux reads `/proc/$PPID/status` PPid field; macOS execs `ps -o ppid= -p $PPID`; both files return same signature
- [x] CHK-009 Grandparent failure: walker error causes hook to omit `pid` field but not fail
- [x] CHK-010 GC throttle: `GCIfDue` returns no-op when `now - last_run_gc < interval`; otherwise sweeps and updates
- [x] CHK-011 GC liveness: entries with `pid` field + dead process (ESRCH) are pruned; entries with live pid preserved; entries without `pid` preserved
- [x] CHK-012 Pane map resolution: agent state resolves by matching `_agents[*].tmux_pane` to pane ID, independent of active-change state
- [x] CHK-013 Server disambiguation: `--server <name>` narrows agent matching to entries whose `tmux_server` matches or is empty
- [x] CHK-014 JSON output fields: `agent_state`/`agent_idle_duration` populate for discussion-mode panes; `change`/`stage` remain null when no active change
- [x] CHK-015 Cache discipline: `.fab-runtime.yaml` read at most once per worktree per `fab pane map` invocation
- [x] CHK-016 Pane send validation: `fab pane send` accepts discussion-mode idle panes, rejects active panes without `--force`
- [x] CHK-017 Clean-slate migration: migration file in `src/kit/migrations/` deletes `.fab-runtime.yaml` at each worktree; idempotent

## Behavioral Correctness

- [x] CHK-018 Discussion-mode visibility: `fab pane map` pane with no active change + running Claude shows `idle (<dur>)` or `active` instead of `—`
- [x] CHK-019 Change-mode pane unchanged: `fab pane map` pane with active change renders Change/Stage/Agent columns with same data as before (no regression)
- [x] CHK-020 Non-tmux agent tracking: Claude invoked outside tmux still produces `_agents[session_id]` entry (with omitted `tmux_*` fields); `fab pane map` shows `—` for panes since no match

## Removal Verification

- [x] CHK-021 Legacy `SetIdle`/`ClearIdle` removed from `src/go/fab/internal/runtime/runtime.go` (no dead code, no wrapper shims)
- [x] CHK-022 Legacy per-folder `agent:` block handling removed from `src/go/fab/internal/pane/pane.go` (`ResolveAgentState`/`ResolveAgentStateWithCache` no longer scan `<folder>.agent`)
- [x] CHK-023 Legacy test fixtures (per-folder schema) removed from `internal/runtime/runtime_test.go` and `internal/pane/pane_test.go`

## Scenario Coverage

- [x] CHK-024 Scenario "Active change with tmux agent": covered by test in `internal/runtime/runtime_test.go`
- [x] CHK-025 Scenario "Discussion-mode agent without change": covered by test in `internal/runtime/runtime_test.go` + `cmd/fab/hook_test.go`
- [x] CHK-026 Scenario "Non-tmux agent": covered by test asserting omitted `tmux_*` fields when env unset
- [x] CHK-027 Scenario "Stop / session-start / user-prompt hook semantics": covered by `cmd/fab/hook_test.go` for all three events
- [x] CHK-028 Scenario "GC prunes dead PID / preserves live PID / preserves pid-less": covered by `internal/runtime/runtime_test.go`
- [x] CHK-029 Scenario "Discussion-mode pane renders in fab pane map": covered by `cmd/fab/panemap_test.go`
- [x] CHK-030 Scenario "Server disambiguation": covered by `internal/pane/pane_test.go`
- [x] CHK-031 Scenario "Migration idempotent": covered by migration text (pre-check + idempotent instructions); no executable migration test harness exists in this repo

## Edge Cases & Error Handling

- [x] CHK-032 Malformed stdin to hook: exits 0 silently (no panic, no partial write)
- [x] CHK-033 Missing `session_id` in payload: exits 0 silently
- [x] CHK-034 First write creates file: hook writing to a worktree with no `.fab-runtime.yaml` succeeds and creates file atomically
- [x] CHK-035 Concurrent hook writes: atomic write via temp-file + rename ensures file is never partially written (behavior inherited from existing `SaveFile`)
- [x] CHK-036 Grandparent walker on exited parent: returns error; hook proceeds without `pid` field

## Code Quality

- [x] CHK-037 Pattern consistency: new code follows existing codebase conventions (naming, error wrapping, yaml tags, subprocess patterns)
- [x] CHK-038 No unnecessary duplication: reuses `SaveFile`/`LoadFile` from existing runtime package; reuses existing `pane_process_{linux,darwin}.go` pattern
- [x] CHK-039 Readability over cleverness: helper functions (e.g., `findAgentByPane`) named clearly; no inline complex logic
- [x] CHK-040 Existing project patterns: platform-split via build tags matches `pane_process_{linux,darwin}.go`; yaml tags match existing runtime schema; atomic file writes match existing `SaveFile`
- [x] CHK-041 Composition over inheritance: runtime package exposes focused functions (`WriteAgent`, `ClearAgent`, `ClearAgentIdle`, `GCIfDue`) rather than a runtime manager struct
- [x] CHK-042 Anti-pattern "God functions": hook handlers delegate to `runtime.*` helpers; no single function exceeds ~50 lines
- [x] CHK-043 Anti-pattern "Duplicating utilities": no re-implemented yaml parsing, process liveness, or env parsing — uses stdlib and existing helpers
- [x] CHK-044 Anti-pattern "Magic strings": env var names (`TMUX`, `TMUX_PANE`), field names (`_agents`, `last_run_gc`, `idle_since`), and signal (`0` for kill liveness) are used as named constants where repeated

## Documentation Accuracy

- [x] CHK-045 New memory file `docs/memory/fab-workflow/runtime-agents.md` created with full schema, hook pipeline, GC, grandparent walker, matching rule
- [x] CHK-046 `docs/memory/fab-workflow/pane-commands.md` updated: three-axis model, new matching rule, discussion-mode scenarios in output tables
- [x] CHK-047 `docs/memory/fab-workflow/schemas.md` updated with cross-reference to runtime-agents
- [x] CHK-048 `docs/memory/fab-workflow/index.md` lists runtime-agents entry with short description
- [x] CHK-049 Migration file in `src/kit/migrations/` documents the schema change and clean-slate rationale

## Cross-References

- [x] CHK-050 `docs/memory/fab-workflow/pane-commands.md` cross-references runtime-agents where appropriate
- [x] CHK-051 Spec for per-skill behavior (if present in `docs/specs/skills/`) is updated to reflect new pane-map semantics — N/A: no SPEC-pane-commands.md exists (T033 notes); `src/kit/skills/_cli-fab.md` is updated with new semantics

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-000 **N/A**: {reason}`
