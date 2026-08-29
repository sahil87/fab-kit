# Plan: Agent-Exit Shell Fallback for Interactive Spawns

**Change**: 260829-1xqx-agent-exit-shell-fallback
**Intake**: `intake.md`

## Requirements

### Runtime: Interactive spawn grammar

#### R1: Interactive spawns carry a shell fallback
Every **interactive** agent spawn fab performs — the operator launcher tab (`fab operator`), `fab batch new`, and `fab batch switch` — MUST pass `tmux new-window` a shell command of the form `<composed-cmd> '<prompt>'; exec "$SHELL"`, so the agent runs as the wrapper shell's foreground child and the pane survives the agent's exit as the user's login shell in the same cwd.

- **GIVEN** `fab operator` is run inside tmux with no existing `operator` window
- **WHEN** the new window is created
- **THEN** the `new-window` shell-command argument ends with the exact suffix `; exec "$SHELL"`
- **AND** the `FAB_AGENT_WORKERS=… ` env prefix (when `--workers` is set) still precedes the agent command only — the fallback is appended after the whole existing command string

- **GIVEN** `fab batch new <id>` or `fab batch switch <change>` opens a window
- **WHEN** the shell command is composed
- **THEN** it carries the same suffix after the quoted `/fab-new …` / `/fab-switch …` prompt

#### R2: One Go owner for the wrapper string
The fallback suffix MUST be produced by a single exported helper `spawn.WithShellFallback(cmd string) string` in `src/go/fab/internal/spawn/spawn.go`, returning `cmd + "; exec \"$SHELL\""`; no call site MAY spell the suffix literally.

- **GIVEN** the three call sites in R1
- **WHEN** they compose their `new-window` command
- **THEN** each calls `spawn.WithShellFallback(...)` as the final composition step

#### R3: Dispatch pane workers keep the unwrapped spawn (Non-Goal made executable)
`pane.OpenWindow`, `pane.OpenSplitPane`, `fab dispatch open`, and `fab pane open` MUST NOT gain the fallback. Existing tests asserting the unwrapped command MUST keep passing unchanged.

- **GIVEN** `fab dispatch open <change> <stage>` or `fab pane open --provider <p>`
- **WHEN** the pane is created
- **THEN** the shell command is the resolved `interactive_command` verbatim, with no `; exec "$SHELL"` suffix

### Runtime: Pane snapshot carries the pane's current command

#### R4: Snapshot rows carry `command`
The pane-map/tick snapshot row (`paneEntry` → `paneRow` in `src/go/fab/cmd/fab/pane_map.go`) MUST carry the pane's current foreground command on BOTH enumeration paths: the delegated `rk mux panes --json` path (its rows already carry `"command"`) and fab's internal `tmux list-panes -F` fallback (append `#{pane_current_command}` as a NINTH tab-separated field to `tmuxPaneFormat`).

- **GIVEN** the internal fallback path and a 9-field line
- **WHEN** `parsePaneLines` parses it
- **THEN** `paneEntry.command` is the ninth field, `windowID` remains field 8, and 5–8-field legacy lines still parse with `command == ""`

- **GIVEN** the rk-delegated path
- **WHEN** a row is converted
- **THEN** `paneEntry.command` is the row's `command` value (`""` when absent)

- **GIVEN** `fab pane map` human/JSON output
- **WHEN** rendered
- **THEN** it is byte-identical to today — `command` is internal to the snapshot and not a new output field

### Runtime: Operator `agent_exited` delta

#### R5: `agent_exited` is emitted when a monitored pane's agent has exited
`fab operator tick-start --diff` MUST emit a delta `kind: agent_exited` for a monitored entry whose pane IS present in the snapshot, whose resolved change matches the entry (i.e. it is not a `pane_mismatch`), and whose `command` basename is in the fixed shell set `sh bash zsh fish dash ksh tcsh csh nu`. The delta carries `change`, `pane`, and the kind-specific field `command` (the shell name).

- **GIVEN** a monitored entry on pane `%3` and a snapshot row for `%3` with `command: zsh` resolving to the same change
- **WHEN** the tick runs
- **THEN** `deltas:` contains `{kind: agent_exited, change: <id>, pane: "%3", command: zsh}`

- **GIVEN** the same pane with `command: claude` (or `node`, `kimi`, or empty)
- **WHEN** the tick runs
- **THEN** no `agent_exited` delta is emitted

#### R6: `agent_exited` is level-triggered and mirrors `pane_mismatch` exclusions
The delta MUST re-emit every tick until `fab operator remove` acks it (no baseline state). An `agent_exited` pane MUST NOT receive a baseline `stage`/`agent`/`last_transition` write, MUST NOT produce `completion`/`stage_advance`/`review_fail`, MUST NOT appear in `candidates:`, and its `fleet:` row MUST be the baseline-identity row (`baselineFleetRow`: null observed fields, `agent_state: null`) so `fleet_summary` counts it under `unknown`.

- **GIVEN** an `agent_exited` pane whose stale `@rk_pane_agent_state` still reads `idle`
- **WHEN** the tick runs
- **THEN** the pane is absent from `candidates:` and the entry's stored `agent` field is unchanged

- **GIVEN** two consecutive ticks with no `fab operator remove`
- **WHEN** both run
- **THEN** both emit the `agent_exited` delta

#### R7: Delta precedence
Evaluation order per entry MUST be `pane_death` (absent) → `pane_mismatch` (different/none change) → `agent_exited` (shell command) → clean join. A pane that is both mismatched and shell-hosted emits only `pane_mismatch`.

- **GIVEN** pane `%3` enrolled for change `a001`, snapshot row `%3` with `command: zsh` and `change: ""`
- **WHEN** the tick runs
- **THEN** exactly one delta for `a001` is emitted and its kind is `pane_mismatch`

### Docs: Skill prose, CLI reference, specs

#### R8: Spawn Composition owns the fallback rule
`src/kit/skills/_cli-agents.md` § Spawn Composition MUST document the raw interactive form with the `; exec "$SHELL"` suffix, its mechanism (wrapper shell cannot exec-optimise a two-command string; `#{pane_current_command}` reports the agent while it runs; `$SHELL` comes from tmux `default-shell`), and the scope rule (interactive spawns only; `fab dispatch open`/`fab pane open` pane workers are excluded because their state machine keys on pane death). `_cli-external.md`'s `new-window` rows and `fab-operator.md` §6 step 7 MUST point at this owner (one-line pointer + the mechanical grammar), never restate the rationale.

- **GIVEN** a repo-wide grep for `exec "$SHELL"` / `exec \"$SHELL\"` in `src/kit/skills/`
- **WHEN** inspected
- **THEN** the mechanism/rationale prose appears only in `_cli-agents.md`; other hits are the grammar line plus a pointer

#### R9: `_cli-fab.md` reflects the new CLI surface
`src/kit/skills/_cli-fab.md` § fab operator tick-start MUST list `agent_exited` in the `deltas:` kind comment and kind-specific fields (`agent_exited → command`), in the "Two delivery classes" level-triggered set, and in "Detection semantics" (predicate + exclusions + precedence). § fab pane map's fallback-path description MUST mention the ninth `#{pane_current_command}` format field as snapshot-internal (no new output column/JSON key).

- **GIVEN** the `deltas:` YAML block in `_cli-fab.md`
- **WHEN** read
- **THEN** the kind comment reads `completion | pane_death | pane_mismatch | agent_exited | stage_advance | review_fail` and an `agent_exited → command` line exists

#### R10: `fab-operator.md` acts on `agent_exited`
`src/kit/skills/fab-operator.md` MUST: add `agent_exited` to Tick Behavior step 1's delta list (report, then remove via step 5; level-triggered sentence updated); add a §4 automation row after `Pane death`: `Agent exited (pane survives as a shell) | 0 | Report gone (pane kept, cwd intact). Respawn only in autopilot (1 attempt): kill the leftover shell pane first (rk mux kill when rk is installed — an uninstrumented/idle pane passes its gate — else tmux kill-pane -t <pane>), then spawn per §6`; update §6 step 7's grammar per R8; and state in Status Frame Format that an `agent_exited` row renders like a `pane_death` row (baseline identity, `—` agent) with a distinguishing marker so it never reads as a live agent.

- **GIVEN** the §4 table and the tick delta list
- **WHEN** read
- **THEN** both carry the `agent_exited` entries above

#### R11: Specs and sibling sweep
`docs/specs/harness-adapters.md` MUST gain one sentence at the pane-adapter row/section: interactive (human-facing) spawns carry a shell fallback; dispatch pane workers do not because the adapter's `running`/`done`/`orphaned` subset depends on pane death. `docs/specs/skills.md` `/fab-operator` section MUST reflect the new delta wherever it restates the delta list or §4 rows. A repo-wide grep for `pane_death` MUST be swept: every prose site listing the delta kinds (known: `_cli-fab.md`, `fab-operator.md`, `docs/specs/skills.md`, `docs/memory/runtime/operator.md` — the memory file is hydrate's, but apply MUST NOT leave `src/kit/` or `docs/specs/` listings incomplete).

- **GIVEN** `grep -rn pane_death src/kit docs/specs`
- **WHEN** inspected after apply
- **THEN** every hit that enumerates delta kinds also names `agent_exited`

### Non-Goals

- Wrapping `fab dispatch open` / `fab pane open` / `pane.OpenWindow` / `pane.OpenSplitPane` — dispatch workers' lifecycle keys on pane death (intake decision 2).
- A `--shell-fallback` flag on `fab pane open` — not needed by any current consumer.
- A new kill verb — `rk mux kill` / `C-d` suffice (intake decision 4).
- Exposing `command` as a new `fab pane map` output column or JSON key — snapshot-internal only.
- Reading tmux's `default-shell` option to extend the shell set — fixed set first; open question in intake.
- `remain-on-exit` / `respawn-pane` designs.

### Design Decisions

#### Shell fallback via `; exec "$SHELL"` wrapper, not shell-then-send-keys
**Decision**: Append `; exec "$SHELL"` to the single-shot interactive spawn command so the agent runs as the wrapper shell's foreground child and the user's login shell takes the pane over on exit.
**Why**: Keeps the spawn atomic (no shell-prompt readiness race, no send-keys into a shell, no alias/quoting detour), preserves every existing readiness/deliver contract (`#{pane_current_command}` still reports the agent while it runs), and is a one-token grammar change per site.
**Rejected**: Open a bare shell then `send-keys` the agent command — introduces a readiness race and a two-step choreography on every spawn. `remain-on-exit` — leaves a dead pane, not a terminal.
*Introduced by*: 260829-1xqx-agent-exit-shell-fallback

#### `agent_exited` keys on `pane_current_command`, not agent state or pid
**Decision**: Detect an exited interactive agent as "pane present AND foreground command is a shell", carried on the snapshot row from `rk mux panes --json`'s `command` or the internal `#{pane_current_command}` field.
**Why**: fab's `parseAgentState` validates and ignores the `:pid` segment (pid reconciliation is rk's), so after exit the pane option still reads the agent's last `idle` — using agent state would list a bare zsh as a nudge candidate. A pid-children heuristic false-positives on exec-optimised unwrapped single-command spawns where `pane_pid` *is* the agent. The shell predicate is provider-agnostic, works on rk-less servers, and also covers a hand-started agent in a user-opened shell pane.
**Rejected**: agent-state-null predicate (stale on fab's path); `pgrep -P pane_pid` empty (false positives on unwrapped spawns).
*Introduced by*: 260829-1xqx-agent-exit-shell-fallback

#### Dispatch pane workers stay "just agent"
**Decision**: The fallback applies only to interactive, human-typed-into spawns; `fab dispatch open`/`fab pane open` are untouched.
**Why**: The pane adapter's `running`/`done`/`orphaned` subset and `dispatch.reap_done` treat pane death as the worker's terminal event; nobody types into a reaped worker.
**Rejected**: Wrapping all spawns and re-deriving `orphaned` from `pane_current_command` — changes the adapter contract for no user-facing gain.
*Introduced by*: 260829-1xqx-agent-exit-shell-fallback

## Tasks

### Phase 1: Setup

- [x] T001 Add `WithShellFallback(cmd string) string` to `src/go/fab/internal/spawn/spawn.go` (doc comment stating the interactive-only scope and naming the excluded dispatch creators) and a table-driven test in `src/go/fab/internal/spawn/spawn_test.go` asserting the exact suffix `; exec "$SHELL"` and idempotence-free plain concatenation <!-- R2 -->

### Phase 2: Core Implementation

- [x] T002 [P] Wrap the operator launcher: in `src/go/fab/cmd/fab/operator.go` `runOperator`, apply `spawn.WithShellFallback` to `shellCmd` AFTER `withWorkersEnv`; extend `src/go/fab/cmd/fab/operator_test.go` so the `new-window` argv's last argument is asserted to end with the suffix and to keep the `FAB_AGENT_WORKERS=` prefix first when `--workers` is set <!-- R1 -->
- [x] T003 [P] Wrap batch spawns: in `src/go/fab/cmd/fab/batch_new.go` and `src/go/fab/cmd/fab/batch_switch.go`, apply `spawn.WithShellFallback` to `shellCmd` after `withWorkersEnv`; extend `batch_new_test.go` (its composed-command extractor helper ~line 356) and the batch-switch test to assert the suffix <!-- R1 -->
- [x] T004 [P] Carry `command` on the snapshot: in `src/go/fab/cmd/fab/pane_map.go` add `command string` to `paneEntry` and `paneRow`; append `\t#{pane_current_command}` to `tmuxPaneFormat` (update its doc comment: NINE fields, `command` is the new never-empty-in-practice trailing field, `window_id` stays field 8); update `parsePaneLines` (`SplitN(line, "\t", 9)`, `case 9:` sets `windowID = parts[7]`, `command = parts[8]`; 8/7/6/5-field tolerance unchanged); add `Command string \`json:"command"\`` to `rkPaneRow` and copy it into `paneEntry` in `discoverPanesViaRK`; propagate `command` in `resolvePane` (both branches); update `pane_map_test.go` format/parser tests (9-field line, legacy lines still parse, rk row conversion) and confirm `fab pane map` output tests are unchanged <!-- R4 -->
- [x] T005 Emit `agent_exited`: in `src/go/fab/cmd/fab/operator_tick_start.go` add `Command *string` to `tickDelta` (MarshalYAML: `case "agent_exited": putNullable("command", d.Command)`), a `shellCommands` set + `isShellCommand(cmd string) bool` (basename match against `sh bash zsh fish dash ksh tcsh csh nu`), and in `diffMonitored` — after the `pane_mismatch` branch and before the clean join — emit `tickDelta{Kind: "agent_exited", Change: id, Pane: entry.Pane, Command: &row.command}`, append `baselineFleetRow(id, entry)`, and `continue` (no baseline write, no candidates). Update the `tickDelta` and `tickFleetSummary` doc comments (level-triggered set now includes `agent_exited`) <!-- R5 R6 R7 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Tests for the tick delta in `src/go/fab/cmd/fab/operator_tick_diff_test.go`: extend `snapRow` (or add `snapRowCmd`) with a `command` argument; add cases — shell command emits `agent_exited` with `command: zsh`; non-shell (`claude`) and empty command emit nothing; stale `idle` agent state on an exited pane yields no candidate and an unchanged stored `agent`; two ticks re-emit; mismatched+shell pane emits only `pane_mismatch`; fleet row has null `agent_state` and quiet-tick summary counts it under `unknown`; update `TestOperatorTickDiff_AllEventKinds` to include the new kind <!-- R5 R6 R7 -->
- [x] T007 Scope-boundary guard: confirm `dispatch_open_test.go`, `pane_open_test.go`, and any `pane/create.go` tests still assert the UNWRAPPED command (add one explicit negative assertion — the `new-window`/`split-window` argv passed by `pane.OpenWindow`/`OpenSplitPane` does not contain `exec "$SHELL"`) <!-- R3 -->

### Phase 4: Polish

- [x] T008 [P] `src/kit/skills/_cli-agents.md` § Spawn Composition: update the raw-form code block to `"<composed-cmd> '<initial-prompt>'; exec \"$SHELL\""`, add a bullet owning the mechanism + scope rule (interactive only; dispatch/`fab pane open` excluded and why); `src/kit/skills/_cli-external.md` `new-window` rows: note the interactive fallback with a pointer to the owner <!-- R8 -->
- [x] T009 [P] `src/kit/skills/_cli-fab.md`: § fab operator tick-start `deltas:` block kind comment + `agent_exited → command` line, level-triggered set, detection-semantics bullet (predicate, exclusions, precedence); § fab pane map fallback-path sentence mentioning the ninth `#{pane_current_command}` field as snapshot-internal <!-- R9 -->
- [x] T010 [P] `src/kit/skills/fab-operator.md`: Tick Behavior step 1 delta list + level-triggered sentence; §4 table row after `Pane death`; §6 step 7 grammar with pointer; Status Frame Format note for exited rows <!-- R10 -->
- [x] T011 Specs + sweep: `docs/specs/harness-adapters.md` one-sentence interactive-vs-worker note at the pane adapter; `docs/specs/skills.md` `/fab-operator` delta/§4 restatements; `docs/specs/glossary.md` entry if delta kinds are listed; then `grep -rn pane_death src/kit docs/specs` and update every kind-enumerating hit <!-- R11 -->
- [x] T012 Run `cd src/go/fab && go build ./... && go test ./internal/spawn/... ./cmd/fab/...`, then `gofmt -l` on touched files (CI trips on gofmt), then the full `go test ./...` <!-- R1 R4 R5 -->

## Execution Order

- T001 blocks T002, T003 (they import the helper)
- T004 blocks T005 (row field), T005 blocks T006
- T007 can run any time after T001
- T008–T011 are docs and independent of Go tasks; T011 after T009/T010 so the sweep sees final wording
- T012 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab operator`, `fab batch new`, and `fab batch switch` each pass a `new-window` command ending in `; exec "$SHELL"`, with any `FAB_AGENT_WORKERS=` prefix still leading — verified in `operator.go:106-109`, `batch_new.go:142-144`, `batch_switch.go:142-144` (wrapper applied after `withWorkersEnv`) and pinned by suffix+prefix assertions in `operator_test.go`, `batch_new_test.go`, `batch_switch_test.go`
- [x] A-002 R2: `spawn.WithShellFallback` exists, is tested, and is the only place the suffix is spelled — `internal/spawn/spawn.go:61-63`, table test in `spawn_test.go`; grep for `exec "$SHELL"` in `src/go` hits only the helper, doc comments, and test assertions
- [x] A-003 R4: `paneEntry`/`paneRow` carry `command` from both enumeration paths; `tmuxPaneFormat` has nine fields; legacy 5–8-field lines still parse — `pane_map.go` (`rkPaneRow.Command`, `tmuxPaneFormat` field 9, `parsePaneLines` `SplitN 9` + graded 5–8-field tolerance, `resolvePane` both branches); parser tests in `pane_map_test.go`
- [x] A-004 R5: `agent_exited` delta emitted with `command` for a shell-hosted, change-matched monitored pane — `operator_tick_start.go:469-479`; covered by `TestOperatorTickDiff_AgentExited` and `TestOperatorTickDiff_AllEventKinds`
- [x] A-005 R6: exited pane excluded from candidates/baseline write/stage diffs; fleet row is the baseline row with null agent state; re-emits until removed — the branch `continue`s before any baseline write/candidate append and appends `baselineFleetRow`; covered by the stale-idle, level-triggered re-emit, and fleet/summary subtests
- [x] A-006 R7: precedence `pane_death` → `pane_mismatch` → `agent_exited` → join, with the mismatched+shell case emitting only `pane_mismatch` — branch order in `diffMonitored` (`operator_tick_start.go:440/450/469`); "mismatch wins over agent_exited" subtest
- [x] A-007 R8: `_cli-agents.md` § Spawn Composition owns the fallback grammar, mechanism, and scope rule; `_cli-external.md` and `fab-operator.md` §6 point at it — `_cli-agents.md:80/84` owns mechanism+scope; `_cli-external.md:185` and `fab-operator.md:496-499` carry grammar + pointer
- [x] A-008 R9: `_cli-fab.md` tick-start section lists `agent_exited` (kind comment, field, delivery class, detection semantics) and pane map mentions the ninth field — `_cli-fab.md:528`, `:1258`, `:1264`, `:1286-1287`
- [x] A-009 R10: `fab-operator.md` carries the tick delta entry, the §4 row, the §6 grammar, and the frame note — `fab-operator.md:138` (§4 row after Pane death), `:271/:273` (tick delta list + level-triggered sentence), `:349` (`⏏ shell` frame marker), `:496-499` (§6 grammar)
- [x] A-010 R11: `harness-adapters.md` and `skills.md` updated; no `pane_death`-enumerating site in `src/kit` or `docs/specs` omits `agent_exited` — `harness-adapters.md:188-191` gained the interactive-vs-worker sentence; `skills.md`'s `/fab-operator` section does not restate the delta list/§4 rows (verified by grep — nothing to update); every `pane_death` enumeration in `src/kit` names `agent_exited`; `docs/specs` has no `pane_death` enumeration

### Behavioral Correctness

- [x] A-011 R1: after the wrapped agent exits, the pane shows the user's shell in the same cwd — verified by live probe: `tmux new-window "true; exec \"$SHELL\""` left the window alive with `pane_current_command` = `zsh`; `exec` preserves the wrapper's cwd
- [x] A-012 R6: a stale `idle:<epoch>:<pid>` pane option on an exited pane does not surface the pane in `candidates:` — the predicate is `row.command` only, never agent state; "stale idle agent state yields no candidate and leaves the baseline untouched" subtest

### Removal Verification

- [x] A-013 R3: **N/A** — nothing removed; recorded to keep the category explicit

### Scenario Coverage

- [x] A-014 R3: dispatch/pane-open tests assert the unwrapped command and a negative assertion for the suffix exists — `internal/pane/create_test.go` `TestPaneCreatorsNeverShellWrap`; `create.go`/`dispatch_start.go`/`pane open` untouched
- [x] A-015 R5: tests cover shell / non-shell / empty `command`, and `TestOperatorTickDiff_AllEventKinds` includes `agent_exited` — `operator_tick_diff_test.go`

### Edge Cases & Error Handling

- [x] A-016 R4: an rk row without `command` and a legacy 8-field tmux line both yield `command == ""` and never emit `agent_exited` — `pane_map_test.go` legacy-line and rk-row-without-command cases; `isShellCommand("")` is false
- [x] A-017 R7: a pane that is both mismatched and shell-hosted emits exactly one delta (`pane_mismatch`) — "mismatch wins over agent_exited" subtest

### Code Quality

- [x] A-018 Pattern consistency: new tick branch mirrors the `pane_mismatch` block's shape and comment style; MarshalYAML key order pinned (`kind, change, pane, command`) — `operator_tick_start.go:461-479` mirrors the mismatch block; `MarshalYAML:158-159` pins the key order
- [x] A-019 No unnecessary duplication: one Go helper for the suffix; one prose owner for the rule (owner-or-pointer) — `internal/spawn.WithShellFallback` is the single Go owner (3 call sites); `_cli-agents.md` § Spawn Composition is the single prose owner
- [x] A-020 Canonical source only: no edits under `.claude/skills/`; all skill edits in `src/kit/skills/` — diff touches only `src/kit/skills/` (4 files); no `.claude/` or `docs/memory/` edits
- [x] A-021 CLI ⇒ docs + tests: `_cli-fab.md` updated for the new delta kind/field and the ninth format field; Go changes ship tests; `gofmt -l` clean — `go build ./...` + `go test ./internal/spawn/... ./cmd/fab/... ./internal/pane/...` all pass; `gofmt -l` over the 13 changed `.go` files is empty
- [x] A-022 Sibling sweep: `pane_death` enumerations swept across `_cli-fab.md`, `fab-operator.md`, `docs/specs/skills.md` — repo-wide grep: every kind-enumerating site names `agent_exited`; `skills.md` does not enumerate kinds
- [x] A-023 No migration needed: no user data restructured (additive delta kind, internal row field) — state-file schema untouched; `command` is snapshot-internal and never rendered

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Hydrate owns `docs/memory/` (the intake's Affected Memory list); apply does not edit memory files.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. `pane_death` stays load-bearing for dispatch pane workers (unwrapped spawns) and any pane killed outright; no symbol, branch, or config became unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fallback wraps the WHOLE existing shell command after `withWorkersEnv` | Preserves the env-prefix semantics exactly (env scopes the agent command only) | S:90 R:95 A:95 D:90 |
| 2 | Confident | `command` is snapshot-internal — no new `fab pane map` column/JSON key | Keeps rk-contract parity ("fab adds no new output fields from the delegation"); nothing consumes it externally yet | S:80 R:90 A:90 D:80 |
| 3 | Confident | Shell set is the fixed list `sh bash zsh fish dash ksh tcsh csh nu`, basename-matched | Intake open question resolved for the first cut; trivially extendable | S:70 R:95 A:85 D:75 |
| 4 | Confident | `agent_exited` fleet row reuses `baselineFleetRow` (null observed fields) rather than a new renderer | Mirrors `pane_death`/`pane_mismatch`; summary buckets it under `unknown` for free | S:80 R:90 A:90 D:85 |
| 5 | Confident | Frame marker wording left to the skill-prose task within "must not read as a live agent" | Cosmetic; intake open question | S:60 R:95 A:80 D:65 |
| 6 | Tentative | `fab batch switch`'s spawn (fire-and-forget `exec.Command(...).Run()`) is wrapped identically without changing its error handling | In scope per intake row 4; error-handling changes are out of scope | S:65 R:90 A:85 D:70 |

6 assumptions (1 certain, 4 confident, 1 tentative).
