# Plan: Read `@rk_pane_agent_state` with legacy `@rk_agent_state` fallback

**Change**: 260828-xdmh-rk-pane-agent-state-option
**Intake**: `intake.md`

## Requirements

### Runtime: agent-state option read contract

#### R1: Canonical option name is `@rk_pane_agent_state`; legacy name is a fallback
`internal/pane` MUST expose `AgentStateOption = "@rk_pane_agent_state"` and `LegacyAgentStateOption = "@rk_agent_state"`. `ReadAgentStateOption(paneID, server)` MUST read the canonical option first via `tmux show-options -pv -t <pane>` and, only when that call errors or returns an empty trimmed value, repeat the identical call for the legacy option. The empty-`paneID` refusal and error→unknown (`""`) mapping are unchanged.

- **GIVEN** a pane with only `@rk_pane_agent_state = idle:1700000000`
- **WHEN** `ReadAgentStateOption` runs
- **THEN** it returns `idle:1700000000` with one tmux call

- **GIVEN** a pane with only legacy `@rk_agent_state = active:1700000000`
- **WHEN** `ReadAgentStateOption` runs
- **THEN** it returns `active:1700000000` (second call)

- **GIVEN** both options set to different values
- **WHEN** `ReadAgentStateOption` runs
- **THEN** the canonical value is returned

- **GIVEN** neither option set (or a missing pane / dead server)
- **WHEN** `ReadAgentStateOption` runs
- **THEN** it returns `""`

#### R2: `parseAgentState` tolerates run-kit's optional third `:pid` segment
run-kit Change 3 widened the value grammar to `"<state>:<epoch_seconds>[:<pid>]"` (`docs/specs/agent-state.md` § The Option: "Readers MUST tolerate its absence (legacy two-segment values). A malformed value — wrong segment count, unknown state, non-integer epoch, or a malformed/non-positive pid — is wholly unknown"). `parseAgentState` MUST accept exactly two or three colon-separated segments: segment 1 ∈ `{active, waiting, idle}`, segment 2 a parseable int64 epoch, and — when present — segment 3 a positive int64 pid. Any other shape is `ok=false`. The returned `(state, epoch)` pair and every caller (`AgentDisplayFromOption`, `ResolvePaneContext`, `parsePaneLines`) are unchanged.

- **GIVEN** raw `waiting:1751790000:48213`
- **WHEN** `parseAgentState` runs
- **THEN** it returns `("waiting", 1751790000, true)`

- **GIVEN** raw `idle:1751800000` (legacy two-segment)
- **WHEN** `parseAgentState` runs
- **THEN** it returns `("idle", 1751800000, true)`

- **GIVEN** raw `idle:1751800000:0`, `idle:1751800000:abc`, `idle:1751800000:1:2`, `idle:1751800000:-5`
- **WHEN** `parseAgentState` runs
- **THEN** every one returns `ok=false`

#### R3: `fab pane map` fallback enumeration carries both keys and prefers the new one
`tmuxPaneFormat` MUST be the eight-field string `#{pane_id}\t#{window_name}\t#{pane_current_path}\t#{session_name}\t#{window_index}\t#{@rk_pane_agent_state}\t#{@rk_agent_state}\t#{window_id}`. `parsePaneLines` MUST split with width 8 and apply graded tolerance: 8 fields → agent state from field 6 if non-empty (after `TrimSpace`) else field 7, `windowID` = field 8; 7 fields (legacy layout) → field 6 agent state, field 7 `windowID`; 6 → field 6 agent state, no `windowID`; 5 → no agent state; < 5 → skipped. The newline-only per-line trim stays. The two comment blocks (format constant, parser) MUST describe the eight-field layout, both possibly-empty middle fields, the prefer-new rule, and the trailing-never-empty invariant. The rk-delegated path (`parseRKPanes`) is untouched.

- **GIVEN** a line with field 6 `idle:1700000000`, field 7 empty, field 8 `@3`
- **WHEN** `parsePaneLines` runs
- **THEN** the entry's `agentState` is `idle` and `windowID` is `@3`

- **GIVEN** a line with field 6 empty, field 7 `active:1700000000`, field 8 `@3`
- **WHEN** `parsePaneLines` runs
- **THEN** `agentState` is `active`

- **GIVEN** a line with field 6 `waiting:1700000000:42` and field 7 `idle:1600000000`
- **WHEN** `parsePaneLines` runs
- **THEN** `agentState` is `waiting` (new wins)

- **GIVEN** the existing legacy 7-, 6-, and 5-field fixture lines
- **WHEN** `parsePaneLines` runs
- **THEN** the pre-change results hold verbatim

#### R4: Kit/doc prose names the canonical option and the deprecation fallback
Every non-historical occurrence of `@rk_agent_state` in `src/kit/skills/*.md`, `src/go/fab/cmd/fab/skill.md`, `docs/site/skill.md`, `docs/specs/*.md`, and `src/go/fab/**/*.go` comments MUST read `@rk_pane_agent_state`. The two canonical owner passages — `src/kit/skills/_cli-fab.md` § agent state and `docs/memory/runtime/runtime-agents.md` § Read Contract (the latter written at hydrate) — MUST additionally state (a) the legacy `@rk_agent_state` is read as a fallback during run-kit's deprecation window, (b) the value grammar `"<state>:<epoch_seconds>[:<pid>]"` with the pid segment ignored by fab, and (c) the rk version floor: the canonical name is written by the first run-kit release after v3.18.7 (the one carrying run-kit PR #755, "tmux Option Dual-Read for Externally-Written Keys"); older `rk agent setup` hook generations write only the legacy name. `@rk_role` mentions MUST read `@rk_win_role`. Historical rows (`docs/memory/**/log.md`), `src/kit/migrations/2.13.6-to-2.14.0.md`, and `fab/changes/archive/` are left untouched. `docs/memory/**` prose is rewritten at hydrate, not apply.

- **GIVEN** the sweep grep `grep -rn 'rk_agent_state' src docs/specs docs/site README.md`
- **WHEN** apply completes
- **THEN** every hit outside `2.13.6-to-2.14.0.md` is either `@rk_pane_agent_state` or an explicit legacy-fallback mention naming `@rk_agent_state` as the retired name

### Non-Goals
- Removing the legacy read — a later follow-up sequenced after run-kit drops its own legacy reads.
- Consuming the `:pid` segment (PID-liveness reconciliation is rk's; fab ignores it).
- Any `fab pane` flag or JSON key change (`agent_state` stays).
- Touching run-kit or the `rk mux panes --json` delegation.

### Design Decisions

#### Two `show-options -pv` calls rather than one format read
**Decision**: `ReadAgentStateOption` issues the canonical read, then the legacy read only on error/empty.
**Why**: `show-options -pv` reads pane scope strictly and already carries a documented error→unknown mapping; `#{@opt}` format expansion walks pane→window→session→global — the scope leak run-kit's rename exists to fix. The second call fires only for legacy-hook and uninstrumented panes.
**Rejected**: `tmux display-message -p -t <pane> -F '#{@rk_pane_agent_state}\t#{@rk_agent_state}'` — one call, but inherits outer-scope values and changes the missing-option failure mode.
*Introduced by*: 260828-xdmh-rk-pane-agent-state-option

#### Tolerate, don't consume, the pid segment
**Decision**: `parseAgentState` accepts 2 or 3 segments, validates the pid as a positive integer, and discards it.
**Why**: run-kit Change 3 writers now emit `state:epoch:pid`; the previous `LastIndex(":")` parse would have read `waiting:1751790000` as the state token and resolved every current hook's value to unknown. Validating (not ignoring) the third segment honours run-kit's "malformed ⇒ wholly unknown" contract.
**Rejected**: splitting only on the first colon and ignoring the rest — accepts garbage tails the writer contract calls malformed.
*Introduced by*: 260828-xdmh-rk-pane-agent-state-option

## Tasks

### Phase 1: Setup

- [x] T001 Precondition + floor: verify run-kit `origin/main` carries `@rk_pane_agent_state` in Go (`git -C ~/code/sahil87/run-kit grep -l rk_pane_agent_state origin/main -- '*.go'`) and record the floor wording "first run-kit release after v3.18.7 (PR #755)" for T004/T005; STOP if the grep is empty <!-- R4 -->

### Phase 2: Core Implementation

- [x] T002 `src/go/fab/internal/pane/pane.go`: `AgentStateOption` → `@rk_pane_agent_state`, add `LegacyAgentStateOption`, dual-read in `ReadAgentStateOption`, 2-or-3-segment `parseAgentState`; update doc comments; extend `TestParseAgentState` (pid cases) and `TestReadAgentStateOption_Integration` (new-only / legacy-only / both / neither) in `pane_test.go`; `go test ./internal/pane/` <!-- R1 R2 -->
- [x] T003 `src/go/fab/cmd/fab/pane_map.go`: eight-field `tmuxPaneFormat`, width-8 `parsePaneLines` with prefer-new rule, rewrite both comment blocks; extend `TestParsePaneLines` (new-only, old-only, both-set, legacy 7/6/5 re-asserted) in `pane_map_test.go`; sweep `pane_capture_test.go` literals; `go test ./cmd/fab/ -run 'ParsePaneLines|Pane'` <!-- R3 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Prose sweep outside `docs/memory/`: `src/kit/skills/{_cli-fab,_cli-agents,_cli-external,fab-operator}.md` (incl. `@rk_role`→`@rk_win_role`), `src/go/fab/cmd/fab/skill.md` + `docs/site/skill.md` twins, `docs/specs/{hooks,architecture,index}.md`; `_cli-fab.md` § agent state gains the fallback sentence, the `[:<pid>]` grammar, and the rk floor; verify with the R4 grep <!-- R4 -->

### Phase 4: Polish

- [x] T005 Full package tests `go test ./internal/pane/ ./cmd/fab/` green; `gofmt -l src/go` empty; `fab sync` so deployed skills match <!-- R1 R2 R3 R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `AgentStateOption == "@rk_pane_agent_state"` and `LegacyAgentStateOption == "@rk_agent_state"` exist in `internal/pane`
- [x] A-002 R1: `ReadAgentStateOption` returns the canonical value when set, the legacy value when only it is set, `""` when neither
- [x] A-003 R2: `parseAgentState` accepts `state:epoch` and `state:epoch:pid` and rejects other segment counts / malformed pid
- [x] A-004 R3: `tmuxPaneFormat` has eight fields with `#{@rk_pane_agent_state}` at 6, `#{@rk_agent_state}` at 7, `#{window_id}` trailing
- [x] A-005 R4: no non-historical `@rk_agent_state` occurrence remains outside an explicit legacy-fallback mention (R4 grep)

### Behavioral Correctness

- [x] A-006 R1: when both options are set with different values the canonical one wins in `ReadAgentStateOption`
- [x] A-007 R3: when both fields are non-empty in a list-panes line the canonical field wins in `parsePaneLines`
- [x] A-008 R2: a current run-kit three-segment value (e.g. `waiting:1751790000:48213`) resolves to `waiting`, not unknown

### Scenario Coverage

- [x] A-009 R1: `TestReadAgentStateOption_Integration` covers new-only, legacy-only, both-set, neither
- [x] A-010 R3: `TestParsePaneLines` covers new-only, old-only, both-set, and re-asserts legacy 7/6/5-field lines
- [x] A-011 R2: `TestParseAgentState` covers the pid-segment accept and reject cases

### Edge Cases & Error Handling

- [x] A-012 R1: empty `paneID` still returns `""` without invoking tmux; a missing pane / dead server still maps to `""`
- [x] A-013 R3: an eight-field line with both agent fields empty resolves to unknown with `windowID` populated; the newline-only trim is retained

### Code Quality

- [x] A-014 Pattern consistency: new code follows the surrounding constant/comment/error-handling style in `pane.go` and `pane_map.go`
- [x] A-015 No unnecessary duplication: the fallback logic lives once in `ReadAgentStateOption`; the parser stays the single grammar authority
- [x] A-016 No magic strings: both option names are referenced via the exported constants in code and tests
- [x] A-017 Canonical source only: no edit under `.claude/skills/`; skill edits in `src/kit/skills/`
- [x] A-018 Owner-or-pointer: the fallback/floor rule is stated in `_cli-fab.md` § agent state (+ memory at hydrate) and only pointed at elsewhere
- [x] A-019 Go changes ship tests; `gofmt` clean

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — the refactor is strictly additive (dual-read + pid tolerance); the legacy `@rk_agent_state` reads it introduces alongside the new ones are the deprecation-window contract, and their removal is an explicitly deferred follow-up, so no existing file, symbol, or branch became redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Tolerating the optional `:pid` third segment is in scope (R2) even though the intake did not name it | run-kit Change 3's shipped spec makes every current writer emit three segments; without R2 the rename would prefer a key whose values fab cannot parse — the change would be functionally dead. Validated, not consumed | S:85 R:90 A:90 D:90 |
| 2 | Certain | Gate treated as satisfied on *merged* (PR #755 on run-kit `main`) even though no tag follows yet | The convention and the dual-write hooks exist on main; the tag only fixes the floor number, expressed as "first release after v3.18.7 (#755)" | S:85 R:85 A:85 D:85 |
| 3 | Certain | `docs/memory/**` prose is rewritten at hydrate, not in T004 | Hydrate owns memory writes; doing both would double-edit the same files | S:85 R:95 A:90 D:90 |
| 4 | Confident | Floor wording names the PR rather than a version literal | No tag exists to cite; a placeholder version would be a guess. Hydrate/memory can be corrected to the literal once tagged | S:70 R:90 A:60 D:75 |

4 assumptions (3 certain, 1 confident).
