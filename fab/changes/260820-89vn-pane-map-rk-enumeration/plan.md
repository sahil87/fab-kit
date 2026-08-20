# Plan: Pane Map on rk Enumeration

**Change**: 260820-89vn-pane-map-rk-enumeration
**Intake**: `intake.md`

## Requirements

### Runtime: `fab pane map` enumeration delegation

#### R1: Delegated enumeration via `rk mux panes --json`
When `rk` resolves on PATH, `fab pane map` SHALL source its pane enumeration from `rk mux panes --json` (appending `-L <server>` when the `--server`/`-L` flag was passed) instead of its own `tmux list-panes` call. Each rk row SHALL map: `pane`←`pane`, `tab`←`window_name`, `cwd`←`cwd`, `session`←`session`, `index`←`window_index`, `windowID`←`window_id`, and agent state ← rk's reconciled `agent_state` + `agent_state_duration` taken structurally — the delegated path MUST NOT re-read the `@rk_agent_state` pane option. rk-only fields (`session_id`, `window_active`, `pane_index`, `pane_active`, `command`) are ignored.

- **GIVEN** rk ≥ 3.17.18 on PATH and a tmux server with panes
- **WHEN** `fab pane map --all-sessions --json` runs
- **THEN** the rows come from rk's enumeration (one `rk mux panes --json` subprocess, no `tmux list-panes`), with `agent_state` carrying rk's reconciled value
- **AND** the per-cwd change/stage/display_state/PR enrichment runs identically to today

#### R2: Fail-open fallback, silent and byte-identical
Any failure of the delegation — `rk` absent from PATH, a pre-`mux panes` rk (unknown command, non-zero exit), a non-zero exit for any reason, or unparseable JSON — SHALL fall back silently to the existing internal `tmux list-panes` enumeration. No warning, no error, no output difference from today's behavior on that path. The attempt IS the capability probe; there is no version check.

- **GIVEN** a machine without rk (or with rk v3.16)
- **WHEN** `fab pane map` runs
- **THEN** output is byte-identical to the pre-change binary — internal enumeration, raw `@rk_agent_state` option parse, raw row set

#### R3: Session scoping is applied by fab as a row filter
`rk mux panes` always enumerates the whole server; fab SHALL preserve its flag contract by filtering the delegated rows: default mode filters to the current session (name resolved via `tmux display-message -p '#{session_name}'`, honoring `--server`; the existing `$TMUX`-required guard stays ahead of discovery), `--session <name>` filters on session-name equality, `--all-sessions` applies no filter. If the current-session name resolution fails in default mode, the run SHALL fall back to the internal enumeration (R2's posture).

- **GIVEN** rk present and panes in sessions `alpha` and `beta`
- **WHEN** `fab pane map --session alpha` runs
- **THEN** only `alpha` rows render; **AND WHEN** run with no targeting flag from a pane in `beta`, **THEN** only `beta` rows render and `$TMUX` unset still errors `not inside a tmux session`

#### R4: Output contracts unchanged
Table columns (`Session`/`Pane`/`WinIdx`/`Tab`/`Worktree`/`Change`/`Stage`/`Agent`) and the JSON field set (`session`, `window_index`, `window_id`, `pane`, `tab`, `worktree`, `repo`, `change`, `stage`, `display_state`, `agent_state`, `agent_idle_duration`, `pr_url`, `pr_number`) SHALL NOT change. `agent_idle_duration` keeps idle-only semantics: rk reports `agent_state_duration` for `waiting` too — the delegated mapping SHALL drop a waiting-row duration rather than surface it. The Agent column keeps `active` / `waiting` / `idle (<dur>)` / `—`.

- **GIVEN** a delegated row with `agent_state: "waiting"`, `agent_state_duration: "3m"`
- **WHEN** rendered
- **THEN** the table shows `waiting` (no duration) and JSON emits `"agent_state": "waiting", "agent_idle_duration": null`
- **AND** a delegated `agent_state: null` row renders `—` / JSON `null` for both fields

#### R5: Row-set semantics differ deliberately between paths
The delegated path SHALL adopt rk's filtered view (no `_rk-pin-*` pin-sessions, no `_rk-ctl` anchor, pinned windows once via their home session). The fallback path SHALL keep today's raw enumeration verbatim — fab does not reimplement rk's filtering conventions.

- **GIVEN** a server with a `_rk-ctl` anchor session
- **WHEN** `fab pane map --all-sessions` runs with rk present
- **THEN** no `_rk-ctl` row appears; **AND** with rk absent the row appears as before

#### R6: CLI docs updated and enumeration-source claims swept
`src/kit/skills/_cli-fab.md` § fab pane map (and its § agent state note) SHALL document the delegation: enumeration source, silent fail-open fallback, the filtered-row-set and reconciled-agent-state deltas. Claims that pin `map`'s mechanics to the internal path — e.g. "read from the SAME `list-panes` call — zero extra subprocesses" — SHALL be scoped to the fallback path. Sweep `src/kit/skills/` and `docs/specs/` for such enumeration-source claims about `pane map` (docs/memory/ updates belong to hydrate).

- **GIVEN** the updated `_cli-fab.md`
- **WHEN** an agent reads § fab pane map
- **THEN** the delegated and fallback enumeration paths are both described, and no skill/spec text asserts the internal mechanics as unconditional

### Non-Goals

- `fab resolve --pane` keeps the internal enumeration — it needs only cwd-per-pane and is not part of the pane-map role Part 8 splits
- No new JSON fields or table columns (rk-only fields not adopted); no exit-code changes (`map` keeps plain exit-1)
- No rk version pin or hard dependency; no reimplementation of rk's session filtering in the fallback
- No skill-guidance re-point (`fab-operator.md`, `_cli-agents.md` keep invoking `fab pane map` unchanged — the delegation is inside the binary)
- No migration (no user-data restructuring)

### Design Decisions

#### Structured agent-state pair replaces the raw-option thread
**Decision**: `paneEntry`/`paneRow` carry a resolved `(agentState, agentIdleDur)` display pair instead of the raw `@rk_agent_state` option string; the internal path resolves the pair at parse time via the existing `pane.AgentDisplayFromOption`, the delegated path fills it from rk's JSON (idle-only duration filter applied at mapping).
**Why**: rk rows carry no raw option — reconciled state + formatted duration only — so the raw string cannot remain the shared row representation; resolving once at discovery gives both paths one downstream render/JSON seam.
**Rejected**: Synthesizing a fake `state:epoch` raw value from rk rows (cannot reconstruct an epoch from a formatted duration; a lie in the data model); dual-field rows with a source discriminator (two render paths to keep in sync).
*Introduced by*: 260820-89vn-pane-map-rk-enumeration

#### Injectable exec seam for the rk call
**Decision**: The rk invocation goes through a package-level function variable (LookPath + run, defaulting to `exec.LookPath` / `exec.Command`), so fallback-trigger tests stub the seam without a live rk or tmux.
**Why**: matches the repo's injectable-seam precedent (`internal/setupcheck.LookPathFunc`) and keeps the trigger matrix (absent / exec error / bad JSON) unit-testable.
**Rejected**: integration-only testing behind `exec.LookPath("rk")` guards (can't exercise the failure matrix deterministically on CI).
*Introduced by*: 260820-89vn-pane-map-rk-enumeration

#### Unknown `--session` name yields an empty result on the delegated path
**Decision**: `--session bogus` with rk present filters to zero rows and prints `No tmux panes found.` (exit 0); the internal path keeps tmux's own error for an unknown `-t` target.
**Why**: the filter model has no natural "session exists but is empty vs. doesn't exist" probe without an extra subprocess; the empty-result answer is honest and consistent with the delegated path's whole-server view.
**Rejected**: a `tmux has-session` pre-check per invocation (extra subprocess on the operator hot path to preserve an error message).
*Introduced by*: 260820-89vn-pane-map-rk-enumeration

## Tasks

### Phase 1: Core Implementation

- [x] T001 rk enumeration source in `src/go/fab/cmd/fab/panemap.go`: `rkPaneRow` JSON struct, pure `parseRKPanes([]byte) ([]paneEntry, error)` mapper (field mapping per R1; `agent_state` null → unknown; waiting-row duration dropped per R4), and `rkPanesArgs(server)` argv builder (`mux panes --json` [+ `-L <server>`]). Unit tests in `panemap_test.go`: happy mapping, null agent fields, waiting-duration drop, malformed JSON error, argv with/without server. <!-- R1, R4 -->
- [x] T002 Structured agent-state refactor in `src/go/fab/cmd/fab/panemap.go`: `paneEntry`/`paneRow` carry `(agentState, agentIdleDur)`; `parsePaneLines` resolves the pair via `pane.AgentDisplayFromOption`; `agentColumn`/`agentJSONFields` consume the pair. Adapt existing tests (`TestParsePaneLines`, `TestAgentColumn`, `TestAgentJSONFields`, JSON/table render tests) to the new seam — internal-path table and JSON output stay byte-identical. <!-- R4 -->
- [x] T003 Delegation wiring in `runPaneMap` (`src/go/fab/cmd/fab/panemap.go`): injectable rk exec seam (package-level var per the design decision), attempt-first delegation with silent fallback on LookPath miss / non-zero exit / parse error, session filtering (default mode resolves the current session via `tmux display-message -p '#{session_name}'` with `pane.WithServer`, falling back to internal enumeration if that fails; `--session` equality; `--all-sessions` unfiltered). Tests: all three fallback triggers use the internal path; filter modes on a stubbed rk payload. <!-- R1, R2, R3, R5 -->

### Phase 2: Docs & Sweep

- [x] T004 [P] Update `src/kit/skills/_cli-fab.md` § fab pane map + § agent state: delegated enumeration (rk-gated, fail-open), filtered-row-set and reconciled-state deltas, scope the "SAME `list-panes` call — zero extra subprocesses" claim to the fallback path. <!-- R6 -->
- [x] T005 Sweep `src/kit/skills/` and `docs/specs/` for `pane map` enumeration-source claims that become fallback-only (grep `pane map`, `list-panes`); fix each; then run `go test ./...` under `src/go/fab/cmd/fab` and `src/go/fab/internal/pane` and confirm green. <!-- R6 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: With rk stubbed present, enumeration comes from the rk seam (no `tmux list-panes` call on the delegated path) and rows map per the R1 field table, enrichment unchanged
- [x] A-002 R2: Each fallback trigger (rk absent, non-zero exit, malformed JSON) silently uses the internal enumeration with output identical to the pre-change path
- [x] A-003 R3: Default/`--session`/`--all-sessions` filtering verified against a stubbed whole-server payload; `$TMUX` guard intact; failed current-session resolution falls back internally

### Behavioral Correctness

- [x] A-004 R4: JSON field set and nullability unchanged; a waiting row emits `agent_idle_duration: null`; an idle row carries rk's duration; table Agent column renders `active`/`waiting`/`idle (<dur>)`/`—`
- [x] A-005 R5: No fab-side reimplementation of rk's session filtering exists on the fallback path (code inspection — fallback enumeration untouched)

### Scenario Coverage

- [x] A-006 R1: Live check on this machine (rk v3.17.18 present, inside tmux): `fab pane map --all-sessions` shows rk's filtered row set with change/stage populated for fab worktree panes
- [x] A-007 R2: `go test` green in `src/go/fab/cmd/fab` and `src/go/fab/internal/pane`

### Edge Cases & Error Handling

- [x] A-008 R3: Unknown `--session` name on the delegated path prints `No tmux panes found.` and exits 0 (documented delta); `agent_state: null` rows render as unknown, not an error

### Code Quality

- [x] A-009 Pattern consistency: rk exec seam follows the repo's injectable-seam precedent; argv building uses `pane.WithServer` where tmux is invoked; no `.claude/skills/` edits (canonical `src/kit/` only)
- [x] A-010 No unnecessary duplication: one downstream render/JSON seam for both enumeration paths (no dual agent-state model); existing helpers (`AgentDisplayFromOption`, `WithServer`, `toNullable`) reused
- [x] A-011 CLI ⇒ docs + tests: `_cli-fab.md` updated in the same change with test updates alongside (constitution Additional Constraints)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant; the internal `tmux list-panes` enumeration is deliberately retained as the fail-open fallback (R2/R5).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Package-level function-var exec seam (not a param-threaded runner) for the rk call | Smallest testable seam; `setupcheck.LookPathFunc` precedent; cmd/fab tests already monkey-friendly | S:60 R:80 A:75 D:70 |
| 2 | Confident | Default-mode current-session name resolved via `tmux display-message -p '#{session_name}'`; resolution failure → internal fallback | Only reliable name source under `$TMUX`; failure is an R2-class degradation, not an error | S:60 R:75 A:80 D:75 |
| 3 | Confident | Unknown `--session` on the delegated path = empty result (exit 0), not tmux's unknown-target error | Filter model has no existence probe without an extra hot-path subprocess; documented as a design decision | S:55 R:75 A:70 D:65 |
| 4 | Certain | Internal-path behavior stays byte-identical (render/JSON seams refactored, semantics preserved by existing tests) | R2/R4 pin it; the existing test suite is the oracle | S:80 R:85 A:90 D:85 |

4 assumptions (1 certain, 3 confident, 0 tentative).
