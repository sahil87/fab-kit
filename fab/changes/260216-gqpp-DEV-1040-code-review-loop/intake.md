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

Introduce a review sub-agent that runs review validation in a separate agent context, spawned via the Task tool. The sub-agent gets:
- The spec, tasks, checklist, and source files as context
- Review-specific instructions prioritizing validation checks
- Ability to produce structured review output (pass/fail with findings)

### Apply-Review Loop

After apply completes, instead of transitioning directly to review-done-or-rework:
1. Spawn the review sub-agent
2. Sub-agent performs validation checks and returns structured findings
3. On pass: advance to hydrate readiness (normal flow)
4. On failure: automatically loop back to apply — the applying agent receives the review findings, fixes the identified issues, and the review sub-agent is re-spawned
<!-- assumed: Auto-loop only in fab-fff autonomous mode; fab-continue still presents interactive rework options — aligns with existing autonomous vs interactive distinction -->

### Review Comment Prioritization

The review sub-agent produces prioritized review comments — structured findings ranked by severity/impact. The applying agent then triages these comments:
- **Must-fix**: Spec mismatches, failing tests, checklist violations — always addressed
- **Should-fix**: Code quality issues, pattern inconsistencies — addressed when clear and low-effort
- **Nice-to-have**: Style suggestions, minor improvements — may be skipped

Not all review comments need to be implemented. The applying agent uses judgment to determine which comments warrant rework and which can be acknowledged but deferred. This prevents infinite rework loops over diminishing-return suggestions.
<!-- assumed: Three-tier priority scheme (must/should/nice-to-have) — natural severity levels; exact tier names and criteria can be refined -->

### Retry Cap and Termination

The apply-review loop has a bounded retry count before escalating to the user.
<!-- assumed: Align with fab-fff's existing 3-cycle retry cap — consistent with established pattern -->

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Add review sub-agent behavior, apply-review loop mechanism, review comment prioritization
- `fab-workflow/model-tiers`: (modify) Add tier classification for the review sub-agent

## Impact

- **`/fab-continue`** — Review behavior gains sub-agent spawning; interactive rework flow may be augmented with sub-agent findings
- **`/fab-fff`** — Autonomous rework loop replaced/enhanced with sub-agent-based review loop
- **`/fab-ff`** — Interactive rework on failure would also use sub-agent review
- **`docs/specs/skills.md`** — Review behavior section needs updating
- **`fab/.kit/skills/fab-continue/`** — Primary implementation target for review dispatch
- **Existing `code-review` agent** — Potential reuse or alignment with the review sub-agent

## Open Questions

- Should the review sub-agent reuse the existing `code-review:code-review` skill/agent, or be a new fab-specific review agent?
- Does the loop mechanism apply only to `/fab-fff` (autonomous) or also change `/fab-continue` interactive flow?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Retry cap aligns with fab-fff's 3-cycle cap | Established pattern in autonomous rework; highly reversible | S:40 R:80 A:75 D:80 |
| 2 | Confident | Review sub-agent uses capable model tier | Review requires deep reasoning and code analysis per model-tiers criteria | S:20 R:85 A:70 D:75 |
| 3 | Tentative | Auto-loop only in fab-fff autonomous mode; fab-continue keeps interactive rework | Aligns with existing autonomous vs interactive distinction, but user may want the loop in both modes | S:50 R:45 A:55 D:45 |
| 4 | Certain | Review comments are prioritized by severity; not all must be implemented | User clarified: "prioritize the comments" and "not necessary to implement all comments" | S:95 R:80 A:95 D:95 |
| 5 | Tentative | Three-tier priority scheme (must-fix / should-fix / nice-to-have) | Natural severity levels for review comments; exact tier names and criteria can be refined | S:55 R:75 A:60 D:50 |
| 6 | Tentative | Loop replaces existing inline review, not augments | Running both inline and sub-agent review would be redundant; replacement is cleaner | S:45 R:35 A:50 D:40 |
| 7 | Tentative | Review sub-agent is fab-specific, not a reuse of code-review skill | fab review has specific validation checks (checklist, spec match, memory drift) beyond general code review | S:55 R:40 A:60 D:50 |

7 assumptions (1 certain, 2 confident, 4 tentative, 0 unresolved). Run /fab-clarify to review.
