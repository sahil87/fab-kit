# Spec: DEV-1040 Code Review Loop

**Change**: 260216-gqpp-DEV-1040-code-review-loop
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/model-tiers.md`

## Non-Goals

- Changing planning stages (intake, spec, tasks) — review loop applies only to the apply-review boundary
- Adding new pipeline stages — review remains a single stage; the loop is internal to review dispatch
- Modifying checklist generation, quality criteria, or the hydrate stage
- Prescribing a specific review agent implementation — the orchestrating LLM uses whatever review agent is available in its environment

## Execution Skills: Review Sub-Agent Dispatch

### Requirement: Review via Sub-Agent

All three pipeline skills (`/fab-continue`, `/fab-ff`, `/fab-fff`) SHALL dispatch review validation to a sub-agent running in a separate execution context, replacing the current inline review.

The sub-agent provides a fresh perspective — it has no shared context with the applying agent beyond the explicitly provided artifacts.

#### Scenario: Normal review dispatch in fab-continue
- **GIVEN** apply stage is complete (all tasks `[x]`)
- **WHEN** `/fab-continue` dispatches to review behavior
- **THEN** a review sub-agent is spawned in a separate execution context
- **AND** the sub-agent receives spec.md, tasks.md, checklist.md, and relevant source files as context
- **AND** the sub-agent performs all validation checks (tasks complete, checklist items, tests, spec match, memory drift, code quality)
- **AND** the sub-agent returns structured review output

#### Scenario: Review dispatch in fab-fff pipeline
- **GIVEN** fab-fff has completed apply (Step 6)
- **WHEN** review runs (Step 7)
- **THEN** review validation is performed by a sub-agent, not inline
- **AND** pass/fail determination uses the sub-agent's structured output

#### Scenario: Review dispatch in fab-ff pipeline
- **GIVEN** fab-ff has completed apply (Step 4)
- **WHEN** review runs (Step 5)
- **THEN** review validation is performed by a sub-agent, not inline

### Requirement: Sub-Agent is Non-Prescriptive

The orchestrating LLM MAY use any review agent available in its environment (e.g., a `code-review` skill, a general-purpose sub-agent with review instructions, or any equivalent). The skill files SHALL NOT hardcode a specific agent name or tool.
<!-- clarified: Review agent selection is not prescriptive — LLM uses whatever review agent is available -->

#### Scenario: Agent selection flexibility
- **GIVEN** the orchestrating LLM has a `code-review` skill available
- **WHEN** review dispatch occurs
- **THEN** the LLM MAY use the `code-review` skill as the review sub-agent

#### Scenario: Fallback agent selection
- **GIVEN** the orchestrating LLM has no specialized review skill
- **WHEN** review dispatch occurs
- **THEN** the LLM MAY use a general-purpose sub-agent with review-specific instructions

## Execution Skills: Review Output and Prioritization

### Requirement: Structured Review Output

The review sub-agent SHALL produce structured findings with priority classifications. Each finding MUST include a severity tier, a description, and a file:line reference where applicable.

#### Scenario: Review produces prioritized findings
- **GIVEN** a review sub-agent completes validation
- **WHEN** the sub-agent identifies issues
- **THEN** each issue is classified into one of three priority tiers: must-fix, should-fix, or nice-to-have
- **AND** each finding includes a description and file:line reference where applicable

### Requirement: Three-Tier Priority Scheme

Review findings SHALL use a three-tier priority scheme:
<!-- clarified: Three-tier priority scheme (must-fix/should-fix/nice-to-have) confirmed by user -->

- **Must-fix**: Spec mismatches, failing tests, checklist violations — always addressed during rework
- **Should-fix**: Code quality issues, pattern inconsistencies — addressed when clear and low-effort
- **Nice-to-have**: Style suggestions, minor improvements — may be skipped

#### Scenario: Must-fix findings always trigger rework
- **GIVEN** a review sub-agent returns findings containing must-fix items
- **WHEN** the applying agent triages the findings
- **THEN** all must-fix items are addressed in the rework cycle

#### Scenario: Nice-to-have findings may be skipped
- **GIVEN** a review sub-agent returns only nice-to-have findings (no must-fix or should-fix)
- **WHEN** the applying agent triages the findings
- **THEN** the review MAY be considered a pass if no must-fix or should-fix items remain

### Requirement: Comment Triage by Applying Agent

The applying agent SHALL triage review comments by priority when executing rework. Not all review comments need to be implemented. The applying agent uses judgment to determine which comments warrant rework and which can be acknowledged but deferred.

#### Scenario: Mixed-priority findings
- **GIVEN** a review returns 2 must-fix, 3 should-fix, and 5 nice-to-have findings
- **WHEN** the applying agent triages for rework
- **THEN** the 2 must-fix items are always addressed
- **AND** should-fix items are addressed when clear and low-effort
- **AND** nice-to-have items may be acknowledged but deferred

## Execution Skills: Apply-Review Loop

### Requirement: fab-continue Manual Rework (Unchanged Flow)

`/fab-continue` SHALL use the review sub-agent for validation but SHALL preserve the existing manual rework flow. On review failure, the user is presented with the same three rework options (fix code, revise tasks, revise spec).
<!-- clarified: fab-continue uses sub-agent but keeps manual rework -->

#### Scenario: fab-continue review failure
- **GIVEN** `/fab-continue` dispatched review to a sub-agent
- **WHEN** the sub-agent returns a failure verdict with prioritized findings
- **THEN** the findings are presented to the user with priority annotations
- **AND** the user is prompted to choose a rework option (fix code, revise tasks, revise spec)
- **AND** no automatic re-review loop occurs

### Requirement: fab-fff Auto-Loop with Bounded Retry

`/fab-fff` SHALL automatically loop between apply and review on failure. The applying agent receives the sub-agent's prioritized findings, triages them, fixes identified issues, and review is re-dispatched to a fresh sub-agent instance. The existing 3-cycle retry cap and escalation rules are preserved.
<!-- clarified: Auto-loop applies to fab-fff — per user direction -->

#### Scenario: fab-fff auto-rework on review failure
- **GIVEN** `/fab-fff` review sub-agent returns a failure verdict
- **WHEN** the orchestrating agent processes the failure
- **THEN** `.status.yaml` is set to `review: failed`, `apply: active`
- **AND** the agent autonomously selects a rework path based on the findings
- **AND** the agent executes the rework (fix code, revise tasks, or revise spec)
- **AND** a new review sub-agent is spawned for re-review

#### Scenario: fab-fff retry cap exhaustion
- **GIVEN** `/fab-fff` has completed 3 rework cycles without passing review
- **WHEN** the 3rd review fails
- **THEN** the pipeline BAILS with a cycle summary
- **AND** the user is directed to `/fab-continue` for manual rework

#### Scenario: fab-fff escalation after consecutive fix-code
- **GIVEN** the agent has chosen "fix code" for 2 consecutive rework cycles
- **WHEN** the 2nd fix-code review fails
- **THEN** the agent MUST escalate to "revise tasks" or "revise spec"
- **AND** the agent SHALL NOT choose "fix code" a third time in a row

### Requirement: fab-ff Auto-Loop with Interactive Fallback

`/fab-ff` SHALL automatically loop between apply and review on failure, using the same mechanism as `/fab-fff` (sub-agent review, prioritized findings, comment triage). The loop is bounded to 3 cycles. When the retry cap is exhausted, `/fab-ff` SHALL fall back to interactive rework options (same 3 options presented to the user), preserving its semi-interactive character.
<!-- clarified: fab-ff auto-loops first, then interactive fallback on cap exhaustion -->

#### Scenario: fab-ff auto-rework on first failure
- **GIVEN** `/fab-ff` review sub-agent returns a failure verdict
- **WHEN** the retry count is below 3
- **THEN** the agent autonomously selects a rework path and executes it
- **AND** a new review sub-agent is spawned for re-review

#### Scenario: fab-ff fallback to interactive after retry cap
- **GIVEN** `/fab-ff` has exhausted 3 auto-rework cycles
- **WHEN** the 3rd review fails
- **THEN** the pipeline presents interactive rework options to the user (fix code, revise tasks, revise spec)
- **AND** the user chooses where to loop back
- **AND** if the user-directed rework also fails, interactive options are re-presented (no further retry cap — user is in the loop)

#### Scenario: fab-ff escalation rule applies during auto-loop
- **GIVEN** `/fab-ff` agent has chosen "fix code" for 2 consecutive auto-rework cycles
- **WHEN** the 2nd fix-code review fails
- **THEN** the agent MUST escalate to "revise tasks" or "revise spec" (same escalation rule as fab-fff)

### Requirement: Retry Cap Alignment

The apply-review loop retry cap SHALL be 3 cycles for both `/fab-ff` and `/fab-fff`, aligned with the existing `/fab-fff` autonomous rework cap. Each cycle consists of one rework action plus one re-review by a fresh sub-agent.
<!-- assumed: Align with fab-fff's existing 3-cycle retry cap — consistent with established pattern -->

#### Scenario: Retry count increments per cycle
- **GIVEN** a review sub-agent returns a failure verdict
- **WHEN** the applying agent executes a rework and re-review occurs
- **THEN** the retry counter increments by 1
- **AND** the counter is checked before each new auto-rework cycle

## Model Tiers: Review Sub-Agent Classification

### Requirement: Review Sub-Agent Uses Capable Tier

The review sub-agent SHALL be classified as `capable` tier. Review requires deep reasoning, code analysis, spec comparison, and checklist validation — all criteria that mandate the capable tier per the existing Tier Selection Criteria.
<!-- assumed: Review sub-agent uses capable model tier — review requires deep reasoning per model-tiers criteria -->

#### Scenario: Sub-agent tier classification
- **GIVEN** a review sub-agent is spawned
- **WHEN** the orchestrating LLM selects the model for the sub-agent
- **THEN** the capable tier is used (not fast tier)

## Documentation: README and Specs Updates

### Requirement: Update README Review Description

`README.md` SHALL be updated to reflect sub-agent review and the apply-review loop. Specifically:

- The stage table review row (line 44) SHOULD mention sub-agent validation
- The review description paragraph (line 49) SHALL describe the sub-agent dispatch and prioritized comments
- The "Code Quality as a Guardrail" section SHALL update the rework table and description to mention sub-agent review, prioritized findings, and the auto-loop behavior of `/fab-fff` and `/fab-ff`

#### Scenario: README reflects sub-agent review
- **GIVEN** the README's "Code Quality as a Guardrail" section describes review rework
- **WHEN** the documentation is updated
- **THEN** the section describes review via a fresh-context sub-agent
- **AND** the rework table notes that findings are prioritized (must-fix / should-fix / nice-to-have)
- **AND** `/fab-fff` auto-rework description mentions sub-agent review with bounded retry
- **AND** `/fab-ff` description mentions auto-loop with interactive fallback

### Requirement: Update Specs Skills Reference

`docs/specs/skills.md` SHALL be updated to reflect the new review behavior across all three pipeline skills:

- **Review Behavior (via `/fab-continue`)** section SHALL describe sub-agent dispatch, structured output with priority tiers, and manual rework flow with priority annotations
- **`/fab-fff`** section SHALL describe sub-agent-based review in the autonomous rework loop
- **`/fab-ff`** section SHALL describe the behavioral change from fully interactive to auto-loop with interactive fallback; update the description line and Step 5 review behavior

#### Scenario: Skills spec reflects sub-agent review dispatch
- **GIVEN** the skills spec describes "Review Behavior (via `/fab-continue`)"
- **WHEN** the documentation is updated
- **THEN** the section describes dispatching review to a sub-agent in a separate context
- **AND** the "On failure" block includes priority annotations on findings presented to the user

#### Scenario: Skills spec reflects fab-ff behavioral change
- **GIVEN** the skills spec describes `/fab-ff` with "interactive rework on review failure"
- **WHEN** the documentation is updated
- **THEN** the description reflects "auto-loop with interactive fallback on retry cap exhaustion"
- **AND** Step 5 describes the 3-cycle auto-loop before falling back to interactive options

### Requirement: Update Specs Overview

`docs/specs/overview.md` SHALL be updated:

- Stage table review row SHALL mention sub-agent validation
- Quick Reference table SHALL update `/fab-ff` and `/fab-fff` entries to reflect updated review behavior

#### Scenario: Overview stage table updated
- **GIVEN** the overview stage table describes stage 5 (Review)
- **WHEN** the documentation is updated
- **THEN** the review row reflects sub-agent-based validation

### Requirement: Update User Flow Diagrams

`docs/specs/user-flow.md` SHALL be updated to reflect the apply-review loop:

- Diagram 2 ("The Same Flow, With Fab") SHALL show the review-to-apply loop arrow for sub-agent rework
- Diagram 3B ("Change Flow") SHALL annotate the review step with sub-agent dispatch and the auto-loop for `/fab-ff` and `/fab-fff`
- Diagram 4 ("Change State Diagram") SHALL add a review → apply loop transition with annotation for auto-rework

#### Scenario: User flow diagrams show apply-review loop
- **GIVEN** the user flow diagrams show the review stage
- **WHEN** the documentation is updated
- **THEN** diagrams include a visible apply ↔ review loop arrow
- **AND** the loop is annotated with "sub-agent review, auto-rework (fab-ff, fab-fff)" or equivalent

## Design Decisions

1. **Sub-agent over inline review**: Sub-agent provides fresh execution context, eliminating the bias of reviewing one's own implementation. The applying agent cannot influence the review's findings — only triage them afterward.
   - *Why*: The primary motivation for this change. Same-context review is fundamentally limited by shared cognitive biases.
   - *Rejected*: Multiple inline review passes (still shares context), external review tool integration (too prescriptive, not portable).

2. **Priority-based triage over fix-everything**: The applying agent triages review comments by severity rather than implementing all of them.
   - *Why*: Prevents infinite rework loops over diminishing-return suggestions. Must-fix items ensure correctness; nice-to-have items allow pragmatic completion.
   - *Rejected*: Fix all comments (leads to infinite loops on style disagreements), ignore all non-critical (misses genuine should-fix quality issues).

3. **fab-ff gains auto-loop (behavioral change)**: fab-ff previously had no retry cap and always presented interactive rework. Now it auto-loops first (up to 3 cycles) and falls back to interactive on exhaustion.
   - *Why*: Sub-agent review enables tighter automated feedback cycles. The interactive fallback preserves fab-ff's semi-interactive character — the user is never locked out of control.
   - *Rejected*: Keep fab-ff fully interactive (wastes the fresh-context benefit on simple fixes), make fab-ff fully autonomous like fab-fff (loses the semi-interactive identity).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Auto-loop applies to fab-fff and fab-ff; fab-continue keeps manual rework | User explicitly directed: loop in fab-fff and fab-ff, manual in fab-continue. Confirmed from intake #3, #8, #9 | S:95 R:45 A:95 D:95 |
| 2 | Certain | Review comments are prioritized by severity; not all must be implemented | User clarified: "prioritize the comments" and "not necessary to implement all comments". Confirmed from intake #4 | S:95 R:80 A:95 D:95 |
| 3 | Certain | Three-tier priority scheme (must-fix / should-fix / nice-to-have) | User confirmed three tiers as right granularity. Confirmed from intake #5 | S:95 R:75 A:95 D:95 |
| 4 | Certain | Sub-agent replaces existing inline review for all three skills | All skills use sub-agent; running both inline and sub-agent would be redundant. Confirmed from intake #6 | S:80 R:35 A:80 D:80 |
| 5 | Certain | Review agent is not prescriptive — LLM uses whatever review agent is available | User explicitly directed. Confirmed from intake #7 | S:95 R:80 A:95 D:95 |
| 6 | Certain | fab-continue uses sub-agent but keeps manual rework | User confirmed: fresh-context evaluation via sub-agent, rework options presented manually. Confirmed from intake #8 | S:95 R:50 A:95 D:95 |
| 7 | Certain | fab-ff auto-loops with bounded retry, falls back to interactive rework on cap exhaustion | User confirmed: preserves fab-ff's semi-interactive character. Confirmed from intake #9 | S:95 R:50 A:95 D:95 |
| 8 | Confident | Retry cap aligns with fab-fff's existing 3-cycle cap for both fab-ff and fab-fff | Established pattern in autonomous rework; highly reversible via config. Confirmed from intake #1 | S:40 R:80 A:75 D:80 |
| 9 | Confident | Review sub-agent uses capable model tier | Review requires deep reasoning and code analysis per model-tiers criteria. Confirmed from intake #2 | S:20 R:85 A:70 D:75 |
| 10 | Confident | fab-ff inherits the same escalation rule as fab-fff (escalate after 2 consecutive fix-code) | Consistent behavior across auto-loop skills; no signal to differentiate | S:30 R:80 A:70 D:75 |
| 11 | Confident | Review pass/fail is determined by presence of must-fix items (not should-fix or nice-to-have alone) | Follows from three-tier semantics — must-fix = blocking, others = advisory. Highly reversible | S:50 R:85 A:70 D:65 |
| 12 | Certain | Documentation updates (README, specs) are in scope for this change | User explicitly requested adding README.md and docs/specs/** updates | S:95 R:90 A:95 D:95 |
| 13 | Confident | User-flow diagrams updated in-place rather than creating new diagram files | Existing diagrams already show review flow; extending them is cleaner than adding new files | S:40 R:90 A:80 D:80 |

13 assumptions (8 certain, 5 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
