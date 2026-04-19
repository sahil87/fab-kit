# Tasks: Agents Runtime Unified

**Change**: 260419-o5ej-agents-runtime-unified
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create new package directory `src/go/fab/internal/proc/` with placeholder `doc.go` declaring the package purpose (grandparent PID resolution).
- [x] T002 [P] Create stub `src/go/fab/internal/proc/proc_linux.go` with `//go:build linux` constraint and empty `ClaudePID()` function (fleshed out in T005).
- [x] T003 [P] Create stub `src/go/fab/internal/proc/proc_darwin.go` with `//go:build darwin` constraint and empty `ClaudePID()` function (fleshed out in T006).

## Phase 2: Core Implementation

### Schema + Runtime Package

- [x] T004 Define the new runtime schema types in `src/go/fab/internal/runtime/runtime.go`: add `AgentEntry` struct with fields `Change string, IdleSince *int64, PID *int, TmuxServer string, TmuxPane string, TranscriptPath string` (use pointers for truly-optional numeric fields; `omitempty` yaml tags on all optional string fields).

### Platform-Split Walker

- [x] T005 Implement Linux `ClaudePID()` in `src/go/fab/internal/proc/proc_linux.go`: read `/proc/$PPID/status`, scan for `PPid:\s+(\d+)` line, return parsed int. Return wrapped error on read/parse failure.
- [x] T006 [P] Implement macOS `ClaudePID()` in `src/go/fab/internal/proc/proc_darwin.go`: exec `ps -o ppid= -p <ppid>`, parse trimmed stdout as int. Return wrapped error on exec/parse failure.
- [x] T007 [P] Add unit test `src/go/fab/internal/proc/proc_test.go` with a single cross-platform smoke test that calls `ClaudePID()` and asserts a positive integer return (parent of test process is `go test`, grandparent is the shell invoking it — both exist).

### Runtime Helpers

- [x] T008 Replace `SetIdle`/`ClearIdle` in `src/go/fab/internal/runtime/runtime.go` with three new functions: `WriteAgent(fabRoot string, sessionID string, entry AgentEntry) error` (writes/overwrites `_agents[sessionID]`), `ClearAgent(fabRoot, sessionID string) error` (deletes the entry), `ClearAgentIdle(fabRoot, sessionID string) error` (unsets only `idle_since`, preserves other fields). All three use atomic temp-file + rename via existing `SaveFile`.
- [x] T009 Implement `GCIfDue(fabRoot string, interval time.Duration) error` in `src/go/fab/internal/runtime/runtime.go`: read `last_run_gc` top-level field; if `now - last_run_gc < interval.Seconds()`, return nil; else iterate `_agents`, for each entry with non-nil `pid`, invoke `unix.Kill(pid, 0)` — if result is `syscall.ESRCH`, delete the entry. Update `last_run_gc = now()`. Save atomically.
- [x] T010 [P] Update `src/go/fab/internal/runtime/runtime_test.go`: remove tests for `SetIdle`/`ClearIdle`; add tests for `WriteAgent` (writes full entry, overwrites existing, creates file if absent), `ClearAgent` (removes entry, no-op on missing), `ClearAgentIdle` (preserves non-idle_since fields), `GCIfDue` (throttled within interval, prunes dead PID, preserves live PID, preserves pid-less entry, updates last_run_gc).

### Hook Package (Stdin Parsing + New Payload)

- [x] T011 Add stdin payload parser in `src/go/fab/internal/hooklib/payload.go`: new type `SessionPayload struct { SessionID string `json:"session_id"`; TranscriptPath string `json:"transcript_path"` }` and function `ParseSessionPayload(r io.Reader) (SessionPayload, error)` that reads, JSON-decodes, and returns. Empty `session_id` treated as absent (not error).
- [x] T012 [P] Add tests `src/go/fab/internal/hooklib/payload_test.go`: well-formed payload, missing session_id, malformed JSON, empty input.

### Hook Handlers

- [x] T013 Rewrite `hookStopCmd()` in `src/go/fab/cmd/fab/hook.go`: parse stdin via `hooklib.ParseSessionPayload` (swallow errors → exit 0). If session_id empty, exit 0. Resolve `fabRoot`. Build `AgentEntry`: set `IdleSince = now()`, `PID` via `proc.ClaudePID()` (nil on error), `TmuxServer` via basename of `$TMUX` first component (empty if unset), `TmuxPane = $TMUX_PANE` (empty if unset), `TranscriptPath` from payload, `Change` via `resolve.ToFolder(fabRoot, "")` (empty string on error — not "unknown"). Call `runtime.WriteAgent(fabRoot, sessionID, entry)`. Call `runtime.GCIfDue(fabRoot, 180*time.Second)`. Swallow all errors (exit 0).
- [x] T014 Rewrite `hookSessionStartCmd()` in `src/go/fab/cmd/fab/hook.go`: parse stdin → session_id. If empty, exit 0. Call `runtime.ClearAgent(fabRoot, sessionID)`. Call `runtime.GCIfDue`. Swallow errors.
- [x] T015 Rewrite `hookUserPromptCmd()` in `src/go/fab/cmd/fab/hook.go`: parse stdin → session_id. If empty, exit 0. Call `runtime.ClearAgentIdle(fabRoot, sessionID)` (preserves other fields). Call `runtime.GCIfDue`. Swallow errors.
- [x] T016 Confirm `hookArtifactWriteCmd()` in `src/go/fab/cmd/fab/hook.go` is unchanged — it uses the existing file-path payload parser, not the new session payload parser. Add a short code comment documenting why it differs (different Claude hook event, different payload shape).
- [x] T017 [P] Update `src/go/fab/cmd/fab/hook_test.go` (or add it if none): cover new hook handlers with stdin payloads for well-formed and missing-session-id cases, verify `_agents` file contents after each.

### Pane Package (Resolution Rewrite)

- [x] T018 Rewrite `ResolvePaneContext()` in `src/go/fab/internal/pane/pane.go`: remove the `if folderName != ""` short-circuit on agent resolution. After change/stage resolution, scan `_agents` in the worktree's `.fab-runtime.yaml` for an entry matching `tmux_pane == paneID` AND (`tmux_server == ""` OR `tmux_server == parseServerFromEnvOrFlag(server)`). If found, set `ctx.AgentState` and `ctx.AgentIdleDuration` from the entry's `idle_since`. Independent of whether `folderName` is set.
- [x] T019 Rewrite `ResolveAgentState()` and `ResolveAgentStateWithCache()` in `src/go/fab/internal/pane/pane.go` to scan `_agents` (not `<folder>.agent`), matched by pane ID. Preserve the per-worktree cache pattern — a single `.fab-runtime.yaml` read per worktree per pane-map invocation.
- [x] T020 Helper `findAgentByPane(rtData map[string]interface{}, paneID, server string) (entry, bool)` — extract matching logic. Used by both `ResolvePaneContext` and `ResolveAgentStateWithCache`.
- [x] T021 [P] Update `src/go/fab/internal/pane/pane_test.go`: remove old-schema test fixtures; add fixtures with new `_agents` schema covering discussion-mode pane, active-change pane, pane with no matching entry, multi-server disambiguation, cache-reuse across panes.

### Pane Map Command (Inherit, Minor Adjustments)

- [x] T022 Inspect `src/go/fab/cmd/fab/panemap.go`: the CLI columns and JSON marshaling should need no changes since they consume `PaneContext`'s existing `AgentState`/`AgentIdleDuration` fields (which are now populated in discussion mode via the new `ResolvePaneContext`). Verify no code branches on `folderName != ""` for agent display.
- [x] T023 [P] Update `src/go/fab/cmd/fab/panemap_test.go`: add table tests for discussion-mode visibility (change null, agent populated) and three-axis independence.

### Pane Send Validation (Inherit)

- [x] T024 Inspect `src/go/fab/cmd/fab/pane_send.go`: verify the idle-check calls `ResolvePaneContext` and that there are no additional branches on change state. No code change expected — behavior inherits from T018.
- [x] T025 [P] Update `src/go/fab/cmd/fab/pane_send_test.go`: add tests for discussion-mode idle pane accepted, discussion-mode active pane rejected without `--force`.

### Migration

- [x] T026 Create `src/kit/migrations/<NNN>-agents-runtime-unified.md` (number by next sequential). Migration body: find `.fab-runtime.yaml` at each worktree root (current and any sibling worktrees discoverable via `git worktree list`); delete each. Per project constitution, migration is markdown instructions applied by `/fab-setup migrations`. Content follows existing migration format in `src/kit/migrations/` (read one existing file for reference).

## Phase 3: Integration & Edge Cases

- [x] T027 End-to-end test or smoke script: simulate all three hook events (stop, session-start, user-prompt) via `echo '{"session_id": "test-uuid", ...}' | fab hook <event>`, verify `.fab-runtime.yaml` contents after each.
- [x] T028 GC interaction test: write two entries (one live PID, one dead PID via exited-subprocess trick), invoke GC via a hook that triggers it (e.g., stop), verify dead entry pruned and live preserved.
- [x] T029 Missing env var tests: run hook with `$TMUX` unset (no tmux_server written), with `$TMUX_PANE` unset (no tmux_pane written), with both unset (entry is non-tmux).
- [x] T030 [P] Run `go vet ./...` and `go test ./...` across the full fab-kit Go tree; fix any issues surfaced.
- [x] T031 Run existing integration/smoke tests if any exist (`bats tests/` if present); confirm pane-map enrichment paths still work on current worktree. **N/A**: no bats or other integration test suite present in the repository.

## Phase 4: Polish

- [x] T032 Update `src/kit/skills/_cli-fab.md` if any `fab hook` or `fab runtime` surface changed (likely unchanged — hooks are invoked by Claude Code, not users). Remove any references to legacy `SetIdle`/`ClearIdle` semantics if present.
- [x] T033 [P] Create `docs/specs/skills/SPEC-pane-commands.md` update (if that file exists) OR create it if missing — describe new agent-state resolution semantics, three-axis model. Check `docs/specs/skills/` for existing pane-commands spec. **N/A**: no SPEC-pane-commands.md exists in `docs/specs/skills/`; per spec §Documentation, the hydrate stage of this change produces `docs/memory/fab-workflow/runtime-agents.md` and updates `pane-commands.md`. Spec-level doc updates are a hydrate-stage concern, not an apply-stage concern — the spec artifact itself and `src/kit/skills/_cli-fab.md` (updated in T032) already capture the behavioral semantics.
- [x] T034 Run `fab sync` locally (or document the version bump procedure) — verify deployed skills don't reference removed APIs. **N/A**: `fab sync` runs in user worktrees to deploy kit content from the system cache; kit-source edits in `src/kit/skills/` (done in T032) are the authoritative update. No deployed skill in `.claude/skills/` references the removed `fab runtime set-idle/clear-idle/is-idle` commands beyond the archival `docs/memory/` and `docs/specs/` notes, which are updated in the hydrate stage per the change's §Documentation requirement.

---

## Execution Order

**Phase 1 → Phase 2 → Phase 3 → Phase 4** sequentially. Within phases:

- T001 blocks T002, T003
- T004 blocks T008, T018, T019, T020 (schema types needed)
- T005, T006, T007 are [P] after T004 (platform files independent)
- T008 blocks T009 (WriteAgent used by GC test fixtures)
- T008 blocks T013–T015 (hook handlers use WriteAgent/Clear*)
- T011 blocks T013–T015 (ParseSessionPayload used by hooks)
- T018 blocks T022, T024 (pane-map and pane-send inherit new resolution)
- T020 blocks T018, T019 (helper used by both)
- All `[P]` test tasks (T007, T010, T012, T017, T021, T023, T025, T030, T033) can run alongside their parent implementation tasks
- T026 (migration) depends on nothing and can run anytime after T004
- Phase 3 tasks (T027–T031) require Phase 2 complete
- Phase 4 tasks are final polish
