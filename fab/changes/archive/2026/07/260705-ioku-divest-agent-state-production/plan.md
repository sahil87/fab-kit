# Plan: Divest Agent Active/Idle State Production from fab-kit

**Change**: 260705-ioku-divest-agent-state-production
**Intake**: `intake.md`

## Requirements

### Agent-State Convention: Read-Side Contract

#### R1: fab reads the `@rk_agent_state` tmux pane option
fab SHALL determine an agent's lifecycle state by reading the tmux **pane user option** `@rk_agent_state`, whose value is `"<state>:<epoch_seconds>"` where `state ∈ {active, waiting, idle}`. fab SHALL parse this with plain tmux commands and never depend on run-kit software being installed. The epoch suffix is mandatory; a value without it, an unknown state token, or an absent option is treated as **unknown**.

- **GIVEN** a pane whose `@rk_agent_state` is `idle:1751800000`
- **WHEN** any reader (`pane send`/`map`/`capture`) resolves that pane's agent state
- **THEN** the state is `idle` and the idle duration is `now - 1751800000` formatted via `FormatIdleDuration`
- **AND** a pane with no `@rk_agent_state` option resolves to unknown (`—` / em-dash in displays)

#### R2: parsing is a pure, tmux-free helper
The `"<state>:<epoch>"` parse SHALL live in a pure function in `internal/pane/pane.go` that maps a raw option value to `(state, epoch, ok)`, so it is unit-testable without a tmux server. `ok` is false for an empty value, a missing `:epoch` suffix, a non-integer epoch, or a state token outside `{active, waiting, idle}`.

- **GIVEN** raw values `""`, `"active"`, `"idle:notanum"`, `"bogus:123"`, `"idle:1751800000"`
- **WHEN** the parser runs on each
- **THEN** only `"idle:1751800000"` returns ok=true (`idle`, `1751800000`); all others return ok=false

### Delete the `_agents` Producer Pipeline

#### R3: the hook state-tracking pipeline is removed
The `_agents` write pipeline SHALL be deleted: `fab hook stop|user-prompt|session-start` stop writing agent state and become one-release no-op exit-0 shims that emit nothing. `WriteAgent`/`ClearAgent`/`ClearAgentIdle`/`UpdateAgent`/`GCIfDue`/`gcSweepIfDue`, the `last_run_gc` throttle, the grandparent PID walker (`internal/proc/`), and the `internal/runtime/` package (`.fab-runtime.yaml` read/write) SHALL be deleted wholesale.

- **GIVEN** any invocation of `fab hook stop`, `fab hook user-prompt`, or `fab hook session-start`
- **WHEN** the shim runs (with any stdin, inside or outside tmux)
- **THEN** it exits 0, writes nothing to stdout, and creates/touches no `.fab-runtime.yaml`

#### R4: `internal/proc/` and `internal/runtime/` are deleted; `internal/lockfile` stays
The `internal/proc/` and `internal/runtime/` packages (and their tests) SHALL be removed. The comment-only reference to `internal/proc` in `internal/dispatch/dispatch.go` SHALL be swept. `internal/lockfile` SHALL remain (consumed by `status.go`, `preflight.go`, `score.go` for `.status.yaml` serialization); only the runtime-lock usage is removed with the runtime package.

- **GIVEN** the built binary after this change
- **WHEN** `go build ./...` and `go vet ./...` run in `src/go/fab`
- **THEN** there is no reference to `internal/proc` or `internal/runtime`, and `internal/lockfile` still compiles and is imported by the status/preflight/score packages

#### R5: `FormatIdleDuration` survives in `internal/pane`
`FormatIdleDuration` SHALL remain in `internal/pane/pane.go` — it formats the epoch-derived idle durations of the new readers. The now-dead resolvers (`ResolveAgentState`, `ResolveAgentStateWithCache`, `findAgentByPane`, `loadRuntimeForCache`, `LoadRuntimeFile`, and the runtime schema-key constants) SHALL be deleted.

- **GIVEN** `internal/pane/pane.go` after this change
- **WHEN** the package compiles
- **THEN** `FormatIdleDuration` is present and exercised; `LoadRuntimeFile`/`ResolveAgentState*`/`findAgentByPane` are absent

### Rewrite the Three Readers

#### R6: `fab pane send` three-state idle gate
`fab pane send` SHALL read `@rk_agent_state` via `tmux [-L <server>] show-options -pv -t <pane> @rk_agent_state`. `idle` → send. `active`/`waiting` → refuse with the same error shape as today, now three-state aware (the state name appears in the message). Absent/unparseable → refuse with a **distinct** "unknown" message pointing the caller at `--force`. `--force` bypasses only the state check; pane existence is still enforced via the targeted probe.

- **GIVEN** a pane whose `@rk_agent_state` is set to `idle:<epoch>` (writer simulated via `tmux set-option -p`)
- **WHEN** `fab pane send <pane> <text>` runs without `--force`
- **THEN** the send succeeds
- **AND** with `active:<epoch>` or `waiting:<epoch>` it refuses with `agent in pane <id> is not idle (state: <state>)` (exit 1)
- **AND** with the option absent it refuses with a distinct unknown-state message naming `--force` (exit 1)
- **AND** `--force` sends regardless of state, but a missing pane still exits 2

#### R7: `fab pane map` Agent column via the list-panes format string
`fab pane map` SHALL add `#{@rk_agent_state}` to the **existing** `list-panes -F` format string (zero extra subprocesses; server disambiguation evaporates because a pane option lives on exactly one server's pane). Column values: `active` / `waiting` / `idle (<duration>)` / `—`. Duration comes from the epoch suffix via `FormatIdleDuration`. The per-worktree runtime cache and the `_agents`-matching resolution are removed.

- **GIVEN** panes carrying `active:<e>`, `waiting:<e>`, `idle:<e>`, and no option
- **WHEN** `fab pane map` renders
- **THEN** the Agent column shows `active`, `waiting`, `idle (<dur>)`, and `—` respectively
- **AND** no additional tmux subprocess beyond the single `list-panes` call is spawned for agent state

#### R8: `fab pane capture` header reads the same option
`fab pane capture` SHALL resolve agent state from `@rk_agent_state` (same read as send) and display it in the header block identically to today's shape (`agent: <state>` or `agent: idle (<dur>)`).

- **GIVEN** a pane with `@rk_agent_state = waiting:<epoch>`
- **WHEN** `fab pane capture <pane>` runs (non-JSON)
- **THEN** the header includes `agent: waiting`

#### R9: JSON field names preserved; `waiting` value added
`pane map --json` and `pane capture --json` SHALL keep the field names `agent_state` and `agent_idle_duration`. `agent_state` SHALL gain the `waiting` value. `agent_idle_duration` is populated only for `idle`; it is null for `active`/`waiting`/unknown. Unknown maps `agent_state` to null (unchanged shape).

- **GIVEN** a pane with `@rk_agent_state = waiting:<epoch>` and one with `idle:<epoch>`
- **WHEN** `pane map --json` / `pane capture --json` render
- **THEN** the `waiting` pane emits `agent_state: "waiting"`, `agent_idle_duration: null`; the idle pane emits `agent_state: "idle"` with a non-null `agent_idle_duration`

### Settings Migration + Sync Registration

#### R10: migration removes the three hook settings entries
A new migration file `src/kit/migrations/2.13.6-to-2.14.0.md` SHALL remove the three session-scoped hook entries (`SessionStart` → `fab hook session-start`, `Stop` → `fab hook stop`, `UserPromptSubmit` → `fab hook user-prompt`) from `.claude/settings.local.json`. It SHALL be sentinel-guarded and idempotent, modeled on `2.10.1-to-2.11.0.md`, and preserve any unrelated custom commands and non-hook top-level keys.

- **GIVEN** a `.claude/settings.local.json` still registering the three entries
- **WHEN** the migration applies
- **THEN** the three entries are gone, unrelated keys/entries are preserved, and re-running is a complete no-op (sentinel trips)

#### R11: `hooklib.Sync` registration list empties; `Sync` retained
`hooklib.DefaultMappings` (fab) and `defaultHookMappings` (fab-kit) SHALL be emptied of the three entries. `hooklib.Sync` / `syncHooks` themselves SHALL be retained for one release — they merge the desired entries into settings (deduplicated by matcher+command; now nothing, since the mapping tables are empty) with **no removal path** (Sync never deletes stale fab-managed entries). The legacy `on-*.sh` → `fab hook …` rewrite rows are also dropped (cycle 2, T011) so Sync can no longer re-mint the very entries the 2.14.0 migration deletes; the migration removes both the inline and legacy forms. Full removal of the sync path is a follow-up. y022's `artifact-write` shim and orphaned hooklib funcs are out of scope.

- **GIVEN** `fab hook sync` (and fab-kit's sync) after this change
- **WHEN** it runs against a fresh settings file
- **THEN** it registers zero session-scoped hook entries and reports the OK/created line accordingly

### Docs & Spec Mirror Sweep

#### R12: SPEC mirrors and aggregate specs updated
The touched skills' SPEC mirrors (`docs/specs/skills/SPEC-hooks.md`, `SPEC-fab-operator.md`, `SPEC-_cli-fab.md`) and aggregate specs restating agent-state facts (`docs/specs/architecture.md`, and `skills.md`/`glossary.md` where they carry the old claims) SHALL be swept to the convention-reader model. The kit skills `src/kit/skills/_cli-fab.md` (fab hook + pane send/map/capture sections) and `src/kit/skills/fab-operator.md` (three-state vocabulary, `waiting`-triggered 90s cadence, pre-send checks) SHALL be updated. `docs/memory/` is NOT rewritten here (hydrate's job). Historical `docs/specs/findings/*` dated review artifacts are left as-is (point-in-time records).

- **GIVEN** `rg "idle_since|_agents|fab-runtime" src/` after this change
- **WHEN** the results are inspected
- **THEN** only convention-reader code and migration shims remain (no producer-pipeline claims in `src/kit/`)

#### R13: `fab-operator` three-state vocabulary + waiting cadence
`src/kit/skills/fab-operator.md` (and its SPEC mirror) SHALL update the two-state active/idle vocabulary to three states + unknown, treat `waiting` as the trigger for the existing tightened 90s heartbeat cadence, and reflect the three-state gate in the pre-send checks.

- **GIVEN** a monitored agent reading `waiting`
- **WHEN** a tick detects it
- **THEN** the operator skill directs tightening the heartbeat to 90s (the same cadence trigger previously keyed on capture-based menu detection)

### Non-Goals

- No staleness heuristic in v1 readers — a stale `active` still refuses sends; `--force` is the escape hatch (A16).
- No change to `pane map`'s Change/Stage/display_state/pr_url columns (sourced from `.fab-status.yaml`, not `_agents`).
- No `docs/memory/` rewrites (hydrate owns those).
- y022's pending deletions (`artifact-write` shim + orphaned hooklib funcs) untouched.

### Design Decisions

1. **Pure parser `parseAgentState`**: approach — a single `parseAgentState(raw string) (state string, epoch int64, ok bool)` in `internal/pane/pane.go`, consumed by all three readers. *Why*: tmux-free unit tests, one authority for the `"<state>:<epoch>"` grammar. *Rejected*: parsing inline at each reader (three drifting copies).
2. **`pane map` reads via the format string; `send`/`capture` via `show-options -pv`**: *Why*: map already runs `list-panes -F`, so `#{@rk_agent_state}` is zero-cost; send/capture operate on a single pane and already probe it, so a targeted `show-options -pv` is the minimal read. *Rejected*: a `show-options` per pane in map (extra subprocess per pane — the intake explicitly forbids it).
3. **`waiting`/`active`/unknown carry no idle duration**: only `idle` computes a duration from the epoch. *Why*: duration is meaningful only for a completed turn; matches today's `active` (no duration) semantics.

## Tasks

### Phase 1: Reader convention parser (foundation)

- [x] T001 Add `parseAgentState(raw string) (state string, epoch int64, ok bool)` and a `resolveAgentDisplay`-style helper (state + optional `idle (<dur>)`) to `src/go/fab/internal/pane/pane.go`; keep `FormatIdleDuration`. <!-- R2 R5 -->
- [x] T002 Delete the dead resolvers from `src/go/fab/internal/pane/pane.go`: `ResolveAgentState`, `ResolveAgentStateWithCache`, `findAgentByPane`, `loadRuntimeForCache`, `LoadRuntimeFile`, `asInt64`, and the `_agents`/`idle_since`/`tmux_pane`/`tmux_server` schema-key constants (keep `WithServer`, `RunCmd`, `StderrError`, `IsPaneMissing`, `PaneNotFoundError`, `ValidatePane`, `ReadWindowName`, `GetPanePID`, `FindMainWorktreeRoot`, `GitWorktreeRoot`, `WorktreeDisplayPath`, `ReadFabCurrent`, `FormatIdleDuration`). <!-- R5 -->

### Phase 2: Rewrite the three readers

- [x] T003 Rewrite the agent-state resolution inside `ResolvePaneContext` (`src/go/fab/internal/pane/pane.go`) to read `@rk_agent_state` via `tmux show-options -pv -t <pane> @rk_agent_state` and set `AgentState` (`active`/`waiting`/`idle`, nil when unknown) + `AgentIdleDuration` (only for idle). <!-- R1 R6 R8 --> <!-- rework: cycle 1 must-fix — resolve agent state BEFORE the not-a-git-repo / no-fab-dir early returns (pane.go:213,223) so send/map/capture agree on non-fab panes -->
- [x] T004 Rewrite `fab pane send` idle gate in `src/go/fab/cmd/fab/pane_send.go`: accept only `idle`; refuse `active`/`waiting` with `agent in pane <id> is not idle (state: <state>)`; refuse unknown with a distinct message naming `--force`; `--force` bypasses the state check only. <!-- R6 --> <!-- rework: cycle 2 nice-to-have — the unknown refusal (pane_send.go:87) says "(no @rk_agent_state option)" even when the option is present but malformed; reword to "(missing or unparseable @rk_agent_state)" and update the pinned message tests -->
- [x] T005 Extend `tmuxPaneFormat` in `src/go/fab/cmd/fab/panemap.go` with a 6th tab-delimited field `#{@rk_agent_state}`, thread it through `paneEntry`/`parsePaneLines`, and compute the Agent column (`active`/`waiting`/`idle (<dur>)`/`—`) from it via the Phase-1 helper; remove the `runtimeCache` and the `ResolveAgentStateWithCache` call. <!-- R7 --> <!-- rework: cycle 1 must-fix — non-git row (panemap.go:308-321) hardcodes the em-dash; use agentColumn(p.agentState) so map matches send on non-git panes -->
- [x] T006 Update `splitAgentState` (or its replacement) in `panemap.go` so `--json` emits `agent_state` ∈ `{active, waiting, idle, null}` and `agent_idle_duration` only for idle. <!-- R9 --> <!-- rework: cycle 2 nice-to-have — --json (panemap.go:455-466,472) re-parses the human display string (agentColumn → splitAgentState) instead of emitting from the structured state/duration pair; refactor so a display-format tweak cannot silently break the JSON contract run-kit consumes -->
- [x] T007 Confirm `fab pane capture` header/JSON (`src/go/fab/cmd/fab/pane_capture.go`) renders the new state via `ResolvePaneContext` (no code change expected beyond what T003 provides; adjust if the `waiting` value needs explicit handling). <!-- R8 R9 -->

### Phase 3: Delete the producer pipeline

- [x] T008 Convert `fab hook stop|user-prompt|session-start` in `src/go/fab/cmd/fab/hook.go` to no-op exit-0 shims (emit nothing, consume nothing); remove `WriteAgent`/`ClearAgent`/`ClearAgentIdle` calls, `buildAgentEntry`, `resolveClaudePID`, `resolveActiveChangeFolder`, `parseTmuxServer`, `gcInterval`, and the `proc`/`runtime`/`resolve` imports no longer used; keep `hookArtifactWriteCmd` (y022 scope) and `hookSyncCmd`. <!-- R3 -->
- [x] T009 Delete the `src/go/fab/internal/runtime/` package (`runtime.go` + `runtime_test.go`) and the `src/go/fab/internal/proc/` package (`doc.go`, `proc_linux.go`, `proc_darwin.go`, `proc_test.go`). <!-- R3 R4 -->
- [x] T010 Sweep the comment-only `internal/proc` reference in `src/go/fab/internal/dispatch/dispatch.go`. <!-- R4 --> <!-- rework: cycle 1 should-fix — sibling missed: dispatch_posix.go:43 still cites deleted internal/runtime.pidAlive; also sweep the stale "registers inline fab hook commands" comment at src/go/fab-kit/internal/sync.go:96 --> <!-- rework: cycle 2 should-fix — two more stale runtime.SaveFile comment cites: internal/statusfile/statusfile.go:342 and internal/change/change.go:176; grep "runtime\." comment cites repo-wide to close the class -->
- [x] T011 Empty the session-scoped entries from `hooklib.DefaultMappings` (`src/go/fab/internal/hooklib/sync.go`) and `defaultHookMappings` (`src/go/fab-kit/internal/hooksync.go`), retaining `Sync`/`syncHooks` and the legacy-script migration map; update the surrounding comments. <!-- R11 --> <!-- rework: cycle 2 should-fix — the retained legacy map (sync.go:43,185; hooksync.go:153) still rewrites on-{stop,user-prompt,session-start}.sh into exactly the three fab-hook entries the 2.14.0 migration deletes (re-minting hazard): DROP those three legacy rewrite rows (keep any others, e.g. artifact-write), and update sync tests accordingly -->

### Phase 4: Migration

- [x] T012 Create `src/kit/migrations/2.13.6-to-2.14.0.md` (sentinel-guarded, idempotent, modeled on `2.10.1-to-2.11.0.md`) removing the three session-scoped hook entries from `.claude/settings.local.json`, preserving unrelated custom commands and non-hook keys. <!-- R10 --> <!-- rework: cycle 1 should-fix — migration must also delete the now-unread .fab-runtime.yaml / .fab-runtime.yaml.lock files (precedent: 1.4.0-to-1.5.0.md deleted .fab-runtime.yaml across worktrees) --> <!-- rework: cycle 2 should-fix — with the legacy rewrite rows dropped (T011), extend the migration to ALSO remove lingering legacy on-{stop,user-prompt,session-start}.sh session-hook entries, so neither form survives -->

### Phase 5: Tests

- [x] T013 [P] Add unit tests for `parseAgentState` (and the display helper) in `src/go/fab/internal/pane/pane_test.go`; delete the `_agents`-fixture tests (`TestLoadRuntimeFile`, `TestResolveAgentState`, `TestResolveAgentStateWithCache`, `TestFindAgentByPane`, `writeRuntimeFixture`, `isIdlePrefix`). <!-- R2 R5 -->
- [x] T014 [P] Update `src/go/fab/cmd/fab/hook_test.go`: delete `_agents`-write assertions (Stop/SessionStart/UserPrompt tests), keep/adjust `TestParseTmuxServer` only if the function survives (it does not — delete it), keep the sync tests, and add shim-is-silent-no-op tests for stop/user-prompt/session-start mirroring `TestHookArtifactWrite_ShimIsSilentNoOp`. <!-- R3 -->
- [x] T015 [P] Update `src/go/fab/cmd/fab/panemap_test.go` and `pane_send_test.go`/`pane_capture_test.go` for the new state model; add reader tests that simulate the writer via `tmux set-option -p -t <pane> @rk_agent_state "<state>:<epoch>"` on the test tmux server (follow the existing tmux-server test pattern where present, else keep pure-helper coverage). <!-- R6 R7 R8 R9 --> <!-- rework: cycle 1 must-fix — pane_send.go:51-56 three-state gate has ZERO test coverage (A-014 unchecked): add gate tests covering idle-send / active-refuse / waiting-refuse / unknown-distinct-refusal / --force bypass using the tmux set-option writer simulation --> <!-- rework: cycle 2 should-fix+nice — pane_send_test.go:117 strptr duplicates strPtr (panemap_test.go:1286), reuse the existing one; add an explicit waiting-state header case to pane_capture_test.go (R8 GIVEN/THEN) -->
- [x] T016 [P] Update `src/go/fab-kit/internal/hooksync_test.go` and `src/go/fab/internal/hooklib/sync_test.go` to expect zero session-scoped registrations (retain legacy-migration coverage). <!-- R11 -->

### Phase 6: Docs & spec mirror sweep

- [x] T017 Rewrite `docs/specs/skills/SPEC-hooks.md` to the convention-reader model: the three session hooks are now no-op shims, `.fab-runtime.yaml`/`_agents` producer schema removed, event-coverage table updated. <!-- R12 -->
- [x] T018 Update `src/kit/skills/_cli-fab.md` (fab hook table + prose; pane send/map/capture sections) and `docs/specs/skills/SPEC-_cli-fab.md` (fab hook row) to the `@rk_agent_state` reader model + `waiting` value + three-state gate. <!-- R12 --> <!-- rework: cycle 1 nice-to-have — fix "see § agent state below" x4 (_cli-fab.md:361,365,369,373; the block sits ABOVE at :353) and soften the "converges/removes stale fab-managed entries" overstatement (sync.go:34 comment, SPEC-hooks.md § Registered Hooks, _cli-fab.md § fab hook — Sync has no removal path) -->
- [x] T019 Update `src/kit/skills/fab-operator.md` and `docs/specs/skills/SPEC-fab-operator.md`: three-state + unknown vocabulary, `waiting` as the 90s-cadence trigger, three-state pre-send gate. <!-- R13 --> <!-- rework: cycle 1 must-fix — fab-operator.md:318-319 §5 lead sentence "Each non-idle agent is checked every tick" inverts the question-detection population (excludes idle agents, the historical primary auto-nudge case); contradicts :217 tick step 2, :321 fallback sentence, and the SPEC mirror — align with the waiting+idle-fallback set --> <!-- rework: cycle 2 must-fix, PINNED replacement text — (a) fab-operator.md:292 Watches 🟡 row: replace the row with `| waiting / idle / new-items | \`waiting\` (blocked on a human) or idle | has new unprocessed items | 🟡 |` so plain idle keeps a health row (frame example :256 shows `review · idle 8m` as 🟡), AND update the SPEC legend (SPEC-fab-operator.md:38) from "🟡 idle/new-items" to "🟡 waiting/idle/new-items"; grep 🟡 across src/kit/skills + docs/specs and align every legend to this set. (b) fab-operator.md:706 quick-ref /loop row: replace "when any monitored agent is menu-waiting" with "when any monitored agent is `waiting` (`@rk_agent_state`) or menu-waiting (capture fallback)" — matching SPEC-fab-operator.md:167. (c) population alignment: keep tick step 2 (:217) as the canonical per-tick population (waiting primary + idle fallback); reword §5 lead (:318) so active/unknown panes are "usable, not swept every tick" — the capture-based patterns remain applicable to `active`/unknown (`—`) panes (uninstrumented, or not yet flipped to `waiting`) but the per-tick sweep is waiting+idle only; align SPEC-fab-operator.md:86 to the same statement -->
- [x] T020 Sweep aggregate specs: `docs/specs/architecture.md` (`.fab-runtime.yaml` file-tree/gitignore mentions — note it no longer exists), and grep-verify `skills.md`/`glossary.md` carry no stale `idle_since`/`_agents`/`fab-runtime`/two-state claims (update if any). <!-- R12 --> <!-- rework: cycle 1 should-fix — docs/specs/skills.md:157 still lists "hook registration" among what fab sync handles (architecture.md got the equivalent correction; this aggregate restatement was missed) -->

- [x] T022 Sweep every doc/comment claim that `fab hook sync` still migrates/rewrites legacy `on-*.sh` scripts — cycle-2's T011 dropped those rewrite rows, so the claim is now false. PINNED canonical statement (adapt grammar per site, keep the semantics verbatim): "sync is retained one release but now fully inert — it registers nothing and no longer rewrites legacy scripts (the rewrite rows were dropped so sync cannot re-mint the entries the 2.14.0 migration deletes); it has no removal path." Known locations (fix ALL): `src/kit/skills/_cli-fab.md:327` + `:335` (sync table row), `docs/specs/skills/SPEC-hooks.md:5,:13,:23,:48` (:48's retained-for-legacy-migration rationale is inverted — restate retention as stale-settings tolerance for one release), `docs/specs/skills/SPEC-_cli-fab.md:26`, `docs/specs/architecture.md:445`, `docs/specs/skills.md:157`, comment `src/go/fab-kit/internal/sync.go:96`, comment `src/go/fab/cmd/fab/hook_test.go:170-171`. Then CLOSE THE CLASS: `rg -in "legacy|migrat" src/kit/skills docs/specs src/go | grep -i "sync\|hook"` (and similar) — fix any location beyond the 11. Also: strengthen `TestHookSync_RegistersNoSessionHooks` (`hook_test.go:172-198`) to assert empty stderr + the `hooks: OK`/`Created` stdout line (sync swallows failures, so the os.IsNotExist early-return can mask a silent no-write); collapse the duplicated sweep-population statement at `fab-operator.md:319-321` to one; rename the "Menu-detected heartbeat" setting label to "Waiting/menu heartbeat" in `fab-operator.md:684` AND `SPEC-fab-operator.md:42` (mirror pair). Restore A-022's text to its plain claim (drop the embedded NOT-MET note) leaving its box `[ ]` for the reviewer. <!-- R10 R11 R12 --> <!-- rework: cycle 3 REVISE-PLAN — added after two fix-code cycles; root cause: each rework's behavior changes need their own doc-claims sweep; this task makes the sweep explicit and grep-closed --> <!-- rework: cycle 4 must-fix, PINNED — hook.go:96 cobra Short still reads "Register hook commands into .claude/settings.local.json" (the only visible line in `fab hook --help`; string literals were missed by the comment-oriented greps): replace with Short: "Deprecated no-op — registers nothing and rewrites nothing; retained one release" ; extend the class-close to USER-FACING STRING LITERALS (rg 'Short:|Long:' src/go | grep -i hook). Cheap nice-to-haves in the same class: prepend "(now registers nothing — the mapping table is empty)" context to the Sync/syncHooks doc-comment openers (sync.go:67, hooksync.go:47); drop the unreachable Created/Updated from the sync output lists (_cli-fab.md:341, SPEC-hooks.md:49) or mark OK-only; fix the ReadAgentStateOption comment (pane.go:376) — unset option exits 1 with stderr on tmux 3.6a, not empty output. Out of scope: SPEC-fab-new.md:27 / SPEC-fab-draft.md:37 artifact-write drift (pre-existing y022 class, follow-up) -->

### Phase 7: Verification

- [x] T021 Run `go build ./... && go vet ./... && go test ./...` in `src/go/fab` and `src/go/fab-kit`; then `rg "idle_since|_agents|fab-runtime" src/` and confirm only convention-reader code + migration shims remain. <!-- R3 R4 R12 --> <!-- rework: cycle 1 — re-verify after rework fixes --> <!-- rework: cycle 2 — re-verify after cycle-2 fixes --> <!-- rework: cycle 3 — re-verify after T022 --> <!-- rework: cycle 4 — re-verify after the help-text fix -->
- [x] T023 [amendment] Remove the hook command family outright (no one-release retention) — fab hook stop/user-prompt/session-start/artifact-write/sync deleted, internal/hooklib + fab-kit hooksync deleted, docs/SPEC/memory retention language swept <!-- R3 R10 R11 R12 --> <!-- amendment: directed by Sahil post-ship 2026-07-06, supersedes the one-release-shim decisions (intake assumptions 11/15/18) -->

## Execution Order

- T001, T002 precede all reader/deleter work (Phase 1 is foundation).
- T003 blocks T004/T007 (both consume the rewritten `ResolvePaneContext`/parser).
- T008–T011 (deletions) must land together — deleting `internal/runtime`/`internal/proc` (T009) breaks `hook.go` until T008 rewires it.
- Phase 5 tests follow their production-code phases; Phase 6 docs are independent of the Go build and can proceed in parallel once the behavior is settled.
- T021 is last (whole-tree verification).

## Acceptance

### Functional Completeness

- [x] A-001 R1: All three readers resolve agent state from `@rk_agent_state`; an absent option yields unknown (`—`/null).
- [x] A-002 R2: `parseAgentState` returns ok=true only for a well-formed `"<state>:<epoch>"` with a known state token and integer epoch.
- [x] A-003 R3: `fab hook stop|user-prompt|session-start` are no-op exit-0 shims that emit nothing and never write `.fab-runtime.yaml`.
- [x] A-004 R4: `internal/proc/` and `internal/runtime/` are deleted; `internal/lockfile` remains and still compiles; the `dispatch.go` proc comment is swept.
- [x] A-005 R5: `FormatIdleDuration` is present in `internal/pane`; the `_agents` resolvers and `LoadRuntimeFile` are gone.
- [x] A-006 R6: `fab pane send` sends on `idle`, refuses `active`/`waiting` (three-state-aware message), refuses unknown with a distinct `--force` message; `--force` bypasses state only.
- [x] A-007 R7: `fab pane map` Agent column reads `#{@rk_agent_state}` from the single `list-panes` format string (no extra subprocess) and renders `active`/`waiting`/`idle (<dur>)`/`—`.
- [x] A-008 R8: `fab pane capture` header shows the resolved state (incl. `waiting`).
- [x] A-009 R9: JSON keeps `agent_state`/`agent_idle_duration`; `agent_state` includes `waiting`; duration only for idle.
- [x] A-010 R10: The `2.13.6-to-2.14.0.md` migration removes the three hook entries, preserves unrelated keys, and is idempotent.
- [x] A-011 R11: `DefaultMappings`/`defaultHookMappings` register zero session-scoped hooks; `Sync`/`syncHooks` are retained.

### Behavioral Correctness

- [x] A-012 R3: All fab commands behave identically outside tmux — no `.fab-runtime.yaml` is written anywhere.
- [x] A-013 R7: The `pane map` server-disambiguation path is removed (a pane option lives on one server's pane) with no regression in the Change/Stage columns.

### Scenario Coverage

- [x] A-014 R6: A codex/copilot pane (previously invisible to the Claude-only pipeline) is gated correctly by `fab pane send` once its `@rk_agent_state` is set — exercised via `tmux set-option -p` writer simulation. — MET (rework cycle 1): the gate is now covered two ways in `pane_send_test.go` — `TestIdleGate` unit-tests the extracted pure decision helper for all five cases (idle-send / active-refuse / waiting-refuse / unknown-distinct-refuse / and pins both message contracts), and `TestPaneSendGate_Integration` drives the full `fab pane send` command against a real ephemeral tmux server with the `tmux set-option -p` writer simulation (codex-pane scenario end-to-end: active/waiting/unknown refuse, idle sends, `--force` bypasses).
- [x] A-015 R2: Reader tests simulate the writer via `tmux set-option -p -t <pane> @rk_agent_state "<state>:<epoch>"`; the manual `waiting`-probe acceptance is deferred to post-writer verification.

### Edge Cases & Error Handling

- [x] A-016 R1: A stale `active` (Esc-interrupted agent) still refuses sends (no staleness heuristic in v1); `--force` is the escape hatch.
- [x] A-017 R6: A missing pane still exits 2 even with `--force` (pane existence enforced independent of the state gate).

### Code Quality

- [x] A-018 Pattern consistency: New reader/parse code follows the surrounding `internal/pane` naming, error-handling (`RunCmd`/`StderrError`), and build/test conventions.
- [x] A-019 No unnecessary duplication: The `"<state>:<epoch>"` parse lives in one helper reused by all three readers; existing tmux argv helpers (`WithServer`) are reused.
- [x] A-020 No god functions / magic strings: state tokens (`active`/`waiting`/`idle`) and the option name (`@rk_agent_state`) are named constants; no reader exceeds the codebase's typical function size.

### Documentation Accuracy

- [x] A-021 Skill/SPEC mirror parity: every touched `src/kit/skills/*.md` (`_cli-fab.md`, `fab-operator.md`) has its `docs/specs/skills/SPEC-*.md` mirror updated in the same change (constitution Additional Constraints); `SPEC-hooks.md` reflects the shim/reader model.
- [x] A-022 CLI-doc parity: `_cli-fab.md` documents the changed `fab hook` (shims) and `fab pane send/map/capture` (`@rk_agent_state` reader, `waiting`) signatures/behavior (constitution: CLI change MUST update `_cli-fab.md`). — MET (review cycle 3, verified against `src/go/fab/cmd/fab/`): § fab hook (:325-341) matches hook.go exactly (silent exit-0 shims, inert sync, exit-0-always contract, `2.13.6-to-2.14.0` migration); § agent state (:353) + map/capture/send sections (:355-373) match the implemented reader (three states + `waiting`, epoch-mandatory, idle-only duration, `—`/null unknown, distinct `--force` unknown refusal, exact `ERROR:`/`Error:` message shapes per main.go's printer, exit 1/2/3 mapping, JSON `agent_state`/`agent_idle_duration` semantics incl. `waiting`).
- [x] A-023 `rg "idle_since|_agents|fab-runtime" src/` returns only convention-reader code and migration shims — no stale producer-pipeline claims in shipped source/kit.
- [x] A-026 Sync-claims accuracy: no shipped doc, skill, SPEC, or code comment claims `fab hook sync` still registers hooks or migrates/rewrites legacy `on-*.sh` scripts — grep-verified (e.g. `rg -in "legacy|migrat" src/kit/skills docs/specs | grep -i sync`) consistent with the emptied rewrite map and `TestSync_LeavesLegacyScriptUntouched`. — MET (review cycle 4): the cycle-4 fix landed — `hook.go:96` cobra `Short` now reads "Deprecated no-op — registers nothing and rewrites nothing; retained one release" (`rg 'Short:|Long:' src/go | grep -i hook` shows no registration claim in any user-facing string literal); re-swept the full class with `rg -in "legacy|migrat" src/kit/skills docs/specs src/go | grep -i "sync\|hook"` and `rg -in "register" docs/specs src/kit | grep -i hook` — every remaining hit is a corrected "registers nothing / no longer rewrites" statement, a historical migration file (`0.46.0-to-1.1.0.md`), or a dated `docs/specs/findings/*` point-in-time record (out of scope per assumption 7); the mechanical `migrateOldHookCommands` function docs are covered by the emptied-map context comments 4 lines above (assumption 14); the scaffold `fragment-settings.local.json` ships permissions only, no hook entries. docs/memory/ still carries producer-model claims — deferred to hydrate by design (R12), not counted here.

### Cross References

- [x] A-024 Aggregate-spec sweep: `docs/specs/architecture.md` no longer presents `.fab-runtime.yaml` as a live runtime file (or explains its removal); `skills.md`/`glossary.md` carry no stale two-state/`_agents` claims.
- [x] A-025 Migration lineage: the new migration references the `2.10.1-to-2.11.0.md` precedent it follows; the retained-`Sync` and out-of-scope-y022 decisions are recorded (comments/decisions) so hydrate can capture them.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- `docs/memory/` rewrites (runtime-agents, pane-commands, operator, schemas, hooks-may-enhance-never-own, kit-architecture) are deferred to hydrate per the intake's Affected Memory section — NOT done in this apply.

## Deletion Candidates

- ~~`src/go/fab/internal/hooklib/payload.go` (`ParseSessionPayload`, `SessionPayload`) + `payload_test.go`~~ — **TAKEN in this change (rework cycle 1):** newly orphaned once T008 reduced the three session-hook handlers (its only production callers) to no-op shims. Verified sole remaining references were its own tests; both files deleted. (Distinct from y022's out-of-scope `artifact.go` orphans, which stay.)
- ~~`src/go/fab/internal/dispatch/dispatch_posix.go:43` — comment references `internal/runtime.pidAlive`~~ — **TAKEN in this change (rework cycle 1, T010):** swept (now reads "It is the POSIX-standard kill(pid, 0) liveness probe"); re-verified cycles 3–4, no `internal/runtime` references remain anywhere in src/go production code.
- ~~`src/go/fab/cmd/fab/hook.go` shims `hookSessionStartCmd`/`hookStopCmd`/`hookUserPromptCmd` (+ `noOpHookShim`, and y022's `hookArtifactWriteCmd`)~~ — **TAKEN in the T023 amendment (2026-07-06):** the whole `cmd/fab/hook.go` file (all shims + `hookCmd` + `noOpHookShim`) was deleted and the `hookCmd()` registration dropped from `newRootCmd()`. No shim period — Sahil's post-ship directive superseded the one-release lifecycle.
- ~~The whole sync path, retained one release per R11 and now **fully inert**: `hooklib.Sync` + `DefaultMappings`/`HookMapping`/`hasDuplicate` (`src/go/fab/internal/hooklib/sync.go`), fab-kit's `syncHooks`/`defaultHookMappings`/`hookMapping`/`hookHasDuplicate` (`src/go/fab-kit/internal/hooksync.go`), and `hookSyncCmd` (`cmd/fab/hook.go`)~~ — **TAKEN in the T023 amendment (2026-07-06):** `src/go/fab/internal/hooklib/sync.go` (+ `sync_test.go`) and `src/go/fab-kit/internal/hooksync.go` (+ `hooksync_test.go`) deleted wholesale; the fab-kit `sync.go` hook-sync step removed (`cleanLegacyAgents` retained). **`internal/hooklib/artifact.go` + `artifact_test.go` are KEPT** — they hold the change-type/section-count parsers that feed `internal/refresh` (`fab status refresh`), not any hook. All the dead sub-pieces below (the merge loops, `oldScriptToSubcommand`/`migrateOldHookCommands`, the unreachable Created/Updated branches, `hasDuplicate`, and the unconditional settings write) went with the deleted files:
  - `oldScriptToSubcommand` + `migrateOldHookCommands` — gone with the deleted sync files.
  - The desired-entry merge loops, the `migrated > 0`/`added > 0` result branches, and `hasDuplicate`/`hookHasDuplicate` — gone with the deleted sync files.
  - The unconditional settings write (`Sync`/`syncHooks`) — gone; `fab sync` no longer materializes a `"hooks": {}` `.claude/settings.local.json` (it does not touch that file at all).

*(Deletion Candidates fully cleared by the T023 amendment: the sync path and all session/artifact-write shims are removed outright, not retained. `internal/hooklib` survives as parser-only — see the scope note in Assumption 17.)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Migration named `2.13.6-to-2.14.0.md` (next release is a feature-level 2.14.0; latest released tag is v2.13.6) | Verified `git log` release tags (v2.13.6 latest) + existing migration naming convention `X-to-Y.md`; subsystem removal is minor-version-worthy | S:85 R:90 A:95 D:90 |
| 2 | Certain | Parse helper `parseAgentState(raw) (state, epoch, ok)` lives in `internal/pane/pane.go`, reused by all three readers | Intake specifies pure tmux-free testability; single-authority avoids the three-drifting-copies anti-pattern (code-quality) | S:85 R:85 A:90 D:90 |
| 2b | Certain | `pane map` reads `#{@rk_agent_state}` from the existing `list-panes -F` format string; `send`/`capture` read via `tmux show-options -pv -t <pane> @rk_agent_state` | Directed by intake (zero-extra-subprocess for map; single-pane targeted read for send/capture) | S:95 R:85 A:90 D:95 |
| 3 | Confident | `active`/`waiting`/unknown carry no `agent_idle_duration`; only `idle` computes a duration from the epoch | Mirrors today's `active` (no duration) semantics; duration is meaningful only for a completed turn | S:65 R:85 A:80 D:75 |
| 4 | Confident | Unknown-state `agent_state` maps to JSON `null` (unchanged shape), matching today's no-match em-dash→null contract | Preserves the additive JSON shape run-kit consumes; intake says field names preserved, `waiting` added | S:70 R:80 A:80 D:75 |
| 5 | Confident | Reader tests use `tmux set-option -p` writer simulation where a tmux test server is available; pure-helper (`parseAgentState`) tests cover the parse grammar tmux-free | Intake directs `tmux set-option -p` simulation; existing pane tests are largely tmux-free pure-helper tests, so parse coverage is the primary automated surface | S:70 R:85 A:75 D:70 |
| 6 | Confident | `TestParseTmuxServer` and `parseTmuxServer` are deleted (the function's sole use was building `_agents` entries) | The server-basename parse only fed `buildAgentEntry`, which is deleted; no reader needs it (map reads the option off the pane directly) | S:65 R:85 A:80 D:75 |
| 7 | Confident | `docs/specs/findings/*` dated review artifacts are left unchanged | Constitution VI: specs/findings are historical, human-curated point-in-time records, not living mirrors of current behavior | S:70 R:90 A:80 D:75 |
| 8 | Tentative | State token constants (`agentStateActive`/`Waiting`/`Idle`) and the option name `@rk_agent_state` are defined in `internal/pane` and referenced by the readers | Reasonable to centralize per code-quality (no magic strings); exact constant placement/names are a low-stakes local choice | S:55 R:85 A:70 D:60 |
| 9 | Confident | (rework cycle 1) Extract the pane-send three-state gate into a pure `idleGate(paneID, *state)` helper in `pane_send.go`, unit-tested by `TestIdleGate`, with `TestPaneSendGate_Integration` covering the full command against a real tmux server | Review directed structuring the gate for testability; mirrors the existing `validatePaneResult` pure-decision-half pattern in `internal/pane`; behavior is byte-identical to the inline switch it replaced | S:85 R:90 A:85 D:85 |
| 10 | Confident | (rework cycle 1) Guard `ReadAgentStateOption` against an empty paneID (return "" unknown without touching tmux, since `-t ""` targets the client's current pane) | Review nice-to-have; a defensive guard that prevents reading a wrong-pane state, fully reversible and codebase-consistent | S:80 R:90 A:85 D:85 |
| 11 | Confident | (rework cycle 2, T006) Thread the RAW `@rk_agent_state` value through a new `paneRow.agentOption` field and derive the JSON `agent_state`/`agent_idle_duration` pair directly from `pane.AgentDisplayFromOption(agentOption)` via a new `agentJSONFields` helper; delete `splitAgentState` (now callerless) | Review directed decoupling JSON from the display string so a display-format tweak cannot break the run-kit-consumed contract; `AgentDisplayFromOption` is the same structured source `agentColumn` already uses, so map/JSON stay byte-identical | S:85 R:85 A:85 D:85 |
| 12 | Confident | (rework cycle 2, T011) Empty the `oldScriptToSubcommand` rewrite map to `{}` (dropping the three legacy on-*.sh rows) while KEEPING `migrateOldHookCommands`/`Sync`/`syncHooks` as a now-no-op path, rather than deleting the sync path outright | Matches the recorded one-release-retention follow-up (Deletion Candidates); the empty map closes the re-minting hazard with the minimal change and the migration removes both the inline and legacy forms | S:80 R:85 A:80 D:80 |
| 13 | Certain | (rework cycle 2, T004) Keep the option name in the reworded unknown refusal as the `%s`=`pane.AgentStateOption` placeholder ("(missing or unparseable %s)") rather than a hardcoded literal | Preserves the no-magic-string convention; `AgentStateOption` = `@rk_agent_state`, so the rendered message matches the pinned target text exactly | S:90 R:90 A:95 D:90 |
| 14 | Confident | (rework cycle 3, T022) Leave the *mechanical* `migrateOldHookCommands` doc-comment and call-site comment ("Migrate old-style commands to inline fab hook commands", sync.go:107/183, hooksync.go:86/153) unchanged — they describe the function's literal loop body, and the correcting "now EMPTY / no-op this release" context sits 4 lines above at the `oldScriptToSubcommand` declaration (sync.go:41, hooksync.go:29) | T022 targets doc/comment CLAIMS that sync still migrates as *current behavior*; the function docs are not such a claim (the map-declaration comment already carries the divestment context), and rewording a function's contract to contradict its own loop would be misleading. Fully reversible once the sync path is deleted next release | S:75 R:90 A:80 D:75 |
| 15 | Certain | (rework cycle 3, T022) Rendered the SPEC-fab-operator.md:42 label rename lowercase ("waiting/menu heartbeat") to fit the running prose sentence, while the skill's Settings-table row (fab-operator.md:684) uses title-case "Waiting/menu heartbeat" as a column label | Mirror-pair intent is the *label* rename ("Menu-detected" → "Waiting/menu"); casing follows each site's existing register (table-cell label vs. inline prose), matching how :42 already lowercased "menu-detected heartbeat" | S:90 R:95 A:90 D:85 |
| 16 | Confident | (rework cycle 3, T022) Collapsed the §5 duplication by keeping the full per-tick population statement in the §5 lead (fab-operator.md:319) as canonical and trimming the redundant restatement from the "primary signal" paragraph (:321), which now defers with "(the population stated above)" | Single-sources the sweep-population fact within §5 while preserving the distinct primary-signal point at :321; SPEC-fab-operator.md:86 already states the population once, so skill↔SPEC stay aligned (both name waiting+idle as the per-tick sweep, active/unknown applicable-not-swept) | S:80 R:90 A:85 D:80 |
| 17 | Certain | Amendment: hook family removed outright (no shims, no inert sync, y022 artifact-write shim also deleted) — un-migrated settings error until the 2.14.0 migration runs; accepted. Scope refinement discovered during work: `internal/hooklib` is NOT deleted wholesale — only its hook-coupled `sync.go`/`sync_test.go` go; `artifact.go`/`artifact_test.go` (change-type + section-count parsers) survive because they feed `internal/refresh`, `internal/status/acceptance`, and `internal/prmeta` (non-hook consumers). Deleting the whole package would break the build | Directed by Sahil 2026-07-06 post-ship ("no retained-one-release, just remove"); the artifact.go carve-out is required for `go build` to pass (verified: only `hooklib.Sync` was hook-coupled) | S:95 R:85 A:95 D:95 |

17 assumptions (6 certain, 10 confident, 1 tentative).
