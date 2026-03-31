# Intake: Operator Never-Ask Monitor Fix

**Change**: 260331-mvhj-operator-never-ask-monitor
**Created**: 2026-03-31
**Status**: Draft

## Origin

> Bug fix: operator asks "Want me to monitor it?" after spawning agents instead of automatically enrolling them. The principle in section 1 says "Never ask whether to monitor a spawned agent" but the spawning sequence in section 6 does not reinforce this strongly enough. Add explicit never-ask language to the Spawning an Agent subsection so the LLM reliably auto-enrolls without prompting.

One-shot input. The user identified a behavioral gap where the operator LLM sometimes asks the user whether to monitor a freshly spawned agent, despite a principle explicitly prohibiting this.

## Why

The operator skill (`fab-operator7.md`) contains a clear principle at line 20:

> **Automate the routine.** … Never ask whether to monitor a spawned agent — if the operator spawned it, monitor it.

However, the "Spawning an Agent" subsection (§6, lines 297–346) only lists "Enroll in monitored set" as step 4 of the spawn sequence without restating the never-ask constraint. LLMs are sensitive to proximity — a principle stated once in an introductory section may not reliably influence behavior deep in a procedural sequence. The result is that the operator sometimes asks "Want me to monitor it?" instead of silently auto-enrolling.

If unfixed, users will continue receiving unnecessary prompts that break the autonomous operator experience and require manual intervention for a routine operation.

## What Changes

### `fab/.kit/skills/fab-operator7.md` — Spawning an Agent subsection

Add explicit never-ask reinforcement language to the "Spawning an Agent" subsection in §6. Specifically:

1. **Step 4 annotation**: Expand the "Enroll in monitored set" step to include a direct prohibition: enrollment is unconditional and silent — never prompt the user about whether to monitor.

2. **Subsection-level admonition**: Add a bold callout or blockquote immediately before or after the spawn sequence steps reinforcing that the entire spawn-then-enroll flow is autonomous. Something like:

   > **Auto-enroll is mandatory.** Every spawned agent MUST be enrolled in the monitored set immediately. Do not ask the user whether to monitor — this decision is already made by the act of spawning.

The exact wording should use RFC 2119 language (MUST/SHALL/MUST NOT) consistent with the rest of the skill file.

### `fab/.kit/skills/fab-operator6.md` — Spawning an Agent subsection (if applicable)

Check whether `fab-operator6.md` has a similar spawning subsection that would benefit from the same reinforcement. If so, apply the same pattern. (The gap analysis shows fab-operator6 has a simpler "Spawning an Agent" subsection at lines 253–259.)

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator skill documentation to reflect the never-ask-monitor reinforcement

## Impact

- **`fab/.kit/skills/fab-operator7.md`** — primary change target; the Spawning an Agent subsection in §6
- **`fab/.kit/skills/fab-operator6.md`** — secondary; may need similar reinforcement if it has a spawn sequence
- **`fab/.kit/skills/fab-operator5.md`** — unlikely affected (predecessor version), but should be checked for completeness
- No code changes — this is a skill file (markdown) fix only
- No template, script, or config changes

## Open Questions

- None — the fix is well-scoped: add never-ask reinforcement to the spawn sequence subsection(s).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Primary target is `fab-operator7.md` | User explicitly referenced the principle (section 1) and spawn sequence (section 6), both in operator7 | S:95 R:95 A:95 D:95 |
| 2 | Certain | Fix is markdown-only — no script or Go code changes | The issue is LLM prompt adherence, fixed by strengthening skill file language | S:90 R:95 A:95 D:95 |
| 3 | Confident | `fab-operator6.md` should also get the reinforcement | It has a spawn subsection (lines 253–259) and could exhibit the same drift; applying consistently is low-cost | S:70 R:90 A:80 D:75 |
| 4 | Confident | `fab-operator5.md` does NOT need the change | Operator5 is a predecessor version without the same spawn sequence structure | S:65 R:90 A:75 D:80 |
| 5 | Certain | Use RFC 2119 language (MUST NOT) for the prohibition | Consistent with constitution and existing skill file conventions | S:90 R:95 A:90 D:90 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
