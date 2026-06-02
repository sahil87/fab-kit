# Intake: Operator Numbered-Menu Classification + Idle-Escalation Auto-Default

**Change**: 260422-hin2-operator-strategic-menu-escalation
**Created**: 2026-04-22
**Status**: Draft

## Origin

Two coupled operator backlog items (`hin2` and `i1l6`, both filed 2026-04-22) discussed together and scoped for a single shipping unit because they interlock: `hin2` introduces strategic escalations; `i1l6` prevents those escalations from stalling headless queues. Shipping either alone produces an incoherent behavior — `hin2` alone stalls overnight autopilot runs; `i1l6` alone has nothing to idle-default against (rule 4 already auto-answers uniformly).

User input (verbatim design decisions reached before `/fab-new` was invoked):

> Operator numbered-menu classification + idle-escalation auto-default (resolves backlog hin2 + i1l6).
>
> **Part 1 — hin2**: before applying §5 Answer Model rule 4, classify the numbered prompt. Routine (tool/permission, binary-framed, semantic-synonym options) → auto-answer `1`. Strategic (multi-option, materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach decisions) → escalate to user. Classification uses LLM judgment from signals visible in the captured terminal (option text length, semantic distinctness, surrounding agent context, reversibility). NO hardcoded keyword list, NO agent-side marker protocol — principle-based, model-judged.
>
> **Part 2 — i1l6**: when a strategic numbered prompt has been escalated and remains idle 30 minutes, auto-pick option `1` (or the agent's stated default if visible) and log with a distinct format: `auto-defaulted after 30m idle: '{summary}' → {answer}`. Applies ONLY to strategic-classification escalations from Part 1; rule-6 "cannot determine keystrokes" escalations stay escalated (different failure mode). Threshold hardcoded at 30m — no config knob, no per-change override.

Interaction mode: pre-formed design handed to `/fab-new`. No SRAD clarification round was needed — the user's description is deliberate, unambiguous, and records both parts' scope boundaries explicitly.

## Why

**Problem (Part 1 — hin2).** `src/kit/skills/fab-operator.md` §5 Answer Model rule 4 currently says "Numbered menu → `1` (first/default)". This fires uniformly across prompt categories that have nothing in common:

- *Routine*: Claude Code tool-permission prompts (`1) Yes  2) Yes, and don't ask again  3) No`), binary yes/no wrapped in a menu, or menus whose options are semantic synonyms with identical effect. Picking `1` here is correct and unblocks the agent.
- *Strategic*: agent-surfaced decisions like "scope choice for this rework" (`1) fix code  2) revise tasks  3) revise spec  4) abort`), "PR split" (`1) single PR  2) split by domain  3) split by stage`), or "pipeline shape" (`1) run ff  2) run fff  3) stop at spec`). These are materially different directions — picking `1` silently commits the queue to a direction the user never saw.

Today, the operator auto-answers both categories identically. That trades correctness for throughput in the exact scenarios where correctness matters most (irreversible pipeline decisions, commit content, checklist compliance).

**Problem (Part 2 — i1l6).** Once `hin2` lands, the operator will escalate strategic prompts to the user. If the user is asleep, in meetings, or otherwise away from the terminal, every strategic escalation stops forward progress on that agent — and potentially on agents downstream of it in a dependency chain. Headless autopilot runs (overnight, multi-hour) become unreliable. This trades oversight for throughput when the alternative is zero throughput.

**Why this approach over alternatives.**

- *Hardcoded keyword list for classification (rejected)*: brittle, fails on novel prompt text, requires ongoing maintenance, conflicts with project principle of "principle-based, model-judged" behavior. The operator already has the terminal capture; the model can read it.
- *Agent-side marker protocol (rejected)*: would require every skill to emit a `[STRATEGIC]` sentinel before numbered menus. Couples the operator to skill-surface-area changes; misses prompts from Claude Code itself or third-party CLIs.
- *Config-knob for idle threshold (rejected)*: more surface area than the problem warrants. 30 minutes is a defensible default for the "user is asleep or in a meeting" scenario that motivates the feature; shorter thresholds risk auto-defaulting while the user is mid-reply; longer ones defeat the feature. Per-change overrides compound this.
- *Extend i1l6 to rule-6 escalations (rejected)*: rule-6 triggers when the operator genuinely cannot determine what keystrokes to send. Auto-defaulting to `1` in that case sends garbage to the agent. The two failure modes warrant different handling.

**Why ship together.** Part 2 has no standalone value without Part 1 (there are no strategic escalations to idle-default against). Part 1 without Part 2 regresses headless-run reliability. The pair is one behavior change to §5 Answer Model, not two.

## What Changes

### 1. Classify numbered prompts before applying rule 4 (§5 Answer Model)

Rule 4 today:

```
4. Numbered menu → `1` (first/default)
```

Replace with a two-step evaluation. Before picking `1`, classify the numbered prompt as **Routine** or **Strategic** using LLM judgment over signals visible in the captured terminal.

**Routine → auto-answer `1`.** Characteristics (any one sufficient):

- Tool / permission prompts (Claude Code `Allow Bash:`, `Allow Edit:`, file-access confirmations).
- Binary-framed choices rendered as a numbered menu — e.g., `1) Yes  2) No`, `1) Yes  2) Yes, and don't ask again  3) No`, `1) always  2) never  3) just this time`.
- Prompts whose options are semantic synonyms with the same effect — e.g., `1) proceed  2) continue  3) go ahead`.
- Short option text, low semantic distinctness, reversible outcome.

**Strategic → escalate to user.** Characteristics (any one sufficient):

- Multi-option menu where options represent materially different directions.
- Scope / PR split / pipeline shape / commit organization / spec or approach decisions.
- Long option text, high semantic distinctness, irreversible or expensive-to-undo outcome.
- Surrounding agent context frames the prompt as a decision the user would normally make (e.g., the rework-menu prompt after a failed review).

**Classification mechanics.** The operator reads the terminal capture that already drove question detection (§5 Question Detection step 1) and applies LLM judgment across these signals:

- Option text length — short options correlate with routine; long options correlate with strategic.
- Semantic distinctness — synonymous or near-synonymous options are routine; options pointing at different directions are strategic.
- Surrounding agent context — preceding agent output often signals intent ("Choose a rework scope:" vs. "Allow this tool?").
- Reversibility — prompts that commit the queue to a direction are strategic; prompts whose effect is immediately undoable are routine.

No hardcoded keyword list. No sentinel protocol requiring skill-side changes. The classifier is principle-based and model-judged, consistent with the project's approach elsewhere (SRAD scoring, question detection).

**On ambiguity, escalate.** If the classification itself is uncertain, treat as strategic and escalate. False-positive strategic (escalating a routine) costs a few seconds of user time; false-negative strategic (auto-answering a strategic) can commit the queue to the wrong branch of work.

Revised rule text for §5 Answer Model:

```
4. Numbered menu:
   - Classify the prompt as Routine or Strategic using LLM judgment over the terminal capture.
     Signals: option text length, semantic distinctness of options, surrounding agent context,
     reversibility of the choice. No hardcoded keyword list.
     - Routine (tool/permission, binary-framed, synonymous-option menus) → `1` (first/default).
     - Strategic (multi-option choices representing materially different directions — scope,
       PR split, pipeline shape, commit organization, spec/approach) → escalate to user.
   - On classification uncertainty, treat as Strategic and escalate.
```

### 2. 30-minute idle auto-default on strategic escalations

When a prompt has been escalated as Strategic per rule 4 above and has remained idle for 30 minutes (no terminal state change, no user keystrokes into the pane), the operator auto-picks:

- The agent's stated default if it is visible in the prompt (e.g., `(default: 2)` or `Press enter for 2`), otherwise
- Option `1`.

It then sends that answer and logs with a distinct format:

```
{change}: auto-defaulted after 30m idle: '{summary}' → {answer}
```

The log line is intentionally different from the normal auto-answer line (`auto-answered`) so that after-action review can distinguish between decisions the operator took confidently vs. decisions it took because the user never returned.

**Idle measurement.** The operator already captures the pane on every tick (§4 Tick Behavior). The 30-minute clock starts when the strategic escalation is logged; it is reset only by terminal state change (new content in the pane capture) or by the user sending keystrokes that change the prompt. Tick cadence already supports sub-minute resolution, so no new polling infrastructure is needed.

**Scope constraints (hard):**

- Applies ONLY to strategic-classification escalations from §5 rule 4. Rule-6 "cannot determine keystrokes" escalations stay escalated — that is a different failure mode (the operator doesn't know what to type, so auto-defaulting to `1` would send nonsense).
- Threshold is hardcoded at 30 minutes. No config knob in `.fab-operator.yaml`. No per-change override.
- Log line format is distinct from the normal auto-answer line. After-action review tooling should be able to grep `auto-defaulted` as a separate signal from `auto-answered`.

### 3. Logging additions (§5 Logging)

Append one bullet to the existing Logging list:

```
- Auto-default (after 30m idle on strategic escalation):
  `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`
```

The two existing bullets (`auto-answered`, `can't determine`) are unchanged.

### 4. Spec mirroring (constitutional requirement)

Per `fab/project/constitution.md` §Additional Constraints: "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file." The behavior additions above MUST be mirrored in `docs/specs/skills/SPEC-fab-operator.md` in the same change.

### 5. Backlog cleanup (at hydrate)

On hydrate, mark both backlog entries done in `fab/backlog.md`:

- `[hin2] 2026-04-22: Operator /fab-operator: rule 4 (numbered menu → 1) is ambiguous...` → `[x]`
- `[i1l6] 2026-04-22: Operator /fab-operator: add configurable auto-default-after-N-minutes...` → `[x]`

Note: the i1l6 entry says "configurable per-change (or globally via .fab-operator.yaml)". The shipped behavior is hardcoded 30m — this is a deliberate narrowing of the original backlog scope, recorded here so hydrate's diff-vs-backlog reconciliation is explicit.

## Affected Memory

- `fab-workflow/execution-skills` (modify): if this file documents operator behavior, update the §5 Answer Model summary to reflect classification; otherwise no change. Hydrate will determine the exact file.

> Hydrate-time scan will confirm which memory files reference §5 Answer Model and adjust accordingly. No new memory domain.

## Impact

**Files modified:**

- `src/kit/skills/fab-operator.md` — §5 Answer Model (rule 4 expanded with classification), §5 Logging (new auto-default line). Canonical source — this file is deployed to `.claude/skills/` by `fab sync`; never edit the deployed copy directly.
- `docs/specs/skills/SPEC-fab-operator.md` — mirror the §5 additions per constitutional requirement.
- `fab/backlog.md` — mark `[hin2]` and `[i1l6]` as `[x]` at hydrate.

**Files unchanged:**

- No CLI (`fab` Go binary) changes — this is pure skill-markdown behavior. §Additional Constraints carve-out about `_cli-fab.md` does not apply.
- No config schema changes. `.fab-operator.yaml` schema in §4 is untouched.
- No template changes.
- Other skills are not affected. `/fab-operator` is the sole consumer of §5 Answer Model.

**Dependencies / ordering:** none. Self-contained skill-doc change.

**Risk surface:**

- Classification false-negative (strategic read as routine) → operator silently commits the queue to option `1` on a decision the user should have made. Mitigation: the rule requires escalating on classification uncertainty, so any borderline case defaults to "ask".
- Classification false-positive (routine read as strategic) → operator escalates on a trivial prompt; user sees an extra nudge. Mitigation: after 30m idle, auto-defaults resolve it anyway.
- 30m threshold too long → user returns to find a stalled pane. Mitigation: the threshold is calibrated for the "asleep / in-meeting" scenario that motivates the feature; the user can always intervene sooner.
- 30m threshold too short → auto-defaults while user is mid-reply. Mitigation: the idle clock resets on terminal state change, including user keystrokes.

## Open Questions

- None at intake time — the design decisions are fully specified by the user's description. Spec stage will translate the prose above into GIVEN/WHEN/THEN scenarios and may surface edge cases (e.g., behavior when the pane is killed mid-idle-timer).

## Clarifications

### Session 2026-04-22 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 9 | Confirmed | — |
| 10 | Confirmed | — |
| 11 | Confirmed | — |
| 12 | Confirmed | — |
| 13 | Confirmed | — |
| 14 | Confirmed | After explanation — dropped "~30 chars per option" from skill text; option text length stays as a qualitative signal only. |
| 15 | Confirmed | After explanation — hydrate target remains TBD; hydrate's diff scan will resolve. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship Part 1 (classification) and Part 2 (idle auto-default) together as one change. | Discussed — user explicitly scoped them as "two coupled behavior changes... shipped together in one change". Part 2 has no standalone value; Part 1 alone regresses headless-run reliability. | S:95 R:80 A:95 D:95 |
| 2 | Certain | Classification is principle-based / LLM-judged, not keyword-driven and not sentinel/marker-driven. | Discussed — user explicit: "NO hardcoded keyword list, NO agent-side marker protocol — principle-based, model-judged." | S:100 R:70 A:95 D:100 |
| 3 | Certain | Idle threshold is hardcoded 30 minutes. No config knob, no per-change override. | Discussed — user explicit: "Threshold is hardcoded at 30 minutes — no config knob, no per-change override." Deliberate narrowing vs. backlog i1l6 which suggested configurability. | S:100 R:60 A:90 D:95 |
| 4 | Certain | Auto-default log line format is distinct from normal auto-answer line. | Discussed — user explicit: "Log line format is distinct from the normal auto-answer log line so auto-defaulted decisions are traceable in after-action review." | S:100 R:90 A:100 D:100 |
| 5 | Certain | Idle auto-default does NOT apply to rule-6 "cannot determine keystrokes" escalations. | Discussed — user explicit: "Does NOT apply to rule-6 'cannot determine keystrokes' escalations — those stay escalated (different failure mode)." | S:100 R:85 A:95 D:100 |
| 6 | Certain | Change type is `feat` (skill behavior addition). | Discussed — user explicit: "Change type: feat (skill behavior additions, not a bug fix)." Keyword inference would also pick `feat` via step 7 of §Step 6 (no fix/refactor/docs/test/ci/chore keywords). | S:100 R:100 A:100 D:100 |
| 7 | Certain | `docs/specs/skills/SPEC-fab-operator.md` MUST be updated in the same change as `src/kit/skills/fab-operator.md`. | Constitution §Additional Constraints: "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file." | S:100 R:100 A:100 D:100 |
| 8 | Certain | Backlog IDs `hin2` and `i1l6` are marked done in `fab/backlog.md` at hydrate. | Discussed — user explicit: "on completion (hydrate stage), mark backlog IDs hin2 and i1l6 as done in fab/backlog.md". | S:100 R:100 A:100 D:100 |
| 9 | Certain | Auto-default picks the agent's stated default if visible in the prompt; otherwise option `1`. | Clarified — user confirmed. | S:95 R:75 A:85 D:85 |
| 10 | Certain | Classification-uncertain prompts are treated as Strategic (escalate). | Clarified — user confirmed. | S:95 R:70 A:85 D:75 |
| 11 | Certain | Idle clock resets on any terminal-state change in the pane (new content or user keystrokes), not only on the final answer. | Clarified — user confirmed. | S:95 R:70 A:85 D:80 |
| 12 | Certain | No new fields in `.fab-operator.yaml` schema (§4 `.fab-operator.yaml`). | Clarified — user confirmed. | S:95 R:80 A:90 D:90 |
| 13 | Certain | No new CLI subcommand or `fab` Go binary change. | Clarified — user confirmed. | S:95 R:85 A:95 D:95 |
| 14 | Certain | Drop the "~30 chars per option" heuristic from the skill text entirely — option text length remains a qualitative signal only. | Clarified — user confirmed after explanation. | S:95 R:70 A:75 D:60 |
| 15 | Certain | Memory hydrate target for §5 Answer Model additions is left as TBD; hydrate's diff scan will confirm or redirect. | Clarified — user confirmed after explanation. | S:95 R:80 A:65 D:55 |

15 assumptions (15 certain, 0 confident, 0 tentative, 0 unresolved).
