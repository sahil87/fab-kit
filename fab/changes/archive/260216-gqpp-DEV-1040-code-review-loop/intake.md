# Intake: DEV-1040 Code Review Loop

**Change**: 260216-gqpp-DEV-1040-code-review-loop
**Created**: 2026-02-16
**Status**: Draft

## Origin

> DEV-1040 Create a code review loop between apply and review stages, that uses review sub-agents, prioritizes review comments, and moves back to apply.
>
> Clarifications: It's important to prioritize the comments. It's not necessary to implement all comments.

Linear issue [DEV-1040](https://linear.app/weaver-ai/issue/DEV-1040/create-code-review-loop-between-apply-and-review-stages): "Create code review loop between apply and review stages" — assigned to Sahil Ahuja, project FabKit: AI Engg Workflow, status Backlog.

## Why

The current review behavior in `/fab-continue` runs inline within the same agent context that performed the apply. This creates two problems:

1. **No fresh perspective**: The applying agent reviews its own work, missing issues that a separate reviewer would catch. The same context biases that led to an implementation choice also bias the review.
2. **No review comment triage**: Review validation runs as part of a general-purpose pipeline step. There's no mechanism to prioritize review comments by severity or impact, nor to selectively act on the most important findings while deferring or skipping low-value ones.
3. **Manual rework loop**: When review fails in `/fab-continue`, the user must manually select a rework option and re-invoke. In `/fab-fff`, autonomous rework exists but still runs in the same context. Neither approach creates a true apply-review-apply loop with clean separation.

A dedicated review sub-agent with an automated loop back to apply would improve review quality through fresh-context evaluation and enable tighter, faster rework cycles.

## What Changes

### Review Sub-Agent Integration

Introduce a review sub-agent that runs review validation in a separate agent context. The sub-agent is not prescriptive about implementation — the orchestrating LLM should use whichever review agent is available in its environment (e.g., a `code-review` skill, a general-purpose sub-agent with review instructions, or any equivalent). The sub-agent gets:
- The spec, tasks, checklist, and source files as context
- Review-specific instructions prioritizing validation checks
- Ability to produce structured review output (pass/fail with findings)
<!-- clarified: Review agent selection is not prescriptive — LLM uses whatever review agent is available -->

### Apply-Review Loop

After apply completes, instead of transitioning directly to review-done-or-rework:
1. Spawn the review sub-agent
2. Sub-agent performs validation checks and returns structured findings
3. On pass: advance to hydrate readiness (normal flow)
4. On failure: automatically loop back to apply — the applying agent receives the review findings, fixes the identified issues, and the review sub-agent is re-spawned
<!-- clarified: Auto-loop applies to both fab-fff and fab-ff; fab-continue keeps manual rework — per user direction -->

### Review Comment Prioritization

The review sub-agent produces prioritized review comments — structured findings ranked by severity/impact. The applying agent then triages these comments:
- **Must-fix**: Spec mismatches, failing tests, checklist violations — always addressed
- **Should-fix**: Code quality issues, pattern inconsistencies — addressed when clear and low-effort
- **Nice-to-have**: Style suggestions, minor improvements — may be skipped

Not all review comments need to be implemented. The applying agent uses judgment to determine which comments warrant rework and which can be acknowledged but deferred. This prevents infinite rework loops over diminishing-return suggestions.
<!-- clarified: Three-tier priority scheme (must-fix/should-fix/nice-to-have) confirmed by user -->

### Retry Cap and Termination

The apply-review loop has a bounded retry count before escalating to the user.
<!-- assumed: Align with fab-fff's existing 3-cycle retry cap — consistent with established pattern; applies to both fab-ff and fab-fff -->

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Add review sub-agent behavior, apply-review loop mechanism, review comment prioritization
- `fab-workflow/model-tiers`: (modify) Add tier classification for the review sub-agent

## Impact

- **`/fab-fff`** — Autonomous rework loop replaced with sub-agent-based apply-review loop (bounded retry)
- **`/fab-ff`** — Gains apply-review loop with sub-agent (bounded retry); falls back to interactive rework options when retry cap is hit
<!-- clarified: fab-ff auto-loops first, then interactive fallback on cap exhaustion -->
- **`/fab-continue`** — Review uses sub-agent for fresh-context evaluation; rework remains manual (user-directed)
<!-- clarified: fab-continue uses review sub-agent but keeps manual rework -->
- **`docs/specs/skills.md`** — Review behavior section needs updating
- **`fab/.kit/skills/fab-continue/`** — Primary implementation target for review dispatch
- **Review agent** — Not prescriptive; LLM uses whatever review agent is available in its environment

## Open Questions

*(All resolved — see Clarifications below)*

## Clarifications

### Session 2026-02-16

1. **Review agent selection**: Not prescriptive — the LLM should use whichever review agent is available in its environment. Different agents will be available to different LLMs.
2. **Loop scope**: Auto-loop applies to both `/fab-fff` and `/fab-ff`. Not `/fab-continue` — manual rework there.
3. **fab-continue review**: Uses the review sub-agent for fresh-context evaluation, but rework remains manual (user-directed, same interactive flow as today).
4. **fab-ff loop behavior**: Auto-loops with bounded retry (same cap as fab-fff). Falls back to interactive rework options when retry cap is hit, preserving fab-ff's semi-interactive character.
5. **Priority tiers**: Three-tier scheme confirmed (must-fix / should-fix / nice-to-have).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Retry cap aligns with fab-fff's 3-cycle cap | Established pattern in autonomous rework; highly reversible | S:40 R:80 A:75 D:80 |
| 2 | Confident | Review sub-agent uses capable model tier | Review requires deep reasoning and code analysis per model-tiers criteria | S:20 R:85 A:70 D:75 |
| 3 | Certain | Auto-loop applies to fab-fff and fab-ff; fab-continue keeps manual rework | User explicitly directed: loop in fab-fff and fab-ff, manual in fab-continue | S:95 R:45 A:95 D:95 |
| 4 | Certain | Review comments are prioritized by severity; not all must be implemented | User clarified: "prioritize the comments" and "not necessary to implement all comments" | S:95 R:80 A:95 D:95 |
| 5 | Certain | Three-tier priority scheme (must-fix / should-fix / nice-to-have) | User confirmed three tiers as right granularity | S:95 R:75 A:95 D:95 |
| 6 | Certain | Sub-agent replaces existing inline review for all three skills | All skills (fab-continue, fab-ff, fab-fff) use sub-agent; running both inline and sub-agent would be redundant | S:80 R:35 A:80 D:80 |
| 7 | Certain | Review agent is not prescriptive — LLM uses whatever review agent is available | User explicitly directed: don't be prescriptive, different agents available to different LLMs | S:95 R:80 A:95 D:95 |
| 8 | Certain | fab-continue uses sub-agent but keeps manual rework | User confirmed: fresh-context evaluation via sub-agent, rework options presented manually | S:95 R:50 A:95 D:95 |
| 9 | Certain | fab-ff auto-loops with bounded retry, falls back to interactive rework on cap exhaustion | User confirmed: preserves fab-ff's semi-interactive character | S:95 R:50 A:95 D:95 |

9 assumptions (7 certain, 2 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
