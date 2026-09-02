# Plan: Labelled-Rung Dispatch Choreography

**Change**: 260902-0i4x-labelled-rung-dispatch-choreography
**Intake**: `intake.md`

## Requirements

### Skills: CLI-Adapter Dispatch choreography

#### R1: Step 1 branches on the labelled rung
`src/kit/skills/_preamble.md` § CLI-Adapter Dispatch step 1 SHALL branch on the resolution's `dispatch.rung` label: `rung: pane` goes straight to § The pane readiness gate (`open` → gate → `deliver`, then step 2's `wait`); `rung: headless` goes straight to `fab dispatch start <change> <stage>` with the full stage prompt on stdin (existing `--timeout` guidance unchanged). The discovery-probe teaching — "`start` is how you learn the mode", "attempt `start` first and let its answer be the discriminator", and the "deliberately does not consume that label until the successor change" clause — SHALL be removed. The "Remember this landing" instruction (the deferred apply reap fires only on the pane arm) SHALL key on `rung: pane` at resolution time.

- **GIVEN** a surfaced `fab agent <stage> -o yaml` result whose `dispatch:` mapping carries `rung: pane`
- **WHEN** an orchestrator follows § CLI-Adapter Dispatch step 1
- **THEN** it runs `open` → gate → `deliver` directly, with no preceding `fab dispatch start` probe
- **AND** it remembers the pane landing for step 3's deferred apply reap

- **GIVEN** a `dispatch:` mapping carrying `rung: headless`
- **WHEN** the orchestrator follows step 1
- **THEN** it sends the stage prompt on stdin to `fab dispatch start` directly

#### R2: `start`'s pane-refusal stays as defense-in-depth
The prose SHALL retain a short note that `fab dispatch start` still refuses a pane landing before stdin is read and before any state write (the unchanged Go contract, owned by `docs/memory/runtime/dispatch.md`), so a stale-environment or mislabelled landing errors cleanly and names `open` — documented as a safety net, never as the discovery mechanism. No `src/go/` file changes.

- **GIVEN** the rewritten step 1
- **WHEN** a reader asks what happens if `start` is invoked on a pane landing anyway
- **THEN** the prose answers: refused pre-stdin/pre-state-write, re-run as `open` — defense-in-depth

#### R3: The "cheap shortcut" parenthetical is removed
The parenthetical "(One cheap shortcut, since the ladder descends only and never ascends: a `dispatch.mode` other than `pane` can never land on pane, so under the default `native` the `dispatch:`-present branch is always headless.)" SHALL be removed — it compensated for not reading the label.

- **GIVEN** the rewritten step 1
- **WHEN** grepping `_preamble.md` for "cheap shortcut"
- **THEN** no match remains

#### R4: Behavior-claim sweep of the discriminator restatements
The two memory restatements SHALL be rewritten to the labelled-rung branch — `docs/memory/pipeline/execution-skills.md` (the wiring sentence in the status-transition-ownership paragraph) and `docs/memory/runtime/dispatch.md` (the skill-wiring paragraph) — keeping their Go-contract statements (`start` SHALL refuse a pane landing; `open` is pane's entry) intact. A repo-wide sweep (token, phrase-class, and contrastive forms: "until the successor change", "learn the mode", "let its answer", "attempt `start` first", "discriminator" in dispatch context, "cheap shortcut") SHALL find no remaining discovery-mechanism claims outside `fab/changes/` artifacts and `fab/plans/` history.

- **GIVEN** the sweep greps run repo-wide after the edits
- **WHEN** matches are classified
- **THEN** every live doc/skill hit is the rewritten labelled-rung prose or an unchanged Go-contract/defense-in-depth statement

### Non-Goals

- No Go changes (`fab dispatch` verbs, `SelectMode`, resolver output/schema all untouched)
- No change to the readiness gate, stall guard, reap timing, recovery budget, or Dispatch-Prompt Obligations
- No edits to `_pipeline.md` / `fab-continue.md` / `fab-adopt.md` (they point at the canon) or to deployed `.claude/skills/` copies
- No spec edits — `docs/specs/harness-adapters.md`'s "start MUST refuse a pane landing" is the unchanged Go contract (verify via sweep)

### Design Decisions

#### Consume the rung label; keep the refusal as a net
**Decision**: Step 1 branches on `dispatch.rung` directly; `fab dispatch start`'s pane-refusal is retained in prose only as defense-in-depth.
**Why**: The label is computed by the same `SelectMode` ladder `fab dispatch` re-resolves, so reading it loses no correctness, saves one refused-CLI-invocation per pane dispatch, and removes probe-and-branch indirection; the Go refusal still catches resolution-to-launch environment skew.
**Rejected**: Keeping the probe as the primary mechanism with the label as a hint — pays the probe forever and leaves the "successor change" IOU dangling; deleting the refusal mention entirely — loses the skew story the reader needs.
*Introduced by*: 260902-0i4x-labelled-rung-dispatch-choreography

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch step 1: branch on `rung: pane|headless`, re-head the two mode bullets, key "Remember this landing" on `rung: pane`, remove discriminator teaching + free-probe rationale + "cheap shortcut" parenthetical, add one defense-in-depth sentence for `start`'s pane-refusal <!-- R1 R2 R3 --> <!-- rework: A-008 owner-or-pointer — defense-in-depth note restated the refusal's full mechanics (pre-stdin, pre-refuse-if-running, pre-write, no-consumption) with no owner pointer; trim to safety-net summary + pointer at the runtime-details owner (`_cli-fab.md` § fab dispatch — the deployed-kit owner; docs/memory is not shipped to user projects, so the reviewer's suggested memory pointer is unportable) -->

### Phase 3: Integration & Edge Cases

- [x] T002 [P] Rewrite the discriminator restatement in `docs/memory/pipeline/execution-skills.md` (wiring sentence ~line 21) to the labelled-rung branch, Go-contract claims untouched <!-- R4 -->
- [x] T003 [P] Rewrite the discriminator restatement in `docs/memory/runtime/dispatch.md` (skill-wiring paragraph ~line 25) likewise <!-- R4 -->

### Phase 4: Polish

- [x] T004 Repo-wide sweep greps (token + phrase-class + contrastive: "until the successor change", "learn the mode", "let its answer", "attempt `start` first", dispatch-context "discriminator", "cheap shortcut") over src/, docs/, README, docs/site; classify every hit; run `fab memory-index --check` and regen if flagged <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `_preamble.md` step 1 branches on `dispatch.rung` (pane → `open`→gate→`deliver`; headless → `start` stdin) with no preceding probe step
- [x] A-002 R2: A defense-in-depth sentence for `start`'s pane-refusal is present; `git diff` touches zero `src/go/` files
- [x] A-003 R4: Both memory restatements teach the labelled-rung branch; their Go-contract requirement statements are unchanged

### Behavioral Correctness

- [x] A-004 R1: The deferred-apply-reap arm gate ("only if step 1 landed on pane") still holds, now keyed on `rung: pane` (plus the existing lost-context record check)

### Removal Verification

- [x] A-005 R3: "cheap shortcut" parenthetical gone from `_preamble.md`
- [x] A-006 R4: Sweep greps return no live discovery-mechanism claims outside `fab/changes/` + `fab/plans/` history

### Code Quality

- [x] A-007 Owner-or-pointer preserved: dispatch sites and memory point at the `_preamble.md` canon without restating the branch mechanics beyond their existing summary depth
- [x] A-008 No unnecessary duplication: the defense-in-depth note states the refusal once, pointing at the Go-contract owner rather than restating its full behavior

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before hydrate
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The free-probe *fact* (refusal fires pre-stdin, pre-state-write) survives inside the defense-in-depth sentence, not as a standalone rationale block | The fact is load-bearing for the skew story; only its probe-justifying framing is removed | S:75 R:85 A:85 D:80 |
| 2 | Confident | "Remember this landing" reword keys the reap gate on `rung: pane` while keeping the `.fab-dispatch/{id}/apply.yaml` `pane:` record check for lost-context orchestrators | The record check exists for orchestrators that lost context; the label is only in-context state | S:70 R:85 A:85 D:75 |

2 assumptions (0 certain, 2 confident, 0 tentative).
