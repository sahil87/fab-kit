# Spec: Add Operator3 — Auto-Nudge for Blocked Agents

**Change**: 260312-ngew-add-operator3-auto-nudge
**Created**: 2026-03-12
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Lifecycle enforcement — operator3 does not advance stages or make pipeline decisions
- Configurable question patterns — v1 uses hardcoded heuristics in the skill file
- Audit trail for auto-answers — deferred to a future iteration
- Skip-on-failure behavior — if an auto-answer leads to a problem, the agent handles it; operator3 does not skip

## Operator Skill: Question Detection

### Requirement: Terminal Heuristic for Question Detection

Operator3 SHALL detect when a monitored agent is waiting for user input by scanning the agent's terminal output via `tmux capture-pane`. For each agent that is **idle** (per the pane-map Agent column), the monitoring tick SHALL capture the last ~10 lines of the agent's terminal and scan for question indicators.

The following patterns SHALL be recognized as question indicators:
- Lines ending with `?`
- `[Y/n]`, `[y/N]`, `(y/n)`, `(yes/no)` patterns
- `Allow?`, `Approve?`, `Confirm?`, `Proceed?` keywords
- Claude Code permission prompts (tool approval patterns)
- `Do you want to...`, `Should I...`, `Would you like...` phrasing

The heuristic SHALL be intentionally broad — false positives are handled by the answer confidence model (escalate when unsure), not by tightening detection. When multiple question indicators appear in the captured output, the operator SHALL evaluate the most recent (bottom-most) indicator, consistent with standard terminal interaction patterns.
<!-- clarified: multiple questions in capture — operator addresses the most recent (bottom-most) question indicator -->

#### Scenario: Agent idle with visible question
- **GIVEN** an agent is idle in the pane map
- **WHEN** the monitoring tick captures the last 10 lines of the agent's terminal
- **AND** a question indicator pattern is found in the captured output
- **THEN** the operator proceeds to the answer confidence assessment

#### Scenario: Agent idle with no question
- **GIVEN** an agent is idle in the pane map
- **WHEN** the monitoring tick captures the last 10 lines of the agent's terminal
- **AND** no question indicator pattern is found
- **THEN** the operator treats this as normal idle behavior (existing operator2 stuck detection applies)

#### Scenario: Agent is active (not idle)
- **GIVEN** an agent is active (not idle) in the pane map
- **WHEN** the monitoring tick processes this agent
- **THEN** the terminal heuristic is NOT run (question detection only applies to idle agents)

## Operator Skill: Answer Confidence Model

### Requirement: Two-Tier Classification

When a question is detected, operator3 SHALL classify it into one of two tiers:

**Auto-answer** tier — one obvious right answer given the pipeline context:
- Permission/approval prompts → approve
- "Should I commit before rebasing?" → yes
- "Proceed with the changes?" → yes
- "Continue?" → yes
- Simple operational confirmations where saying "no" would stall the pipeline

**Escalate** tier — multiple valid paths, judgment required:
- Questions involving tradeoffs or architecture decisions
- Questions involving destructive or irreversible actions ("Should I delete these files?")
- Questions offering multiple approaches ("Which approach: A or B?")
- Questions about failing tests ("Tests are failing, should I fix or skip?")

The classification heuristic: if the question is a binary yes/no where "yes" continues the work and "no" stalls it, and the question does NOT involve destructive or branching choices → auto-answer. Everything else → escalate.

#### Scenario: Auto-answerable permission prompt
- **GIVEN** an idle agent's terminal shows "Allow this tool call? [Y/n]"
- **WHEN** the answer confidence model evaluates the question
- **THEN** the question is classified as auto-answer tier
- **AND** the operator sends the appropriate affirmative response via `tmux send-keys`
- **AND** reports "{change}: auto-answered 'Allow this tool call?' → y"

#### Scenario: Auto-answerable continuation prompt
- **GIVEN** an idle agent's terminal shows "Proceed with the changes? (y/n)"
- **WHEN** the answer confidence model evaluates the question
- **THEN** the question is classified as auto-answer tier
- **AND** the operator sends "y" via `tmux send-keys`
- **AND** reports "{change}: auto-answered 'Proceed with the changes?' → y"

#### Scenario: Escalation — destructive action
- **GIVEN** an idle agent's terminal shows "Should I delete these 5 files? [Y/n]"
- **WHEN** the answer confidence model evaluates the question
- **THEN** the question is classified as escalate tier
- **AND** the operator reports "{change}: waiting for input — 'Should I delete these 5 files?'. Please respond."
- **AND** the operator does NOT send any answer

#### Scenario: Escalation — branching decision
- **GIVEN** an idle agent's terminal shows "Tests are failing. Should I fix them or skip and continue?"
- **WHEN** the answer confidence model evaluates the question
- **THEN** the question is classified as escalate tier
- **AND** the operator reports "{change}: waiting for input — 'Tests are failing. Should I fix them or skip?'. Please respond."

### Requirement: No Nudge Budget

Operator3 SHALL NOT impose a cooldown or retry limit on auto-answers. Each question SHALL be evaluated independently on its own merits by the confidence model. The PR review step (human and/or Copilot review before merge) serves as the safety net against compounding bad auto-answers.

#### Scenario: Multiple sequential auto-answers
- **GIVEN** an agent receives an auto-answer for a question
- **AND** the agent becomes idle again with a new question
- **WHEN** the next monitoring tick detects the new question
- **THEN** the new question is evaluated independently (no cooldown)
- **AND** may be auto-answered if it meets auto-answer criteria

## Operator Skill: Updated Monitoring Tick

### Requirement: Input-Waiting Detection Step

The monitoring tick SHALL include a new detection step for input-waiting agents, positioned between the existing detection steps. The full tick order SHALL be:

1. Stage advance detection *(existing)*
2. Pipeline completion detection *(existing)*
3. Review failure detection *(existing)*
4. Pane death detection *(existing)*
5. **Input-waiting detection** *(new)* — for idle agents, run the terminal heuristic
6. Stuck detection *(existing, modified)* — only for agents NOT detected as input-waiting

Input-waiting detection MUST run **before** stuck detection. An agent waiting for input is not "stuck" — it's blocked on user input. Stuck detection SHALL only flag agents that are idle without a visible question.

#### Scenario: Idle agent with question — not flagged as stuck
- **GIVEN** an agent has been idle for 20 minutes (above the 15m stuck threshold)
- **AND** the agent's terminal shows a question prompt
- **WHEN** the monitoring tick processes this agent
- **THEN** the agent is processed by input-waiting detection (step 5)
- **AND** the agent is NOT flagged as stuck (step 6 skipped for this agent)

#### Scenario: Idle agent without question — stuck detection applies
- **GIVEN** an agent has been idle for 20 minutes (above the 15m stuck threshold)
- **AND** the agent's terminal shows no question prompt
- **WHEN** the monitoring tick processes this agent
- **THEN** input-waiting detection finds no question (step 5 no-op)
- **AND** stuck detection flags the agent as potentially stuck (step 6)

## Operator Skill: Inheritance Model

### Requirement: Operator3 Extends Operator2

Operator3 SHALL inherit all of operator2's behavior: monitoring, enrollment, `/loop` ticks, all use cases (UC1–UC8), confirmation model, pre-send validation, context discipline, and configuration. The new behavior (question detection + answer confidence + input-waiting detection step) is purely additive — no existing operator2 behavior is modified or removed.

#### Scenario: Standard operator2 behavior preserved
- **GIVEN** operator3 is running
- **WHEN** the user requests any operator2 operation (broadcast, sequenced rebase, merge, spawn, status, unstick, notification, autopilot)
- **THEN** operator3 handles it identically to operator2

### Requirement: Skill File Structure

The skill file `fab/.kit/skills/fab-operator3.md` SHALL reference operator2's skill file for inherited behavior and define only the new behavior (question detection, answer confidence, monitoring tick modification). It SHALL use the same frontmatter pattern as operator1 and operator2.

#### Scenario: Skill file references operator2
- **GIVEN** the operator3 skill file exists at `fab/.kit/skills/fab-operator3.md`
- **WHEN** an agent reads the skill file
- **THEN** it finds a directive to read `fab/.kit/skills/fab-operator2.md` for inherited behavior
- **AND** the new behavior sections are defined after the inheritance directive

### Requirement: Launcher Script

A launcher script `fab/.kit/scripts/fab-operator3.sh` SHALL be created, mirroring `fab-operator2.sh`. It SHALL use the same singleton tab name `operator` and launch `/fab-operator3` instead of `/fab-operator2`.

#### Scenario: Launching operator3
- **GIVEN** the user is in a tmux session
- **WHEN** the user runs `fab/.kit/scripts/fab-operator3.sh`
- **THEN** a singleton tmux tab named `operator` is created (or switched to if it already exists)
- **AND** the tab runs `claude --dangerously-skip-permissions '/fab-operator3'`

### Requirement: Per-Skill Spec

A new spec file `docs/specs/skills/SPEC-fab-operator3.md` SHALL be created documenting operator3's behavior, primitives, question detection, answer confidence model, and relationship to operator2.

#### Scenario: Spec file exists
- **GIVEN** the operator3 spec exists at `docs/specs/skills/SPEC-fab-operator3.md`
- **WHEN** a developer looks up operator3's design
- **THEN** it documents question detection, answer confidence, monitoring tick changes, and inheritance from operator2

## Design Decisions

1. **Terminal heuristic over time-based detection**: Detection uses `tmux capture-pane` content analysis rather than idle-time thresholds.
   - *Why*: Terminal content directly reveals whether an agent is waiting for input; time-based detection cannot distinguish "waiting for input" from "slow but working"
   - *Rejected*: Time-gated heuristic (idle > N minutes → assume stuck), autopilot-only detection

2. **Two-tier model over multi-tier**: A binary classification (auto-answer vs escalate) is sufficient for v1.
   - *Why*: Simpler mental model, easier to reason about safety; the critical distinction is "one obvious answer" vs "judgment needed"
   - *Rejected*: Multi-tier with "ask first then auto-answer" middle tier

3. **No audit trail for v1**: Auto-answer actions are reported inline in the operator's output but not logged to a persistent file.
   - *Why*: Overkill for v1; PR review catches compounding errors
   - *Rejected*: JSON audit log, `.history.jsonl` entries per auto-answer

4. **Hardcoded question patterns**: Detection patterns are embedded in the skill file, not configurable via config.yaml.
   - *Why*: Simpler for v1; patterns are stable across projects
   - *Rejected*: Configurable pattern list in config.yaml

5. **No cooldown between auto-answers**: Each question is evaluated independently; PR review is the safety net.
   - *Why*: Cooldowns reduce utility without meaningfully improving safety when each question is independently assessed
   - *Rejected*: 2-minute cooldown between consecutive auto-answers

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Operator3 extends operator2 (not replaces) | Confirmed from intake #1 — user confirmed incremental evolution model | S:95 R:90 A:95 D:95 |
| 2 | Certain | "Stuck" means waiting for user input, not slow | Confirmed from intake #2 — user explicitly defined scope | S:95 R:95 A:95 D:95 |
| 3 | Certain | Detection via terminal heuristic (tmux capture-pane) | Confirmed from intake #3 — user chose option 1 over alternatives | S:90 R:80 A:85 D:90 |
| 4 | Certain | Two-tier confidence model: auto-answer vs escalate | Confirmed from intake #4 — user agreed on approach | S:90 R:85 A:85 D:90 |
| 5 | Certain | No audit trail for v1 | Confirmed from intake #5 — user agreed overkill for v1 | S:95 R:95 A:90 D:95 |
| 6 | Certain | "Not a Lifecycle Enforcer" still holds | Confirmed from intake #6 — user confirmed "not enforcing, just nudging" | S:90 R:90 A:90 D:90 |
| 7 | Confident | Nudge is answering the actual question (not /fab-continue) | Confirmed from intake #7 — user clarified with commit example | S:85 R:80 A:80 D:75 |
| 8 | Confident | No skip-on-failure behavior | Confirmed from intake #8 — user said "No, don't skip" | S:85 R:85 A:80 D:80 |
| 9 | Certain | No cooldown between auto-answers | Confirmed from intake #9 — PR review is the safety net | S:90 R:90 A:85 D:90 |
| 10 | Certain | Terminal heuristic patterns are hardcoded in skill file | Confirmed from intake #10 — user confirmed for v1 | S:90 R:85 A:75 D:90 |
| 11 | Certain | Auto-answer sends via tmux send-keys (same mechanism as operator2 interaction) | Codebase pattern — operator2 already sends commands via resolve --pane + tmux send-keys | S:90 R:90 A:95 D:95 |
| 12 | Confident | Reporting format matches operator2 monitoring tick output style | Codebase pattern — "{change}: {status}" style consistent with operator2 | S:80 R:90 A:80 D:80 |

12 assumptions (9 certain, 3 confident, 0 tentative, 0 unresolved).
