# Intake: Swap fab-ff and fab-fff Review Failure Behavior

**Change**: 260216-knmw-DEV-1030-swap-ff-fff-review-rework
**Created**: 2026-02-16
**Status**: Draft

## Origin

> Swap review failure behavior between fab-ff and fab-fff: fab-ff gets interactive rework (3 options presented to user), fab-fff gets fully autonomous rework with agent choosing the rework path, retry cap of 3 cycles, and escalation logic (fix code twice -> force escalate to tasks/spec)

One-shot mode. The change was discussed conversationally before `/fab-new` — the user proposed inverting the current review failure handling and the agent suggested bounded autonomous retry with escalation. Key decisions from the discussion:

- fab-ff should present the user with the 3 rework options (interactive) because the user is "still in the loop"
- fab-fff should be FULLY autonomous — the agent chooses the rework path itself and retries
- Autonomous retry needs a cap (agreed: 3 rework cycles) to prevent infinite loops
- Escalation logic: if "fix code" fails twice consecutively, force escalation to tasks/spec
- After the retry cap is exhausted, bail with a summary of what was tried

## Why

The current design has the review failure behavior backwards relative to the naming intuition:

1. **Naming mismatch**: `ff` (two f's) suggests "fast but I'm watching" — yet it bails immediately on failure, giving the user no chance to course-correct. `fff` (three f's) suggests "maximum autonomy, don't bother me" — yet it stops and presents an interactive menu.

2. **Confidence gate inconsistency**: `/fab-ff` already has a confidence gate, meaning the user asserted trust in the spec quality. When review *does* fail despite high confidence, letting the user steer the correction (interactive rework) makes more sense than bailing. Conversely, `/fab-fff` has no confidence gate and is designed for full-pipeline autonomy — autonomous rework is the natural completion of that contract.

3. **If we don't fix it**: Users will continue to find the behavior counterintuitive. The command that promises more autonomy (`fff`) interrupts more, and the command that implies user involvement (`ff`) gives the user the least control over failure recovery.

## What Changes

### 1. `/fab-ff`: Bail on failure -> Interactive rework

**Current behavior**: On review failure, `/fab-ff` bails immediately with `"Review failed. Run /fab-continue for rework options."`

**New behavior**: On review failure, `/fab-ff` presents the same 3 rework options currently described in `/fab-continue` review behavior:

- **Fix code** -> unchecks affected tasks in `tasks.md` with `<!-- rework: reason -->`, re-runs apply behavior
- **Revise tasks** -> user edits `tasks.md`, re-runs apply behavior
- **Revise spec** -> resets to spec stage, regenerates `spec.md`, invalidates downstream

The user chooses, the pipeline re-runs from the chosen point, and review runs again. This is a single rework cycle — if review fails again, the options are presented again.

### 2. `/fab-fff`: Interactive rework -> Autonomous rework with bounded retry

**Current behavior**: On review failure, `/fab-fff` presents an interactive rework menu (same 3 options).

**New behavior**: On review failure, `/fab-fff` autonomously chooses the rework path based on the review failure details. The agent analyzes the failure (code bugs vs. missing tasks vs. spec gaps) and selects the appropriate option without user input.

#### Retry Cap

Maximum **3 rework cycles**. Each cycle = rework + re-review. After 3 failed cycles, the pipeline bails with a summary:

```
Review failed after 3 rework attempts. Summary:
  Cycle 1: Fix code — {what was fixed}
  Cycle 2: Fix code — {what was fixed}
  Cycle 3: Revise tasks — {what was changed}
Run /fab-continue for manual rework options.
```

#### Escalation Logic

If the agent chooses "fix code" and the subsequent review fails again on the same or similar issues, the agent MUST escalate:

- **After 2 consecutive "fix code" attempts**: Force escalation to "revise tasks" or "revise spec" (agent decides which based on failure analysis)
- This prevents the agent from repeatedly patching symptoms when the root cause is upstream

The escalation is a hard rule, not a suggestion — even if the agent believes another code fix would work, it must escalate after 2 consecutive code-fix failures.

#### Agent Decision Heuristics

The agent chooses the rework path based on the review failure type:

- **Test failures, code quality issues, pattern violations** -> "Fix code"
- **Missing functionality, incomplete coverage, wrong task breakdown** -> "Revise tasks"
- **Spec drift, requirements mismatch, fundamental approach issues** -> "Revise spec"

### 3. Documentation Updates

The following files need updating to reflect the swap:

- `docs/memory/fab-workflow/planning-skills.md` — `/fab-ff` and `/fab-fff` sections, escape valve table, design decisions
- `docs/memory/fab-workflow/execution-skills.md` — pipeline invocation note, review failure section
- `docs/specs/user-flow.md` — diagram labels for fab-ff and fab-fff
- `docs/specs/skills.md` — if it contains review failure behavior descriptions
- `fab/.kit/skills/fab-ff.md` — review failure behavior
- `fab/.kit/skills/fab-fff.md` — review failure behavior
- `fab/.kit/skills/_context.md` — autonomy levels table (escape valve row)

## Affected Memory

- `fab-workflow/planning-skills`: (modify) Swap review failure behavior descriptions for `/fab-ff` and `/fab-fff`, update escape valve table, update Scope Differentiation design decision
- `fab-workflow/execution-skills`: (modify) Update pipeline invocation note to reflect swapped behavior

## Impact

- **Skills**: `fab-ff.md`, `fab-fff.md` — core behavioral change in review failure handling
- **Memory**: Two memory files need the swap reflected
- **Specs**: `user-flow.md` diagrams (already partially updated by user), `skills.md` if applicable
- **Context**: `_context.md` autonomy levels table
- **No script changes**: Review behavior is agent-driven from skill prompts, not shell scripts
- **No schema changes**: `.status.yaml` already supports `review: failed` state

## Open Questions

- None — all behavioral details were agreed in conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Retry cap is 3 cycles | Explicitly agreed in conversation | S:95 R:85 A:95 D:95 |
| 2 | Certain | Escalation triggers after 2 consecutive "fix code" failures | Explicitly agreed in conversation | S:90 R:80 A:90 D:90 |
| 3 | Certain | fab-ff gets interactive rework, fab-fff gets autonomous rework | Core requirement — explicitly stated by user | S:95 R:85 A:95 D:95 |
| 4 | Confident | Escalation path: fix code -> revise tasks -> revise spec (ordered by distance from root cause) | Logical ordering; agent can skip to spec if failure analysis warrants it | S:75 R:70 A:80 D:65 |
| 5 | Confident | After retry cap, bail message includes per-cycle summary | Suggested by agent, agreed by user; format not explicitly specified | S:80 R:75 A:85 D:75 |
| 6 | Certain | Agent decision heuristics are based on review failure type analysis | Standard pattern — agent already does failure classification in review | S:90 R:85 A:90 D:90 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
