# Plan: Guidance Re-Point — capture/kill/process to rk mux Twins (cli-layering Part 7)

**Change**: 260820-4un7-guidance-repoint-rk-mux-twins
**Intake**: `intake.md`

## Requirements

### Kit Skills: Skill-Facing Peek/Kill/Process Guidance Rides the rk mux Twins

#### R1: `_cli-external.md` points capture guidance at rk and trims the fab copies from skill-facing prose
`_cli-external.md` SHALL repoint its § tmux "Pane capture" usage note at `rk mux capture` when rk is present (`command -v rk`-gated), failing open to raw `tmux capture-pane` when rk is absent — never an error — with usage ownership pointed at `_cli-agents.md` § Peek. The "Send keys" note's rk-absent state-read parenthetical SHALL name `fab pane map` only (dropping `fab pane capture --json`). The § Reference Model intro and the § rk (run-kit) fab-owned-usage index SHALL reflect the repoint (a pointer covering peek — owner stays `_cli-agents.md`), keeping owner-or-pointer discipline.

- **GIVEN** an operator-skill consumer reading `_cli-external.md` § tmux with rk installed
- **WHEN** it needs a pane's scrollback
- **THEN** the guidance directs it to `rk mux capture` (rk-gated), not `fab pane capture`
- **AND** with rk absent the guidance degrades to raw `tmux capture-pane` without error

#### R2: `_cli-agents.md` § Peek / § Pre-Send Validation / Scope Boundary repoint to the rk twins
`_cli-agents.md` SHALL: (a) drop `fab pane capture` from the Scope Boundary fab-commands list and extend the rk-riding sentence to cover peek/kill/process (`rk mux capture`/`kill`/`process`, raw-tmux fallback); (b) in § Pre-Send Validation step 2's rk-absent path, read state via `fab pane map` alone; (c) in § Peek, make the output axis `rk mux capture <pane> [-l N] [--json|--raw]` when rk is present (`command -v rk`-gated) with raw `tmux capture-pane -p -t <pane>` (tail-piped for a last-N window) as the rk-absent fallback, and make the state axis `fab pane map`'s Agent column plus `rk mux capture --json`'s reconciled agent-state fields; (d) name `rk mux process` (instrumentation-authoritative agent classification) and `rk mux kill` (agent-state-gated, `--force` override) where a natural slot exists — adding no new procedures. The state-writer caveat and capture-is-the-universal-fallback rule survive with only the command swap.

- **GIVEN** a consumer following § Peek with rk installed
- **WHEN** it reads a pane's output and agent state
- **THEN** the output axis is `rk mux capture` and the state axis is `fab pane map` / `rk mux capture --json` — `fab pane capture` appears nowhere in the section
- **GIVEN** the same consumer with rk absent
- **WHEN** it peeks
- **THEN** the fallback is raw `tmux capture-pane` for output and `fab pane map` for state, never an error

#### R3: `fab-operator.md`'s two capture call sites become rk-gated dual-path
`fab-operator.md` §5 Question Detection step 1 SHALL read `rk mux capture --raw -l 20 [-L <server>] <pane>` when rk is installed (`command -v rk`-gated), with a raw `tmux capture-pane` fallback when rk is absent — matching the skill's existing rk-present/rk-absent dual-path pattern for sends. The §5 answer-delivery re-capture sentence (which self-references step 1's command) SHALL follow the same form.

- **GIVEN** the operator scanning a `waiting` pane for a question, rk installed
- **WHEN** it captures the pane per §5 step 1
- **THEN** it runs `rk mux capture --raw -l 20 [-L <server>] <pane>` (flag-for-flag the old form, on the twin)

#### R4: `_cli-fab.md` § fab pane demotes capture/process/kill to dispatch-internal
`_cli-fab.md` SHALL mark `fab pane capture`, `fab pane process`, and `fab pane kill` as **dispatch-internal** — kept because the dispatch pane arm must work rk-less (cli-layering Part 7); skill-facing guidance uses the rk twins, owned by `_cli-agents.md` § Peek / `_cli-external.md` § tmux. The `### kill` entry's operator-facing rationale sentence SHALL be updated (operator removal points at `rk mux kill`; fab's kill remains for dispatch-internal/rk-less use). Command behavior, flags, exit codes, and help text are UNCHANGED — no Go edits.

- **GIVEN** an agent reading `_cli-fab.md` § fab pane
- **WHEN** it reaches the capture/process/kill entries
- **THEN** each carries the dispatch-internal demotion note with the rk-twin pointer, and the documented flag/exit-code surface is byte-equivalent to before

#### R5: Dispatch-internal call sites stay on `fab pane capture`; a repo-wide sweep enforces the boundary
The dispatch-orchestrator usages SHALL stay verbatim: `_preamble.md` § CLI-Adapter Dispatch's pane peek (`fab pane capture [-L <server>] <pane>`), `fab dispatch logs`' suggested capture command (`_cli-fab.md` § fab dispatch prose), and the `status --json`-assembles-the-capture-command claim. A repo-wide sweep of `src/kit/skills/*.md` for `fab pane capture`, `fab pane kill`, `fab pane process`, and contrastive phrases ("instead of raw `tmux capture-pane`", "stop falling back to raw `tmux kill-pane`") SHALL classify every hit as skill-facing (repoint) or dispatch-internal (keep) with no unclassified remainder.

- **GIVEN** the sweep over `src/kit/skills/`
- **WHEN** every `fab pane capture|kill|process` occurrence is classified
- **THEN** skill-facing occurrences are repointed, dispatch-internal occurrences (`_preamble.md` CLI-Adapter section, `_cli-fab.md` § fab dispatch, the § fab pane reference entries themselves) are unchanged, and none is left unclassified

### Non-Goals

- No Go/CLI changes: the three `fab pane` verbs keep behavior, flags, exit codes, and help visibility (no cobra hiding — the spec's mechanism is guidance-level).
- No operator kill/await *adoption* (a separate follow-up per the yxyi notes) — kill/process guidance additions are pointer-minimal.
- No changes to `fab pane open`/`ready`/`deliver`/`map`/`window-name` guidance (choreography/hybrid dispositions per the spec's split table).
- No memory edits during apply — `docs/memory/` updates are hydrate's (Affected Memory: `runtime/agent-primitives.md`, `runtime/operator.md`, `runtime/pane-commands.md`, `distribution/kit-architecture.md`).

### Design Decisions

#### Skill-Facing Fallback Is Raw tmux + `fab pane map`, Not `fab pane capture`
**Decision**: When rk is absent, skill-facing guidance degrades to raw `tmux capture-pane` (output) and `fab pane map` (agent state) — `fab pane capture/kill/process` disappear from skill-facing prose entirely, surviving only in dispatch-internal call sites.
**Why**: cli-layering rule 2 states the degradation target ("absence degrades to raw tmux / fab's internal builders"), and Part 5 set the same fallback shape for send/await; naming `fab pane capture` as the skill-facing fallback would contradict the Part 7 row's "dropped from skill-facing guidance".
**Rejected**: `fab pane capture` as the rk-absent fallback (keeps the verb in skill-facing guidance, defeating the demotion; two capture guidances is the drift liability being removed).
*Introduced by*: 260820-4un7-guidance-repoint-rk-mux-twins

#### Demotion Is Prose-Only — No Cobra Hiding
**Decision**: The three verbs stay visible in `fab pane --help`; the demotion is documentation status only (dispatch-internal notes in `_cli-fab.md`, removal from skill-facing guidance).
**Why**: The spec's stated mechanism is "dropped from skill-facing guidance"; hiding was an rk-side device for machine-invoked plumbing; `open`/`ready`/`deliver` stay visible with the same dual role, and a help-surface change would trigger the shll standards audit for zero layering gain.
**Rejected**: Marking the verbs hidden in cobra (a CLI surface change with Go edits, tests, and standards audit — disproportionate, and the pane arm's rk-less operators still legitimately type these verbs).
*Introduced by*: 260820-4un7-guidance-repoint-rk-mux-twins

## Tasks

### Phase 2: Core Implementation

- [x] T001 [P] Edit `src/kit/skills/_cli-external.md`: § tmux "Pane capture" bullet → rk-gated `rk mux capture` with raw-tmux fallback + ownership pointer; "Send keys" bullet's rk-absent state read → `fab pane map` only; § Reference Model intro parenthetical; § rk fab-owned-usage pointer extended to peek/kill/process <!-- R1 -->
- [x] T002 [P] Edit `src/kit/skills/_cli-agents.md`: Scope Boundary list + rk sentence; § Pre-Send Validation step 2 rk-absent state read; § Peek output/state axes rewritten to `rk mux capture` (rk-gated) with raw-tmux fallback; minimal `rk mux process`/`rk mux kill` mentions <!-- R2 -->
- [x] T003 [P] Edit `src/kit/skills/fab-operator.md`: §5 Question Detection step 1 capture command and the §5 answer-delivery re-capture self-reference → rk-gated dual path <!-- R3 -->
- [x] T004 [P] Edit `src/kit/skills/_cli-fab.md`: dispatch-internal demotion notes on `### capture`/`### process`/`### kill` (and/or the § fab pane header), rk-twin pointers, kill-entry rationale sentence update <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Repo-wide sweep of `src/kit/skills/*.md` (+ `docs/` spot-check) for `fab pane capture|kill|process` and contrastive phrases; classify each hit skill-facing vs dispatch-internal; verify `_preamble.md` CLI-Adapter peek and `_cli-fab.md` § fab dispatch prose unchanged <!-- R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `_cli-external.md` § tmux directs capture at rk-gated `rk mux capture` with raw-tmux fallback; rk-absent state read names `fab pane map` only; § rk pointer covers peek/kill/process
- [x] A-002 R2: `_cli-agents.md` § Peek's output axis is `rk mux capture` (rk-gated, raw-tmux fallback); state axis is `fab pane map` + `rk mux capture --json`; Scope Boundary list drops `fab pane capture`
- [x] A-003 R3: both `fab-operator.md` §5 capture sites read `rk mux capture --raw -l 20 [-L <server>]` (rk-gated) with raw-tmux fallback
- [x] A-004 R4: `_cli-fab.md` capture/process/kill entries carry the dispatch-internal demotion note + rk-twin pointer; documented flags/exit codes unchanged

### Removal Verification

- [x] A-005 R2: no skill-facing section of `_cli-agents.md`/`_cli-external.md`/`fab-operator.md` names `fab pane capture`, `fab pane kill`, or `fab pane process` as the verb to use

### Scenario Coverage

- [x] A-006 R5: sweep log classifies every `fab pane capture|kill|process` occurrence in `src/kit/skills/` with zero unclassified; dispatch-internal sites (`_preamble.md` CLI-Adapter, `_cli-fab.md` § fab dispatch) byte-unchanged

### Edge Cases & Error Handling

- [x] A-007 R1,R2: every rk usage added is `command -v rk`-gated and fail-open (never an error when rk is absent), per `_preamble.md` § Run-Kit (rk) Reference

### Code Quality

- [x] A-008 Owner-or-pointer: no file both states a repointed rule and points at its owner; `_cli-external.md` points, `_cli-agents.md` owns peek usage
- [x] A-009 Canonical source only: all edits under `src/kit/skills/`, none under `.claude/skills/`
- [x] A-010 Sibling sweep done up front (contrastive phrases + user-facing string literals checked), not reactively

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `_cli-fab.md`'s § fab dispatch prose naming `fab pane capture` (logs suggestion, status-json assembly, escalation peek) is dispatch-internal and stays | It documents the dispatch orchestrator's own path — the rk-less pane arm the verbs are kept for | S:70 R:85 A:85 D:75 |
| 2 | Confident | The `### kill` entry keeps its content but reframes the operator-facing rationale; no restructuring of `_cli-fab.md` § fab pane beyond added notes | Reference completeness is the file's job; demotion is status, not deletion | S:65 R:85 A:80 D:70 |

2 assumptions (0 certain, 2 confident, 0 tentative).
