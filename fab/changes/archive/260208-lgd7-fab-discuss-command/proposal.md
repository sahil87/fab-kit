# Proposal: Add fab-discuss Command and Confidence Scoring to fab-new

**Change**: 260208-lgd7-fab-discuss-command
**Created**: 2026-02-08
**Status**: Draft

## Why

The current workflow has two gaps:

1. **No conversational entry point.** `/fab-new` is a one-shot capture — great for clear ideas, but it doesn't help you explore whether a change is even needed, evaluate alternatives, or iteratively sharpen a vague idea into a solid proposal. The pattern of discussing an idea back-and-forth until it crystallizes should be a first-class skill.

2. **No confidence scoring on `/fab-new`.** The SRAD confidence scoring framework exists in `_context.md` and the `status.yaml` template includes the `confidence` block, but `/fab-new` doesn't actually compute and write it. This means proposals created via `/fab-new` lack the score needed by `/fab-fff`'s gate (>= 3.0), requiring an extra `/fab-clarify` step just to populate it.

Together, these changes create a path where: discuss idea → solid proposal with high confidence → `/fab-fff` straight through.

## What Changes

- **New `/fab-discuss` command**: A conversational skill for developing proposals through discussion. Two modes:
  - **From scratch**: Start with a vague idea or description. Runs gap analysis ("is this even needed?"), walks through clarifying questions in free-form conversation, and outputs a solid `proposal.md` in a new change folder.
  - **On existing proposal**: Takes an existing change name, loads its `proposal.md`, and refines it through discussion — resolving tentative/unresolved decisions, sharpening scope, driving confidence score up.
- **Does not switch active change**: Unlike `/fab-new`, `fab-discuss` creates the change folder but does NOT update `fab/current`. The user must `/fab-switch` to it explicitly. This keeps current work context undisturbed.
- **No git integration**: `fab-discuss` skips branch creation entirely — branching belongs to the "I'm working on this now" moment (`/fab-switch` or `/fab-new`), not the "let's think about this" moment.
<!-- clarified: fab-discuss skips git integration — confirmed by user, branching deferred to fab-switch or fab-new -->
- **Confidence scoring goal**: After a thorough discussion, the confidence score should be high enough (>= 3.0) to go straight to `/fab-fff`. The skill explicitly tracks and reports score progression during the conversation.
- **Conversation termination**: The discussion ends when confidence score >= 3.0 AND the user signals satisfaction (e.g., "looks good", "done"). When the score crosses the threshold, fab-discuss proactively suggests wrapping up — but the user always has the last word. The user can also end early at any time.
<!-- clarified: conversation termination uses score threshold + user signal — user confirmed option 1 -->
- **Context-driven mode selection**: `fab-discuss` uses the active change (`fab/current`) to determine its mode. If an active change exists, it defaults to refining that change's proposal. If the user's description seems significantly different from the active change's scope, `fab-discuss` confirms whether this is a new change. If no active change exists, it starts a new change from scratch. No special arguments needed — just `/fab-discuss <description>`.
<!-- clarified: mode selection uses active change context rather than argument-based auto-detection -->
- **Modify `/fab-new`**: Add confidence score computation (Step 8 in fab-new.md already describes this but the `.status.yaml` output in Step 5 doesn't include the confidence block populated with actual counts). Ensure fab-new writes the computed `confidence` block to `.status.yaml` after proposal generation.
- **Update `_context.md`**: Add `/fab-discuss` to the SRAD skill autonomy table and update the Next Steps lookup table.

## Affected Docs

### New Docs
- (none — fab-discuss will be documented via updates to existing docs)

### Modified Docs
- `fab-workflow/planning-skills.md`: Add `/fab-discuss` documentation alongside `/fab-new`
- `fab-workflow/change-lifecycle.md`: Add the discuss → fff path as an alternative entry point

### Removed Docs
(none)

## Impact

### New files
- `fab/.kit/skills/fab-discuss.md` — The new skill definition
- `.claude/skills/fab-discuss/SKILL.md` — Symlink for Claude Code skill registration

### Modified files
- `fab/.kit/skills/fab-new.md` — Ensure confidence scoring is wired through (Step 8 already exists in the spec but verify `.status.yaml` init includes confidence block)
- `fab/.kit/skills/_context.md` — Add `fab-discuss` to SRAD skill autonomy table, update Next Steps table

### Relationship to existing proposal
There is an earlier proposal at `fab/changes/260208-r4w1-add-fab-discuss/` that covers the `fab-discuss` skill design. This change supersedes it with broader scope (also includes fab-new confidence scoring and _context.md updates) and more concrete design decisions from the user's detailed description.

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | fab-discuss skips git integration entirely | Branching belongs to the "working on this" moment, not the "thinking about this" moment — user can branch later via fab-switch or fab-new |
| 2 | Confident | Conversational style with no fixed question cap | Unlike fab-clarify (5-question cap) or fab-new (3-question cap), fab-discuss is meant for thorough exploration — artificial caps would undermine the purpose |
| 3 | Confident | Context-driven mode selection via active change | User clarified: use fab/current to determine mode — active change = refine, no active change = new, divergent description = confirm |

3 assumptions made (3 confident, 0 tentative).

## Clarifications

### Session 2026-02-08

- **Q**: How should fab-discuss distinguish its two modes (new vs existing)?
  **A**: Use active change context — if fab/current points to a change, refine it; if no active change, start new; if description diverges significantly from active change, confirm with user.
- **Q**: Should fab-discuss skip git integration?
  **A**: Accepted recommendation: skip git, branching deferred to fab-switch or fab-new.
- **Q**: When does the discussion end and the proposal finalize?
  **A**: Accepted recommendation: suggest wrapping up when confidence >= 3.0, but only finalize when user agrees.
