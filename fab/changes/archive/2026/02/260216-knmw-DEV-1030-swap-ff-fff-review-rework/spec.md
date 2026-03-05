# Spec: Swap fab-ff and fab-fff Review Failure Behavior

**Change**: 260216-knmw-DEV-1030-swap-ff-fff-review-rework
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing the confidence gate logic on `/fab-ff` — the gate stays as-is
- Modifying `/fab-continue` review behavior — it remains the canonical rework flow with interactive options
- Adding retry logic to `/fab-continue` — autonomous retry is exclusive to `/fab-fff`
- Changing any shell scripts — review behavior is agent-driven from skill prompts

## Skills: `/fab-ff` Review Failure — Interactive Rework

### Requirement: Interactive Rework Menu on Review Failure

On review failure, `/fab-ff` SHALL present the user with the same three rework options defined in `/fab-continue` review behavior:

1. **Fix code** — unchecks affected tasks in `tasks.md` with `<!-- rework: reason -->`, re-runs apply and review
2. **Revise tasks** — user edits `tasks.md`, re-runs apply and review
3. **Revise spec** — resets to spec stage via stage reset, regenerates `spec.md`, invalidates downstream

After the user chooses, the pipeline SHALL re-run from the chosen re-entry point and review SHALL execute again. If review fails again, the options SHALL be presented again (no retry cap — the user is in the loop).

`/fab-ff` SHALL call `lib/stageman.sh log-review <change_dir> "failed" "<rework-option>"` after the user selects a rework option.

#### Scenario: Review fails, user chooses "Fix code"

- **GIVEN** `/fab-ff` has completed apply and entered review
- **WHEN** review fails with code quality or test failures
- **AND** the user selects "Fix code"
- **THEN** the agent unchecks affected tasks in `tasks.md` with `<!-- rework: reason -->` annotations
- **AND** apply behavior re-executes for the unchecked tasks
- **AND** review runs again

#### Scenario: Review fails, user chooses "Revise tasks"

- **GIVEN** `/fab-ff` review has failed
- **WHEN** the user selects "Revise tasks"
- **THEN** the user edits `tasks.md` (add/modify tasks)
- **AND** apply behavior re-executes for unchecked tasks
- **AND** review runs again

#### Scenario: Review fails, user chooses "Revise spec"

- **GIVEN** `/fab-ff` review has failed
- **WHEN** the user selects "Revise spec"
- **THEN** the pipeline resets to spec stage (sets `spec: active`, all stages after spec → `pending`)
- **AND** `spec.md` is regenerated
- **AND** downstream artifacts (tasks, checklist) are invalidated and regenerated
- **AND** apply and review execute on the new artifacts

#### Scenario: Review fails twice in a row

- **GIVEN** `/fab-ff` review has failed and the user chose "Fix code"
- **WHEN** the re-review also fails
- **THEN** the interactive rework menu is presented again
- **AND** the user can choose any of the three options again (no retry cap)

### Requirement: Remove Bail-on-Failure Behavior

`/fab-ff` SHALL NOT bail immediately on review failure. The current behavior (`"Review failed. Run /fab-continue for rework options."`) SHALL be replaced by the interactive rework menu described above.

#### Scenario: No more bail on review failure

- **GIVEN** `/fab-ff` is running and review fails
- **WHEN** the review verdict is "fail"
- **THEN** the skill does NOT stop with a bail message
- **AND** instead presents the three rework options to the user

## Skills: `/fab-fff` Review Failure — Autonomous Rework

### Requirement: Autonomous Rework Path Selection

On review failure, `/fab-fff` SHALL autonomously choose the rework path based on the review failure details. The agent SHALL NOT present an interactive menu or ask the user which option to take.

The agent SHALL use the following heuristics to select the rework path:

- **Test failures, code quality issues, pattern violations** → "Fix code"
- **Missing functionality, incomplete coverage, wrong task breakdown** → "Revise tasks"
- **Spec drift, requirements mismatch, fundamental approach issues** → "Revise spec"

#### Scenario: Autonomous fix code selection

- **GIVEN** `/fab-fff` review has failed
- **WHEN** the failure details indicate test failures or code quality issues
- **THEN** the agent selects "Fix code" without user input
- **AND** unchecks affected tasks in `tasks.md` with `<!-- rework: reason -->`
- **AND** re-runs apply and review

#### Scenario: Autonomous revise tasks selection

- **GIVEN** `/fab-fff` review has failed
- **WHEN** the failure details indicate missing functionality or wrong task breakdown
- **THEN** the agent selects "Revise tasks" without user input
- **AND** adds or modifies tasks in `tasks.md`
- **AND** re-runs apply and review

#### Scenario: Autonomous revise spec selection

- **GIVEN** `/fab-fff` review has failed
- **WHEN** the failure details indicate spec drift or requirements mismatch
- **THEN** the agent selects "Revise spec" without user input
- **AND** resets to spec stage and regenerates downstream artifacts
- **AND** re-runs apply and review

### Requirement: Retry Cap of 3 Cycles

`/fab-fff` autonomous rework SHALL be bounded to a maximum of **3 rework cycles**. Each cycle consists of one rework action followed by one re-review. After 3 failed cycles, the pipeline SHALL bail with a per-cycle summary.

The bail message format SHALL be:

```
Review failed after 3 rework attempts. Summary:
  Cycle 1: {action} — {what was done}
  Cycle 2: {action} — {what was done}
  Cycle 3: {action} — {what was done}
Run /fab-continue for manual rework options.
```

#### Scenario: Pipeline bails after 3 failed rework cycles

- **GIVEN** `/fab-fff` review has failed
- **AND** the agent has completed 3 rework cycles, each failing review
- **WHEN** the 3rd re-review fails
- **THEN** the pipeline stops with a bail message containing per-cycle summaries
- **AND** the bail message suggests `/fab-continue` for manual rework

#### Scenario: Rework succeeds within 3 cycles

- **GIVEN** `/fab-fff` review has failed
- **AND** the agent has completed 1 rework cycle (fix code)
- **WHEN** the re-review passes
- **THEN** the pipeline continues to hydrate without further rework
- **AND** `lib/stageman.sh log-review <change_dir> "passed"` is called

### Requirement: Escalation After 2 Consecutive Fix-Code Failures

If the agent chooses "Fix code" and the subsequent review fails again on the same or similar issues, the agent MUST escalate to "Revise tasks" or "Revise spec" (agent decides which based on failure analysis). This escalation is a **hard rule** — even if the agent believes another code fix would work, it MUST escalate after 2 consecutive "fix code" attempts.

The escalation resets the consecutive fix-code counter. A "Revise tasks" or "Revise spec" action followed by a "Fix code" is NOT consecutive — only sequential fix-code actions trigger escalation.

#### Scenario: Forced escalation after 2 consecutive fix-code attempts

- **GIVEN** `/fab-fff` review has failed
- **AND** the agent chose "Fix code" in cycle 1
- **AND** the re-review failed again on similar issues
- **AND** the agent chose "Fix code" in cycle 2
- **WHEN** the cycle 2 re-review also fails
- **THEN** the agent MUST escalate to "Revise tasks" or "Revise spec" in cycle 3
- **AND** the agent SHALL NOT choose "Fix code" in cycle 3

#### Scenario: Non-consecutive fix-code does not trigger escalation

- **GIVEN** the agent chose "Fix code" in cycle 1
- **AND** the agent chose "Revise tasks" in cycle 2
- **WHEN** cycle 2 re-review fails
- **THEN** the agent MAY choose "Fix code" in cycle 3 (the "Revise tasks" reset the consecutive counter)

### Requirement: Logging for Autonomous Rework

`/fab-fff` SHALL call `lib/stageman.sh log-review <change_dir> "failed" "<rework-option>"` for each autonomous rework cycle, recording the action the agent selected. On final bail, the review log SHALL also be written.

#### Scenario: Review log records autonomous choice

- **GIVEN** `/fab-fff` review has failed
- **WHEN** the agent autonomously selects "Fix code"
- **THEN** `lib/stageman.sh log-review <change_dir> "failed" "fix-code"` is called before rework begins

## Documentation: Skill Files

### Requirement: Update `/fab-ff` Skill File

The skill file `fab/.kit/skills/fab-ff.md` SHALL be updated to:

1. Replace the "bail on review failure" behavior in Step 5 (Review) with the interactive rework menu
2. Update the Error Handling table: the "Review fails" row SHALL say "Present interactive rework menu" instead of the current bail message
3. Update the Purpose section to reference "interactive rework on review failure" instead of "bail on review failure"
4. Update the skill description frontmatter from "bail on review failure" to "interactive rework on review failure"

#### Scenario: fab-ff.md reflects interactive rework

- **GIVEN** the skill file `fab/.kit/skills/fab-ff.md` has been updated
- **WHEN** an agent reads Step 5 (Review)
- **THEN** the Fail block describes presenting 3 rework options to the user
- **AND** there is no reference to bailing on review failure

### Requirement: Update `/fab-fff` Skill File

The skill file `fab/.kit/skills/fab-fff.md` SHALL be updated to:

1. Replace the interactive rework menu in Step 7 (Review) with autonomous rework behavior including retry cap and escalation logic
2. Update the Error Handling table: the "Review fails" row SHALL describe autonomous rework with retry cap
3. Update the Purpose section to reference "autonomous rework with bounded retry" instead of "interactive rework"
4. Update the skill description frontmatter from "interactive rework" to "autonomous rework with bounded retry"

#### Scenario: fab-fff.md reflects autonomous rework

- **GIVEN** the skill file `fab/.kit/skills/fab-fff.md` has been updated
- **WHEN** an agent reads Step 7 (Review)
- **THEN** the Fail block describes autonomous path selection, retry cap, and escalation logic
- **AND** there is no reference to presenting an interactive rework menu

## Documentation: Context File

### Requirement: Update Autonomy Levels Table

The Skill-Specific Autonomy Levels table in `fab/.kit/skills/_context.md` SHALL be updated:

- **`/fab-fff` Escape valve** row: change from `/fab-clarify` (interactive rework on review failure) to `/fab-continue` (autonomous rework with retry cap, bail after 3 cycles)
- **`/fab-ff` Escape valve** row: change from `/fab-continue` (bail on review failure) to `/fab-clarify` (interactive rework on review failure)

#### Scenario: Autonomy table reflects swapped behavior

- **GIVEN** `_context.md` has been updated
- **WHEN** an agent reads the Skill-Specific Autonomy Levels table
- **THEN** `/fab-fff` escape valve says autonomous rework / bail after retry cap
- **AND** `/fab-ff` escape valve says interactive rework on review failure

## Documentation: Memory Files

### Requirement: Update Planning Skills Memory

`docs/memory/fab-workflow/planning-skills.md` SHALL be updated to:

1. Swap the review failure behavior descriptions in the `/fab-ff` and `/fab-fff` requirement sections
2. Update the "Scope Differentiation" design decision to reflect the swap (fab-ff gets interactive rework, fab-fff gets autonomous rework)
3. Record this change in the Changelog

#### Scenario: Planning skills memory reflects swap

- **GIVEN** `planning-skills.md` has been updated
- **WHEN** a reader checks the `/fab-ff` section
- **THEN** it describes interactive rework on review failure
- **AND** the `/fab-fff` section describes autonomous rework with retry cap and escalation

### Requirement: Update Execution Skills Memory

`docs/memory/fab-workflow/execution-skills.md` SHALL be updated to:

1. Update the "Pipeline invocation" note in the Overview section to reflect the swapped behavior (fab-fff → autonomous rework, fab-ff → interactive rework)
2. Record this change in the Changelog

#### Scenario: Execution skills memory reflects swap

- **GIVEN** `execution-skills.md` has been updated
- **WHEN** a reader checks the Pipeline invocation note
- **THEN** it states `/fab-ff` presents interactive rework options on review failure
- **AND** `/fab-fff` uses autonomous rework with bounded retry

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Retry cap is 3 cycles | Confirmed from intake #1 — explicitly agreed in conversation | S:95 R:85 A:95 D:95 |
| 2 | Certain | Escalation triggers after 2 consecutive "fix code" failures | Confirmed from intake #2 — explicitly agreed | S:90 R:80 A:90 D:90 |
| 3 | Certain | fab-ff gets interactive rework, fab-fff gets autonomous rework | Confirmed from intake #3 — core requirement | S:95 R:85 A:95 D:95 |
| 4 | Confident | Escalation path: agent decides between "revise tasks" and "revise spec" based on failure analysis | Confirmed from intake #4 — agent has freedom to choose escalation target | S:80 R:70 A:80 D:65 |
| 5 | Confident | Bail message includes per-cycle summary with action and description | Confirmed from intake #5 — format agreed but exact wording not specified | S:80 R:75 A:85 D:75 |
| 6 | Certain | Agent decision heuristics based on review failure type | Confirmed from intake #6 — standard pattern already in use | S:90 R:85 A:90 D:90 |
| 7 | Certain | fab-ff interactive rework has no retry cap (user is in the loop) | User explicitly stated "I'm still in the loop" for fab-ff — no need for autonomous bounds | S:90 R:80 A:90 D:90 |
| 8 | Certain | No script changes needed — all behavior is agent-driven from skill prompts | Confirmed from intake — review logic is prompt-based, not shell-based | S:95 R:90 A:95 D:95 |
| 9 | Confident | docs/specs/user-flow.md and skills.md updates deferred to apply stage | Intake lists these as impacted, but they are spec files (human-curated per constitution) — checking and updating as needed during apply | S:70 R:80 A:70 D:65 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
