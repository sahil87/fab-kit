# Proposal: fab-clarify Dual Modes + fab-ff Clarify Checkpoints

**Change**: 260207-m3qf-clarify-dual-modes
**Created**: 2026-02-07
**Status**: Draft

## Why

fab-clarify currently operates in a single autonomous mode where the agent resolves gaps itself and only asks the user as a last resort. For developer tooling where the cost of wrong assumptions is high, the user should be the authority on ambiguity resolution. Simultaneously, fab-ff skips clarification entirely — gaps in each artifact compound downstream, producing tasks built on unverified assumptions.

This change introduces dual modes for both fab-clarify (interactive vs autonomous) and fab-ff (checkpoint vs full-auto), giving users control when they want it and speed when they trust the agent.

## What Changes

- **fab-clarify gains two modes**:
  - `suggest` (default, when user calls `/fab:clarify`) — SpecKit-inspired interactive mode with systematic taxonomy scan, structured multiple-choice/short-answer questions (max 5 per invocation), incremental artifact updates after each answer, and a coverage report at completion
  - `auto` (internal, when called by fab-ff) — current autonomous behavior preserved, returns a machine-readable result (resolved/blocking/non-blocking counts) so fab-ff can decide whether to continue or bail

- **fab-ff gains two modes**:
  - Default `/fab:ff` — interleaves auto-clarify between stage generations (`spec → auto-clarify → plan → auto-clarify → tasks → auto-clarify`). Stops and reports if auto-clarify finds blocking issues the agent can't resolve autonomously. Resumable — re-running `/fab:ff` after `/fab:clarify` picks up where it left off
  - Full-auto `/fab:ff --auto` — same pipeline but never stops. Makes best-guess decisions on blockers, marks them with `<!-- auto-guess: {description} -->` markers, and warns the user in output

- **Suggest mode specifics** (borrowed from SpecKit, adapted for fab):
  - Stage-scoped taxonomy scan (not a fixed 10-category list — categories vary by proposal/specs/plan/tasks)
  - One question at a time, never reveals future queued questions
  - Each question has a recommendation with reasoning + options table (multiple-choice) or suggested answer (short-answer)
  - User accepts with "yes"/"recommended" or picks an option/provides own answer
  - `## Clarifications > ### Session YYYY-MM-DD` audit trail with `Q: → A:` bullets
  - Coverage summary table at completion (Resolved / Clear / Deferred / Outstanding)
  - Early termination with "done" / "good" / "no more"

- **fab-ff default mode stop behavior**:
  - Reports blocking issues found by auto-clarify
  - Suggests: `Run /fab:clarify to resolve these, then /fab:ff to resume.`
  - Updates `.status.yaml` — completed stages marked `done`, blocked stage stays at current status

## Affected Docs

### New Docs
- `fab-workflow/clarify`: fab-clarify skill — dual modes (suggest/auto), taxonomy scan, structured questions, coverage reports

### Modified Docs
- `fab-workflow/index.md`: Add clarify doc entry

### Removed Docs
(none)

## Impact

- **`fab/.kit/skills/fab-clarify.md`** — Major rewrite: suggest mode as default, auto mode as internal contract, taxonomy scan per stage, structured question format, completion report, clarifications audit trail
- **`fab/.kit/skills/fab-ff.md`** — Add `--auto` flag, interleave auto-clarify between stage generations, bail-on-blocker logic for default mode, guess-warning output for auto mode
- **`fab/.kit/skills/_context.md`** — Update Next Steps table for `/fab:ff --auto` variant
- **No changes to**: `/fab:continue`, `/fab:new`, `/fab:apply`, `/fab:review`, `/fab:archive`, templates, `.status.yaml` schema

## Open Questions

(none — design decisions resolved during proposal discussion)

### Design Decisions (for reference)

1. **Mode selection by call context, not flags**: `/fab:clarify` = suggest, internal fab-ff call = auto. No `--suggest`/`--auto` flags on clarify — avoids a confusing third path with no clear use case.

2. **Max 5 questions per invocation, not unlimited**: Beyond 5, diminishing returns and user fatigue. Run `/fab:clarify` again if needed (it's idempotent, taxonomy scan reprioritizes).

3. **No auto-clarify in `/fab:continue`**: Continue is the deliberate one-stage-at-a-time path. Users call `/fab:clarify` manually between stages. Injecting auto-clarify would slow it without being asked.

4. **`<!-- auto-guess: ... -->` markers in fab-ff --auto**: Makes guesses visible and reviewable. `/fab:review` can flag them. Subsequent `/fab:clarify` (suggest mode) can find and resolve them interactively.
