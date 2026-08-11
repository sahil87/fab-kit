# Plan: `fab pane send --answer` Mode

**Change**: 260811-answ-pane-send-answer-mode
**Intake**: `intake.md`

## Requirements

### CLI: the `--answer` gate mode

#### R1: `--answer` relaxes the state gate for answer-sends
`fab pane send` SHALL accept a new `--answer` bool flag. Under `--answer`, the agent-state gate SHALL permit `waiting` (and `idle`) and SHALL still refuse `active` with the existing exit-1, state-naming refusal. Pane-existence validation (exit 2 pane missing / exit 3 other tmux failure) SHALL apply unchanged in every mode. Plain-send behavior (no flags) SHALL be unchanged.

- **GIVEN** a pane whose `@rk_agent_state` is `waiting`
- **WHEN** `fab pane send %5 "2" --answer` runs
- **THEN** the text is sent (exit 0, `Sent to %5`)

- **GIVEN** a pane whose `@rk_agent_state` is `active`
- **WHEN** `fab pane send %5 "text" --answer` runs
- **THEN** the command refuses with exit 1 and an error naming the state
- **AND** nothing is typed into the pane

#### R2: Unknown state under `--answer` warns and proceeds
Under `--answer`, an unknown agent state (absent/unparseable `@rk_agent_state`) SHALL warn-and-send exactly as plain send does (`warning: agent state unknown — sending anyway` on stderr, exit 0). *(Settled at intake — user chose posture parity with plain send.)*

- **GIVEN** an uninstrumented pane (no `@rk_agent_state` option)
- **WHEN** `fab pane send %5 "y" --answer` runs
- **THEN** the warning goes to stderr and the send proceeds (exit 0)

#### R3: `--force` stays the superset override
`--force` SHALL retain its skip-everything meaning. When `--answer` and `--force` are both given, `--force` SHALL win: the state check is skipped entirely (pane-existence still enforced).

- **GIVEN** a pane whose `@rk_agent_state` is `active`
- **WHEN** `fab pane send %5 "text" --answer --force` runs
- **THEN** the send proceeds (state check skipped)

#### R4: Tests cover the gate matrix
`pane_send_test.go` SHALL cover the full four-state × three-mode matrix at the pure-function seam (the gate function extended from `idleGate`), including the `--answer`+`--force` combination, without cobra/tmux plumbing.

- **GIVEN** the extended gate function
- **WHEN** `go test` runs on the package
- **THEN** all matrix cells (idle/waiting/active/unknown × plain/answer/force) are asserted and pass

### Docs: CLI reference

#### R5: `_cli-fab.md` documents the flag
`src/kit/skills/_cli-fab.md` § fab pane send SHALL document `--answer` in the usage line and extend the validation-pipeline prose with the mode's gate behavior (waiting → send under `--answer`; active → refuse in both non-force modes; unknown → warn-and-send in both).

- **GIVEN** the updated section
- **WHEN** a skill consumer reads it
- **THEN** the four-state gate outcome is derivable for each of plain / `--answer` / `--force`

### Skills: rewire send paths onto the gated binary

#### R6: Operator send paths use `fab pane send`
`src/kit/skills/fab-operator.md` SHALL route text sends through the gated binary: the auto-answer delivery path (§ answer delivery, currently raw `tmux send-keys` at ~:375) SHALL use `fab pane send --answer <pane> <text>`; the routed-command path (§3, ~:103-106) SHALL keep its confirm-before-send policy and, after explicit confirmation targeting a `waiting` agent, use `--answer` (not `--force`, not raw keys); the header prose (~:23 "routes commands via `tmux send-keys`") SHALL name the gated binary. Answers that are key names rather than literal text (bare Enter, arrows, `C-c`) SHALL be called out as the one remaining raw `tmux send-keys` path for delivered workers.

- **GIVEN** the operator detects a `waiting` agent's menu prompt with a text answer
- **WHEN** it delivers the answer
- **THEN** the documented command is `fab pane send --answer`, with the surrounding choreography (re-capture, abort-on-change, delivery probe) unchanged

#### R7: `_cli-agents.md` § Pre-Send Validation is satisfiable
`src/kit/skills/_cli-agents.md` § Pre-Send Validation SHALL state the mode split — plain send for command routing to `idle` agents, `--answer` for answering a detected prompt on a `waiting` agent, `--force` as the deliberate skip-everything override — so its "prefer `fab pane send` … let the binary hold the gate" instruction no longer contradicts the operator's primary use case. Same key-name carve-out note.

- **GIVEN** the updated section
- **WHEN** the operator follows it for a `waiting`-agent answer
- **THEN** no route-around via raw `tmux send-keys` is required for literal-text answers

#### R8: Behavior-claim sweep
The old contract's claims ("the gate refuses `waiting`", "`active`/`waiting` refuse without `--force`" stated as the complete story, operator "routes commands via `tmux send-keys`") SHALL be swept across `src/kit/skills/` and `docs/specs/` (including test-file comments and contrastive phrases); every stale occurrence updated or annotated. `docs/memory/` is hydrate's territory and is NOT swept at apply.

- **GIVEN** the completed apply diff
- **WHEN** grepping `src/kit/` and `docs/specs/` for the old-contract phrases
- **THEN** no occurrence claims the pre-change refusal contract without the `--answer` mode

### Non-Goals

- No key-name send surface on `fab pane send` (arrows/`C-c`/bare Enter stay raw `tmux send-keys`) — new flag surface out of scope
- No change to `fab pane deliver` / `fab dispatch deliver` choreography — they gate on readiness, not the idle gate
- No migration — no user data restructured
- No SPEC-mirror work — mirror tree retired (constitution 1.6.0)
- No `docs/memory/` edits at apply — hydrate owns them (including the stale unknown-refuses claim in `runtime/operator.md` §49/§448)

### Design Decisions

#### Unknown state under `--answer` warns and proceeds
**Decision**: `--answer` keeps plain send's warn-and-send posture for unknown agent state.
**Why**: The operator's capture-based fallback detects prompts on uninstrumented (`—`) panes, so answers legitimately target unknown-state panes; refusing would force `--force` there and recreate the route-around problem the flag exists to kill.
**Rejected**: Stricter refuse-unknown under `--answer` — punishes exactly the uninstrumented-pane case the auto-answer fallback exists for.
*Introduced by*: 260811-answ-pane-send-answer-mode

#### `--force` precedence over `--answer`
**Decision**: When both flags are given, `--force` semantics win — the state check is skipped entirely.
**Why**: `--force` is fixed as the skip-everything override; a mode flag must never silently narrow it. Superset semantics is the only non-surprising combination.
**Rejected**: Erroring on the combination — needless friction; the flags are not contradictory, one strictly contains the other.
*Introduced by*: 260811-answ-pane-send-answer-mode

#### Key-name answers stay raw tmux
**Decision**: `fab pane send` continues to send literal text only (`send-keys -l`); key-name answers remain raw `tmux send-keys`, named explicitly as the one remaining raw path.
**Why**: The `-l` literal send is a deliberate safety design (key names inside answer text must not be interpreted); expressing key names would need new flag surface out of this change's scope.
**Rejected**: A `--key` mode in this change — scope creep on a refusal-contract change; backlog it if demand appears.
*Introduced by*: 260811-answ-pane-send-answer-mode

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add `--answer` flag to `src/go/fab/cmd/fab/pane_send.go`; extend the gate (`idleGate` → mode-aware pure function) to the four-state × three-mode matrix; keep pane-existence validation and exit-code family untouched <!-- R1, R2, R3 -->
- [x] T002 Extend `src/go/fab/cmd/fab/pane_send_test.go` with the full gate matrix (idle/waiting/active/unknown × plain/answer/force) at the pure-function seam; run `go test` on the package <!-- R4 -->

### Phase 2: Docs & Skill Rewiring

- [x] T003 [P] Update `src/kit/skills/_cli-fab.md` § fab pane send — usage line + validation-pipeline prose with the `--answer` gate behavior <!-- R5 -->
- [x] T004 [P] Rewire `src/kit/skills/fab-operator.md` (~:23, ~:103-106, ~:375) and `src/kit/skills/_cli-agents.md` § Pre-Send Validation onto `fab pane send --answer` (policy unchanged; key-name carve-out named in both) <!-- R6, R7 -->

### Phase 3: Sweep

- [x] T005 Behavior-claim sweep across `src/kit/` and `docs/specs/` for the old-contract phrases (incl. test comments and contrastive phrases); update every stale occurrence; `docs/memory/` deferred to hydrate <!-- R8 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `--answer` sends to a `waiting` pane (exit 0) and refuses an `active` pane (exit 1, state-naming error); pane-missing exits 2 even with `--answer`
- [x] A-002 R2: unknown agent state under `--answer` warns on stderr and sends (exit 0)
- [x] A-003 R3: `--answer --force` together behaves as `--force` (state check skipped, pane-existence enforced)

### Behavioral Correctness

- [x] A-004 R1: plain-send behavior is byte-identical to before — `idle` sends; `waiting`/`active` refuse; unknown warns-and-sends

### Scenario Coverage

- [x] A-005 R4: `go test` passes on `src/go/fab/cmd/fab/` with new matrix cases covering all twelve cells including the flag combination

### Edge Cases & Error Handling

- [x] A-006 R1: the `--answer` refusal of `active` keeps the exit-code family intact (1 gate refusal / 2 pane missing / 3 other tmux failure) and the error names the state

### Documentation Accuracy

- [x] A-007 R5: `_cli-fab.md` § fab pane send documents `--answer` and the four-state gate outcome for all three modes
- [x] A-008 R6: `fab-operator.md` text-answer sends ride `fab pane send --answer`; routed-path confirm policy intact; header prose names the gated binary; key-name carve-out present
- [x] A-009 R7: `_cli-agents.md` § Pre-Send Validation names the mode split and is satisfiable for `waiting`-agent answers
- [x] A-010 R8: no stale pre-change refusal-contract claim remains in `src/kit/` or `docs/specs/` (grep evidence)

### Code Quality

- [x] A-011 Pattern consistency: new Go code follows the pane-family patterns (exit-code scheme, pure-function gate seam, stderr warning style)
- [x] A-012 No unnecessary duplication: the gate is extended, not duplicated; skill edits reuse existing section structure
- [x] A-013 Owner-or-pointer: no skill edit restates a rule another file owns (state a rule it owns, or point at the owner — never both)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Gate refactor shape: extend the existing pure function (mode parameter) rather than a sibling function | Keeps the unit-testable seam `idleGate` established; comment says it was extracted precisely for testability | S:70 R:90 A:85 D:75 |
| 2 | Confident | Refusal message under `--answer` reuses the state-naming shape (wording may say the state directly rather than "not idle") | Intake row 4 fixes exit codes, leaves prose flexible; state-naming keeps three-state awareness | S:65 R:90 A:80 D:70 |
| 3 | Confident | Sweep scope at apply = `src/kit/` + `docs/specs/`; `docs/memory/` (incl. stale operator.md unknown-refuses claim) deferred to hydrate | Constitution II: memory is hydrate's artifact; sweeping it at apply would double-touch | S:75 R:85 A:85 D:80 |

3 assumptions (0 certain, 3 confident, 0 tentative).

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant. The raw `tmux send-keys` prose paths in `fab-operator.md` (header ~:23, §3 routed-command, §5 answer delivery) that the change retires were already replaced in-place by the apply diff; no Go symbol, file, or config became unused (the `idleGate` seam is extended, not superseded, and its sole production caller is unchanged).
