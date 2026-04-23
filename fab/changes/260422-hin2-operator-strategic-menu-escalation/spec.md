# Spec: Operator Numbered-Menu Classification + Idle-Escalation Auto-Default

**Change**: 260422-hin2-operator-strategic-menu-escalation
**Created**: 2026-04-22
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Hardcoded keyword list for classification — principle-based, LLM-judged only.
- Agent-side sentinel or marker protocol (`[STRATEGIC]` etc.) — classification operates on terminal capture.
- Configuration knob for the idle threshold (`.fab-operator.yaml`, per-change override, environment variable) — the 30-minute value is fixed.
- Extending the idle auto-default to rule-6 "cannot determine keystrokes" escalations — that failure mode stays escalated.
- Any change to the `fab` Go binary, CLI subcommands, or `.fab-operator.yaml` schema — skill-doc-only change.
- Retroactive classification of prompts answered before this change shipped — forward-only behavior.

## Operator: Numbered-Menu Classification (§5 Answer Model, rule 4)

### Requirement: Classify before answering

The operator SHALL classify every numbered-menu prompt as either **Routine** or **Strategic** before applying a default answer. The classifier MUST use LLM judgment over the terminal capture already produced by §5 Question Detection step 1, weighing at minimum these signals: option text length, semantic distinctness of options, surrounding agent context, and reversibility of the choice. The classifier MUST NOT consult a hardcoded keyword list and MUST NOT require any sentinel token from the emitting agent.

#### Scenario: Routine tool-permission prompt
- **GIVEN** a monitored agent is idle with a Claude Code tool-permission prompt `1) Yes  2) Yes, and don't ask again  3) No` visible in the terminal capture
- **WHEN** the operator's tick reaches §5 rule 4
- **THEN** the operator SHALL classify the prompt as Routine
- **AND** SHALL auto-answer `1` per rule 4
- **AND** SHALL log the action using the existing `auto-answered` log format

#### Scenario: Routine binary-framed menu
- **GIVEN** a monitored agent shows a prompt `1) Yes  2) No` with short option text and synonymous intent across options
- **WHEN** the operator classifies the prompt
- **THEN** the prompt SHALL be classified as Routine
- **AND** SHALL be auto-answered `1`

#### Scenario: Strategic rework menu
- **GIVEN** a monitored agent shows a post-review rework menu `1) fix code  2) revise tasks  3) revise spec  4) abort` — options represent materially different directions with long, semantically distinct text
- **WHEN** the operator classifies the prompt
- **THEN** the prompt SHALL be classified as Strategic
- **AND** SHALL NOT be auto-answered
- **AND** SHALL be escalated to the user via the existing §5 escalation log format

#### Scenario: Strategic pipeline-shape choice
- **GIVEN** a monitored agent shows `1) run ff  2) run fff  3) stop at spec`
- **WHEN** the operator classifies the prompt
- **THEN** the prompt SHALL be classified as Strategic and escalated

### Requirement: Escalate on classification uncertainty

When the classifier cannot confidently assign Routine or Strategic, the operator MUST treat the prompt as Strategic and escalate. False-negative strategic commits the queue to an unchosen direction; false-positive strategic costs at most a user nudge recovered by the 30-minute auto-default (see below). The asymmetric cost structure makes escalate-on-uncertainty the required behavior, not a recommendation.

#### Scenario: Ambiguous classification
- **GIVEN** a numbered prompt whose options mix short and long text, or whose surrounding context does not clearly indicate intent
- **WHEN** the operator classifies the prompt
- **THEN** the prompt SHALL be classified as Strategic
- **AND** SHALL be escalated to the user

### Requirement: Revised rule 4 text in fab-operator.md §5 Answer Model

The canonical skill source `src/kit/skills/fab-operator.md` SHALL replace the current rule 4 text with the classification-aware variant. The revised text MUST name the two classes (Routine, Strategic), list the four classifier signals (option text length, semantic distinctness, surrounding agent context, reversibility), state the "no hardcoded keyword list" constraint, and state the escalate-on-uncertainty rule.

#### Scenario: Skill source reflects classification
- **GIVEN** the shipped `src/kit/skills/fab-operator.md`
- **WHEN** a reader inspects §5 Answer Model rule 4
- **THEN** the rule SHALL describe Routine vs Strategic classification with all four signals named
- **AND** SHALL NOT reference a keyword list or sentinel protocol
- **AND** SHALL state that uncertainty escalates

## Operator: Idle Auto-Default on Strategic Escalations (§5)

### Requirement: 30-minute idle threshold

When the operator has escalated a Strategic numbered-menu prompt per the rule above, it SHALL start a per-prompt idle timer measured in real time. If the prompt remains in the pane capture unchanged for 30 minutes with no user keystrokes and no terminal-state change, the operator SHALL auto-answer the prompt and log the action with a distinct log format (see the distinct-log-line requirement below).

The threshold MUST be hardcoded at 30 minutes. No `.fab-operator.yaml` field, no per-change override, no environment variable SHALL expose it.

#### Scenario: Idle exceeds threshold, auto-default fires
- **GIVEN** a Strategic escalation was logged at time T
- **AND** the pane capture has not changed since T
- **AND** the user has not sent keystrokes to the pane
- **WHEN** the operator tick at time T + 30 minutes runs
- **THEN** the operator SHALL send the auto-default answer to the pane
- **AND** SHALL log the action using the `auto-defaulted after 30m idle` format

#### Scenario: User responds before threshold
- **GIVEN** a Strategic escalation was logged at time T
- **WHEN** the user sends keystrokes that change the prompt before T + 30 minutes
- **THEN** the idle timer SHALL NOT fire
- **AND** no auto-default log entry SHALL be written

### Requirement: Idle clock resets on terminal state change

The idle timer SHALL reset whenever the pane's terminal capture changes in any way — new content appended by the agent, user keystrokes typed into the pane, the prompt's own redraw. The timer is a watchdog on pane-idle-ness, not on escalation-open-ness.

#### Scenario: Agent emits output mid-wait
- **GIVEN** a Strategic escalation was logged at time T
- **AND** the agent appends a status line to the pane at time T + 20 minutes (without answering the prompt)
- **WHEN** the operator's next tick runs
- **THEN** the idle timer SHALL be reset to time T + 20 minutes
- **AND** the 30-minute countdown SHALL restart from that point

#### Scenario: User types partial keystrokes
- **GIVEN** a Strategic escalation was logged at time T
- **AND** the user types keystrokes into the pane at time T + 10 minutes that alter the prompt display
- **WHEN** the operator's next tick runs
- **THEN** the idle timer SHALL be reset to time T + 10 minutes

### Requirement: Auto-default answer selection

When the idle threshold fires, the operator SHALL select the answer by the following priority:

1. If the prompt text visibly states a default (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), the operator SHALL send that stated default.
2. Otherwise, the operator SHALL send `1`.

This matches §5 rule 4's existing "first/default" semantics for routine menus.

#### Scenario: Prompt states a default
- **GIVEN** a Strategic escalation on a prompt containing `(default: 2)`
- **WHEN** the idle threshold fires
- **THEN** the operator SHALL send `2`
- **AND** the log entry's `{answer}` field SHALL be `2`

#### Scenario: Prompt has no stated default
- **GIVEN** a Strategic escalation on a prompt with no visible default marker
- **WHEN** the idle threshold fires
- **THEN** the operator SHALL send `1`
- **AND** the log entry's `{answer}` field SHALL be `1`

### Requirement: Scope is strategic-classification escalations only

The idle auto-default SHALL apply only to escalations produced by §5 rule 4's Strategic classification path. Escalations produced by §5 rule 6 ("cannot determine keystrokes") MUST NOT trigger the idle auto-default — the operator does not know what the correct keystrokes are, so sending `1` or the stated default would emit nonsense into the pane.

#### Scenario: Rule-6 escalation does not auto-default
- **GIVEN** the operator escalated a prompt via rule 6 (cannot determine keystrokes)
- **WHEN** the prompt remains idle beyond 30 minutes
- **THEN** the operator SHALL NOT send any auto-default answer
- **AND** the escalation SHALL remain open pending user action

### Requirement: Distinct log line format

The auto-default action SHALL log with a format distinct from the normal `auto-answered` line so that after-action review tooling can distinguish confidently-auto-answered decisions from decisions the operator took because the user never returned.

The exact format:

```
{change}: auto-defaulted after 30m idle: '{summary}' → {answer}
```

Where `{summary}` is the same short question summary already used by existing §5 log lines, and `{answer}` is the keystroke sent.

#### Scenario: Log line format
- **GIVEN** a Strategic escalation for change `hin2` on a rework prompt summarized as `rework scope`
- **AND** the idle threshold fires with selected answer `1`
- **WHEN** the operator writes the log entry
- **THEN** the entry SHALL be exactly `hin2: auto-defaulted after 30m idle: 'rework scope' → 1`
- **AND** the entry SHALL NOT begin with `auto-answered`

## Spec Mirroring and Documentation

### Requirement: SPEC-fab-operator.md mirrors behavior

Per `fab/project/constitution.md` §Additional Constraints, the behavior added to `src/kit/skills/fab-operator.md` SHALL be mirrored in `docs/specs/skills/SPEC-fab-operator.md` within the same change. The mirrored spec MUST describe both classification (Routine/Strategic) and the idle auto-default behavior with enough detail that a reader of the spec alone can reconstruct the operator's rule 4 behavior.

#### Scenario: SPEC file mirrors the skill source
- **GIVEN** the shipped `docs/specs/skills/SPEC-fab-operator.md`
- **WHEN** a reader searches for "classification" and "auto-defaulted"
- **THEN** both terms SHALL be documented with their mechanics
- **AND** the spec SHALL reference the 30-minute threshold and the rule-6 exclusion

### Requirement: Memory hydrate updates execution-skills.md

On hydrate, the change SHALL update `docs/memory/fab-workflow/execution-skills.md` to reflect the operator's new classification and auto-default behavior. Hydrate's diff scan is the authoritative confirmation of which memory file needs editing; if the scan identifies additional or different memory files that reference §5 Answer Model, those SHALL be updated instead or in addition.

#### Scenario: Hydrate edits memory
- **GIVEN** the apply stage has completed
- **WHEN** hydrate runs
- **THEN** `docs/memory/fab-workflow/execution-skills.md` (or the file(s) identified by hydrate's diff scan) SHALL describe the Routine/Strategic classification and the 30-minute idle auto-default

## Backlog Cleanup

### Requirement: Mark backlog entries done on hydrate

On hydrate, `fab/backlog.md` SHALL have both `[hin2]` and `[i1l6]` entries transitioned from `[ ]` to `[x]`. The entries' text MUST remain unchanged — only the checkbox flips.

#### Scenario: Backlog updated
- **GIVEN** the hydrate stage has completed
- **WHEN** `fab/backlog.md` is read
- **THEN** the line beginning `- [x] [hin2]` SHALL exist
- **AND** the line beginning `- [x] [i1l6]` SHALL exist

## Design Decisions

1. **Principle-based classification, not keyword-based or marker-based**
   - *Why*: The operator already has terminal capture access and LLM reasoning capacity. A keyword list is brittle, fails on novel prompt text, and requires ongoing maintenance as new agents and CLIs emit new prompt shapes. A sentinel protocol couples the operator to every skill's surface area and cannot cover prompts from Claude Code itself or third-party CLIs.
   - *Rejected*: (a) Hardcoded keyword list — brittle, high-maintenance, inconsistent with SRAD and question-detection's model-judged approach elsewhere. (b) Agent-side `[STRATEGIC]` sentinel — requires cooperation from every skill and external tool, fails on CC-native and third-party prompts.

2. **Escalate on classification uncertainty**
   - *Why*: False-negative strategic (auto-answering a strategic prompt) commits the queue to an uninspected direction — expensive to unwind. False-positive strategic (escalating a routine) costs a user nudge recovered by the 30-minute auto-default. Asymmetric costs require the safer default.
   - *Rejected*: Best-guess-then-answer — accepts unbounded downside (wrong queue direction) to save a user nudge, reversing the cost structure.

3. **Hardcoded 30-minute idle threshold**
   - *Why*: 30 minutes is a defensible calibration for the "user is asleep or in a meeting" scenario that motivates the feature. Shorter risks auto-defaulting mid-reply. Longer defeats the feature. A config knob is more surface area than the problem warrants and invites per-project bikeshedding.
   - *Rejected*: Configurable threshold (`.fab-operator.yaml`, per-change override) — added surface area for marginal benefit; the one threshold serves the single motivating scenario well.

4. **Idle auto-default excludes rule-6 escalations**
   - *Why*: Rule-6 escalation means the operator does not know what keystrokes to send. Auto-defaulting to `1` or the stated default would emit nonsense into the pane and corrupt the agent's state.
   - *Rejected*: Uniform 30-minute auto-default across all escalation types — conflates two distinct failure modes.

5. **Distinct log line format (`auto-defaulted` vs `auto-answered`)**
   - *Why*: After-action review needs to distinguish decisions the operator took confidently (classified Routine) from decisions it took because the user never returned. Grep-friendly distinction enables simple tooling.
   - *Rejected*: Reusing the `auto-answered` line — loses the distinction, muddies audit trails.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship Part 1 (classification) and Part 2 (idle auto-default) together as one change. | Confirmed from intake #1. User scoped them as coupled; shipping either alone produces incoherent behavior. | S:95 R:80 A:95 D:95 |
| 2 | Certain | Classification is principle-based / LLM-judged, not keyword-driven and not sentinel-driven. | Confirmed from intake #2. Clarified — user confirmed during /fab-clarify. | S:95 R:70 A:95 D:100 |
| 3 | Certain | Idle threshold is hardcoded 30 minutes. No config knob, no per-change override. | Confirmed from intake #3. | S:100 R:60 A:90 D:95 |
| 4 | Certain | Auto-default log line format is `{change}: auto-defaulted after 30m idle: '{summary}' → {answer}`. | Confirmed from intake #4 and discussed in conversation. Exact string form committed in this spec. | S:100 R:90 A:100 D:100 |
| 5 | Certain | Idle auto-default does NOT apply to rule-6 "cannot determine keystrokes" escalations. | Confirmed from intake #5. | S:100 R:85 A:95 D:100 |
| 6 | Certain | Change type is `feat` (skill behavior addition). | Confirmed from intake #6 and corrected in .status.yaml during /fab-clarify (keyword inference initially picked `fix` due to body text; user override stands). | S:100 R:100 A:100 D:100 |
| 7 | Certain | `docs/specs/skills/SPEC-fab-operator.md` MUST be updated in the same change as `src/kit/skills/fab-operator.md`. | Confirmed from intake #7. Constitution §Additional Constraints. | S:100 R:100 A:100 D:100 |
| 8 | Certain | Backlog IDs `hin2` and `i1l6` are marked done in `fab/backlog.md` at hydrate. | Confirmed from intake #8. | S:100 R:100 A:100 D:100 |
| 9 | Certain | Auto-default picks the agent's stated default if visible in the prompt; otherwise option `1`. | Confirmed from intake #9. Clarified — user confirmed during bulk confirm. Priority order codified in this spec's auto-default-answer-selection requirement. | S:95 R:75 A:85 D:85 |
| 10 | Certain | Classification-uncertain prompts are treated as Strategic (escalate). | Confirmed from intake #10. Clarified — user confirmed during bulk confirm. Codified in this spec as a MUST rule, not advisory. | S:95 R:70 A:85 D:75 |
| 11 | Certain | Idle clock resets on any terminal-state change in the pane (new content or user keystrokes). | Confirmed from intake #11. Clarified — user confirmed during bulk confirm. Codified as scenarios in this spec. | S:95 R:70 A:85 D:80 |
| 12 | Certain | No new fields in `.fab-operator.yaml` schema. | Confirmed from intake #12. | S:95 R:80 A:90 D:90 |
| 13 | Certain | No new CLI subcommand or `fab` Go binary change. | Confirmed from intake #13. | S:95 R:85 A:95 D:95 |
| 14 | Certain | Drop the "~30 chars per option" heuristic from the skill text entirely — option text length remains a qualitative signal only. | Confirmed from intake #14 (Tentative → Certain). Clarified — user confirmed after explanation during /fab-clarify; option-length stays qualitative in the spec text. | S:95 R:70 A:75 D:60 |
| 15 | Certain | Memory hydrate target remains TBD; hydrate's diff scan confirms. | Confirmed from intake #15 (Tentative → Certain). Clarified — user confirmed after explanation. Spec's memory-hydrate requirement defers target resolution to hydrate. | S:95 R:80 A:65 D:55 |
| 16 | Certain | The four classifier signals are: option text length, semantic distinctness of options, surrounding agent context, and reversibility of the choice. | Derived from intake prose and codified as an enumerated list in this spec's rule-4 revision requirement. | S:100 R:70 A:90 D:85 |
| 17 | Certain | Idle timer is a per-prompt real-time wall clock measured from escalation log time. | Derived from intake ("the 30-minute clock starts when the strategic escalation is logged"). Codified as GIVEN/WHEN/THEN in this spec. | S:90 R:80 A:90 D:85 |
| 18 | Confident | Idle-timer state does not need to persist across `/fab-operator` restarts — escalation-time is recoverable from existing session log. | Not discussed; follows from assumption 12 (no schema changes). If operator restarts mid-wait, existing logs contain the escalation timestamp, so the timer can be reconstructed from log history at startup. Apply-stage implementation confirms. | S:70 R:75 A:80 D:80 |

18 assumptions (17 certain, 1 confident, 0 tentative, 0 unresolved).
