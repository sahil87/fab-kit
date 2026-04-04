# Spec: Operator Never-Ask Monitor Fix

**Change**: 260331-mvhj-operator-never-ask-monitor
**Created**: 2026-03-31
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Restructuring the operator spawn sequence or changing enrollment mechanics
- Modifying `fab-operator5.md` — it predates the spawn sequence pattern and has no "Spawning an Agent" subsection

## Operator Skills: Never-Ask Monitor Reinforcement

### Requirement: Spawn Sequence Auto-Enroll Prohibition (operator7)

The "Spawning an Agent" subsection in `fab-operator7.md` §6 (lines 297–306) SHALL include an explicit MUST NOT prohibition against asking the user whether to monitor a spawned agent. The prohibition SHALL appear as:

1. A blockquote admonition immediately after the spawn sequence steps, using RFC 2119 language: every spawned agent MUST be enrolled in the monitored set unconditionally — the operator MUST NOT prompt the user about monitoring.
2. Step 4 ("Enroll in monitored set") SHALL be annotated to reinforce that enrollment is unconditional and silent.

This reinforces the principle already stated in §1 ("Never ask whether to monitor a spawned agent — if the operator spawned it, monitor it") at the point where the LLM actually executes the spawn flow.

#### Scenario: Operator spawns agent for existing change
- **GIVEN** the operator is spawning an agent for an existing change via the spawn sequence
- **WHEN** step 4 (enroll in monitored set) is reached
- **THEN** the agent is enrolled in the monitored set without any user prompt
- **AND** no question like "Want me to monitor it?" is generated

#### Scenario: Operator spawns agent for new backlog item
- **GIVEN** the operator is spawning an agent for a new backlog item
- **WHEN** the spawn sequence completes and enrollment occurs
- **THEN** the agent is enrolled in the monitored set without any user prompt

### Requirement: Spawn Section Auto-Enroll Prohibition (operator6)

The "Spawning an Agent" subsection in `fab-operator6.md` (lines 253–259) SHALL include the same MUST NOT prohibition pattern. Additionally, the "Automate the routine" principle in §1 (line 20) SHALL be updated to include the "Never ask whether to monitor a spawned agent" sentence, matching operator7's wording.

#### Scenario: Operator6 spawns agent
- **GIVEN** `fab-operator6.md` is the active operator skill
- **WHEN** the operator spawns an agent via the spawn sequence
- **THEN** the agent is enrolled in the monitored set without any user prompt
- **AND** the §1 principle and §6 spawn subsection both contain the never-ask language

### Requirement: Principle-Procedure Consistency

The never-ask language SHALL appear in both the principles section (§1) and the procedural spawn section (§6) of each affected operator skill. The principle provides the "why"; the procedure provides the "what" at the execution point.

#### Scenario: LLM reads spawn sequence in isolation
- **GIVEN** an LLM is following the spawn sequence steps without re-reading §1
- **WHEN** it reaches the enrollment step
- **THEN** the local prohibition text is sufficient to prevent asking the user about monitoring

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Primary target is `fab-operator7.md` spawn subsection | Confirmed from intake #1 — user explicitly referenced §1 and §6 | S:95 R:95 A:95 D:95 |
| 2 | Certain | Fix is markdown-only — no script or Go code changes | Confirmed from intake #2 — LLM prompt adherence issue | S:90 R:95 A:95 D:95 |
| 3 | Confident | `fab-operator6.md` gets both spawn subsection and §1 principle updates | Upgraded from intake #3 — confirmed operator6 is missing the "Never ask" sentence in §1 entirely | S:80 R:90 A:85 D:75 |
| 4 | Certain | `fab-operator5.md` is excluded | Confirmed from intake #4 — no "Spawning an Agent" subsection exists; predecessor version | S:85 R:95 A:90 D:90 |
| 5 | Certain | Use RFC 2119 MUST NOT for the prohibition | Confirmed from intake #5 — consistent with constitution and skill conventions | S:90 R:95 A:90 D:90 |
| 6 | Certain | Admonition uses blockquote format after spawn steps | Codebase pattern — other subsections use blockquotes for callouts | S:85 R:95 A:90 D:90 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
